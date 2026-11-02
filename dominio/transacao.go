package dominio

import (
	"fmt"
	"time"
)

// PrepareReservaLocal cria holds persistíveis nos trechos locais
func (g *GerenciadorDeRotas) PrepareReservaLocal(tx TransacaoLocal) (TransacaoLocal, error) {
	if tx.ID == "" {
		return tx, fmt.Errorf("tx_id é obrigatório")
	}
	if tx.Operacao != "RESERVAR" {
		return tx, fmt.Errorf("operação não suportada: %s", tx.Operacao)
	}

	g.muOperacoes.Lock()
	g.muTransacoes.Lock()
	g.muTrechos.Lock()
	defer g.muTrechos.Unlock()
	defer g.muTransacoes.Unlock()
	defer g.muOperacoes.Unlock()

	if existente, ok := g.Transacoes[tx.ID]; ok {
		if existente.Estado == EstadoTransacaoAbortado {
			return *existente, fmt.Errorf("transação foi abortada")
		}
		if existente.Estado == EstadoTransacaoPreparado || existente.Estado == EstadoTransacaoCommitConfirmado || existente.Estado == EstadoTransacaoCommitPersistido {
			mesmoConteudo := existente.IdempotencyKey == tx.IdempotencyKey && existente.Operacao == tx.Operacao && len(existente.Itens) == len(tx.Itens)
			for i := 0; mesmoConteudo && i < len(existente.Itens); i++ {
				mesmoConteudo = existente.Itens[i] == tx.Itens[i]
			}
			if !mesmoConteudo {
				return *existente, fmt.Errorf("mesmo tx_id usado com conteúdo diferente")
			}
			copia := *existente
			return copia, nil
		}
	}

	g.SequenciaPrecedencia++
	tx.SequenciaProprietario = g.SequenciaPrecedencia
	tx.CriadoEm = time.Now().Unix()

	for _, item := range tx.Itens {
		if item.Quantidade != 1 {
			return tx, fmt.Errorf("quantidade inválida para trecho %s", item.TrechoID)
		}
		trecho, existe := g.Trechos[item.TrechoID]
		if !existe {
			return tx, fmt.Errorf("trecho %s não existe", item.TrechoID)
		}
		livres := trecho.AssentosTotais - trecho.AssentosOcupados - trecho.AssentosPreparados
		if livres < item.Quantidade {
			return tx, fmt.Errorf("sem vaga disponível em %s", item.TrechoID)
		}
	}

	for _, item := range tx.Itens {
		g.Trechos[item.TrechoID].AssentosPreparados += item.Quantidade
	}
	tx.Papel = "PARTICIPANTE"
	tx.Estado = EstadoTransacaoPreparado
	g.Transacoes[tx.ID] = &tx

	if err := g.SalvarDados(); err != nil {
		return tx, err
	}
	return tx, nil
}

// CommitReservaLocal converte os holds em ocupação confirmada
func (g *GerenciadorDeRotas) CommitReservaLocal(txID string) (TransacaoLocal, error) {
	g.muOperacoes.Lock()
	g.muTransacoes.Lock()
	g.muTrechos.Lock()
	defer g.muTrechos.Unlock()
	defer g.muTransacoes.Unlock()
	defer g.muOperacoes.Unlock()

	tx, existe := g.Transacoes[txID]
	if !existe {
		return TransacaoLocal{}, fmt.Errorf("transação %s não existe", txID)
	}
	if tx.Estado == EstadoTransacaoAbortado {
		return *tx, fmt.Errorf("transação já foi abortada")
	}
	if tx.Estado == EstadoTransacaoCommitConfirmado {
		copia := *tx
		return copia, nil
	}
	if tx.Estado != EstadoTransacaoPreparado && tx.Estado != EstadoTransacaoCommitPersistido {
		return *tx, fmt.Errorf("estado inválido para commit: %s", tx.Estado)
	}

	for _, item := range tx.Itens {
		trecho, ok := g.Trechos[item.TrechoID]
		if !ok || trecho.AssentosPreparados < item.Quantidade {
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

	if err := g.SalvarDados(); err != nil {
		return copia, err
	}
	return copia, nil
}

// AbortReservaLocal libera holds próprios da transação
func (g *GerenciadorDeRotas) AbortReservaLocal(txID string, motivo string) (TransacaoLocal, error) {
	g.muOperacoes.Lock()
	g.muTransacoes.Lock()
	g.muTrechos.Lock()
	defer g.muTrechos.Unlock()
	defer g.muTransacoes.Unlock()
	defer g.muOperacoes.Unlock()

	tx, existe := g.Transacoes[txID]
	if !existe {
		return TransacaoLocal{}, fmt.Errorf("transação %s não existe", txID)
	}
	if tx.Estado == EstadoTransacaoAbortado {
		copia := *tx
		return copia, nil
	}
	if tx.Estado == EstadoTransacaoCommitConfirmado {
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

func (g *GerenciadorDeRotas) CancelarReservaLocal(txID string) (TransacaoLocal, error) {
	g.muOperacoes.Lock()
	g.muTransacoes.Lock()
	g.muTrechos.Lock()
	defer g.muTrechos.Unlock()
	defer g.muTransacoes.Unlock()
	defer g.muOperacoes.Unlock()

	tx, existe := g.Transacoes[txID]
	if !existe {
		return TransacaoLocal{}, fmt.Errorf("transação %s não existe", txID)
	}
	if tx.Estado == EstadoTransacaoCancelado {
		return *tx, nil
	}

	for _, item := range tx.Itens {
		if trecho, ok := g.Trechos[item.TrechoID]; ok && trecho.AssentosOcupados >= item.Quantidade {
			trecho.AssentosOcupados -= item.Quantidade
		}
	}
	tx.Estado = EstadoTransacaoCancelado
	copia := *tx

	_ = g.SalvarDados()
	return copia, nil
}

// LimparTransacoesExpiradas aborta transações preparadas que excederam o TTL
func (g *GerenciadorDeRotas) LimparTransacoesExpiradas(timeoutSegundos int64) int {
	agora := time.Now().Unix()
	var expiradas []string

	g.muTransacoes.RLock()
	for id, tx := range g.Transacoes {
		if tx.Estado == EstadoTransacaoPreparado && tx.CriadoEm > 0 && (agora-tx.CriadoEm) > timeoutSegundos {
			expiradas = append(expiradas, id)
		}
	}
	g.muTransacoes.RUnlock()

	for _, id := range expiradas {
		_, _ = g.AbortReservaLocal(id, "timeout - expirada por inatividade do participante")
	}
	return len(expiradas)
}