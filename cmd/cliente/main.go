//Sistema-de-Caronas/cmd/cliente/main.go
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"projeto_redes/protocolo"
	"projeto_redes/dominio"
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

// fluxoAutenticacao lida com a escolha entre Login ou Cadastro para o tipo de usuário selecionado
func fluxoAutenticacao(reader *bufio.Reader, tipoUsuario string) {
	for {
		fmt.Printf("\n--- MENU %s ---\n", strings.ToUpper(tipoUsuario))
		fmt.Println("1 - Login")
		fmt.Println("2 - Cadastro")
		fmt.Println("0 - Voltar")
		fmt.Print("Escolha uma opção: ")

		op, _ := reader.ReadString('\n')
		op = strings.TrimSpace(op)

		if op == "0" {
			break
		}

		switch op {
		case "1":
			if realizarLogin(reader, tipoUsuario) {
				// Se logou com sucesso, entra no menu específico daquele papel
				if tipoUsuario == "motorista" {
					menuMotorista(reader)
				} else {
					menuPassageiro(reader)
				}
			}
		case "2":
			realizarCadastro(reader, tipoUsuario)
		default:
			fmt.Println("Opção inválida!")
		}
	}
}

// realizarCadastro envia os dados para o servidor criar a conta e gerar o ID único
func realizarCadastro(reader *bufio.Reader, tipoUsuario string) {
	fmt.Print("Insira seu Nome Completo: ")
	nome, _ := reader.ReadString('\n')
	nome = strings.TrimSpace(nome)

	fmt.Print("Insira uma Senha: ")
	senha, _ := reader.ReadString('\n')
	senha = strings.TrimSpace(senha)

	req := protocolo.Requisicao{
		TipoAcao:    "CADASTRAR_USUARIO",
		TipoUsuario: tipoUsuario,
		Nome:        nome,
		Senha:       senha,
		Tipo:        tipoUsuario,
	}

	resp, err := enviarRequisicao(req)
	if err != nil {
		fmt.Println("Erro de comunicação com o servidor:", err)
		return
	}

	fmt.Printf("\n[RESPOSTA] %s\n", resp.Mensagem)
	if resp.Sucesso {
		fmt.Printf("-> ANOTE SEU ID DE ACESSO: **%s**\n", resp.DadosJSON)
	}
}

// realizarLogin valida o ID e senha informados no servidor
func realizarLogin(reader *bufio.Reader, tipoUsuario string) bool {
	fmt.Print("Insira seu ID de Usuário (ex: joao_1234): ")
	idUsuario, _ := reader.ReadString('\n')
	idUsuario = strings.TrimSpace(idUsuario)

	fmt.Print("Insira sua Senha: ")
	senha, _ := reader.ReadString('\n')
	senha = strings.TrimSpace(senha)

	req := protocolo.Requisicao{
		TipoAcao:    "LOGIN",
		TipoUsuario: tipoUsuario,
		IDUsuario:   idUsuario,
		Senha:       senha,
	}

	resp, err := enviarRequisicao(req)
	if err != nil {
		fmt.Println("Erro de comunicação com o servidor:", err)
		return false
	}

	fmt.Printf("\n[RESPOSTA] %s\n", resp.Mensagem)
	return resp.Sucesso
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
	for {
		fmt.Println("\n--- MOTORISTA (LOGADO) ---")
		fmt.Println("1 - Adicionar Rota Completa (Origem, Destino, Paradas e Preços)")
		fmt.Println("0 - Deslogar")
		fmt.Print("Escolha: ")

		op, _ := reader.ReadString('\n')
		op = strings.TrimSpace(op)

		if op == "0" {
			break
		}

		if op == "1" {
			fmt.Print("Seu ID de Motorista: ")
			idMotorista, _ := reader.ReadString('\n')
			idMotorista = strings.TrimSpace(idMotorista)

			fmt.Print("Data da Viagem (ex: 2026-06-15): ")
			dataViagem, _ := reader.ReadString('\n')
			dataViagem = strings.TrimSpace(dataViagem)

			fmt.Println("\n--- Configuração de Paradas ---")
			fmt.Println("Você precisa informar pelo menos 2 paradas (Ex: Origem e Destino).")
			
			var paradas []protocolo.ParadaRequisicao
			
			for i := 1; ; i++ {
				fmt.Printf("\nParada %d:\n", i)
				fmt.Print("  Nome da Cidade (ou digite 'fim' para encerrar): ")
				cidade, _ := reader.ReadString('\n')
				cidade = strings.TrimSpace(cidade)

				if strings.ToLower(cidade) == "fim" {
					break
				}

				fmt.Print("  Horário de Saída desta cidade (ex: 800): ")
				var horarioSaida int
				fmt.Scanln(&horarioSaida)

				var assentos int
				var valor float64

				// Se não for a última parada, ela tem um trecho que sai dela para a próxima
				fmt.Print("  Assentos disponíveis para o trecho que SAI desta cidade: ")
				fmt.Scanln(&assentos)

				fmt.Print("  Valor em R$ para o trecho que SAI desta cidade: ")
				fmt.Scanln(&valor)
				reader.ReadString('\n') // Limpa o buffer

				paradas = append(paradas, protocolo.ParadaRequisicao{
					Cidade:         cidade,
					HorarioSaida:   horarioSaida,
					AssentosTotais: assentos,
					Valor:          valor,
				})

				// Se já tivermos 2 ou mais, perguntamos se deseja adicionar mais paradas intermediárias
				if len(paradas) >= 2 {
					fmt.Print("Deseja adicionar outra parada intermediária? (s/n): ")
					respContinuar, _ := reader.ReadString('\n')
					respContinuar = strings.TrimSpace(respContinuar)
					if strings.ToLower(respContinuar) != "s" {
						break
					}
				}
			}

			if len(paradas) < 2 {
				fmt.Println("Erro: Uma rota precisa ter pelo menos 2 paradas (Origem e Destino). Operação cancelada.")
				continue
			}

			req := protocolo.Requisicao{
				TipoAcao:    "CADASTRAR_ROTA",
				TipoUsuario: "motorista",
				IDUsuario:   idMotorista,
				Data:        dataViagem,
				Paradas:     paradas,
			}

			resp, err := enviarRequisicao(req)
			if err != nil {
				fmt.Println("Erro de comunicação:", err)
				continue
			}
			fmt.Printf("\n[RESPOSTA] %s\n", resp.Mensagem)
		}
	}
}

// menuPassageiro gerencia as interações específicas para o passageiro
func menuPassageiro(reader *bufio.Reader) {
	for {
		fmt.Println("\n--- PASSAGEIRO (LOGADO) ---")
		fmt.Println("1 - Buscar e Reservar Itinerário")
		fmt.Println("0 - Deslogar")
		fmt.Print("Escolha: ")

		op, _ := reader.ReadString('\n')
		op = strings.TrimSpace(op)

		if op == "0" {
			break
		}

		if op == "1" {
			fmt.Print("Origem desejada: ")
			origem, _ := reader.ReadString('\n')
			origem = strings.TrimSpace(origem)

			fmt.Print("Destino desejado: ")
			destino, _ := reader.ReadString('\n')
			destino = strings.TrimSpace(destino)

			fmt.Print("Data da Viagem (ex: 2026-06-15): ")
			data, _ := reader.ReadString('\n')
			data = strings.TrimSpace(data)

			req := protocolo.Requisicao{
				TipoAcao:    "BUSCAR_ITINERARIO",
				TipoUsuario: "passageiro",
				Origem:      origem,
				Destino:     destino,
				Data:        data,
			}

			resp, err := enviarRequisicao(req)
			if err != nil {
				fmt.Println("Erro de comunicação:", err)
				continue
			}

			if !resp.Sucesso || resp.DadosJSON == "" || resp.DadosJSON == "null" {
				fmt.Println("\nNenhum itinerário encontrado para esta rota e data.")
				continue
			}

			var itinerarios []dominio.Itinerario
			json.Unmarshal([]byte(resp.DadosJSON), &itinerarios)

			fmt.Printf("\n--- ITINERÁRIOS DISPONÍVEIS (%d) ---\n", len(itinerarios))
			for i, itin := range itinerarios {
				fmt.Printf("[%d] Valor Total: R$ %.2f | Vagas Disponíveis: %d\n", i+1, itin.ValorTotal, itin.AssentosDisponivel)
				for _, t := range itin.Trechos {
					fmt.Printf("    -> Trecho: %s para %s | Saída: %d | R$ %.2f (Motorista: %s)\n", 
						t.Origem, t.Destino, t.HorarioSaida, t.Valor, t.MotoristaID)
				}
			}

			fmt.Print("\nDigite o número do itinerário que deseja reservar (ou 0 para cancelar): ")
			var escolha int
			fmt.Scanln(&escolha)
			reader.ReadString('\n') // Limpa buffer

			if escolha <= 0 || escolha > len(itinerarios) {
				fmt.Println("Operação cancelada.")
				continue
			}

			itinerarioEscolhido := itinerarios[escolha-1]

			// Coleta os IDs de todos os trechos que compõem o itinerário escolhido
			var idsTrechos []string
			for _, t := range itinerarioEscolhido.Trechos {
				idsTrechos = append(idsTrechos, t.ID)
			}

			reqReserva := protocolo.Requisicao{
				TipoAcao:   "RESERVAR_ITINERARIO",
				IDsTrechos: idsTrechos,
			}

			respReserva, err := enviarRequisicao(reqReserva)
			if err != nil {
				fmt.Println("Erro na reserva:", err)
				continue
			}

			fmt.Printf("\n[STATUS DA RESERVA] %s\n", respReserva.Mensagem)
		}
	}
}