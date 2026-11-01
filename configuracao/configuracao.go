// Package configuracao define a identidade e os endereços da instância.
package configuracao

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Empresa struct {
	ID   string `json:"id"`
	Nome string `json:"nome"`
	URL  string `json:"url"`
}

type Configuracao struct {
	Empresa        Empresa   `json:"empresa"`
	EnderecoTCP    string    `json:"endereco_tcp"`
	EnderecoHTTP   string    `json:"endereco_http"`
	DiretorioDados string    `json:"diretorio_dados"`
	Pares          []Empresa `json:"pares"`
	ChaveHMAC      string    `json:"chave_hmac,omitempty"`
}

// Carregar não consulta a rede: um par fora do ar não impede a inicialização.
// Caminhos relativos de dados usam a pasta do arquivo JSON como referência.
func Carregar(caminho string) (Configuracao, error) {
	var c Configuracao
	f, err := os.Open(caminho)
	if err != nil {
		return c, err
	}
	defer f.Close()
	d := json.NewDecoder(f)
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return c, fmt.Errorf("configuração inválida: %w", err)
	}
	var extra interface{}
	if err := d.Decode(&extra); err != io.EOF {
		return c, fmt.Errorf("o arquivo deve conter somente um objeto JSON")
	}
	if err := c.Validar(); err != nil {
		return c, err
	}
	if !filepath.IsAbs(c.DiretorioDados) {
		c.DiretorioDados = filepath.Join(filepath.Dir(caminho), c.DiretorioDados)
	}
	c.DiretorioDados, err = filepath.Abs(c.DiretorioDados)
	return c, err
}

func (c Configuracao) Validar() error {
	for _, endereco := range []string{c.EnderecoTCP, c.EnderecoHTTP} {
		_, porta, err := net.SplitHostPort(endereco)
		if err != nil {
			return fmt.Errorf("endereço inválido %q: use host:porta", endereco)
		}
		n, err := net.LookupPort("tcp", porta)
		if err != nil || n <= 0 {
			return fmt.Errorf("porta inválida em %q", endereco)
		}
	}
	if c.EnderecoTCP == c.EnderecoHTTP {
		return fmt.Errorf("TCP e HTTP devem usar endereços distintos")
	}
	if strings.TrimSpace(c.DiretorioDados) == "" {
		return fmt.Errorf("diretorio_dados é obrigatório")
	}
	ids := make(map[string]bool)
	urls := make(map[string]bool)
	for _, e := range append([]Empresa{c.Empresa}, c.Pares...) {
		if strings.TrimSpace(e.ID) == "" || strings.TrimSpace(e.Nome) == "" {
			return fmt.Errorf("toda empresa precisa de id e nome")
		}
		if ids[e.ID] {
			return fmt.Errorf("id de empresa repetido: %s", e.ID)
		}
		ids[e.ID] = true
		u, err := url.Parse(e.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("URL base inválida para %s: %q", e.ID, e.URL)
		}
		chave := strings.TrimRight(strings.ToLower(e.URL), "/")
		if urls[chave] {
			return fmt.Errorf("URL repetida: %s", e.URL)
		}
		urls[chave] = true
	}
	return nil
}
