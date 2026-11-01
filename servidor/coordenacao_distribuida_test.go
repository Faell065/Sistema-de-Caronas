package servidor

import (
	"net/http"
	"net/http/httptest"
	"projeto_redes/configuracao"
	"projeto_redes/dominio"
	"testing"
)

func TestCoordenadorReservaSomenteLocal(t *testing.T) {
	dir := t.TempDir()
	g := dominio.NovoGerenciadorComDiretorio(dir)
	g.AdicionarTrechos([]dominio.Trecho{{ID: "ta", AssentosTotais: 1, AssentosOcupados: 0}})
	c := configuracao.Configuracao{
		Empresa:     configuracao.Empresa{ID: "empresa_a", Nome: "A", URL: "http://empresa_a:8081"},
		EnderecoTCP: "127.0.0.1:8080", EnderecoHTTP: "127.0.0.1:8081",
		DiretorioDados: dir,
	}
	s := &Servidor{Gerenciador: g, Configuracao: &c}

	tx, err := s.ExecutarReservaDistribuida("passageiro_a", []dominio.ItemTransacao{{
		EmpresaID: "empresa_a", TrechoID: "ta", Quantidade: 1,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if tx.Estado != dominio.EstadoCoordenadorConfirmado {
		t.Fatalf("estado global = %s; esperado CONFIRMED", tx.Estado)
	}
	trecho, _ := g.ObterTrecho("ta")
	if trecho.AssentosOcupados != 1 || trecho.AssentosPreparados != 0 {
		t.Fatalf("estado do trecho: %+v", trecho)
	}
}

func TestCoordenadorAbortaHoldLocalQuandoParticipanteRemotoRecusa(t *testing.T) {
	participante := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := http.Header{}
		_ = h
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"erro":"sem vaga"}`))
	}))
	defer participante.Close()

	dir := t.TempDir()
	g := dominio.NovoGerenciadorComDiretorio(dir)
	g.AdicionarTrechos([]dominio.Trecho{{ID: "ta", AssentosTotais: 1, AssentosOcupados: 0}})
	c := configuracao.Configuracao{
		Empresa:     configuracao.Empresa{ID: "empresa_a", Nome: "A", URL: "http://empresa_a:8081"},
		EnderecoTCP: "127.0.0.1:8080", EnderecoHTTP: "127.0.0.1:8081",
		DiretorioDados: dir,
		Pares:          []configuracao.Empresa{{ID: "empresa_b", Nome: "B", URL: participante.URL}},
	}
	s := &Servidor{Gerenciador: g, Configuracao: &c}

	itens := []dominio.ItemTransacao{
		{EmpresaID: "empresa_a", TrechoID: "ta", Quantidade: 1},
		{EmpresaID: "empresa_b", TrechoID: "tb", Quantidade: 1},
	}
	tx, err := s.ExecutarReservaDistribuida("passageiro_a", itens)
	if err == nil {
		t.Fatal("esperada recusa do participante remoto")
	}
	if tx.Estado != dominio.EstadoCoordenadorAborted {
		t.Fatalf("estado global = %s; esperado ABORTED", tx.Estado)
	}
	trecho, _ := g.ObterTrecho("ta")
	if trecho.AssentosPreparados != 0 || trecho.AssentosOcupados != 0 {
		t.Fatalf("hold local não foi revertido: %+v", trecho)
	}
}

func TestCoordenadorIndisponibilidadeRemotaAbortaEChegaAoEstadoPersistido(t *testing.T) {
	dir := t.TempDir()
	g := dominio.NovoGerenciadorComDiretorio(dir)
	g.AdicionarTrechos([]dominio.Trecho{{ID: "ta", AssentosTotais: 1}})
	c := configuracao.Configuracao{
		Empresa:     configuracao.Empresa{ID: "empresa_a", Nome: "A", URL: "http://empresa_a:8081"},
		EnderecoTCP: "127.0.0.1:8080", EnderecoHTTP: "127.0.0.1:8081",
		DiretorioDados: dir,
		Pares:          []configuracao.Empresa{{ID: "empresa_b", Nome: "B", URL: "http://127.0.0.1:1"}},
	}
	s := &Servidor{Gerenciador: g, Configuracao: &c}

	tx, err := s.ExecutarReservaDistribuida("passageiro_a", []dominio.ItemTransacao{
		{EmpresaID: "empresa_a", TrechoID: "ta", Quantidade: 1},
		{EmpresaID: "empresa_b", TrechoID: "tb", Quantidade: 1},
	})
	if err == nil {
		t.Fatal("esperada indisponibilidade do participante remoto")
	}
	if tx.Estado != dominio.EstadoCoordenadorAborted {
		t.Fatalf("estado persistido: %s", tx.Estado)
	}

	reiniciado := dominio.NovoGerenciadorComDiretorio(dir)
	txReiniciado, existe := reiniciado.ConsultarTransacaoCoordenador(tx.ID)
	if !existe || txReiniciado.Estado != dominio.EstadoCoordenadorAborted {
		t.Fatalf("decisão não persistiu após reinício: %+v existe=%v", txReiniciado, existe)
	}
	trecho, _ := reiniciado.ObterTrecho("ta")
	if trecho.AssentosPreparados != 0 || trecho.AssentosOcupados != 0 {
		t.Fatalf("hold remanescente após abort persistido: %+v", trecho)
	}
}

func TestBuscaDistribuidaIndicaFalhaSemPerderResultadoLocal(t *testing.T) {
	dir := t.TempDir()
	g := dominio.NovoGerenciadorComDiretorio(dir)
	g.AdicionarTrechos([]dominio.Trecho{{
		ID: "ta", Origem: "A", Destino: "B", Data: "01/11/2026",
		HorarioSaida: 800, HorarioChegada: 900, AssentosTotais: 1,
	}})
	c := configuracao.Configuracao{
		Empresa:        configuracao.Empresa{ID: "empresa_a", Nome: "A", URL: "http://empresa_a:8081"},
		DiretorioDados: dir,
		Pares:          []configuracao.Empresa{{ID: "empresa_b", Nome: "B", URL: "http://127.0.0.1:1"}},
	}
	s := &Servidor{Gerenciador: g, Configuracao: &c}

	resultado := s.BuscarDistribuido("A", "B", "01/11/2026")
	if len(resultado.Caminhos) != 1 {
		t.Fatalf("caminho local esperado; obtidos %d", len(resultado.Caminhos))
	}
	if len(resultado.Falhas) != 1 || resultado.Falhas[0] != "empresa_b" {
		t.Fatalf("falha remota não registrada: %+v", resultado.Falhas)
	}
}
