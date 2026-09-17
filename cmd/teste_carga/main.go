//Sistema-de-Caronas/cmd/servidor/main.go
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
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
	fmt.Printf("   INICIANDO TESTE DE CONCORRÊNCIA E ESTRESSE\n")
	fmt.Printf("   Alvo: %s\n", enderecoServidor)
	fmt.Println("==================================================")

	// 1. Cadastra Motorista e captura o ID dinâmico gerado pelo servidor
	respMot := enviarRequisicao(protocolo.Requisicao{
		TipoAcao:    "CADASTRAR_USUARIO",
		TipoUsuario: "motorista",
		Nome:        "Motorista Estresse",
		Senha:       "123",
		Tipo:        "motorista",
	})

	idMotorista := strings.ReplaceAll(respMot.DadosJSON, "\"", "") // Remove aspas caso venham no JSON
	if idMotorista == "" {
		fmt.Println("[ERRO] Não foi possível capturar o ID do Motorista.")
		return
	}
	fmt.Printf("[SETUP] Motorista de teste cadastrado com ID: %s\n", idMotorista)

	// 2. Cadastra uma Rota com APENAS 2 VAGAS
	dataFutura := time.Now().AddDate(1, 0, 0).Format("02/01/2006")
	respRota := enviarRequisicao(protocolo.Requisicao{
		TipoAcao:    "CADASTRAR_ROTA",
		TipoUsuario: "motorista",
		IDUsuario:   idMotorista, // Usa o ID dinâmico capturado
		Data:        dataFutura,
		Paradas: []protocolo.ParadaRequisicao{
			{Cidade: "Feira Teste", HorarioSaida: 800, AssentosTotais: 2, Valor: 25.0},
			{Cidade: "Salvador Teste", HorarioSaida: 0, AssentosTotais: 0, Valor: 0.0},
		},
	})

	if !respRota.Sucesso {
		fmt.Printf("[ERRO AO CRIAR ROTA] %s\n", respRota.Mensagem)
		return
	}

	// 3. Busca o itinerário que acabamos de criar para capturar o ID do Trecho Real
	respBusca := enviarRequisicao(protocolo.Requisicao{
		TipoAcao:    "BUSCAR_ITINERARIO",
		TipoUsuario: "passageiro",
		Origem:      "Feira Teste",
		Destino:     "Salvador Teste",
		Data:        dataFutura,
	})

	var itinerarios []dominio.Itinerario
	json.Unmarshal([]byte(respBusca.DadosJSON), &itinerarios)

	if len(itinerarios) == 0 || len(itinerarios[0].Trechos) == 0 {
		fmt.Println("[ERRO] Falha ao encontrar a rota recém-criada no sistema.")
		return
	}

	idTrechoCriado := itinerarios[0].Trechos[0].ID
	fmt.Printf("[SETUP] Rota localizada! ID do trecho alvo: '%s' (Apenas 2 vagas disponíveis)\n\n", idTrechoCriado)

	// 4. Dispara 10 Passageiros Simultâneos tentando reservar a vaga ao mesmo tempo
	numClientesConcorrentes := 10
	var wg sync.WaitGroup

	fmt.Printf("[DISPARO] Lançando %d conexões TCP concorrentes simultâneas...\n\n", numClientesConcorrentes)
	timeInicio := time.Now()

	for i := 1; i <= numClientesConcorrentes; i++ {
		wg.Add(1)
		idPassageiro := fmt.Sprintf("passageiro_simultaneo_%d_%d", time.Now().UnixNano(), i)

		go func(id string) {
			defer wg.Done()

			// Cadastra o passageiro concorrente
			enviarRequisicao(protocolo.Requisicao{
				TipoAcao:    "CADASTRAR_USUARIO",
				TipoUsuario: "passageiro",
				Nome:        id,
				Senha:       "123",
				Tipo:        "passageiro",
			})

			// Tenta reservar o trecho concorrido
			respReserva := enviarRequisicao(protocolo.Requisicao{
				TipoAcao:   "RESERVAR_ITINERARIO",
				IDUsuario:  id, // Pode passar o nome pois no teste ignoramos a autenticação rigorosa
				IDsTrechos: []string{idTrechoCriado},
			})

			if respReserva.Sucesso {
				fmt.Printf("  🟢 [SUCESSO] O cliente %s CONSEGUIU a vaga!\n", id)
			} else {
				fmt.Printf("  🔴 [RECUSADO] O cliente %s NÃO conseguiu: %s\n", id, respReserva.Mensagem)
			}
		}(idPassageiro)
	}

	wg.Wait()
	fmt.Printf("\n[FIM DO TESTE] Concluído em %v! (Apenas 2 clientes devem ter obtido sucesso)\n", time.Since(timeInicio))
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