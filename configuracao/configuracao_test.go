package configuracao

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func base() Configuracao {
	return Configuracao{Empresa: Empresa{ID: "a", Nome: "A", URL: "http://localhost:8081"}, EnderecoTCP: "127.0.0.1:8080", EnderecoHTTP: "127.0.0.1:8081", DiretorioDados: "dados"}
}

func TestQuantidadeVariavel(t *testing.T) {
	c := base()
	for i := 0; i < 20; i++ {
		c.Pares = append(c.Pares, Empresa{ID: fmt.Sprintf("p%d", i), Nome: "Par", URL: fmt.Sprintf("http://localhost:%d", 9000+i)})
	}
	if err := c.Validar(); err != nil {
		t.Fatal(err)
	}
}

func TestRejeitarIdentidadesAmbiguas(t *testing.T) {
	for _, par := range []Empresa{{ID: "a", Nome: "Outro", URL: "http://localhost:9999"}, {ID: "b", Nome: "B", URL: "http://localhost:8081/"}} {
		c := base()
		c.Pares = []Empresa{par}
		if err := c.Validar(); err == nil {
			t.Fatal("aceitou ID próprio ou URL repetida")
		}
	}
}

func TestCaminhoRelativoAoJSON(t *testing.T) {
	dir := t.TempDir()
	c := base()
	c.DiretorioDados = "../dados_a"
	b, _ := json.Marshal(c)
	p := filepath.Join(dir, "a.json")
	if err := os.WriteFile(p, b, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := Carregar(p)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Clean(filepath.Join(dir, "../dados_a"))
	if got.DiretorioDados != want {
		t.Fatalf("got %s, want %s", got.DiretorioDados, want)
	}
}

func TestRejeitarCampoDesconhecido(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.json")
	os.WriteFile(p, []byte(`{"empresa_errada": "a"}`), 0600)
	if _, err := Carregar(p); err == nil {
		t.Fatal("aceitou campo desconhecido")
	}
}
