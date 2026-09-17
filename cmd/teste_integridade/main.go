package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"projeto_redes/dominio"
	"projeto_redes/protocolo"
)

var enderecoServidor = "127.0.0.1:8080"

func main() {
	if len(os.Args) > 1 {
		enderecoServidor = os.Args[1]
		if !strings.Contains(enderecoServidor, ":") {
			enderecoServidor += ":8080"
		}
	}

	fmt.Println("==================================================")
	fmt.Printf("   TESTE DE INTEGRIDADE E REGRAS DE NEGÓCIO\n")
	fmt.Printf("   Alvo: %s\n", enderecoServidor)
	fmt.Println("==================================================\n")

	// SETUP: Cadastra Motorista
	respMot := enviarRequisicao(protocolo.Requisicao{
		TipoAcao:    "CADASTRAR_USUARIO",
		TipoUsuario: "motorista",
		Nome:        "Motorista Integridade",
		Senha:       "123",
		Tipo:        "motorista",
	})
	idMotorista := strings.ReplaceAll(respMot.DadosJSON, "\"", "")

	// SETUP: Cadastra Passageiro
	idPassageiro := fmt.Sprintf("pass_test_%d", time.Now().Unix())
	enviarRequisicao(protocolo.Requisicao{
		TipoAcao:    "CADASTRAR_USUARIO",
		TipoUsuario: "passageiro",
		Nome:        idPassageiro,
		Senha:       "123",
		Tipo:        "passageiro",
	})

	dataFutura := time.Now().AddDate(1, 0, 0).Format("02/01/2006")

	// ------------------------------------------------------------------
	// CENÁRIO 1: Testar Atomicidade (Reserva de Rota Multi-Trecho)
	// ------------------------------------------------------------------
	fmt.Println("--- CENÁRIO 1: Testando Atomicidade da Reserva Multi-Trecho ---")
	
	// Cria Rota A -> B (1 VAGA) e B -> C (5 VAGAS)
	enviarRequisicao(protocolo.Requisicao{
		TipoAcao:    "CADASTRAR_ROTA",
		TipoUsuario: "motorista",
		IDUsuario:   idMotorista,
		Data:        dataFutura,
		Paradas: []protocolo.ParadaRequisicao{
			{Cidade: "Cidade A", HorarioSaida: 800, AssentosTotais: 1, Valor: 10.0},
			{Cidade: "Cidade B", HorarioSaida: 1200, AssentosTotais: 5, Valor: 10.0},
			{Cidade: "Cidade C", HorarioSaida: 0, AssentosTotais: 0, Valor: 0.0},
		},
	})

	// Localiza o Itinerário (A -> B -> C)
	respBusca := enviarRequisicao(protocolo.Requisicao{
		TipoAcao:    "BUSCAR_ITINERARIO",
		TipoUsuario: "passageiro",
		Origem:      "Cidade A",
		Destino:     "Cidade C",
		Data:        dataFutura,
	})

	var itinerarios []dominio.Itinerario
	json.Unmarshal([]byte(respBusca.DadosJSON), &itinerarios)

	if len(itinerarios) == 0 || len(itinerarios[0].Trechos) < 2 {
		fmt.Println("  ❌ [FALHA] Não foi possível estruturar o itinerário de teste.")
		return
	}

	idTrechoAB := itinerarios[0].Trechos[0].ID
	idTrechoBC := itinerarios[0].Trechos[1].ID

	// Esgota a única vaga do Trecho A -> B efetuando uma reserva direta
	enviarRequisicao(protocolo.Requisicao{
		TipoAcao:   "RESERVAR_ITINERARIO",
		IDUsuario:  idPassageiro,
		IDsTrechos: []string{idTrechoAB},
	})
	fmt.Println("  [SETUP] Trecho A -> B foi completamente esgotado (0 vagas restantes).")

	// Tenta reservar a viagem COMPLETA (A -> B -> C)
	// Como A->B está esgotado mas B->C tem vagas, a operação DEVE FALHAR POR COMPLETO (Atomicidade)
	respReservaAtomica := enviarRequisicao(protocolo.Requisicao{
		TipoAcao:   "RESERVAR_ITINERARIO",
		IDUsuario:  idPassageiro,
		IDsTrechos: []string{idTrechoAB, idTrechoBC},
	})

	if !respReservaAtomica.Sucesso {
		fmt.Println("  🟢 [SUCESSO] Atomicidade Garantida! A reserva multi-trecho foi recusada por falta de vaga em um dos segmentos.")
	} else {
		fmt.Println("  🔴 [ERRO] Falha de Atomicidade! O sistema aceitou a reserva mesmo com um dos trechos esgotado.")
	}

	// ------------------------------------------------------------------
	// CENÁRIO 2: Restauração de Vagas após Cancelamento
	// ------------------------------------------------------------------
	fmt.Println("\n--- CENÁRIO 2: Cancelamento e Reciclagem de Vagas ---")

	// O passageiro cancela a primeira reserva que fez no Trecho A -> B
	// Precisamos buscar as reservas do passageiro para pegar o ID da Reserva
	respMinhasRes := enviarRequisicao(protocolo.Requisicao{
		TipoAcao:    "LISTAR_RESERVAS_PASSAGEIRO",
		TipoUsuario: "passageiro",
		IDUsuario:   idPassageiro,
	})

	var reservas []dominio.Reserva
	json.Unmarshal([]byte(respMinhasRes.DadosJSON), &reservas)

	if len(reservas) > 0 {
		idReservaParaCancelar := reservas[0].ID
		
		// Cancela
		respCancel := enviarRequisicao(protocolo.Requisicao{
			TipoAcao:  "CANCELAR_RESERVA_PASSAGEIRO",
			IDReserva: idReservaParaCancelar,
		})

		if respCancel.Sucesso {
			// Tenta reservar novamente o mesmo trecho A -> B para provar que a vaga voltou ao sistema
			respNovaReserva := enviarRequisicao(protocolo.Requisicao{
				TipoAcao:   "RESERVAR_ITINERARIO",
				IDUsuario:  idPassageiro,
				IDsTrechos: []string{idTrechoAB},
			})

			if respNovaReserva.Sucesso {
				fmt.Println("  🟢 [SUCESSO] Vaga liberada e reutilizada com sucesso após o cancelamento!")
			} else {
				fmt.Println("  🔴 [ERRO] A vaga não foi restaurada corretamente após o cancelamento.")
			}
		} else {
			fmt.Println("  🔴 [ERRO] Não foi possível cancelar a reserva.")
		}
	}

	fmt.Println("\n==================================================")
	fmt.Println("   FIM DOS TESTES DE INTEGRIDADE")
	fmt.Println("==================================================")
}

func enviarRequisicao(req protocolo.Requisicao) protocolo.Resposta {
	conn, err := net.Dial("tcp", enderecoServidor)
	if err != nil {
		return protocolo.Resposta{Sucesso: false, Mensagem: fmt.Sprintf("Erro de conexão com %s: %v", enderecoServidor, err)}
	}
	defer conn.Close()

	bytesReq, _ := json.Marshal(req)
	conn.Write(append(bytesReq, '\n'))

	respostaBytes, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return protocolo.Resposta{Sucesso: false, Mensagem: "Erro ao ler resposta"}
	}

	var resp protocolo.Resposta
	json.Unmarshal(respostaBytes, &resp)
	return resp
}