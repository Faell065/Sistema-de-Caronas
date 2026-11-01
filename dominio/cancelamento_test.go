package dominio

import "testing"

func TestReservaDistribuidaPersisteItensParaCancelamento(t *testing.T) {
	dir := t.TempDir()
	g := NovoGerenciadorComDiretorio(dir)
	itens := []ItemTransacao{
		{EmpresaID: "empresa_a", TrechoID: "ta", Quantidade: 1},
		{EmpresaID: "empresa_b", TrechoID: "tb", Quantidade: 1},
	}
	if err := g.CriarReservaDistribuida("res-1", "empresa_a", "p1", itens); err != nil {
		t.Fatal(err)
	}

	reiniciado := NovoGerenciadorComDiretorio(dir)
	itensRecarregados, existe := reiniciado.RemoverReservaDistribuida("p1", "res-1")
	if !existe || len(itensRecarregados) != 2 {
		t.Fatalf("reserva não recuperada: itens=%v existe=%v", itensRecarregados, existe)
	}
	if _, existe := reiniciado.RemoverReservaDistribuida("p1", "res-1"); existe {
		t.Fatal("cancelamento repetido não deveria remover novamente")
	}
}

func TestCancelarReservaLocalIdempotente(t *testing.T) {
	g := NovoGerenciadorComDiretorio(t.TempDir())
	g.AdicionarTrechos([]Trecho{{ID: "t1", AssentosTotais: 1}})
	tx := TransacaoLocal{
		ID: "tx-cancel", Operacao: "RESERVAR", Estado: EstadoTransacaoPreparado,
		Itens: []ItemTransacao{{EmpresaID: "empresa_a", TrechoID: "t1", Quantidade: 1}},
	}
	if _, err := g.PrepareReservaLocal(tx); err != nil {
		t.Fatal(err)
	}
	if _, err := g.CommitReservaLocal(tx.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := g.CancelarReservaLocal(tx.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := g.CancelarReservaLocal(tx.ID); err != nil {
		t.Fatal(err)
	}
	trecho, _ := g.ObterTrecho("t1")
	if trecho.AssentosOcupados != 0 || trecho.AssentosPreparados != 0 {
		t.Fatalf("cancelamento não liberou ocupação: %+v", trecho)
	}
}
