package dominio

import (
	"fmt"
	"sort"
	"time"
)

// CriarTransacaoCoordenador inicia a máquina de estados global (no papel de Coordenador do 2PC).
// Registra as intenções de reserva e as empresas participantes antes da chamada HTTP/REST remota.
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
	// Salva as intenções antes de avisar os participantes
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

// SetEstadoCoordenador atualiza persistentemente o progresso global da transação.
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

// SetEstadoParticipanteCoordenador marca em qual fase um participante individual está.
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

// CriarReservaDistribuida consolida no banco de dados local a posse do itinerário inter-empresas 
// associado ao passageiro, após a fase 2 (Commit) ser bem-sucedida.
func (g *GerenciadorDeRotas) CriarReservaDistribuida(txID, empresaID, passageiroID string, itens []ItemTransacao, trechos []Trecho, valorTotal float64) error {
	g.muReservas.Lock()
	ids := make([]string, 0, len(itens))
	for _, item := range itens {
		ids = append(ids, item.EmpresaID+"/"+item.TrechoID)
	}
	g.Reservas[txID] = &Reserva{
		ID:           txID,
		PassageiroID: passageiroID,
		IDsTrechos:   ids,
		Itens:        append([]ItemTransacao(nil), itens...),
		EmpresaID:    empresaID,
		Data:         time.Now().Format("02/01/2006"),
		Trechos:      append([]Trecho(nil), trechos...),
		ValorTotal:   valorTotal,
	}
	g.muReservas.Unlock()
	return g.SalvarDados()
}

// RemoverReservaDistribuida exclui os artefatos de uma reserva global caso o passageiro solicite cancelamento.
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