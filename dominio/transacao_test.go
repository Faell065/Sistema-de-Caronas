package dominio

import "testing"

func transacaoTeste(txID, trechoID string) TransacaoLocal {
	return TransacaoLocal{
		ID: txID, CoordenadorID: "empresa_a", EmpresaID: "empresa_a",
		Operacao: "RESERVAR", IdempotencyKey: "idem-" + txID, Lamport: 1,
		Itens: []ItemTransacao{{EmpresaID: "empresa_a", TrechoID: trechoID, Quantidade: 1}},
	}
}

func TestPrepareCommitIdempotente(t *testing.T) {
	dir := t.TempDir()
	g := NovoGerenciadorComDiretorio(dir)
	g.AdicionarTrechos([]Trecho{{ID: "t1", AssentosTotais: 1, AssentosOcupados: 0}})
	tx := transacaoTeste("tx-1", "t1")

	preparado, err := g.PrepareReservaLocal(tx)
	if err != nil || preparado.Estado != EstadoTransacaoPreparado {
		t.Fatalf("prepare falhou: estado=%s err=%v", preparado.Estado, err)
	}

	trecho, _ := g.ObterTrecho("t1")
	if trecho.AssentosPreparados != 1 || trecho.AssentosOcupados != 0 {
		t.Fatalf("hold incorreto: preparado=%d ocupado=%d", trecho.AssentosPreparados, trecho.AssentosOcupados)
	}

	preparado, err = g.PrepareReservaLocal(tx)
	if err != nil || preparado.Estado != EstadoTransacaoPreparado {
		t.Fatalf("prepare repetido falhou: estado=%s err=%v", preparado.Estado, err)
	}
	trecho, _ = g.ObterTrecho("t1")
	if trecho.AssentosPreparados != 1 {
		t.Fatalf("prepare repetido duplicou hold: %d", trecho.AssentosPreparados)
	}

	confirmado, err := g.CommitReservaLocal("tx-1")
	if err != nil || confirmado.Estado != EstadoTransacaoCommitConfirmado {
		t.Fatalf("commit falhou: estado=%s err=%v", confirmado.Estado, err)
	}
	confirmado, err = g.CommitReservaLocal("tx-1")
	if err != nil || confirmado.Estado != EstadoTransacaoCommitConfirmado {
		t.Fatalf("commit repetido falhou: estado=%s err=%v", confirmado.Estado, err)
	}

	trecho, _ = g.ObterTrecho("t1")
	if trecho.AssentosOcupados != 1 || trecho.AssentosPreparados != 0 {
		t.Fatalf("commit incorreto: ocupado=%d preparado=%d", trecho.AssentosOcupados, trecho.AssentosPreparados)
	}

	reiniciado := NovoGerenciadorComDiretorio(dir)
	trecho, _ = reiniciado.ObterTrecho("t1")
	transacao, existe := reiniciado.ConsultarTransacaoLocal("tx-1")
	if !existe || transacao.Estado != EstadoTransacaoCommitConfirmado {
		t.Fatalf("transação não recuperou commit: %+v existe=%v", transacao, existe)
	}
	if trecho.AssentosOcupados != 1 || trecho.AssentosPreparados != 0 {
		t.Fatalf("estado do trecho não recuperou: %+v", trecho)
	}
}

func TestAbortLiberaHoldEImpedeCommitPosterior(t *testing.T) {
	dir := t.TempDir()
	g := NovoGerenciadorComDiretorio(dir)
	g.AdicionarTrechos([]Trecho{{ID: "t2", AssentosTotais: 1}})

	if _, err := g.PrepareReservaLocal(transacaoTeste("tx-abort", "t2")); err != nil {
		t.Fatal(err)
	}
	repetido, err := g.AbortReservaLocal("tx-abort", "participante recusou")
	if err != nil || repetido.Estado != EstadoTransacaoAbortado {
		t.Fatalf("abort falhou: %+v err=%v", repetido, err)
	}
	repetido, err = g.AbortReservaLocal("tx-abort", "repetição")
	if err != nil || repetido.Estado != EstadoTransacaoAbortado {
		t.Fatalf("abort repetido falhou: %+v err=%v", repetido, err)
	}
	if _, err := g.CommitReservaLocal("tx-abort"); err == nil {
		t.Fatal("commit após abort deveria ser recusado")
	}

	trecho, _ := g.ObterTrecho("t2")
	if trecho.AssentosPreparados != 0 || trecho.AssentosOcupados != 0 {
		t.Fatalf("abort deixou estado incorreto: %+v", trecho)
	}
}

func TestPrepareRespeitaUltimaVagaInclusoPreparada(t *testing.T) {
	g := NovoGerenciadorComDiretorio(t.TempDir())
	g.AdicionarTrechos([]Trecho{{ID: "t3", AssentosTotais: 1}})

	if _, err := g.PrepareReservaLocal(transacaoTeste("tx-a", "t3")); err != nil {
		t.Fatal(err)
	}
	if _, err := g.PrepareReservaLocal(transacaoTeste("tx-b", "t3")); err == nil {
		t.Fatal("segunda transação deveria perder a última vaga")
	}

	trecho, _ := g.ObterTrecho("t3")
	if trecho.AssentosPreparados != 1 {
		t.Fatalf("apenas um hold deve existir: %d", trecho.AssentosPreparados)
	}
}
