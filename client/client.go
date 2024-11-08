package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	for {
		if ok, dolar := buscaCotacao(); ok {
			log.Printf("Cotação do Dolar: %s", dolar)
			err := criaArquivo(dolar)
			if err != nil {
				log.Printf("Não foi possível gravar o arquivo: %v", err)
				continue
			} else {
				log.Println("Arquivo gravado com sucesso!")
				break
			}
		} else {
			log.Println("Tentando novamente...")
			time.Sleep(5 * time.Second)
		}
	}

}

func criaArquivo(valor string) error {
	f, err := os.Create("cotacao.txt")
	if err != nil {
		log.Fatalf("Não foi possível abrir o arquivo: %v", err)
		return err
	}
	defer f.Close()
	f.WriteString(fmt.Sprintf("Dólar: %s", valor))
	return nil
}
func buscaCotacao() (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)
	if err != nil {
		log.Printf("Não foi possível fazer a requisição. Erro: %v", err)
		return false, ""
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, ""
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		log.Printf("Não foi possível fazer a requisição. Status Code: %d", res.StatusCode)
		return false, ""
	}
	var response map[string]string
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		log.Printf("Não foi possível decodificar a resposta. Erro: %v", err)
		return false, ""
	}
	return true, response["dolar"]

}
