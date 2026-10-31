package dominio

import (
	"sync"
	"testing"
)

func TestUltimaVagaComReservasConcorrentes(t *testing.T) {
	g := NovoGerenciadorComDiretorio(t.TempDir())
	g.AdicionarTrechos([]Trecho{{
		ID: "trecho_unico", Origem: "A", Destino: "B",
		AssentosTotais: 1, AssentosOcupados: 0,
	}})

	const reservas = 20
	var wg sync.WaitGroup
	sucessos := make(chan string, reservas)

	for i := 0; i < reservas; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sucesso, _, reservaID := g.ReservarItinerarioComID(
				"passageiro_"+string(rune(i)), []string{"trecho_unico"},
			)
			if sucesso {
				sucessos <- reservaID
			}
		}(i)
	}
	wg.Wait()
	close(sucessos)

	confirmadas := 0
	for range sucessos {
		confirmadas++
	}
	if confirmadas != 1 {
		t.Fatalf("esperada 1 reserva confirmada, obtidas %d", confirmadas)
	}

	trecho, existe := g.ObterTrecho("trecho_unico")
	if !existe {
		t.Fatal("trecho não encontrado")
	}
	if trecho.AssentosOcupados != 1 {
		t.Fatalf("assentos ocupados = %d; esperado 1", trecho.AssentosOcupados)
	}
}

func TestCancelamentoReabreVagaERestauraComRepositorioIsolado(t *testing.T) {
	diretorio := t.TempDir()
	g := NovoGerenciadorComDiretorio(diretorio)
	g.AdicionarTrechos([]Trecho{{
		ID: "trecho_cancelavel", Origem: "A", Destino: "B",
		AssentosTotais: 1, AssentosOcupados: 0,
	}})

	sucesso, _, reservaID := g.ReservarItinerarioComID(
		"passageiro_teste", []string{"trecho_cancelavel"},
	)
	if !sucesso {
		t.Fatal("a primeira reserva deveria ser confirmada")
	}
	if _, msg := g.CancelarReservaPassageiro("passageiro_teste", reservaID); msg == "" {
		t.Fatal("cancelamento não retornou mensagem")
	}

	trecho, existe := g.ObterTrecho("trecho_cancelavel")
	if !existe || trecho.AssentosOcupados != 0 {
		t.Fatal("vaga não foi liberada pelo cancelamento")
	}

	gRecarregado := NovoGerenciadorComDiretorio(diretorio)
	trechoRecarregado, existe := gRecarregado.ObterTrecho("trecho_cancelavel")
	if !existe || trechoRecarregado.AssentosOcupados != 0 {
		t.Fatal("estado cancelado não foi restaurado corretamente")
	}
}

func TestAtomicidadeDeReservaComTrechoSemVaga(t *testing.T) {
	g := NovoGerenciadorComDiretorio(t.TempDir())
	g.AdicionarTrechos([]Trecho{
		{ID: "com_vaga", Origem: "A", Destino: "B", AssentosTotais: 1},
		{ID: "sem_vaga", Origem: "B", Destino: "C", AssentosTotais: 1, AssentosOcupados: 1},
	})

	sucesso, _, _ := g.ReservarItinerarioComID(
		"passageiro_teste", []string{"com_vaga", "sem_vaga"},
	)
	if sucesso {
		t.Fatal("reserva com trecho sem vaga não deveria ser confirmada")
	}

	trecho, _ := g.ObterTrecho("com_vaga")
	if trecho.AssentosOcupados != 0 {
		t.Fatal("primeiro trecho foi alterado apesar da recusa global")
	}
}
