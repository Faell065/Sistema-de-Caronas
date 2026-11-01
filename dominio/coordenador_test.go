package dominio

import "testing"

func TestCoordenadorPersisteEstadosPorEmpresa(t *testing.T) {
	dir := t.TempDir()
	g := NovoGerenciadorComDiretorio(dir)
	itens := []ItemTransacao{
		{EmpresaID: "empresa_c", TrechoID: "tc", Quantidade: 1},
		{EmpresaID: "empresa_a", TrechoID: "ta", Quantidade: 1},
		{EmpresaID: "empresa_b", TrechoID: "tb", Quantidade: 1},
	}
	if err := g.CriarTransacaoCoordenador("tx-c1", "idem-c1", "empresa_a", "passageiro_a", itens); err != nil {
		t.Fatal(err)
	}
	if err := g.SetEstadoCoordenador("tx-c1", EstadoCoordenadorPreparando); err != nil {
		t.Fatal(err)
	}
	if err := g.SetEstadoParticipanteCoordenador("tx-c1", "empresa_b", EstadoTransacaoPreparado); err != nil {
		t.Fatal(err)
	}
	if err := g.SetEstadoParticipanteCoordenador("tx-c1", "empresa_b", EstadoTransacaoPreparado); err != nil {
		t.Fatal(err)
	}

	reiniciado := NovoGerenciadorComDiretorio(dir)
	tx, existe := reiniciado.ConsultarTransacaoCoordenador("tx-c1")
	if !existe || tx.Estado != EstadoCoordenadorPreparando {
		t.Fatalf("transação coordenadora não recuperou: %+v existe=%v", tx, existe)
	}
	if len(tx.Participantes) != 3 {
		t.Fatalf("participantes esperados=3 obtidos=%d", len(tx.Participantes))
	}
	for _, participante := range tx.Participantes {
		if participante.EmpresaID == "empresa_b" && participante.Estado != EstadoTransacaoPreparado {
			t.Fatalf("estado de B não recuperou: %+v", participante)
		}
	}

	ids := []string{tx.Participantes[0].EmpresaID, tx.Participantes[1].EmpresaID, tx.Participantes[2].EmpresaID}
	if ids[0] > ids[1] || ids[1] > ids[2] {
		t.Fatalf("participantes não estão ordenados por empresa: %v", ids)
	}
}

func TestCoordenadorIdempotenteComChaveDiferente(t *testing.T) {
	g := NovoGerenciadorComDiretorio(t.TempDir())
	itens := []ItemTransacao{{EmpresaID: "empresa_a", TrechoID: "ta", Quantidade: 1}}
	if err := g.CriarTransacaoCoordenador("tx-c2", "idem-a", "empresa_a", "p1", itens); err != nil {
		t.Fatal(err)
	}
	if err := g.CriarTransacaoCoordenador("tx-c2", "idem-a", "empresa_a", "p1", itens); err != nil {
		t.Fatal(err)
	}
	if err := g.CriarTransacaoCoordenador("tx-c2", "idem-b", "empresa_a", "p1", itens); err == nil {
		t.Fatal("mesmo tx_id com chave diferente deveria ser conflito")
	}
}
