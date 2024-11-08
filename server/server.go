package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	urlBanco = "root:root@tcp(localhost:3306)/client-server-api?charset=utf8mb4&parseTime=True&loc=Local"
)

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("mysql", urlBanco)
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
			id INT(19) PRIMARY KEY AUTO_INCREMENT,
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

	var response map[string]map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		log.Printf("Erro ao decodificar a resposta. Erro: %v", err)
		http.Error(w, "Erro ao decodificar a resposta", http.StatusInternalServerError)
		return
	}
	bid, ok := response["USDBRL"]["bid"].(string)
	if !ok {
		log.Println("Erro ao converter o bid para string")
		http.Error(w, "Erro ao processar a resposta", http.StatusInternalServerError)
		return
	}

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
