package main

import (
	"flag"
	"log"
	"projeto_redes/configuracao"
	"projeto_redes/servidor"
)

func main() {
	caminho := flag.String("config", "config/empresa_a.json", "arquivo de configuração da empresa")
	flag.Parse()
	c, err := configuracao.Carregar(*caminho)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Empresa: %s (%s)", c.Empresa.Nome, c.Empresa.ID)
	log.Printf("TCP: %s | HTTP: %s | URL anunciada: %s", c.EnderecoTCP, c.EnderecoHTTP, c.Empresa.URL)
	log.Printf("Dados: %s | Pares configurados: %d", c.DiretorioDados, len(c.Pares))
	s := servidor.NovoServidorConfigurado(c)
	if err := s.Executar(c); err != nil {
		log.Fatal(err)
	}
}
