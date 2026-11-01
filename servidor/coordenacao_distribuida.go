package servidor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"projeto_redes/dominio"
	"sort"
	"strings"
	"time"
)

func (s *Servidor) ExecutarReservaDistribuida(passageiroID string, itens []dominio.ItemTransacao) (dominio.TransacaoCoordenador, error) {
	if s.Configuracao == nil {
		return dominio.TransacaoCoordenador{}, fmt.Errorf("configuração do servidor indisponível")
	}

	agrupados := make(map[string][]dominio.ItemTransacao)
	for _, item := range itens {
		if item.EmpresaID == "" || item.TrechoID == "" || item.Quantidade != 1 {
			return dominio.TransacaoCoordenador{}, fmt.Errorf("item distribuído inválido")
		}
		agrupados[item.EmpresaID] = append(agrupados[item.EmpresaID], item)
	}

	empresas := make([]string, 0, len(agrupados))
	for empresaID := range agrupados {
		empresas = append(empresas, empresaID)
	}
	sort.Strings(empresas)

	txID := fmt.Sprintf("tx_%s_%d", s.Configuracao.Empresa.ID, time.Now().UnixNano())
	chave := txID
	if err := s.Gerenciador.CriarTransacaoCoordenador(txID, chave, s.Configuracao.Empresa.ID, passageiroID, itens); err != nil {
		return dominio.TransacaoCoordenador{}, err
	}
	_ = s.Gerenciador.SetEstadoCoordenador(txID, dominio.EstadoCoordenadorPreparando)

	urls := make(map[string]string)
	for _, par := range append([]struct{ ID, Nome, URL string }{}, s.ParesParaSlice()...) {
		urls[par.ID] = par.URL
	}
	if _, ok := urls[s.Configuracao.Empresa.ID]; !ok {
		urls[s.Configuracao.Empresa.ID] = s.Configuracao.Empresa.URL
	}

	cliente := NovoClienteREST(s.Configuracao.Empresa.ID, s.Configuracao.ChaveHMAC, 1500*time.Millisecond)
	preparados := make([]string, 0, len(empresas))

	for _, empresaID := range empresas {
		txParticipante := dominio.TransacaoLocal{
			ID: txID, CoordenadorID: s.Configuracao.Empresa.ID, EmpresaID: empresaID,
			Operacao: "RESERVAR", IdempotencyKey: chave,
			PassageiroID: passageiroID, Lamport: s.proximoLamport(), Itens: agrupados[empresaID],
		}

		if empresaID == s.Configuracao.Empresa.ID {
			resultado, err := s.Gerenciador.PrepareReservaLocal(txParticipante)
			if err != nil {
				motivo := "prepare local recusado: " + err.Error()
				s.abortarPreparados(txID, preparados, urls, cliente, motivo)
				return s.estadoCoordenador(txID), fmt.Errorf("%s", motivo)
			}
			if resultado.Estado != dominio.EstadoTransacaoPreparado {
				motivo := "prepare local retornou estado inesperado"
				s.abortarPreparados(txID, preparados, urls, cliente, motivo)
				return s.estadoCoordenador(txID), fmt.Errorf("%s", motivo)
			}
			preparados = append(preparados, empresaID)
			_ = s.Gerenciador.SetEstadoParticipanteCoordenador(txID, empresaID, dominio.EstadoTransacaoPreparado)
			continue
		}

		url, ok := urls[empresaID]
		if !ok {
			motivo := fmt.Sprintf("empresa participante não configurada: %s", empresaID)
			s.abortarPreparados(txID, preparados, urls, cliente, motivo)
			return s.estadoCoordenador(txID), fmt.Errorf("%s", motivo)
		}
		status, body, err := cliente.ChamarComRepeticao(http.MethodPost,
			strings.TrimRight(url, "/")+"/api/v1/transacoes/"+txID+"/prepare",
			chave, txParticipante, 2)
		if err != nil || status >= 300 {
			motivo := fmt.Sprintf("prepare recusado ou indisponível em %s", empresaID)
			if err != nil {
				motivo += ": " + err.Error()
			} else {
				motivo += ": HTTP " + string(body)
			}
			s.abortarPreparados(txID, preparados, urls, cliente, motivo)
			return s.estadoCoordenador(txID), fmt.Errorf("%s", motivo)
		}
		preparados = append(preparados, empresaID)
		_ = s.Gerenciador.SetEstadoParticipanteCoordenador(txID, empresaID, dominio.EstadoTransacaoPreparado)
	}

	_ = s.Gerenciador.SetEstadoCoordenador(txID, dominio.EstadoCoordenadorPreparedAll)
	_ = s.Gerenciador.SetEstadoCoordenador(txID, dominio.EstadoCoordenadorCommitted)

	for _, empresaID := range empresas {
		if empresaID == s.Configuracao.Empresa.ID {
			if _, err := s.Gerenciador.CommitReservaLocal(txID); err != nil {
				return s.estadoCoordenador(txID), fmt.Errorf("falha ao confirmar participante local: %v", err)
			}
			_ = s.Gerenciador.SetEstadoParticipanteCoordenador(txID, empresaID, dominio.EstadoTransacaoCommitConfirmado)
			continue
		}

		url := urls[empresaID]
		status, _, err := cliente.ChamarComRepeticao(http.MethodPost,
			strings.TrimRight(url, "/")+"/api/v1/transacoes/"+txID+"/commit",
			chave, DecisaoTransacaoRest{ID: txID}, 3)
		if err != nil || status >= 300 {
			return s.estadoCoordenador(txID), fmt.Errorf("falha ao confirmar %s: status=%d err=%v", empresaID, status, err)
		}
		_ = s.Gerenciador.SetEstadoParticipanteCoordenador(txID, empresaID, dominio.EstadoTransacaoCommitConfirmado)
	}

	_ = s.Gerenciador.SetEstadoCoordenador(txID, dominio.EstadoCoordenadorConfirmado)
	if err := s.Gerenciador.CriarReservaDistribuida(txID, s.Configuracao.Empresa.ID, passageiroID, itens); err != nil {
		return s.estadoCoordenador(txID), err
	}
	return s.estadoCoordenador(txID), nil
}

func (s *Servidor) abortarPreparados(txID string, preparados []string, urls map[string]string, cliente *ClienteREST, motivo string) {
	_ = s.Gerenciador.SetEstadoCoordenador(txID, dominio.EstadoCoordenadorAborted)
	for _, empresaID := range preparados {
		if s.Configuracao != nil && empresaID == s.Configuracao.Empresa.ID {
			_, _ = s.Gerenciador.AbortReservaLocal(txID, motivo)
			continue
		}
		if url, ok := urls[empresaID]; ok {
			_, _, _ = cliente.ChamarComRepeticao(http.MethodPost,
				strings.TrimRight(url, "/")+"/api/v1/transacoes/"+txID+"/abort",
				txID, DecisaoTransacaoRest{ID: txID, Motivo: motivo}, 3)
		}
	}
}

func (s *Servidor) estadoCoordenador(txID string) dominio.TransacaoCoordenador {
	if tx, existe := s.Gerenciador.ConsultarTransacaoCoordenador(txID); existe {
		return tx
	}
	return dominio.TransacaoCoordenador{ID: txID}
}

func (s *Servidor) proximoLamport() uint64 {
	s.muLamport.Lock()
	defer s.muLamport.Unlock()
	s.Lamport++
	return s.Lamport
}

// ParesParaSlice adapta temporariamente os pares sem alterar configuracao.Empresa.
func (s *Servidor) ParesParaSlice() []struct{ ID, Nome, URL string } {
	if s.Configuracao == nil {
		return nil
	}
	resultado := make([]struct{ ID, Nome, URL string }, 0, len(s.Configuracao.Pares))
	for _, par := range s.Configuracao.Pares {
		resultado = append(resultado, struct{ ID, Nome, URL string }{ID: par.ID, Nome: par.Nome, URL: par.URL})
	}
	return resultado
}

func (s *Servidor) TransacaoEmJSON(tx dominio.TransacaoCoordenador) []byte {
	body, _ := json.Marshal(tx)
	return body
}

func (s *Servidor) RecuperarTransacoes() {
	if s.Configuracao == nil {
		return
	}
	transacoes := s.Gerenciador.TransacoesCoordenadorasParaRetomar()
	decisoes := s.Gerenciador.TransacoesComDecisaoPersistida()
	cliente := NovoClienteREST(s.Configuracao.Empresa.ID, s.Configuracao.ChaveHMAC, 1500*time.Millisecond)
	urls := make(map[string]string)
	for _, par := range s.ParesParaSlice() {
		urls[par.ID] = par.URL
	}
	for _, tx := range transacoes {
		if tx.Estado == dominio.EstadoCoordenadorCommitted || tx.Estado == dominio.EstadoCoordenadorPreparedAll {
			_ = s.Gerenciador.SetEstadoCoordenador(tx.ID, dominio.EstadoCoordenadorCommitted)
		}
	}
	for _, tx := range decisoes {
		if tx.Estado != dominio.EstadoCoordenadorCommitted && tx.Estado != dominio.EstadoCoordenadorAborted {
			continue
		}
		acao := "abort"
		if tx.Estado == dominio.EstadoCoordenadorCommitted {
			acao = "commit"
		}
		for _, participante := range tx.Participantes {
			if participante.EmpresaID == s.Configuracao.Empresa.ID {
				if acao == "commit" {
					_, _ = s.Gerenciador.CommitReservaLocal(tx.ID)
				} else {
					_, _ = s.Gerenciador.AbortReservaLocal(tx.ID, tx.Motivo)
				}
				continue
			}
			url, ok := urls[participante.EmpresaID]
			if !ok {
				continue
			}
			_, _, _ = cliente.ChamarComRepeticao(http.MethodPost,
				strings.TrimRight(url, "/")+"/api/v1/transacoes/"+tx.ID+"/"+acao,
				tx.IdempotencyKey, DecisaoTransacaoRest{ID: tx.ID, Motivo: tx.Motivo}, 3)
		}
	}
}
