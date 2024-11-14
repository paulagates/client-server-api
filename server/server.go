package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

const (
	urlBanco = "file:client-server-api.db?cache=shared&_timeout=5000"
)

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqlite", urlBanco)
	if err != nil {
		log.Fatalf("Erro ao abrir o banco de dados: %v", err)
	}
	defer db.Close()
	preparaTabela()

	http.HandleFunc("/cotacao", buscaCotacaoDolar)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func preparaTabela() {
	stmt, err := db.Prepare(`
		CREATE TABLE IF NOT EXISTS cotacoes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			data DATETIME NOT NULL,
			valor FLOAT NOT NULL
		);
	`)
	if err != nil {
		log.Fatalf("Erro ao preparar statement: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec()
	if err != nil {
		log.Fatalf("Erro ao executar statement: %v", err)
	}
}

func buscaCotacaoDolar(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 200*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		log.Printf("Não foi possível fazer a requisição à API. Erro: %v", err)
		http.Error(w, "Não foi possível fazer a requisição à API", http.StatusInternalServerError)
		return
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Não foi possível fazer a requisição à API. Erro: %v", err)
		http.Error(w, "Não foi possível fazer a requisição à API", http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("Erro ao buscar a cotação do dólar. Código de Status: %v", res.StatusCode)
		http.Error(w, "Erro ao buscar a cotação do dólar", http.StatusInternalServerError)
		return
	}

	var response struct {
		Cotacao struct {
			Valor string `json:"bid"`
		} `json:"USDBRL"`
	}
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		log.Printf("Erro ao decodificar o JSON: %v", err)
		http.Error(w, "Erro ao processar resposta", http.StatusInternalServerError)
		return
	}

	bid := response.Cotacao.Valor
	log.Printf("Cotação do dólar: %s", bid)

	dbCtx, dbCancel := context.WithTimeout(r.Context(), 10*time.Millisecond)
	defer dbCancel()

	err = insereCotacao(bid, dbCtx)
	if err != nil {
		log.Printf("Erro ao inserir a cotação no banco de dados. Erro: %v", err)
		http.Error(w, "Erro ao inserir a cotação no banco de dados", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"dolar": bid})
}

func insereCotacao(valor string, ctx context.Context) error {
	stmt, err := db.PrepareContext(ctx, "INSERT INTO cotacoes(data, valor) VALUES(?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(time.Now(), valor)
	if err != nil {
		return err
	}

	return nil
}
