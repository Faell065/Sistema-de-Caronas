//Sistema-de-Caronas/dominio/persistencia.go
package dominio

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SalvarDados grava o estado atual das memórias nos arquivos JSON
func (g *GerenciadorDeRotas) SalvarDados() error {
	// Garante que a pasta 'dados' existe antes de salvar
	err := os.MkdirAll(g.diretorioDados, 0755)
	if err != nil {
		return err
	}
	// Salva Usuários
	bytesUsuarios, err := json.MarshalIndent(g.Usuarios, "", "  ")
	if err == nil {
		os.WriteFile(filepath.Join(g.diretorioDados, "usuarios.json"), bytesUsuarios, 0644)
	}

	// Salva Trechos
	bytesTrechos, err := json.MarshalIndent(g.Trechos, "", "  ")
	if err == nil {
		os.WriteFile(filepath.Join(g.diretorioDados, "trechos.json"), bytesTrechos, 0644)
	}

	// Salva Reservas
	bytesReservas, err := json.MarshalIndent(g.Reservas, "", "  ")
	if err == nil {
		os.WriteFile(filepath.Join(g.diretorioDados, "reservas.json"), bytesReservas, 0644)
	}

	return nil
}

// CarregarDados restaura o estado do sistema a partir dos arquivos JSON ao ligar o servidor
func (g *GerenciadorDeRotas) CarregarDados() {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Carrega Usuários
	if bytesUsuarios, err := os.ReadFile(filepath.Join(g.diretorioDados, "usuarios.json")); err == nil {
		json.Unmarshal(bytesUsuarios, &g.Usuarios)
	}

	// Carrega Trechos
	if bytesTrechos, err := os.ReadFile(filepath.Join(g.diretorioDados, "trechos.json")); err == nil {
		json.Unmarshal(bytesTrechos, &g.Trechos)
	}

	// Carrega Reservas
	if bytesReservas, err := os.ReadFile(filepath.Join(g.diretorioDados, "reservas.json")); err == nil {
		json.Unmarshal(bytesReservas, &g.Reservas)
	}

	fmt.Println("[SISTEMA] Dados carregados do disco com sucesso!")
}