package servidor

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"projeto_redes/configuracao"
	"projeto_redes/dominio"
	"time"
)

func NovoServidorConfigurado(c configuracao.Configuracao) *Servidor {
	return &Servidor{Gerenciador: dominio.NovoGerenciadorComDiretorio(c.DiretorioDados)}
}

// HandlerEmpresa oferece somente a identidade; reservas distribuídas são outra etapa.
func HandlerEmpresa(e configuracao.Empresa) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/empresa", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"erro": "use GET"})
			return
		}
		json.NewEncoder(w).Encode(e)
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
		Handler:           HandlerEmpresa(c.Empresa),
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
