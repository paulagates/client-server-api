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

const url_banco = "root:root@tcp(localhost:3306)/client-server-api?charset=utf8mb4&parseTime=True&loc=Local"

func main() {
	preparaTabela()
	http.HandleFunc("/cotacao", buscaCotacaoDolar)
	http.ListenAndServe(":8080", nil)
}

func preparaTabela() {
	db, err := sql.Open("mysql", url_banco)
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()
	stmt, err := db.Prepare(`CREATE TABLE IF NOT EXISTS cotacoes (
							id INT(19) PRIMARY KEY AUTO_INCREMENT,
							data DATETIME NOT NULL,
							valor FLOAT NOT NULL
							);`)
	if err != nil {
		log.Println("Erro ao preparar statement.")
		panic(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec()
	if err != nil {
		log.Println("Erro ao executar statement.")
		panic(err)
	}
}
func buscaCotacaoDolar(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		log.Printf("Não foi possível fazer a requisição ao API. Erro: %v", err)
		http.Error(w, "Não foi possível fazer a requisição ao API", http.StatusInternalServerError)
		return
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Não foi possível fazer a requisição ao API. Erro: %v", err)
		http.Error(w, "Não foi possível fazer a requisição ao API", http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Printf("Não foi possível buscar a cotação do dólar. Código de Status: %v", res.StatusCode)
		http.Error(w, "Não foi possível buscar a cotação do dólar", http.StatusInternalServerError)
		return
	}
	var response map[string]map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		log.Printf("Não foi possível decodificar a resposta. Erro: %v", err)
		http.Error(w, "Não foi possível decodificar a resposta", http.StatusInternalServerError)
		return
	}
	bid, ok := response["USDBRL"]["bid"].(string)
	if !ok {
		log.Println("Erro ao converter bid para string")
		return
	}
	db, err := sql.Open("mysql", url_banco)
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()
	err = insereCotacao(db, bid)
	if err != nil {
		log.Printf("Não foi possível inserir a cotação no banco de dados. Erro: %v", err)
		http.Error(w, "Não foi possível inserir a cotação no banco de dados", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"cotacao_dolar": bid})
}

func insereCotacao(db *sql.DB, valor string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
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
