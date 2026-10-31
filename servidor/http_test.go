package servidor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"projeto_redes/configuracao"
	"testing"
)

func TestIdentidadeHTTP(t *testing.T) {
	e := configuracao.Empresa{ID: "empresa_b", Nome: "B", URL: "http://localhost:8091"}
	h := HandlerEmpresa(e)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/empresa", nil))
	var got configuracao.Empresa
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || got != e {
		t.Fatalf("resposta inesperada: %s", w.Body.String())
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/empresa", nil))
	if w.Code != 405 || w.Header().Get("Allow") != "GET" {
		t.Fatal("método incorreto aceito")
	}
}
