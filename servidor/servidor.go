package servidor

import (
	"fmt"
	"bufio"
	"time"
	"encoding/json"
	"net"
	"projeto_redes/dominio" 
	"projeto_redes/protocolo"
)

// O server tem de encapsular o gerenciador de rotas, que é o responsável por armazenar as rotas e trechos, para poder gerenciar as conxões de rede
type Servidor struct {
	Gerenciador *dominio.GerenciadorDeRotas
}

//Novo Servidor cria um novo servidor com um gerenciador de rotas
func NovoServidor() *Servidor{
	Gerenciador := dominio.NovoGerenciador()
	return &Servidor{Gerenciador: Gerenciador}
}

// Função para iniciar o servidor e escutar por conexões
func (s *Servidor) Iniciar(porta string) {
	// Inicia o listener TCP na porta especificada
	listener, err := net.Listen("tcp", ":"+porta)
	if err != nil {
		fmt.Println("Erro ao iniciar o servidor na porta %s: %v\n:",porta, err)
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
		// Valida as credenciais no gerenciador
		valido, usuario := s.Gerenciador.Autenticar(req.IDUsuario, req.Senha)
		if !valido {
			resp = protocolo.Resposta{Sucesso: false, Mensagem: "ID ou senha incorretos."}
		} else {
			resp = protocolo.Resposta{
				Sucesso:  true,
				Mensagem: fmt.Sprintf("Login realizado com sucesso! Bem-vindo, %s.", usuario.Nome),
			}
		}

	case "BUSCAR_ITINERARIO":
		itinerarios := s.Gerenciador.BuscarItinerarios(req.Origem, req.Destino, req.HorarioMin)
		
		// Converte os resultados encontrados em JSON para enviar de volta
		dadosJson, _ := json.Marshal(itinerarios)
		resp = protocolo.Resposta{
			Sucesso:   true,
			Mensagem:  fmt.Sprintf("Encontrados %d itinerários", len(itinerarios)),
			DadosJSON: string(dadosJson),
		}

	case "RESERVAR_ITINERARIO":
		errReserva := s.Gerenciador.ReservarItinerario(req.IDsTrechos)
		if errReserva != nil {
			resp = protocolo.Resposta{Sucesso: false, Mensagem: errReserva.Error()}
		} else {
			resp = protocolo.Resposta{Sucesso: true, Mensagem: "Reserva realizada com sucesso!"}
		}

	case "CADASTRAR_ROTA":
		// Cria o trecho utilizando os dados reais recebidos da requisição do cliente
		novoTrecho := dominio.Trecho{
			ID:               fmt.Sprintf("t_%d", time.Now().UnixNano()),
			MotoristaID:      req.IDUsuario,
			Origem:           req.Origem,
			Destino:          req.Destino,
			HorarioSaida:     req.HorarioSaida,     
			HorarioChegada:   req.HorarioChegada,   
			AssentosTotais:   req.AssentosTotais,   
			AssentosOcupados: 0,
		}

		// Adiciona no Gerenciador (protegido por Mutex)
		s.Gerenciador.AdicionarTrechos([]dominio.Trecho{novoTrecho})

		resp = protocolo.Resposta{
			Sucesso:  true,
			Mensagem: fmt.Sprintf("Rota de %s para %s (%d - %d) com %d assentos cadastrada com sucesso!", 
				req.Origem, req.Destino, req.HorarioSaida, req.HorarioChegada, req.AssentosTotais),
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
