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

func NovoServidor() *Servidor {
	Gerenciador := dominio.NovoGerenciador()
	return &Servidor{Gerenciador: Gerenciador}
}

func (s *Servidor) Iniciar(porta string) {
	listener, err := net.Listen("tcp", ":"+porta)
	if err != nil {
		fmt.Printf("Erro ao iniciar o servidor na porta %s: %v\n", porta, err)
		return
	}
	defer listener.Close()
	fmt.Printf("Servidor TCP rodando e escutando na porta %s\n", porta)
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Erro ao aceitar conexão:", err)
			continue
		}
		go s.atenderCliente(conn)
	}
}

func (s *Servidor) atenderCliente(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	mensagemBytes, err := reader.ReadBytes('\n')
	if err != nil {
		fmt.Println("Erro ao ler mensagem do cliente:", err)
		return
	}

	var req protocolo.Requisicao
	err = json.Unmarshal(mensagemBytes, &req)
	if err != nil {
		fmt.Println("Erro ao desserializar a requisição:", err)
		s.enviarResposta(conn, protocolo.Resposta{Sucesso: false, Mensagem: "JSON inválida"})
		return
	}
	fmt.Printf("[REDE] Ação recebida: %s do usuário: %s\n", req.TipoAcao, req.IDUsuario)

	var resp protocolo.Resposta

	switch req.TipoAcao {

	case "CADASTRAR_USUARIO":
		idGerado, err := s.Gerenciador.CadastrarUsuario(req.Nome, req.Senha, req.Tipo)
		if err != nil {
			resp = protocolo.Resposta{Sucesso: false, Mensagem: err.Error()}
		} else {
			resp = protocolo.Resposta{
				Sucesso:   true,
				Mensagem:  fmt.Sprintf("Usuário cadastrado com sucesso! Seu ID gerado é: %s", idGerado),
				DadosJSON: idGerado,
			}
		}

	case "LOGIN":
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
		sucesso, msg, afetadas := s.Gerenciador.CancelarRotaMotorista(req.IDUsuario, req.IDRota)
		if sucesso && s.Configuracao != nil && len(afetadas) > 0 {
			cliente := NovoClienteREST(s.Configuracao.Empresa.ID, s.Configuracao.ChaveHMAC, 1500*time.Millisecond)
			urls := make(map[string]string)
			for _, par := range s.Configuracao.Pares {
				urls[par.ID] = par.URL
			}
			urls[s.Configuracao.Empresa.ID] = s.Configuracao.Empresa.URL

			for _, af := range afetadas {
				if af.CoordenadorID == s.Configuracao.Empresa.ID {
					// Nós somos o coordenador! O motorista era nosso mesmo.
					// Deletamos a reserva do passageiro local e mandamos as outras empresas liberarem a vaga.
					if itens, existe := s.Gerenciador.RemoverReservaForcada(af.TxID); existe {
						for _, item := range itens {
							if item.EmpresaID == s.Configuracao.Empresa.ID {
								_, _ = s.Gerenciador.CancelarReservaLocal(af.TxID)
							} else if url, ok := urls[item.EmpresaID]; ok {
								_, _, _ = cliente.ChamarComRepeticao(http.MethodPost,
									strings.TrimRight(url, "/")+"/api/v1/transacoes/"+af.TxID+"/cancelar",
									af.TxID, DecisaoTransacaoRest{ID: af.TxID}, 3)
							}
						}
					}
				} else if url, ok := urls[af.CoordenadorID]; ok {
					// O Coordenador da reserva é outro servidor! Avisamos ele para forçar o cancelamento em cascata.
					_, _, _ = cliente.ChamarComRepeticao(http.MethodPost,
						strings.TrimRight(url, "/")+"/api/v1/transacoes/"+af.TxID+"/forcar_cancelamento",
						af.TxID, DecisaoTransacaoRest{ID: af.TxID}, 3)
				}
			}
		}
		resp = protocolo.Resposta{Sucesso: sucesso, Mensagem: msg}

	case "RESERVAR_ITINERARIO":
		if len(req.ItensTrecho) > 0 && s.Configuracao != nil {
			itens := make([]dominio.ItemTransacao, 0, len(req.ItensTrecho))
			trechosDet := make([]dominio.Trecho, 0, len(req.ItensTrecho))
			var valorTotal float64

			for _, item := range req.ItensTrecho {
				itens = append(itens, dominio.ItemTransacao{
					EmpresaID: item.EmpresaID, TrechoID: item.TrechoID, Quantidade: item.Quantidade,
				})
				trechosDet = append(trechosDet, dominio.Trecho{
					ID:             item.TrechoID,
					EmpresaID:      item.EmpresaID,
					Origem:         item.Origem,
					Destino:        item.Destino,
					Data:           item.Data,
					HorarioSaida:   item.HorarioSaida,
					HorarioChegada: item.HorarioChegada,
					Valor:          item.Valor,
					MotoristaID:    item.MotoristaID,
				})
				valorTotal += item.Valor
			}

			tx, err := s.ExecutarReservaDistribuida(req.IDUsuario, itens, trechosDet, valorTotal)
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

	s.enviarResposta(conn, resp)
}

func (s *Servidor) enviarResposta(conn net.Conn, resp protocolo.Resposta) {
	bytesResp, _ := json.Marshal(resp)
	conn.Write(append(bytesResp, '\n'))
}