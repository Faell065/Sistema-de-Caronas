package dominio

import "fmt"

// PrepareReservaLocal cria holds persistíveis em todos os trechos pertencentes
// à instância participante. Não confirma assentos nem espera chamadas de rede.
func (g *GerenciadorDeRotas) PrepareReservaLocal(tx TransacaoLocal) (TransacaoLocal, error) {
	if tx.ID == "" {
		return tx, fmt.Errorf("tx_id é obrigatório")
	}
	if tx.Operacao != "RESERVAR" {
		return tx, fmt.Errorf("operação não suportada: %s", tx.Operacao)
	}

	g.muTransacoes.Lock()
	g.muOperacoes.Lock()
	g.muTrechos.Lock()

	if existente, ok := g.Transacoes[tx.ID]; ok {
		if existente.Estado == EstadoTransacaoAbortado {
			g.muTrechos.Unlock()
			g.muOperacoes.Unlock()
			g.muTransacoes.Unlock()
			return *existente, fmt.Errorf("transação foi abortada")
		}
		if existente.Estado == EstadoTransacaoPreparado || existente.Estado == EstadoTransacaoCommitConfirmado || existente.Estado == EstadoTransacaoCommitPersistido {
			g.muTrechos.Unlock()
			g.muOperacoes.Unlock()
			g.muTransacoes.Unlock()
			copia := *existente
			return copia, nil
		}
	}

	for _, item := range tx.Itens {
		if item.Quantidade != 1 {
			g.muTrechos.Unlock()
			g.muOperacoes.Unlock()
			g.muTransacoes.Unlock()
			return tx, fmt.Errorf("quantidade inválida para trecho %s", item.TrechoID)
		}
		trecho, existe := g.Trechos[item.TrechoID]
		if !existe {
			g.muTrechos.Unlock()
			g.muOperacoes.Unlock()
			g.muTransacoes.Unlock()
			return tx, fmt.Errorf("trecho %s não existe", item.TrechoID)
		}
		livres := trecho.AssentosTotais - trecho.AssentosOcupados - trecho.AssentosPreparados
		if livres < item.Quantidade {
			g.muTrechos.Unlock()
			g.muOperacoes.Unlock()
			g.muTransacoes.Unlock()
			return tx, fmt.Errorf("sem vaga disponível em %s", item.TrechoID)
		}
	}

	for _, item := range tx.Itens {
		g.Trechos[item.TrechoID].AssentosPreparados += item.Quantidade
	}
	tx.Papel = "PARTICIPANTE"
	tx.Estado = EstadoTransacaoPreparado
	g.Transacoes[tx.ID] = &tx

	g.muTrechos.Unlock()
	g.muOperacoes.Unlock()
	g.muTransacoes.Unlock()

	if err := g.SalvarDados(); err != nil {
		return tx, err
	}
	return tx, nil
}

// CommitReservaLocal converte os holds em ocupação confirmada de forma idempotente.
func (g *GerenciadorDeRotas) CommitReservaLocal(txID string) (TransacaoLocal, error) {
	g.muTransacoes.Lock()
	g.muOperacoes.Lock()
	g.muTrechos.Lock()

	tx, existe := g.Transacoes[txID]
	if !existe {
		g.muTrechos.Unlock()
		g.muOperacoes.Unlock()
		g.muTransacoes.Unlock()
		return TransacaoLocal{}, fmt.Errorf("transação %s não existe", txID)
	}
	if tx.Estado == EstadoTransacaoAbortado {
		g.muTrechos.Unlock()
		g.muOperacoes.Unlock()
		g.muTransacoes.Unlock()
		return *tx, fmt.Errorf("transação já foi abortada")
	}
	if tx.Estado == EstadoTransacaoCommitConfirmado {
		copia := *tx
		g.muTrechos.Unlock()
		g.muOperacoes.Unlock()
		g.muTransacoes.Unlock()
		return copia, nil
	}
	if tx.Estado != EstadoTransacaoPreparado && tx.Estado != EstadoTransacaoCommitPersistido {
		g.muTrechos.Unlock()
		g.muOperacoes.Unlock()
		g.muTransacoes.Unlock()
		return *tx, fmt.Errorf("estado inválido para commit: %s", tx.Estado)
	}

	for _, item := range tx.Itens {
		trecho, ok := g.Trechos[item.TrechoID]
		if !ok || trecho.AssentosPreparados < item.Quantidade {
			g.muTrechos.Unlock()
			g.muOperacoes.Unlock()
			g.muTransacoes.Unlock()
			return *tx, fmt.Errorf("hold inválido no trecho %s", item.TrechoID)
		}
	}
	for _, item := range tx.Itens {
		trecho := g.Trechos[item.TrechoID]
		trecho.AssentosPreparados -= item.Quantidade
		trecho.AssentosOcupados += item.Quantidade
	}

	tx.Estado = EstadoTransacaoCommitConfirmado
	copia := *tx
	g.muTrechos.Unlock()
	g.muOperacoes.Unlock()
	g.muTransacoes.Unlock()

	if err := g.SalvarDados(); err != nil {
		return copia, err
	}
	return copia, nil
}

// AbortReservaLocal é idempotente e só libera holds próprios da transação.
func (g *GerenciadorDeRotas) AbortReservaLocal(txID string, motivo string) (TransacaoLocal, error) {
	g.muTransacoes.Lock()
	g.muOperacoes.Lock()
	g.muTrechos.Lock()

	tx, existe := g.Transacoes[txID]
	if !existe {
		g.muTrechos.Unlock()
		g.muOperacoes.Unlock()
		g.muTransacoes.Unlock()
		return TransacaoLocal{}, fmt.Errorf("transação %s não existe", txID)
	}
	if tx.Estado == EstadoTransacaoAbortado {
		copia := *tx
		g.muTrechos.Unlock()
		g.muOperacoes.Unlock()
		g.muTransacoes.Unlock()
		return copia, nil
	}
	if tx.Estado == EstadoTransacaoCommitConfirmado {
		g.muTrechos.Unlock()
		g.muOperacoes.Unlock()
		g.muTransacoes.Unlock()
		return *tx, fmt.Errorf("transação já confirmada não pode ser abortada")
	}

	for _, item := range tx.Itens {
		if trecho, ok := g.Trechos[item.TrechoID]; ok && trecho.AssentosPreparados >= item.Quantidade {
			trecho.AssentosPreparados -= item.Quantidade
		}
	}
	tx.Estado = EstadoTransacaoAbortado
	tx.Motivo = motivo
	copia := *tx

	g.muTrechos.Unlock()
	g.muOperacoes.Unlock()
	g.muTransacoes.Unlock()

	if err := g.SalvarDados(); err != nil {
		return copia, err
	}
	return copia, nil
}

func (g *GerenciadorDeRotas) ConsultarTransacaoLocal(txID string) (TransacaoLocal, bool) {
	g.muTransacoes.RLock()
	defer g.muTransacoes.RUnlock()
	tx, existe := g.Transacoes[txID]
	if !existe {
		return TransacaoLocal{}, false
	}
	copia := *tx
	copia.Itens = append([]ItemTransacao(nil), tx.Itens...)
	return copia, true
}
