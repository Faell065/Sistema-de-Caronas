package dominio

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
	EmpresaID          string  `json:"empresa_id,omitempty"`
	Recurso            string  `json:"recurso,omitempty"`
}

const (
	EstadoTransacaoPreparando       = "PREPARANDO"
	EstadoTransacaoPreparado        = "PREPARADO"
	EstadoTransacaoCommitConfirmado = "COMMIT_CONFIRMADO"
	EstadoTransacaoAbortado         = "ABORTADO"
	EstadoTransacaoCommitPersistido = "COMMIT_PERSISTIDO"
	EstadoTransacaoConfirmadoGlobal = "CONFIRMADO"
	EstadoTransacaoCancelado        = "CANCELADO"
)

type ItemTransacao struct {
	EmpresaID  string `json:"empresa_id"`
	TrechoID   string `json:"trecho_id"`
	Quantidade int    `json:"quantidade"`
}

type TransacaoLocal struct {
	ID                    string          `json:"id"`
	CoordenadorID         string          `json:"coordenador_id"`
	EmpresaID             string          `json:"empresa_id"`
	Papel                 string          `json:"papel"`
	Operacao              string          `json:"operacao"`
	Estado                string          `json:"estado"`
	IdempotencyKey        string          `json:"idempotency_key,omitempty"`
	PassageiroID          string          `json:"passageiro_id,omitempty"`
	Data                  string          `json:"data,omitempty"`
	Lamport               uint64          `json:"lamport"`
	Itens                 []ItemTransacao `json:"itens"`
	Motivo                string          `json:"motivo,omitempty"`
	SequenciaProprietario uint64          `json:"sequencia_proprietario,omitempty"`
	CriadoEm              int64           `json:"criado_em,omitempty"`
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

type Rota struct {
	ID          string
	Trechos     []Trecho
	MotoristaID string
}

type Itinerario struct {
	ID                 string   `json:"id"`
	Trechos            []Trecho `json:"trechos"`
	ValorTotal         float64  `json:"valor_total"`
	AssentosDisponivel int      `json:"assentos_disponivel"`
}

type Usuario struct {
	ID    string `json:"id"`
	Nome  string `json:"nome"`
	Senha string `json:"senha"`
	Tipo  string `json:"tipo"`
}

type Reserva struct {
	ID           string          `json:"id"`
	PassageiroID string          `json:"passageiro_id"`
	IDsTrechos   []string        `json:"ids_trechos"`
	Data         string          `json:"data"`
	EmpresaID    string          `json:"empresa_id,omitempty"`
	Recurso      string          `json:"recurso,omitempty"`
	Itens        []ItemTransacao `json:"itens,omitempty"`
	Trechos      []Trecho        `json:"trechos,omitempty"`
	ValorTotal   float64         `json:"valor_total,omitempty"`
}

const (
	EstadoCoordenadorInit        = "INIT"
	EstadoCoordenadorPreparando  = "PREPARANDO"
	EstadoCoordenadorPreparedAll = "PREPARED_ALL"
	EstadoCoordenadorCommitted   = "COMMITTED"
	EstadoCoordenadorAborted     = "ABORTED"
	EstadoCoordenadorConfirmado  = "CONFIRMED"
)