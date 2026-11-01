// Sistema-de-Caronas/dominio/trecho.go
package dominio

//apenas os modelos de dados

type Trecho struct {
	ID                 string  `json:"id"`
	RotaID             string  `json:"rota_id"`
	MotoristaID        string  `json:"motorista_id"`
	Origem             string  `json:"origem"`
	Destino            string  `json:"destino"`
	Data               string  `json:"data"`
	HorarioSaida       int     `json:"horario_saida"`
	HorarioChegada     int     `json:"horario_chegada"`
	AssentosTotais     int     `json:"assentos_totais"`
	AssentosOcupados   int     `json:"assentos_ocupados"`
	AssentosPreparados int     `json:"assentos_preparados,omitempty"`
	Valor              float64 `json:"valor"`
}

const (
	EstadoTransacaoPreparando       = "PREPARANDO"
	EstadoTransacaoPreparado        = "PREPARADO"
	EstadoTransacaoCommitConfirmado = "COMMIT_CONFIRMADO"
	EstadoTransacaoAbortado         = "ABORTADO"
	EstadoTransacaoCommitPersistido = "COMMIT_PERSISTIDO"
	EstadoTransacaoConfirmadoGlobal = "CONFIRMADO"
)

type ItemTransacao struct {
	EmpresaID  string `json:"empresa_id"`
	TrechoID   string `json:"trecho_id"`
	Quantidade int    `json:"quantidade"`
}

type TransacaoLocal struct {
	ID             string          `json:"id"`
	CoordenadorID  string          `json:"coordenador_id"`
	EmpresaID      string          `json:"empresa_id"`
	Papel          string          `json:"papel"`
	Operacao       string          `json:"operacao"`
	Estado         string          `json:"estado"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	PassageiroID   string          `json:"passageiro_id,omitempty"`
	Data           string          `json:"data,omitempty"`
	Lamport        uint64          `json:"lamport"`
	Itens          []ItemTransacao `json:"itens"`
	Motivo         string          `json:"motivo,omitempty"`
}

type EstadoParticipante struct {
	EmpresaID string `json:"empresa_id"`
	Estado    string `json:"estado"`
}

type TransacaoCoordenador struct {
	ID             string               `json:"id"`
	CoordenadorID  string               `json:"coordenador_id"`
	Estado         string               `json:"estado"`
	IdempotencyKey string               `json:"idempotency_key"`
	PassageiroID   string               `json:"passageiro_id,omitempty"`
	Lamport        uint64               `json:"lamport"`
	Itens          []ItemTransacao      `json:"itens"`
	Participantes  []EstadoParticipante `json:"participantes"`
	Motivo         string               `json:"motivo,omitempty"`
}

// Rota é a viagem completa de um motorista, composta por vários trechos
type Rota struct {
	ID          string
	Trechos     []Trecho // lista dinamica
	MotoristaID string
}

// Itinerario representa o caminho encontrado para o passageiro
type Itinerario struct {
	ID                 string   `json:"id"`
	Trechos            []Trecho `json:"trechos"`
	ValorTotal         float64  `json:"valor_total"`
	AssentosDisponivel int      `json:"assentos_disponivel"` // Mínimo de vagas entre os trechos do itinerario
}

// Usuario representa qualquer pessoa cadastrada (motorista ou passageiro)
type Usuario struct {
	ID    string `json:"id"`    // ID único gerado automaticamente (ex: joao_4892)
	Nome  string `json:"nome"`  // Nome informado pelo usuário
	Senha string `json:"senha"` // Senha para login futuro
	Tipo  string `json:"tipo"`  // "motorista" ou "passageiro"
}

type Reserva struct {
	ID           string   `json:"id"`
	PassageiroID string   `json:"passageiro_id"`
	IDsTrechos   []string `json:"ids_trechos"`
	Data         string   `json:"data"`
}

const (
	EstadoCoordenadorInit        = "INIT"
	EstadoCoordenadorPreparando  = "PREPARANDO"
	EstadoCoordenadorPreparedAll = "PREPARED_ALL"
	EstadoCoordenadorCommitted   = "COMMITTED"
	EstadoCoordenadorAborted     = "ABORTED"
	EstadoCoordenadorConfirmado  = "CONFIRMED"
)
