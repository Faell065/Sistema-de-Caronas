package servidor

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"projeto_redes/configuracao"
	"projeto_redes/dominio"
	"strings"
	"time"
)

type RequisicaoBuscaRest struct {
	Origem  string `json:"origem"`
	Destino string `json:"destino"`
	Data    string `json:"data"`
}

type RespostaErroRest struct {
	Erro string `json:"erro"`
}

type TrechoRest struct {
	Recurso             string  `json:"recurso"`
	EmpresaID           string  `json:"empresa_id"`
	TrechoID            string  `json:"trecho_id"`
	RotaID              string  `json:"rota_id"`
	MotoristaID         string  `json:"motorista_id"`
	Origem              string  `json:"origem"`
	Destino             string  `json:"destino"`
	Data                string  `json:"data"`
	HorarioSaida        int     `json:"horario_saida"`
	HorarioChegada      int     `json:"horario_chegada"`
	AssentosDisponiveis int     `json:"assentos_disponiveis"`
	Valor               float64 `json:"valor"`
}

type ItinerarioRest struct {
	ID                  string       `json:"id"`
	Trechos             []TrechoRest `json:"trechos"`
	ValorTotal          float64      `json:"valor_total"`
	AssentosDisponiveis int          `json:"assentos_disponiveis"`
}

type RespostaBuscaRest struct {
	EmpresaID   string           `json:"empresa_id"`
	Itinerarios []ItinerarioRest `json:"itinerarios"`
}

type DecisaoTransacaoRest struct {
	ID      string `json:"tx_id"`
	Motivo  string `json:"motivo,omitempty"`
	Lamport uint64 `json:"lamport,omitempty"`
}

func NovoServidorConfigurado(c configuracao.Configuracao) *Servidor {
	return &Servidor{
		Gerenciador:  dominio.NovoGerenciadorComDiretorio(c.DiretorioDados),
		Configuracao: &c,
	}
}

func escreverJSON(w http.ResponseWriter, status int, valor interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(valor)
}

func rejeitarMetodo(w http.ResponseWriter, metodos ...string) {
	w.Header().Set("Allow", strings.Join(metodos, ", "))
	escreverJSON(w, http.StatusMethodNotAllowed, RespostaErroRest{Erro: "método não permitido"})
}

func trechoRest(e configuracao.Empresa, t dominio.Trecho) TrechoRest {
	return TrechoRest{
		Recurso: fmt.Sprintf("%s/%s", e.ID, t.ID), EmpresaID: e.ID,
		TrechoID: t.ID, RotaID: t.RotaID, MotoristaID: t.MotoristaID,
		Origem: t.Origem, Destino: t.Destino, Data: t.Data,
		HorarioSaida: t.HorarioSaida, HorarioChegada: t.HorarioChegada,
		AssentosDisponiveis: t.AssentosTotais - t.AssentosOcupados,
		Valor:               t.Valor,
	}
}

func HandlerEmpresa(e configuracao.Empresa) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/empresa", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			rejeitarMetodo(w, http.MethodGet)
			return
		}
		escreverJSON(w, http.StatusOK, e)
	})
	return mux
}

func HandlerEmpresaComGerenciador(e configuracao.Empresa, g *dominio.GerenciadorDeRotas) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/empresa", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			rejeitarMetodo(w, http.MethodGet)
			return
		}
		escreverJSON(w, http.StatusOK, e)
	})

	mux.HandleFunc("/api/v1/empresa", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			rejeitarMetodo(w, http.MethodGet)
			return
		}
		escreverJSON(w, http.StatusOK, e)
	})

	mux.HandleFunc("/api/v1/trechos/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			rejeitarMetodo(w, http.MethodGet)
			return
		}
		trechoID := strings.TrimPrefix(r.URL.Path, "/api/v1/trechos/")
		if trechoID == "" {
			escreverJSON(w, http.StatusNotFound, RespostaErroRest{Erro: "trecho não encontrado"})
			return
		}
		trecho, existe := g.ObterTrecho(trechoID)
		if !existe {
			escreverJSON(w, http.StatusNotFound, RespostaErroRest{Erro: "trecho não encontrado"})
			return
		}
		escreverJSON(w, http.StatusOK, trechoRest(e, *trecho))
	})

	mux.HandleFunc("/api/v1/itinerarios/buscar", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			rejeitarMetodo(w, http.MethodPost)
			return
		}
		var req RequisicaoBuscaRest
		decodificador := json.NewDecoder(r.Body)
		decodificador.DisallowUnknownFields()
		if err := decodificador.Decode(&req); err != nil {
			escreverJSON(w, http.StatusBadRequest, RespostaErroRest{Erro: "corpo JSON inválido"})
			return
		}
		if strings.TrimSpace(req.Origem) == "" || strings.TrimSpace(req.Destino) == "" || strings.TrimSpace(req.Data) == "" {
			escreverJSON(w, http.StatusUnprocessableEntity, RespostaErroRest{Erro: "origem, destino e data são obrigatórios"})
			return
		}

		itinerarios := g.BuscarItinerarios(req.Origem, req.Destino, req.Data)
		resposta := RespostaBuscaRest{EmpresaID: e.ID, Itinerarios: make([]ItinerarioRest, 0, len(itinerarios))}
		for _, itinerario := range itinerarios {
			trechos := make([]TrechoRest, 0, len(itinerario.Trechos))
			for _, t := range itinerario.Trechos {
				trechos = append(trechos, trechoRest(e, t))
			}
			resposta.Itinerarios = append(resposta.Itinerarios, ItinerarioRest{
				ID: itinerario.ID, Trechos: trechos,
				ValorTotal:          itinerario.ValorTotal,
				AssentosDisponiveis: itinerario.AssentosDisponivel,
			})
		}
		escreverJSON(w, http.StatusOK, resposta)
	})

	mux.HandleFunc("/api/v1/transacoes/", func(w http.ResponseWriter, r *http.Request) {
		caminho := strings.TrimPrefix(r.URL.Path, "/api/v1/transacoes/")
		partes := strings.Split(strings.Trim(caminho, "/"), "/")
		if len(partes) == 0 || partes[0] == "" {
			escreverJSON(w, http.StatusNotFound, RespostaErroRest{Erro: "transação não encontrada"})
			return
		}
		txID := partes[0]

		if len(partes) == 1 && r.Method == http.MethodGet {
			tx, existe := g.ConsultarTransacaoLocal(txID)
			if !existe {
				escreverJSON(w, http.StatusNotFound, RespostaErroRest{Erro: "transação não encontrada"})
				return
			}
			escreverJSON(w, http.StatusOK, tx)
			return
		}

		if len(partes) != 2 {
			escreverJSON(w, http.StatusNotFound, RespostaErroRest{Erro: "recurso de transação inválido"})
			return
		}
		acao := partes[1]
		if r.Method != http.MethodPost {
			rejeitarMetodo(w, http.MethodPost)
			return
		}

		if acao == "prepare" {
			var tx dominio.TransacaoLocal
			decodificador := json.NewDecoder(r.Body)
			decodificador.DisallowUnknownFields()
			if err := decodificador.Decode(&tx); err != nil {
				escreverJSON(w, http.StatusBadRequest, RespostaErroRest{Erro: "corpo JSON inválido"})
				return
			}
			if tx.ID == "" {
				tx.ID = txID
			}
			if tx.ID != txID {
				escreverJSON(w, http.StatusConflict, RespostaErroRest{Erro: "tx_id da URL difere do corpo"})
				return
			}
			tx.EmpresaID = e.ID
			resultado, err := g.PrepareReservaLocal(tx)
			if err != nil {
				if _, existe := g.ConsultarTransacaoLocal(txID); existe && resultado.Estado == dominio.EstadoTransacaoAbortado {
					escreverJSON(w, http.StatusConflict, resultado)
					return
				}
				escreverJSON(w, http.StatusConflict, RespostaErroRest{Erro: err.Error()})
				return
			}
			escreverJSON(w, http.StatusOK, resultado)
			return
		}

		var decisao DecisaoTransacaoRest
		decodificador := json.NewDecoder(r.Body)
		decodificador.DisallowUnknownFields()
		if err := decodificador.Decode(&decisao); err != nil {
			escreverJSON(w, http.StatusBadRequest, RespostaErroRest{Erro: "corpo JSON inválido"})
			return
		}
		if decisao.ID != "" && decisao.ID != txID {
			escreverJSON(w, http.StatusConflict, RespostaErroRest{Erro: "tx_id da URL difere do corpo"})
			return
		}

		switch acao {
		case "commit":
			resultado, err := g.CommitReservaLocal(txID)
			if err != nil {
				escreverJSON(w, http.StatusConflict, RespostaErroRest{Erro: err.Error()})
				return
			}
			escreverJSON(w, http.StatusOK, resultado)
		case "abort":
			resultado, err := g.AbortReservaLocal(txID, decisao.Motivo)
			if err != nil {
				escreverJSON(w, http.StatusConflict, RespostaErroRest{Erro: err.Error()})
				return
			}
			escreverJSON(w, http.StatusOK, resultado)
		default:
			escreverJSON(w, http.StatusNotFound, RespostaErroRest{Erro: "ação de transação inválida"})
		}
	})

	return mux
}

// Executar abre as duas portas antes de começar a atender.
func (s *Servidor) Executar(c configuracao.Configuracao) error {
	if err := c.Validar(); err != nil {
		return err
	}
	tcp, err := net.Listen("tcp", c.EnderecoTCP)
	if err != nil {
		return fmt.Errorf("abrir TCP: %w", err)
	}
	defer tcp.Close()
	hl, err := net.Listen("tcp", c.EnderecoHTTP)
	if err != nil {
		return fmt.Errorf("abrir HTTP: %w", err)
	}
	defer hl.Close()
	hs := &http.Server{
		Handler:           HandlerEmpresaComGerenciador(c.Empresa, s.Gerenciador),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	defer hs.Close()
	erros := make(chan error, 2)
	go func() { erros <- hs.Serve(hl) }()
	go func() { erros <- s.atenderConexoes(tcp) }()
	return <-erros
}

func (s *Servidor) atenderConexoes(listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go s.atenderCliente(conn)
	}
}

func (s *Servidor) HandlerREST(c configuracao.Configuracao) http.Handler {
	return HandlerEmpresaComGerenciador(c.Empresa, s.Gerenciador)
}
