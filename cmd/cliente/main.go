//Sistema-de-Caronas/cmd/cliente/main.go
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
	"projeto_redes/protocolo"
	"projeto_redes/dominio"
)

// Variável global para armazenar o IP e porta do servidor
var enderecoServidor = "127.0.0.1:8080"

func main() {
	reader := bufio.NewReader(os.Stdin)

	// Permite passar o IP do servidor direto pelo terminal: go run main.go 192.168.1.15:8080
	if len(os.Args) > 1 {
		enderecoServidor = os.Args[1]
	} else {
		// Se não passar nada na execução, pergunta ao abrir o programa
		fmt.Print("Digite o IP do Servidor (Pressione ENTER para '127.0.0.1:8080'): ")
		ipInput, _ := reader.ReadString('\n')
		ipInput = strings.TrimSpace(ipInput)
		if ipInput != "" {
			if !strings.Contains(ipInput, ":") {
				enderecoServidor = ipInput + ":8080"
			} else {
				enderecoServidor = ipInput
			}
		}
	}

	fmt.Printf("[CONECTANDO] Servidor configurado para: %s\n", enderecoServidor)

	for {
		fmt.Println("\n========================================")
		fmt.Println("      VAIJUNTO - MENU PRINCIPAL         ")
		fmt.Println("========================================")
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
			fluxoAutenticacao(reader, "motorista")
		case "2":
			fluxoAutenticacao(reader, "passageiro")
		default:
			fmt.Println("Opção inválida! Tente novamente.")
		}
	}
}

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
			sucesso, idLogado := realizarLoginComRetorno(reader, tipoUsuario)
			if sucesso {
				if tipoUsuario == "motorista" {
					menuMotorista(reader, idLogado)
				} else {
					menuPassageiro(reader, idLogado)
				}
			}
		case "2":
			realizarCadastro(reader, tipoUsuario)
		default:
			fmt.Println("Opção inválida!")
		}
	}
}

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

func realizarLoginComRetorno(reader *bufio.Reader, tipoUsuario string) (bool, string) {
	fmt.Print("Insira seu ID de Usuário: ")
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
		return false, ""
	}

	fmt.Printf("\n[RESPOSTA] %s\n", resp.Mensagem)
	if resp.Sucesso {
		return true, idUsuario
	}
	return false, ""
}

func enviarRequisicao(req protocolo.Requisicao) (protocolo.Resposta, error) {
	conn, err := net.Dial("tcp", enderecoServidor)
	if err != nil {
		return protocolo.Resposta{}, fmt.Errorf("não foi possível conectar ao servidor: %v", err)
	}
	defer conn.Close()

	bytesReq, err := json.Marshal(req)
	if err != nil {
		return protocolo.Resposta{}, err
	}

	_, err = conn.Write(append(bytesReq, '\n'))
	if err != nil {
		return protocolo.Resposta{}, err
	}

	respostaBytes, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return protocolo.Resposta{}, err
	}

	var resp protocolo.Resposta
	err = json.Unmarshal(respostaBytes, &resp)
	if err != nil {
		return protocolo.Resposta{}, err
	}

	return resp, nil
}

func menuMotorista(reader *bufio.Reader, idMotorista string) {
	for {
		fmt.Printf("\n--- MOTORISTA LOGADO (%s) ---\n", idMotorista)
		fmt.Println("1 - Adicionar Rota Completa")
		fmt.Println("2 - Minhas Rotas Publicadas")
		fmt.Println("3 - Cancelar Rota Publicada")
		fmt.Println("0 - Deslogar")
		fmt.Print("Escolha: ")

		op, _ := reader.ReadString('\n')
		op = strings.TrimSpace(op)

		if op == "0" {
			break
		}
		switch op{
		case "1" :
			dataViagem := lerDataValida(reader, "Data da Viagem (DD/MM/AAAA): ")

			fmt.Println("\n--- Configuração de Paradas ---")
			var paradas []protocolo.ParadaRequisicao
			
			horarioAnterior := -1
			i := 1
			for {
				fmt.Printf("\nParada %d:\n", i)
				cidade := lerTextoValido(reader, "  Nome da Cidade: ")

				isUltima := false
				if len(paradas) >= 1 {
					fmt.Print("Deseja adicionar outra parada intermediária após esta? (s/n): ")
					respInterm, _ := reader.ReadString('\n')
					respInterm = strings.TrimSpace(respInterm)
					if strings.ToLower(respInterm) != "s" {
						isUltima = true
					}
				}

				horarioSaida := 0
				assentos := 0
				valor := 0.0

				// Se NÃO for o destino final, solicita dados do trecho de saída
				if !isUltima {
					horarioSaida = lerHorarioValido(reader, "  Horário de Saída (ex: 08:00 ou 800): ", horarioAnterior)
					horarioAnterior = horarioSaida

					assentos = lerIntValido(reader, "  Assentos disponíveis para o trecho que SAI desta cidade: ")
					valor = lerFloatValido(reader, "  Valor em R$ para o trecho que SAI desta cidade: ")
				}

				paradas = append(paradas, protocolo.ParadaRequisicao{
					Cidade:         cidade,
					HorarioSaida:   horarioSaida,
					AssentosTotais: assentos,
					Valor:          valor,
				})

				if isUltima {
					break
				}
				i++
			}

			if len(paradas) < 2 {
				fmt.Println("Erro: Uma rota precisa ter pelo menos origem e destino.")
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

		case "2" :
			req := protocolo.Requisicao{TipoAcao: "LISTAR_ROTAS_MOTORISTA", IDUsuario: idMotorista}
			resp, err := enviarRequisicao(req)
			if err != nil {
				fmt.Println("Erro de comunicação:", err)
				continue
			}

			var rotas map[string][]dominio.Trecho
			json.Unmarshal([]byte(resp.DadosJSON), &rotas)

			if len(rotas) == 0 {
				fmt.Println("\nNenhuma rota cadastrada no momento.")
				continue
			}

			fmt.Println("\n--- SUAS ROTAS PUBLICADAS ---")
			for rotaID, trechos := range rotas {
				fmt.Printf("\n[ROTA ID: %s] (%d trecho(s))\n", rotaID, len(trechos))
				for _, t := range trechos {
					fmt.Printf("   * %s -> %s | Saída: %s | Assentos: %d/%d ocupados | R$ %.2f\n",
						t.Origem, t.Destino, formatarHorario(t.HorarioSaida), t.AssentosOcupados, t.AssentosTotais, t.Valor)
				}
			}

		case "3":
			fmt.Print("\nDigite o ID da Rota que deseja cancelar: ")
			idRota, _ := reader.ReadString('\n')
			idRota = strings.TrimSpace(idRota)

			req := protocolo.Requisicao{TipoAcao: "CANCELAR_ROTA", IDUsuario: idMotorista, IDRota: idRota}
			resp, err := enviarRequisicao(req)
			if err != nil {
				fmt.Println("Erro de comunicação:", err)
				continue
			}
			fmt.Printf("\n[RESPOSTA] %s\n", resp.Mensagem)
		}
	}
}

func menuPassageiro(reader *bufio.Reader, idPassageiro string) {
	for {
		fmt.Printf("\n--- PASSAGEIRO LOGADO (%s) ---\n", idPassageiro)
		fmt.Println("1 - Buscar e Reservar Itinerário")
		fmt.Println("2 - Minhas Reservas")
		fmt.Println("3 - Cancelar Reserva")
		fmt.Println("0 - Deslogar")
		fmt.Print("Escolha: ")

		op, _ := reader.ReadString('\n')
		op = strings.TrimSpace(op)

		if op == "0" {
			break
		}

		switch op{
			case "1":
			origem := lerTextoValido(reader, "Origem desejada: ")
			destino := lerTextoValido(reader, "Destino desejado: ")
			data := lerDataValida(reader, "Data da Viagem (DD/MM/AAAA): ")

			req := protocolo.Requisicao{
				TipoAcao:    "BUSCAR_ITINERARIO",
				TipoUsuario: "passageiro",
				IDUsuario:   idPassageiro,
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
					fmt.Printf("    -> Trecho: %s para %s | Saída: %s | R$ %.2f (Motorista: %s)\n", 
						t.Origem, t.Destino, formatarHorario(t.HorarioSaida), t.Valor, t.MotoristaID)
				}
			}

			fmt.Print("\nDigite o número do itinerário que deseja reservar (ou 0 para cancelar): ")
			escolhaStr, _ := reader.ReadString('\n')
			escolha, _ := strconv.Atoi(strings.TrimSpace(escolhaStr))

			if escolha <= 0 || escolha > len(itinerarios) {
				fmt.Println("Operação cancelada.")
				continue
			}

			itinerarioEscolhido := itinerarios[escolha-1]

			var idsTrechos []string
			for _, t := range itinerarioEscolhido.Trechos {
				idsTrechos = append(idsTrechos, t.ID)
			}

			reqReserva := protocolo.Requisicao{
				TipoAcao:   "RESERVAR_ITINERARIO",
				IDUsuario:  idPassageiro,
				IDsTrechos: idsTrechos,
			}

			respReserva, err := enviarRequisicao(reqReserva)
			if err != nil {
				fmt.Println("Erro na reserva:", err)
				continue
			}

			fmt.Printf("\n[STATUS DA RESERVA] %s\n", respReserva.Mensagem)

		case "2":
			req := protocolo.Requisicao{TipoAcao: "LISTAR_RESERVAS_PASSAGEIRO", IDUsuario: idPassageiro}
			resp, err := enviarRequisicao(req)
			if err != nil {
				fmt.Println("Erro de comunicação:", err)
				continue
			}

			var reservas []map[string]interface{}
			json.Unmarshal([]byte(resp.DadosJSON), &reservas)

			if len(reservas) == 0 {
				fmt.Println("\nVocê não possui reservas ativas.")
				continue
			}

			fmt.Println("\n--- SUAS RESERVAS ATIVAS ---")
			for _, r := range reservas {
				fmt.Printf("\n[RESERVA ID: %v] - Valor Total: R$ %.2f\n", r["id_reserva"], r["valor_total"])
				
				// Desserializa os trechos da reserva
				bytesTrechos, _ := json.Marshal(r["trechos"])
				var trechos []dominio.Trecho
				json.Unmarshal(bytesTrechos, &trechos)

				for _, t := range trechos {
					fmt.Printf("   * Trecho: %s para %s | Saída: %s | Motorista: %s\n",
						t.Origem, t.Destino, formatarHorario(t.HorarioSaida), t.MotoristaID)
				}
			}

		case "3":
			fmt.Print("\nDigite o ID da Reserva que deseja cancelar: ")
			idReserva, _ := reader.ReadString('\n')
			idReserva = strings.TrimSpace(idReserva)

			req := protocolo.Requisicao{TipoAcao: "CANCELAR_RESERVA", IDUsuario: idPassageiro, IDReserva: idReserva}
			resp, err := enviarRequisicao(req)
			if err != nil {
				fmt.Println("Erro de comunicação:", err)
				continue
			}
			fmt.Printf("\n[STATUS DO CANCELAMENTO] %s\n", resp.Mensagem)	
		}
	}
}

// ============================================================================
// FUNÇÕES AUXILIARES DE VALIDAÇÃO E ENTRADA RECURSIVA/LOOP
// ============================================================================

func lerTextoValido(reader *bufio.Reader, mensagem string) string {
	for {
		fmt.Print(mensagem)
		texto, _ := reader.ReadString('\n')
		texto = strings.TrimSpace(texto)
		if texto != "" {
			return texto
		}
		fmt.Println(" Erro: O campo não pode ficar em branco. Tente novamente.")
	}
}

func lerDataValida(reader *bufio.Reader, mensagem string) string {
	for {
		fmt.Print(mensagem)
		entrada, _ := reader.ReadString('\n')
		entrada = strings.TrimSpace(entrada)

		t, err := time.Parse("02/01/2006", entrada)
		if err != nil {
			fmt.Println(" Erro: Data inválida! Use o formato DD/MM/AAAA (ex: 25/12/2026).")
			continue
		}

		// Zera o horário para comparar apenas o dia
		hoje := time.Now()
		hojeApenasDia := time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, hoje.Location())

		if t.Before(hojeApenasDia) {
			fmt.Println(" Erro: A data não pode ser inferior ao dia de hoje.")
			continue
		}

		return entrada
	}
}

func lerHorarioValido(reader *bufio.Reader, mensagem string, horarioMinimo int) int {
	for {
		fmt.Print(mensagem)
		entrada, _ := reader.ReadString('\n')
		entrada = strings.TrimSpace(entrada)
		entradaClean := strings.ReplaceAll(entrada, ":", "")

		val, err := strconv.Atoi(entradaClean)
		if err != nil {
			fmt.Println(" Erro: Horário inválido! Digite apenas números (ex: 800 ou 08:00).")
			continue
		}

		hora := val / 100
		minuto := val % 100

		if hora < 0 || hora > 23 || minuto < 0 || minuto > 59 {
			fmt.Println(" Erro: Horário inexistente! A hora deve ser entre 00 e 23 e os minutos entre 00 e 59.")
			continue
		}

		if horarioMinimo >= 0 && val <= horarioMinimo {
			fmt.Printf(" Erro: O horário de saída (%s) precisa ser posterior ao horário do trecho anterior (%s).\n", 
				formatarHorario(val), formatarHorario(horarioMinimo))
			continue
		}

		return val
	}
}

func lerIntValido(reader *bufio.Reader, mensagem string) int {
	for {
		fmt.Print(mensagem)
		entrada, _ := reader.ReadString('\n')
		entrada = strings.TrimSpace(entrada)

		val, err := strconv.Atoi(entrada)
		if err != nil || val <= 0 {
			fmt.Println(" Erro: Digite um número inteiro maior que zero.")
			continue
		}

		return val
	}
}

func lerFloatValido(reader *bufio.Reader, mensagem string) float64 {
	for {
		fmt.Print(mensagem)
		entrada, _ := reader.ReadString('\n')
		entrada = strings.TrimSpace(entrada)
		entrada = strings.ReplaceAll(entrada, ",", ".")

		val, err := strconv.ParseFloat(entrada, 64)
		if err != nil || val < 0 {
			fmt.Println(" Erro: Digite um valor numérico válido (ex: 15.50).")
			continue
		}

		return val
	}
}

func formatarHorario(h int) string {
	hora := h / 100
	minuto := h % 100
	return fmt.Sprintf("%02d:%02d", hora, minuto)
}