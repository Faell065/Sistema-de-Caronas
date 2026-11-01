// Sistema-de-Caronas/dominio/persistencia.go
package dominio

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var persistenciaMu sync.Mutex

func cloneUsuarios(src map[string]*Usuario) map[string]*Usuario {
	dst := make(map[string]*Usuario, len(src))
	for id, usuario := range src {
		copiado := *usuario
		dst[id] = &copiado
	}
	return dst
}

func cloneTrechos(src map[string]*Trecho) map[string]*Trecho {
	dst := make(map[string]*Trecho, len(src))
	for id, trecho := range src {
		copiado := *trecho
		dst[id] = &copiado
	}
	return dst
}

func cloneReservas(src map[string]*Reserva) map[string]*Reserva {
	dst := make(map[string]*Reserva, len(src))
	for id, reserva := range src {
		copiado := *reserva
		copiado.IDsTrechos = append([]string(nil), reserva.IDsTrechos...)
		copiado.Itens = append([]ItemTransacao(nil), reserva.Itens...)
		dst[id] = &copiado
	}
	return dst
}

func cloneTransacoes(src map[string]*TransacaoLocal) map[string]*TransacaoLocal {
	dst := make(map[string]*TransacaoLocal, len(src))
	for id, transacao := range src {
		copiado := *transacao
		copiado.Itens = append([]ItemTransacao(nil), transacao.Itens...)
		dst[id] = &copiado
	}
	return dst
}

func cloneTransacoesCoordenador(src map[string]*TransacaoCoordenador) map[string]*TransacaoCoordenador {
	dst := make(map[string]*TransacaoCoordenador, len(src))
	for id, transacao := range src {
		copiado := *transacao
		copiado.Itens = append([]ItemTransacao(nil), transacao.Itens...)
		copiado.Participantes = append([]EstadoParticipante(nil), transacao.Participantes...)
		dst[id] = &copiado
	}
	return dst
}

// SalvarDados cria snapshots antes de acessar o disco. Assim persistência não
// mantém uma trava do gerenciador durante operações de arquivo.
func (g *GerenciadorDeRotas) SalvarDados() error {
	g.muUsuarios.RLock()
	usuarios := cloneUsuarios(g.Usuarios)
	g.muUsuarios.RUnlock()

	g.muTrechos.RLock()
	trechos := cloneTrechos(g.Trechos)
	g.muTrechos.RUnlock()

	g.muReservas.RLock()
	reservas := cloneReservas(g.Reservas)
	g.muReservas.RUnlock()

	g.muTransacoes.RLock()
	transacoes := cloneTransacoes(g.Transacoes)
	transacoesCoordenador := cloneTransacoesCoordenador(g.TransacoesCoordenador)
	g.muTransacoes.RUnlock()

	persistenciaMu.Lock()
	defer persistenciaMu.Unlock()

	if err := os.MkdirAll(g.diretorioDados, 0755); err != nil {
		return err
	}

	for arquivo, dados := range map[string]interface{}{
		"usuarios.json":               usuarios,
		"trechos.json":                trechos,
		"reservas.json":               reservas,
		"transacoes.json":             transacoes,
		"transacoes_coordenador.json": transacoesCoordenador,
	} {
		bytes, err := json.MarshalIndent(dados, "", "  ")
		if err != nil {
			return err
		}
		destino := filepath.Join(g.diretorioDados, arquivo)
		temporario, err := os.CreateTemp(g.diretorioDados, arquivo+".tmp-")
		if err != nil {
			return err
		}
		if err := temporario.Chmod(0644); err != nil {
			temporario.Close()
			return err
		}
		if _, err := temporario.Write(bytes); err != nil {
			temporario.Close()
			return err
		}
		if err := temporario.Sync(); err != nil {
			temporario.Close()
			return err
		}
		if err := temporario.Close(); err != nil {
			return err
		}
		if err := os.Rename(temporario.Name(), destino); err != nil {
			return err
		}
	}

	return nil
}

func (g *GerenciadorDeRotas) CarregarDados() {
	g.muUsuarios.Lock()
	if dados, err := os.ReadFile(filepath.Join(g.diretorioDados, "usuarios.json")); err == nil {
		_ = json.Unmarshal(dados, &g.Usuarios)
	}
	if g.Usuarios == nil {
		g.Usuarios = make(map[string]*Usuario)
	}
	g.muUsuarios.Unlock()

	g.muTrechos.Lock()
	if dados, err := os.ReadFile(filepath.Join(g.diretorioDados, "trechos.json")); err == nil {
		_ = json.Unmarshal(dados, &g.Trechos)
	}
	if g.Trechos == nil {
		g.Trechos = make(map[string]*Trecho)
	}
	g.muTrechos.Unlock()

	g.muReservas.Lock()
	if dados, err := os.ReadFile(filepath.Join(g.diretorioDados, "reservas.json")); err == nil {
		_ = json.Unmarshal(dados, &g.Reservas)
	}
	if g.Reservas == nil {
		g.Reservas = make(map[string]*Reserva)
	}
	g.muReservas.Unlock()

	g.muTransacoes.Lock()
	if dados, err := os.ReadFile(filepath.Join(g.diretorioDados, "transacoes.json")); err == nil {
		_ = json.Unmarshal(dados, &g.Transacoes)
	}
	if g.Transacoes == nil {
		g.Transacoes = make(map[string]*TransacaoLocal)
	}
	for _, tx := range g.Transacoes {
		if tx.SequenciaProprietario > g.SequenciaPrecedencia {
			g.SequenciaPrecedencia = tx.SequenciaProprietario
		}
	}
	g.muTransacoes.Unlock()

	g.muTransacoes.Lock()
	if dados, err := os.ReadFile(filepath.Join(g.diretorioDados, "transacoes_coordenador.json")); err == nil {
		_ = json.Unmarshal(dados, &g.TransacoesCoordenador)
	}
	if g.TransacoesCoordenador == nil {
		g.TransacoesCoordenador = make(map[string]*TransacaoCoordenador)
	}
	g.muTransacoes.Unlock()

	fmt.Println("[SISTEMA] Dados carregados do disco com sucesso!")
}
