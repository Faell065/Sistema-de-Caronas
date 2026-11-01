package servidor

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"projeto_redes/configuracao"
	"projeto_redes/dominio"
)

func TestDisputaUltimaVagaEntreDoisCoordenadores(t *testing.T) {
	ownerConfig := configuracao.Configuracao{
		Empresa:     configuracao.Empresa{ID: "empresa_a", Nome: "A", URL: "http://empresa_a:8081"},
		EnderecoTCP: "127.0.0.1:8080", EnderecoHTTP: "127.0.0.1:8081",
		DiretorioDados: t.TempDir(),
	}
	ownerGerenciador := dominio.NovoGerenciadorComDiretorio(ownerConfig.DiretorioDados)
	ownerGerenciador.AdicionarTrechos([]dominio.Trecho{{ID: "ultima", AssentosTotais: 1}})
	owner := httptest.NewServer(HandlerEmpresaComGerenciador(ownerConfig.Empresa, ownerGerenciador, ownerConfig))
	defer owner.Close()

	ownerConfig.Pares = nil
	newCoordinator := func(id, dir string) *Servidor {
		c := configuracao.Configuracao{
			Empresa:        configuracao.Empresa{ID: id, Nome: id, URL: "http://" + id + ":8081"},
			DiretorioDados: dir,
			Pares:          []configuracao.Empresa{{ID: "empresa_a", Nome: "A", URL: owner.URL}},
		}
		return &Servidor{Gerenciador: dominio.NovoGerenciadorComDiretorio(dir), Configuracao: &c}
	}

	coord1 := newCoordinator("empresa_b", t.TempDir())
	coord2 := newCoordinator("empresa_c", t.TempDir())

	itens1 := []dominio.ItemTransacao{{EmpresaID: "empresa_a", TrechoID: "ultima", Quantidade: 1}}
	itens2 := append([]dominio.ItemTransacao(nil), itens1...)

	tx1, err1 := coord1.ExecutarReservaDistribuida("p1", itens1)
	tx2, err2 := coord2.ExecutarReservaDistribuida("p2", itens2)

	if err1 == nil && err2 == nil {
		t.Fatal("duas coordenadoras confirmaram a mesma última vaga")
	}
	if err1 != nil && err2 != nil {
		t.Fatalf("nenhuma reserva foi processada: tx1=%s tx2=%s", err1, err2)
	}

	confirmado := 0
	if tx1.Estado == dominio.EstadoCoordenadorConfirmado && err1 == nil {
		confirmado++
	}
	if tx2.Estado == dominio.EstadoCoordenadorConfirmado && err2 == nil {
		confirmado++
	}
	if confirmado != 1 {
		t.Fatalf("reservas confirmadas = %d; esperado 1; tx1=%s tx2=%s", confirmado, tx1.Estado, tx2.Estado)
	}

	trecho, existe := ownerGerenciador.ObterTrecho("ultima")
	if !existe || trecho.AssentosOcupados != 1 || trecho.AssentosPreparados != 0 {
		t.Fatalf("estado final da última vaga: %+v existe=%v", trecho, existe)
	}
}

func TestMensagensRepetidasNaoDuplicamCommit(t *testing.T) {
	dir := t.TempDir()
	g := dominio.NovoGerenciadorComDiretorio(dir)
	g.AdicionarTrechos([]dominio.Trecho{{ID: "t1", AssentosTotais: 1}})

	tx := dominio.TransacaoLocal{
		ID: "tx-repetida", CoordenadorID: "empresa_a", EmpresaID: "empresa_a",
		Operacao: "RESERVAR", Itens: []dominio.ItemTransacao{{EmpresaID: "empresa_a", TrechoID: "t1", Quantidade: 1}},
	}
	if _, err := g.PrepareReservaLocal(tx); err != nil {
		t.Fatal(err)
	}
	if _, err := g.CommitReservaLocal(tx.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := g.CommitReservaLocal(tx.ID); err != nil {
		t.Fatal(err)
	}

	trecho, _ := g.ObterTrecho("t1")
	if trecho.AssentosOcupados != 1 || trecho.AssentosPreparados != 0 {
		t.Fatalf("commit repetido alterou a ocupação: %+v", trecho)
	}
}

func TestHMACNegativoNoHandlerDeTransacao(t *testing.T) {
	dir := t.TempDir()
	g := dominio.NovoGerenciadorComDiretorio(dir)
	g.AdicionarTrechos([]dominio.Trecho{{ID: "t1", AssentosTotais: 1}})
	c := configuracao.Configuracao{
		Empresa:        configuracao.Empresa{ID: "empresa_a", Nome: "A", URL: "http://empresa_a:8081"},
		DiretorioDados: dir, ChaveHMAC: "segredo-teste",
	}
	h := HandlerEmpresaComGerenciador(c.Empresa, g, c)

	corpo := `{"id":"tx-hmac","coordenador_id":"empresa_b","operacao":"RESERVAR","itens":[{"empresa_id":"empresa_a","trecho_id":"t1","quantidade":1}]}`
	semAssinatura := httptest.NewRecorder()
	h.ServeHTTP(semAssinatura, httptest.NewRequest(http.MethodPost, "/api/v1/transacoes/tx-hmac/prepare", strings.NewReader(corpo)))
	if semAssinatura.Code != http.StatusUnauthorized {
		t.Fatalf("esperado 401 sem HMAC, obtido %d", semAssinatura.Code)
	}

	assinada := httptest.NewRequest(http.MethodPost, "/api/v1/transacoes/tx-hmac/prepare", strings.NewReader(corpo))
	timestampTexto := strconv.FormatInt(time.Now().UnixNano(), 10)
	assinada.Header.Set("X-VaiJunto-Timestamp", timestampTexto)
	assinada.Header.Set("X-VaiJunto-Empresa-ID", "empresa_b")
	assinada.Header.Set("X-VaiJunto-Idempotency-Key", "tx-hmac")
	assinada.Header.Set("X-VaiJunto-Signature", AssinarCorpo("segredo-teste", http.MethodPost,
		assinada.URL.RequestURI(), timestampTexto, "empresa_b", "tx-hmac", corpo))
	respostaAssinada := httptest.NewRecorder()
	h.ServeHTTP(respostaAssinada, assinada)
	if respostaAssinada.Code != http.StatusOK {
		t.Fatalf("esperado 200 com HMAC válido, obtido %d: %s", respostaAssinada.Code, respostaAssinada.Body.String())
	}
}
