package servidor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"projeto_redes/configuracao"
	"projeto_redes/dominio"
	"strings"
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

func TestContratoRestBuscaETrechoLocal(t *testing.T) {
	e := configuracao.Empresa{ID: "empresa_a", Nome: "Empresa A", URL: "http://localhost:8081"}
	g := dominio.NovoGerenciadorComDiretorio(t.TempDir())
	g.AdicionarTrechos([]dominio.Trecho{{
		ID:               "t_a1",
		RotaID:           "rota_a",
		MotoristaID:      "motorista_a",
		Origem:           "Salvador",
		Destino:          "Feira de Santana",
		Data:             "31/10/2026",
		HorarioSaida:     800,
		HorarioChegada:   930,
		AssentosTotais:   3,
		AssentosOcupados: 0,
		Valor:            45,
	}})
	h := HandlerEmpresaComGerenciador(e, g)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/trechos/t_a1", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET trecho: status %d, corpo %s", w.Code, w.Body.String())
	}
	var trecho TrechoRest
	if err := json.Unmarshal(w.Body.Bytes(), &trecho); err != nil {
		t.Fatal(err)
	}
	if trecho.Recurso != "empresa_a/t_a1" || trecho.AssentosDisponiveis != 3 {
		t.Fatalf("trecho REST inesperado: %+v", trecho)
	}

	w = httptest.NewRecorder()
	requisicao := httptest.NewRequest(http.MethodPost, "/api/v1/itinerarios/buscar",
		strings.NewReader(`{"origem":"Salvador","destino":"Feira de Santana","data":"31/10/2026"}`))
	h.ServeHTTP(w, requisicao)
	if w.Code != http.StatusOK {
		t.Fatalf("POST busca: status %d, corpo %s", w.Code, w.Body.String())
	}
	var busca RespostaBuscaRest
	if err := json.Unmarshal(w.Body.Bytes(), &busca); err != nil {
		t.Fatal(err)
	}
	if busca.EmpresaID != "empresa_a" || len(busca.Itinerarios) != 1 {
		t.Fatalf("busca REST inesperada: %+v", busca)
	}
}

func TestContratoRestValidacaoEResposta404(t *testing.T) {
	e := configuracao.Empresa{ID: "empresa_a", Nome: "Empresa A", URL: "http://localhost:8081"}
	h := HandlerEmpresaComGerenciador(e, dominio.NovoGerenciadorComDiretorio(t.TempDir()))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/trechos/inexistente", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("esperado 404, obtido %d", w.Code)
	}

	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/itinerarios/buscar", strings.NewReader(`{"origem":"A"}`)))
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("esperado 422, obtido %d", w.Code)
	}

	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/v1/empresa", nil))
	if w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") != http.MethodGet {
		t.Fatal("resposta de método não permitido incorreta")
	}
}

func TestContratoRestTransacoes(t *testing.T) {
	e := configuracao.Empresa{ID: "empresa_a", Nome: "Empresa A", URL: "http://localhost:8081"}
	g := dominio.NovoGerenciadorComDiretorio(t.TempDir())
	g.AdicionarTrechos([]dominio.Trecho{{ID: "t1", AssentosTotais: 1}})
	h := HandlerEmpresaComGerenciador(e, g)

	prepare := `{"id":"tx-rest-1","coordenador_id":"empresa_a","operacao":"RESERVAR","idempotency_key":"idem-rest-1","lamport":1,"itens":[{"empresa_id":"empresa_a","trecho_id":"t1","quantidade":1}]}`
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/transacoes/tx-rest-1/prepare", strings.NewReader(prepare)))
	if w.Code != http.StatusOK {
		t.Fatalf("prepare: status %d corpo %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/transacoes/tx-rest-1/prepare", strings.NewReader(prepare)))
	if w.Code != http.StatusOK {
		t.Fatalf("prepare repetido: status %d corpo %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/transacoes/tx-rest-1", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("consulta: status %d corpo %s", w.Code, w.Body.String())
	}

	for i := 0; i < 2; i++ {
		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/transacoes/tx-rest-1/commit", strings.NewReader(`{"tx_id":"tx-rest-1"}`)))
		if w.Code != http.StatusOK {
			t.Fatalf("commit %d: status %d corpo %s", i+1, w.Code, w.Body.String())
		}
	}

	trecho, existe := g.ObterTrecho("t1")
	if !existe || trecho.AssentosOcupados != 1 || trecho.AssentosPreparados != 0 {
		t.Fatalf("estado REST incorreto: %+v existe=%v", trecho, existe)
	}
}

func TestContratoRestTransacaoAbort(t *testing.T) {
	e := configuracao.Empresa{ID: "empresa_a", Nome: "Empresa A", URL: "http://localhost:8081"}
	g := dominio.NovoGerenciadorComDiretorio(t.TempDir())
	g.AdicionarTrechos([]dominio.Trecho{{ID: "t1", AssentosTotais: 1}})
	h := HandlerEmpresaComGerenciador(e, g)

	prepare := `{"id":"tx-rest-abort","coordenador_id":"empresa_a","operacao":"RESERVAR","lamport":1,"itens":[{"empresa_id":"empresa_a","trecho_id":"t1","quantidade":1}]}`
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/transacoes/tx-rest-abort/prepare", strings.NewReader(prepare)))
	if w.Code != http.StatusOK {
		t.Fatalf("prepare: status %d corpo %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/transacoes/tx-rest-abort/abort", strings.NewReader(`{"tx_id":"tx-rest-abort","motivo":"outro participante recusou"}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("abort: status %d corpo %s", w.Code, w.Body.String())
	}

	trecho, _ := g.ObterTrecho("t1")
	if trecho.AssentosPreparados != 0 || trecho.AssentosOcupados != 0 {
		t.Fatalf("abort REST deixou estado incorreto: %+v", trecho)
	}
}
