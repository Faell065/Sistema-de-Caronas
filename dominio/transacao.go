package dominio

import (
	"fmt"
	"time"
)

// PrepareReservaLocal atua como o PARTICIPANTE na Fase 1 do Two-Phase Commit.
// Ele realiza a trava lógica (hold) das vagas incrementando "AssentosPreparados".
func (g *GerenciadorDeRotas) PrepareReservaLocal(tx TransacaoLocal) (TransacaoLocal, error) {
	if tx.ID == "" {
		return tx, fmt.Errorf("tx_id é obrigatório")
	}
	if tx.Operacao != "RESERVAR" {
		return tx, fmt.Errorf("operação não suportada: %s", tx.Operacao)
	}

	// Ordem Estrita dos Locks (Operacoes -> Transacoes -> Trechos)
	// Essencial para evitar Deadlocks de múltiplas chamadas concorrentes.
	g.muOperacoes.Lock()
	g.muTransacoes.Lock()
	g.muTrechos.Lock()

	// Checa Idempotência: caso a rede falhe e o coordenador reenvie o mesmo ID,
	// evita abater a vaga física mais de uma vez.
	if existente, ok := g.Transacoes[tx.ID]; ok {
		if existente.Estado == EstadoTransacaoAbortado {
			g.muTrechos.Unlock()
			g.muTransacoes.Unlock()
			g.muOperacoes.Unlock()
			return *existente, fmt.Errorf("transação foi abortada")
		}
		if existente.Estado == EstadoTransacaoPreparado || existente.Estado == EstadoTransacaoCommitConfirmado || existente.Estado == EstadoTransacaoCommitPersistido {
			mesmoConteudo := existente.IdempotencyKey == tx.IdempotencyKey && existente.Operacao == tx.Operacao && len(existente.Itens) == len(tx.Itens)
			for i := 0; mesmoConteudo && i < len(existente.Itens); i++ {
				mesmoConteudo = existente.Itens[i] == tx.Itens[i]
			}
			if !mesmoConteudo {
				g.muTrechos.Unlock()
				g.muTransacoes.Unlock()
				g.muOperacoes.Unlock()
				return *existente, fmt.Errorf("mesmo tx_id usado com conteúdo diferente")
			}
			copia := *existente
			g.muTrechos.Unlock()
			g.muTransacoes.Unlock()
			g.muOperacoes.Unlock()
			return copia, nil
		}
	}

	g.SequenciaPrecedencia++
	tx.SequenciaProprietario = g.SequenciaPrecedencia
	tx.CriadoEm = time.Now().Unix()

	// Validação de vagas físicas 
	for _, item := range tx.Itens {
		if item.Quantidade != 1 {
			g.muTrechos.Unlock()
			g.muTransacoes.Unlock()
			g.muOperacoes.Unlock()
			return tx, fmt.Errorf("quantidade inválida para trecho %s", item.TrechoID)
		}
		trecho, existe := g.Trechos[item.TrechoID]
		if !existe {
			g.muTrechos.Unlock()
			g.muTransacoes.Unlock()
			g.muOperacoes.Unlock()
			return tx, fmt.Errorf("trecho %s não existe", item.TrechoID)
		}
		livres := trecho.AssentosTotais - trecho.AssentosOcupados - trecho.AssentosPreparados
		if livres < item.Quantidade {
			g.muTrechos.Unlock()
			g.muTransacoes.Unlock()
			g.muOperacoes.Unlock()
			return tx, fmt.Errorf("sem vaga disponível em %s", item.TrechoID)
		}
	}

	// Fase de Retenção Atômica (Holds)
	for _, item := range tx.Itens {
		g.Trechos[item.TrechoID].AssentosPreparados += item.Quantidade
	}
	tx.Papel = "PARTICIPANTE"
	tx.Estado = EstadoTransacaoPreparado
	g.Transacoes[tx.ID] = &tx

	// DESTRAVA TUDO ANTES DE ACESSAR O DISCO
	// Evita um Deadlock de Reentrância fatal se o SalvarDados tentar dar RLock no mesmo Goroutine!
	g.muTrechos.Unlock()
	g.muTransacoes.Unlock()
	g.muOperacoes.Unlock()

	if err := g.SalvarDados(); err != nil {
		return tx, err
	}
	return tx, nil
}

// CommitReservaLocal converte os holds (Preparados) em Ocupação Confirmada física (Fase 2).
func (g *GerenciadorDeRotas) CommitReservaLocal(txID string) (TransacaoLocal, error) {
	g.muOperacoes.Lock()
	g.muTransacoes.Lock()
	g.muTrechos.Lock()

	tx, existe := g.Transacoes[txID]
	if !existe {
		g.muTrechos.Unlock()
		g.muTransacoes.Unlock()
		g.muOperacoes.Unlock()
		return TransacaoLocal{}, fmt.Errorf("transação %s não existe", txID)
	}
	if tx.Estado == EstadoTransacaoAbortado {
		g.muTrechos.Unlock()
		g.muTransacoes.Unlock()
		g.muOperacoes.Unlock()
		return *tx, fmt.Errorf("transação já foi abortada")
	}
	if tx.Estado == EstadoTransacaoCommitConfirmado {
		copia := *tx
		g.muTrechos.Unlock()
		g.muTransacoes.Unlock()
		g.muOperacoes.Unlock()
		return copia, nil
	}
	if tx.Estado != EstadoTransacaoPreparado && tx.Estado != EstadoTransacaoCommitPersistido {
		g.muTrechos.Unlock()
		g.muTransacoes.Unlock()
		g.muOperacoes.Unlock()
		return *tx, fmt.Errorf("estado inválido para commit: %s", tx.Estado)
	}

	for _, item := range tx.Itens {
		trecho, ok := g.Trechos[item.TrechoID]
		if !ok || trecho.AssentosPreparados < item.Quantidade {
			g.muTrechos.Unlock()
			g.muTransacoes.Unlock()
			g.muOperacoes.Unlock()
			return *tx, fmt.Errorf("hold inválido no trecho %s", item.TrechoID)
		}
	}
	// Permuta os status dos Assentos ativando o assento definitivo
	for _, item := range tx.Itens {
		trecho := g.Trechos[item.TrechoID]
		trecho.AssentosPreparados -= item.Quantidade
		trecho.AssentosOcupados += item.Quantidade
	}

	tx.Estado = EstadoTransacaoCommitConfirmado
	copia := *tx

	// DESTRAVA TUDO ANTES DE ACESSAR O DISCO
	g.muTrechos.Unlock()
	g.muTransacoes.Unlock()
	g.muOperacoes.Unlock()

	if err := g.SalvarDados(); err != nil {
		return copia, err
	}
	return copia, nil
}

// AbortReservaLocal desfaz os holds caso um nó na rede tenha falhado (Rollback).
func (g *GerenciadorDeRotas) AbortReservaLocal(txID string, motivo string) (TransacaoLocal, error) {
	g.muOperacoes.Lock()
	g.muTransacoes.Lock()
	g.muTrechos.Lock()

	tx, existe := g.Transacoes[txID]
	if !existe {
		g.muTrechos.Unlock()
		g.muTransacoes.Unlock()
		g.muOperacoes.Unlock()
		return TransacaoLocal{}, fmt.Errorf("transação %s não existe", txID)
	}
	if tx.Estado == EstadoTransacaoAbortado {
		copia := *tx
		g.muTrechos.Unlock()
		g.muTransacoes.Unlock()
		g.muOperacoes.Unlock()
		return copia, nil
	}
	if tx.Estado == EstadoTransacaoCommitConfirmado {
		g.muTrechos.Unlock()
		g.muTransacoes.Unlock()
		g.muOperacoes.Unlock()
		return *tx, fmt.Errorf("transação já confirmada não pode ser abortada")
	}

	// Libera os assentos "Zumbis"
	for _, item := range tx.Itens {
		if trecho, ok := g.Trechos[item.TrechoID]; ok && trecho.AssentosPreparados >= item.Quantidade {
			trecho.AssentosPreparados -= item.Quantidade
		}
	}
	tx.Estado = EstadoTransacaoAbortado
	tx.Motivo = motivo
	copia := *tx

	// DESTRAVA TUDO ANTES DE ACESSAR O DISCO
	g.muTrechos.Unlock()
	g.muTransacoes.Unlock()
	g.muOperacoes.Unlock()

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

	tx, existe := g.Transacoes[txID]
	if !existe {
		g.muTrechos.Unlock()
		g.muTransacoes.Unlock()
		g.muOperacoes.Unlock()
		return TransacaoLocal{}, fmt.Errorf("transação %s não existe", txID)
	}
	if tx.Estado == EstadoTransacaoCancelado {
		copia := *tx
		g.muTrechos.Unlock()
		g.muTransacoes.Unlock()
		g.muOperacoes.Unlock()
		return copia, nil
	}

	for _, item := range tx.Itens {
		if trecho, ok := g.Trechos[item.TrechoID]; ok && trecho.AssentosOcupados >= item.Quantidade {
			trecho.AssentosOcupados -= item.Quantidade
		}
	}
	tx.Estado = EstadoTransacaoCancelado
	copia := *tx

	// DESTRAVA TUDO ANTES DE ACESSAR O DISCO
	g.muTrechos.Unlock()
	g.muTransacoes.Unlock()
	g.muOperacoes.Unlock()

	_ = g.SalvarDados()
	return copia, nil
}

// LimparTransacoesExpiradas aborta transações preparadas que excederam o TTL
// Evita "Assentos Zumbis": assentos presos em Holds infinitos se a rede cair durante o Prepare.
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