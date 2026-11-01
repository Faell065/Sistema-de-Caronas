// Sistema-de-Caronas/servidor/servidor.go
package servidor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"projeto_redes/configuracao"
	"projeto_redes/dominio"
	"projeto_redes/protocolo"
)

type Servidor struct {
	Gerenciador  *dominio.GerenciadorDeRotas
	Configuracao *configuracao.Configuracao

	muLamport sync.Mutex
	Lamport   uint64
}

// Novo Servidor cria um novo servidor com um gerenciador de rotas
func NovoServidor() *Servidor {
	Gerenciador := dominio.NovoGerenciador()
	return &Servidor{Gerenciador: Gerenciador}
}

// Função para iniciar o servidor e escutar por conexões
func (s *Servidor) Iniciar(porta string) {
	// Inicia o listener TCP na porta especificada
	listener, err := net.Listen("tcp", ":"+porta) // anteriormente estava "localhost:"+porta, mas para aceitar conexões externas, usamos ":"+porta
	if err != nil {
		fmt.Printf("Erro ao iniciar o servidor na porta %s: %v\n", porta, err)
		return
	}
	defer listener.Close()
	fmt.Printf("Servidor TCP rodando e escutando na porta %s\n", porta)
	for {
		// Aceita novas conexões de clientes (trava aqui até alguém conectar)
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Erro ao aceitar conexão:", err)
			continue
		}
		// Cria uma goroutine para lidar com a conexão do cliente de forma assincrona/concorrente
		go s.atenderCliente(conn)
	}
}

// Metodo da classe Servidor para atender a conexão de um cliente (gerencia a conversa individual com cada um)
func (s *Servidor) atenderCliente(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	mensagemBytes, err := reader.ReadBytes('\n') // lê até o final da linha
	if err != nil {
		fmt.Println("Erro ao ler mensagem do cliente:", err)
		return
	}

	// Desserializa o JSON recebido para a struct Requisicao do protocolo
	var req protocolo.Requisicao
	err = json.Unmarshal(mensagemBytes, &req)
	if err != nil {
		fmt.Println("Erro ao desserializar a requisição:", err)
		s.enviarResposta(conn, protocolo.Resposta{Sucesso: false, Mensagem: "JSON inválida"})
		return
	}
	fmt.Printf("[REDE] Ação recebida: %s do usuário: %s\n", req.TipoAcao, req.IDUsuario)

	// Processa a regra de negócios com base na ação solicitada
	var resp protocolo.Resposta

	switch req.TipoAcao {

	case "CADASTRAR_USUARIO":
		// Chama o gerenciador para criar o usuário e gerar o ID único
		idGerado, err := s.Gerenciador.CadastrarUsuario(req.Nome, req.Senha, req.Tipo)
		if err != nil {
			resp = protocolo.Resposta{Sucesso: false, Mensagem: err.Error()}
		} else {
			resp = protocolo.Resposta{
				Sucesso:   true,
				Mensagem:  fmt.Sprintf("Usuário cadastrado com sucesso! Seu ID gerado é: %s", idGerado),
				DadosJSON: idGerado, // Retorna o ID gerado para que o cliente possa guardá-lo
			}
		}

	case "LOGIN":
		// Valida as credenciais e o tipo de conta no gerenciador
		valido, usuario := s.Gerenciador.Autenticar(req.IDUsuario, req.Senha, req.TipoUsuario)
		if !valido {
			resp = protocolo.Resposta{Sucesso: false, Mensagem: "ID, senha incorretos ou tipo de conta incompatível."}
		} else {
			resp = protocolo.Resposta{
				Sucesso:  true,
				Mensagem: fmt.Sprintf("Login realizado com sucesso! Bem-vindo, %s.", usuario.Nome),
			}
		}

	case "BUSCAR_ITINERARIO":
		// Busca itinerários com base na origem, destino e data informados
		itinerarios := s.Gerenciador.BuscarItinerarios(req.Origem, req.Destino, req.Data)
		if s.Configuracao != nil {
			busca := s.BuscarDistribuido(req.Origem, req.Destino, req.Data)
			itinerariosDistribuidos := make([]dominio.Itinerario, 0, len(busca.Caminhos))
			for _, caminho := range busca.Caminhos {
				trechos := make([]dominio.Trecho, 0, len(caminho.Trechos))
				for _, trechoRest := range caminho.Trechos {
					trechos = append(trechos, dominio.Trecho{
						ID:             trechoRest.TrechoID,
						RotaID:         trechoRest.RotaID,
						MotoristaID:    trechoRest.MotoristaID,
						Origem:         trechoRest.Origem,
						Destino:        trechoRest.Destino,
						Data:           trechoRest.Data,
						HorarioSaida:   trechoRest.HorarioSaida,
						HorarioChegada: trechoRest.HorarioChegada,
						AssentosTotais: trechoRest.AssentosDisponiveis,
						Valor:          trechoRest.Valor,
						EmpresaID:      trechoRest.EmpresaID,
						Recurso:        trechoRest.Recurso,
					})
				}
				itinerariosDistribuidos = append(itinerariosDistribuidos, dominio.Itinerario{
					ID:                 caminho.ID,
					Trechos:            trechos,
					ValorTotal:         caminho.ValorTotal,
					AssentosDisponivel: caminho.AssentosDisponiveis,
				})
			}
			itinerarios = itinerariosDistribuidos
		}
		dadosBytes, _ := json.Marshal(itinerarios)
		resp = protocolo.Resposta{
			Sucesso: true, Mensagem: fmt.Sprintf("Encontrados %d itinerários.", len(itinerarios)),
			DadosJSON: string(dadosBytes),
		}

	case "LISTAR_ROTAS_MOTORISTA":
		rotas := s.Gerenciador.ListarRotasMotorista(req.IDUsuario)
		bytesData, _ := json.Marshal(rotas)
		resp = protocolo.Resposta{
			Sucesso:   true,
			Mensagem:  fmt.Sprintf("Encontradas %d rotas.", len(rotas)),
			DadosJSON: string(bytesData),
		}

	case "CANCELAR_ROTA":
		sucesso, msg := s.Gerenciador.CancelarRotaMotorista(req.IDUsuario, req.IDRota)
		resp = protocolo.Resposta{Sucesso: sucesso, Mensagem: msg}

	case "RESERVAR_ITINERARIO":
		if len(req.ItensTrecho) > 0 && s.Configuracao != nil {
			itens := make([]dominio.ItemTransacao, 0, len(req.ItensTrecho))
			for _, item := range req.ItensTrecho {
				itens = append(itens, dominio.ItemTransacao{
					EmpresaID: item.EmpresaID, TrechoID: item.TrechoID, Quantidade: item.Quantidade,
				})
			}
			tx, err := s.ExecutarReservaDistribuida(req.IDUsuario, itens)
			if err != nil {
				resp = protocolo.Resposta{Sucesso: false, Mensagem: err.Error(), DadosJSON: tx.ID}
			} else {
				resp = protocolo.Resposta{
					Sucesso: true, Mensagem: "Itinerário reservado com sucesso!",
					DadosJSON: tx.ID,
				}
			}
			break
		}
		sucesso, msg, resID := s.Gerenciador.ReservarItinerarioComID(req.IDUsuario, req.IDsTrechos)
		resp = protocolo.Resposta{
			Sucesso: sucesso, Mensagem: fmt.Sprintf("%s (ID da Reserva: %s)", msg, resID), DadosJSON: resID,
		}

	case "LISTAR_RESERVAS_PASSAGEIRO":
		reservas := s.Gerenciador.ListarReservasPassageiro(req.IDUsuario)
		bytesData, _ := json.Marshal(reservas)
		resp = protocolo.Resposta{
			Sucesso:   true,
			Mensagem:  fmt.Sprintf("Encontradas %d reservas.", len(reservas)),
			DadosJSON: string(bytesData),
		}

	case "CANCELAR_RESERVA":
		if s.Configuracao != nil {
			if itens, existe := s.Gerenciador.RemoverReservaDistribuida(req.IDUsuario, req.IDReserva); existe {
				for _, item := range itens {
					if item.EmpresaID == s.Configuracao.Empresa.ID {
						_, _ = s.Gerenciador.CancelarReservaLocal(req.IDReserva)
						continue
					}
					for _, par := range s.Configuracao.Pares {
						if par.ID == item.EmpresaID {
							cliente := NovoClienteREST(s.Configuracao.Empresa.ID, s.Configuracao.ChaveHMAC, 1500*time.Millisecond)
							_, _, _ = cliente.ChamarComRepeticao(http.MethodPost,
								strings.TrimRight(par.URL, "/")+"/api/v1/transacoes/"+req.IDReserva+"/cancelar",
								req.IDReserva, DecisaoTransacaoRest{ID: req.IDReserva}, 3)
						}
					}
				}
				resp = protocolo.Resposta{Sucesso: true, Mensagem: "Reserva distribuída cancelada."}
				break
			}
		}
		sucesso, msg := s.Gerenciador.CancelarReservaPassageiro(req.IDUsuario, req.IDReserva)
		resp = protocolo.Resposta{Sucesso: sucesso, Mensagem: msg}

	case "CADASTRAR_ROTA":
		// Valida se o ID do motorista foi enviado e existe
		if req.IDUsuario == "" {
			resp = protocolo.Resposta{Sucesso: false, Mensagem: "Usuário não autenticado."}
			break
		}

		var paradasDominio []dominio.ParadaRota
		for _, p := range req.Paradas {
			paradasDominio = append(paradasDominio, dominio.ParadaRota{
				Cidade:         p.Cidade,
				HorarioSaida:   p.HorarioSaida,
				AssentosTotais: p.AssentosTotais,
				Valor:          p.Valor,
			})
		}

		idsGerados := s.Gerenciador.CadastrarRotaCompleta(req.IDUsuario, req.Data, paradasDominio)

		if len(idsGerados) == 0 {
			resp = protocolo.Resposta{
				Sucesso:  false,
				Mensagem: "A rota precisa ter pelo menos origem e destino (2 paradas).",
			}
		} else {
			resp = protocolo.Resposta{
				Sucesso:   true,
				Mensagem:  fmt.Sprintf("Rota cadastrada com sucesso! Foram gerados %d trechos independentes.", len(idsGerados)),
				DadosJSON: fmt.Sprintf("%v", idsGerados),
			}
		}

	default:
		resp = protocolo.Resposta{Sucesso: false, Mensagem: "Ação desconhecida"}
	}

	// Envia a resposta de volta ao cliente via socket
	s.enviarResposta(conn, resp)
}

// enviarResposta serializa a resposta em JSON e envia pela rede com uma quebra de linha delimitadora
func (s *Servidor) enviarResposta(conn net.Conn, resp protocolo.Resposta) {
	bytesResp, _ := json.Marshal(resp)
	// Adicionamos '\n' no final para o cliente saber onde a mensagem termina
	conn.Write(append(bytesResp, '\n'))
}
