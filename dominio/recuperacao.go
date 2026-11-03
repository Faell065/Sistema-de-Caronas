package dominio

// TransacoesQuePrecisamRecuperacao varre o banco local à procura de logs do 2PC que
// foram interrompidos na fase PREPARED caso o servidor sofra um crash (queda de energia).
func (g *GerenciadorDeRotas) TransacoesQuePrecisamRecuperacao() []TransacaoLocal {
	g.muTransacoes.RLock()
	defer g.muTransacoes.RUnlock()
	resultado := make([]TransacaoLocal, 0)
	for _, tx := range g.Transacoes {
		if tx.Estado == EstadoTransacaoPreparado || tx.Estado == EstadoTransacaoCommitPersistido {
			copia := *tx
			copia.Itens = append([]ItemTransacao(nil), tx.Itens...)
			resultado = append(resultado, copia)
		}
	}
	return resultado
}

// TransacoesCoordenadorasParaRetomar recupera transações globais pausadas.
func (g *GerenciadorDeRotas) TransacoesCoordenadorasParaRetomar() []TransacaoCoordenador {
	g.muTransacoes.RLock()
	defer g.muTransacoes.RUnlock()
	resultado := make([]TransacaoCoordenador, 0)
	for _, tx := range g.TransacoesCoordenador {
		if tx.Estado == EstadoCoordenadorPreparedAll || tx.Estado == EstadoCoordenadorCommitted || tx.Estado == EstadoCoordenadorPreparando {
			copia := *tx
			copia.Itens = append([]ItemTransacao(nil), tx.Itens...)
			copia.Participantes = append([]EstadoParticipante(nil), tx.Participantes...)
			resultado = append(resultado, copia)
		}
	}
	return resultado
}

// TransacoesComDecisaoPersistida resgata as respostas (COMMIT ou ABORT) e reencontra os logs.
func (g *GerenciadorDeRotas) TransacoesComDecisaoPersistida() []TransacaoCoordenador {
	g.muTransacoes.RLock()
	defer g.muTransacoes.RUnlock()
	resultado := make([]TransacaoCoordenador, 0)
	for _, tx := range g.TransacoesCoordenador {
		if tx.Estado == EstadoCoordenadorCommitted || tx.Estado == EstadoCoordenadorAborted || tx.Estado == EstadoCoordenadorConfirmado {
			copia := *tx
			copia.Itens = append([]ItemTransacao(nil), tx.Itens...)
			copia.Participantes = append([]EstadoParticipante(nil), tx.Participantes...)
			resultado = append(resultado, copia)
		}
	}
	return resultado
}