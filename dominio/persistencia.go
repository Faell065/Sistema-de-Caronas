package dominio

import (
	"encoding/json"
	"fmt"
	"os"
)

// SalvarDados grava o estado atual das memórias nos arquivos JSON
func (g *GerenciadorDeRotas) SalvarDados() error {
	// Salva Usuários
	bytesUsuarios, err := json.MarshalIndent(g.Usuarios, "", "  ")
	if err == nil {
		os.WriteFile("usuarios.json", bytesUsuarios, 0644)
	}

	// Salva Trechos
	bytesTrechos, err := json.MarshalIndent(g.Trechos, "", "  ")
	if err == nil {
		os.WriteFile("trechos.json", bytesTrechos, 0644)
	}

	// Salva Reservas
	bytesReservas, err := json.MarshalIndent(g.Reservas, "", "  ")
	if err == nil {
		os.WriteFile("reservas.json", bytesReservas, 0644)
	}

	return nil
}

// CarregarDados restaura o estado do sistema a partir dos arquivos JSON ao ligar o servidor
func (g *GerenciadorDeRotas) CarregarDados() {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Carrega Usuários
	if bytesUsuarios, err := os.ReadFile("usuarios.json"); err == nil {
		json.Unmarshal(bytesUsuarios, &g.Usuarios)
	}

	// Carrega Trechos
	if bytesTrechos, err := os.ReadFile("trechos.json"); err == nil {
		json.Unmarshal(bytesTrechos, &g.Trechos)
	}

	// Carrega Reservas
	if bytesReservas, err := os.ReadFile("reservas.json"); err == nil {
		json.Unmarshal(bytesReservas, &g.Reservas)
	}

	fmt.Println("[SISTEMA] Dados carregados do disco com sucesso!")
}