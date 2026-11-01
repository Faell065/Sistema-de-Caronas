package dominio

import (
	"fmt"
	"sort"
	"time"
)

func (g *GerenciadorDeRotas) CriarTransacaoCoordenador(txID, chave, coordenadorID, passageiroID string, itens []ItemTransacao) error {
	if txID == "" || chave == "" {
		return fmt.Errorf("tx_id e chave de idempotência são obrigatórios")
	}
	participantes := make(map[string]bool)
	for _, item := range itens {
		if item.EmpresaID == "" || item.TrechoID == "" || item.Quantidade != 1 {
			return fmt.Errorf("item de transação inválido")
		}
		participantes[item.EmpresaID] = true
	}
	nomes := make([]string, 0, len(participantes))
	for empresaID := range participantes {
		nomes = append(nomes, empresaID)
	}
	sort.Strings(nomes)

	g.muTransacoes.Lock()
	if existente, existe := g.TransacoesCoordenador[txID]; existe {
		g.muTransacoes.Unlock()
		if existente.IdempotencyKey != chave {
			return fmt.Errorf("tx_id já pertence a outra operação")
		}
		return nil
	}
	record := &TransacaoCoordenador{
		ID: txID, CoordenadorID: coordenadorID, Estado: EstadoCoordenadorInit,
		IdempotencyKey: chave, PassageiroID: passageiroID,
		Lamport: uint64(time.Now().UnixNano()), Itens: append([]ItemTransacao(nil), itens...),
	}
	for _, empresaID := range nomes {
		record.Participantes = append(record.Participantes, EstadoParticipante{EmpresaID: empresaID, Estado: EstadoTransacaoPreparando})
	}
	g.TransacoesCoordenador[txID] = record
	g.muTransacoes.Unlock()
	return g.SalvarDados()
}

func (g *GerenciadorDeRotas) SetEstadoCoordenador(txID, estado string) error {
	g.muTransacoes.Lock()
	tx, existe := g.TransacoesCoordenador[txID]
	if !existe {
		g.muTransacoes.Unlock()
		return fmt.Errorf("transação coordenadora %s não existe", txID)
	}
	tx.Estado = estado
	g.muTransacoes.Unlock()
	return g.SalvarDados()
}

func (g *GerenciadorDeRotas) SetEstadoParticipanteCoordenador(txID, empresaID, estado string) error {
	g.muTransacoes.Lock()
	tx, existe := g.TransacoesCoordenador[txID]
	if !existe {
		g.muTransacoes.Unlock()
		return fmt.Errorf("transação coordenadora %s não existe", txID)
	}
	for i := range tx.Participantes {
		if tx.Participantes[i].EmpresaID == empresaID {
			tx.Participantes[i].Estado = estado
			g.muTransacoes.Unlock()
			return g.SalvarDados()
		}
	}
	g.muTransacoes.Unlock()
	return fmt.Errorf("participante %s não pertence à transação", empresaID)
}

func (g *GerenciadorDeRotas) ConsultarTransacaoCoordenador(txID string) (TransacaoCoordenador, bool) {
	g.muTransacoes.RLock()
	defer g.muTransacoes.RUnlock()
	tx, existe := g.TransacoesCoordenador[txID]
	if !existe {
		return TransacaoCoordenador{}, false
	}
	copia := *tx
	copia.Itens = append([]ItemTransacao(nil), tx.Itens...)
	copia.Participantes = append([]EstadoParticipante(nil), tx.Participantes...)
	return copia, true
}

func (g *GerenciadorDeRotas) CriarReservaDistribuida(txID, empresaID, passageiroID string, itens []ItemTransacao) error {
	g.muReservas.Lock()
	ids := make([]string, 0, len(itens))
	for _, item := range itens {
		ids = append(ids, item.EmpresaID+"/"+item.TrechoID)
	}
	g.Reservas[txID] = &Reserva{
		ID: txID, PassageiroID: passageiroID,
		IDsTrechos: ids, Itens: append([]ItemTransacao(nil), itens...),
		EmpresaID: empresaID,
		Data:      "",
	}
	g.muReservas.Unlock()
	return g.SalvarDados()
}

func (g *GerenciadorDeRotas) RemoverReservaDistribuida(passageiroID, reservaID string) ([]ItemTransacao, bool) {
	g.muReservas.Lock()
	reserva, existe := g.Reservas[reservaID]
	if !existe || reserva.PassageiroID != passageiroID {
		g.muReservas.Unlock()
		return nil, false
	}
	itens := append([]ItemTransacao(nil), reserva.Itens...)
	delete(g.Reservas, reservaID)
	g.muReservas.Unlock()
	_ = g.SalvarDados()
	return itens, true
}
