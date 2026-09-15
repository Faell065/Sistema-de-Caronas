package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"projeto_redes/protocolo"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("      VAIJUNTO - MENU PRINCIPAL         ")
		fmt.Println("")
		fmt.Println("1 - Motorista")
		fmt.Println("2 - Passageiro")
		fmt.Println("0 - Sair")
		fmt.Print("Escolha uma opção: ")

		opcaoStr, _ := reader.ReadString('\n')
		opcaoStr = strings.TrimSpace(opcaoStr)

		if opcaoStr == "0" {
			fmt.Println("Saindo do sistema. Até logo!")
			break
		}

		switch opcaoStr {
		case "1":
			menuMotorista(reader)
		case "2":
			menuPassageiro(reader)
		default:
			fmt.Println("Opção inválida! Tente novamente.")
		}
	}
}

// enviarRequisicao encapsula a abertura do socket TCP, envio e leitura da resposta
func enviarRequisicao(req protocolo.Requisicao) (protocolo.Resposta, error) {
	// Conecta ao servidor TCP rodando na máquina local na porta 8080
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		return protocolo.Resposta{}, fmt.Errorf("não foi possível conectar ao servidor: %v", err)
	}
	defer conn.Close()

	// Serializa a struct Requisicao para formato JSON
	bytesReq, err := json.Marshal(req)
	if err != nil {
		return protocolo.Resposta{}, err
	}

	// Envia os bytes para o servidor via socket, adicionando '\n' como delimitador de fim de mensagem
	_, err = conn.Write(append(bytesReq, '\n'))
	if err != nil {
		return protocolo.Resposta{}, err
	}

	// Lê a resposta enviada de volta pelo servidor
	respostaBytes, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return protocolo.Resposta{}, err
	}

	// Desserializa os bytes recebidos para a struct Resposta do protocolo
	var resp protocolo.Resposta
	err = json.Unmarshal(respostaBytes, &resp)
	if err != nil {
		return protocolo.Resposta{}, err
	}

	return resp, nil
}

// menuMotorista gerencia as interações específicas para o motorista
func menuMotorista(reader *bufio.Reader) {
	fmt.Println("\n--- MENU MOTORISTA ---")
	fmt.Println("1 - Adicionar Rota")
	fmt.Println("0 - Voltar")
	fmt.Print("Escolha: ")

	op, _ := reader.ReadString('\n')
	op = strings.TrimSpace(op)

	if op == "1" {
		fmt.Print("Seu ID de Motorista (ex: joao_123): ")
		idMotorista, _ := reader.ReadString('\n')
		idMotorista = strings.TrimSpace(idMotorista)

		fmt.Print("Cidade de Origem: ")
		origem, _ := reader.ReadString('\n')
		origem = strings.TrimSpace(origem)

		fmt.Print("Cidade de Destino: ")
		destino, _ := reader.ReadString('\n')
		destino = strings.TrimSpace(destino)

		// Monta a requisição para cadastrar (vamos integrar no servidor em breve)
		req := protocolo.Requisicao{
			TipoAcao:    "CADASTRAR_ROTA",
			TipoUsuario: "motorista",
			IDUsuario:   idMotorista,
			Origem:      origem,
			Destino:     destino,
		}

		resp, err := enviarRequisicao(req)
		if err != nil {
			fmt.Println("Erro de comunicação com o servidor:", err)
			return
		}
		fmt.Printf("Servidor respondeu: %s\n", resp.Mensagem)
	}
}

// menuPassageiro gerencia as interações específicas para o passageiro
func menuPassageiro(reader *bufio.Reader) {
	fmt.Println("\n--- MENU PASSAGEIRO ---")
	fmt.Println("1 - Buscar Itinerário")
	fmt.Println("0 - Voltar")
	fmt.Print("Escolha: ")

	op, _ := reader.ReadString('\n')
	op = strings.TrimSpace(op)

	if op == "1" {
		fmt.Print("Sua Cidade Origem: ")
		origem, _ := reader.ReadString('\n')
		origem = strings.TrimSpace(origem)

		fmt.Print("Sua Cidade Destino: ")
		destino, _ := reader.ReadString('\n')
		destino = strings.TrimSpace(destino)

		// Monta a requisição baseada no nosso protocolo JSON
		req := protocolo.Requisicao{
			TipoAcao:    "BUSCAR_ITINERARIO",
			TipoUsuario: "passageiro",
			Origem:      origem,
			Destino:     destino,
			HorarioMin:  0, // Para teste inicial, aceita qualquer horário
		}

		fmt.Println("Buscando itinerários no servidor...")
		resp, err := enviarRequisicao(req)
		if err != nil {
			fmt.Println("Erro de conexão:", err)
			return
		}

		fmt.Printf("\n[RESPOSTA] %s\n", resp.Mensagem)
		if resp.Sucesso && resp.DadosJSON != "" {
			fmt.Printf("Itinerários encontrados (JSON Bruto): %s\n", resp.DadosJSON)
		}
	}
}