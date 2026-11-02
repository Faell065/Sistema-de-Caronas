package protocolo

type Requisicao struct {
	TipoAcao       string                 `json:"tipo_acao"`
	TipoUsuario    string                 `json:"tipo_usuario"`
	IDUsuario      string                 `json:"id_usuario"`
	Nome           string                 `json:"nome,omitempty"`
	Senha          string                 `json:"senha,omitempty"`
	Tipo           string                 `json:"tipo,omitempty"`
	Origem         string                 `json:"origem,omitempty"`
	Destino        string                 `json:"destino,omitempty"`
	Data           string                 `json:"data,omitempty"`
	Paradas        []ParadaRequisicao     `json:"paradas,omitempty"`
	HorarioMin     int                    `json:"horario_min,omitempty"`
	HorarioSaida   int                    `json:"horario_saida,omitempty"`
	HorarioChegada int                    `json:"horario_chegada,omitempty"`
	AssentosTotais int                    `json:"assentos_totais,omitempty"`
	IDsTrechos     []string               `json:"ids_trechos,omitempty"`
	ItensTrecho    []ItemTrechoRequisicao `json:"itens_trecho,omitempty"`
	IDItinerario   string                 `json:"id_itinerario,omitempty"`
	IDRota         string                 `json:"id_rota,omitempty"`
	IDReserva      string                 `json:"id_reserva,omitempty"`
}

type Resposta struct {
	Sucesso   bool   `json:"sucesso"`
	Mensagem  string `json:"mensagem"`
	DadosJSON string `json:"dados_json,omitempty"`
}

type ParadaRequisicao struct {
	Cidade         string  `json:"cidade"`
	HorarioSaida   int     `json:"horario_saida"`
	AssentosTotais int     `json:"assentos_totais"`
	Valor          float64 `json:"valor"`
}

type ItemTrechoRequisicao struct {
	EmpresaID      string  `json:"empresa_id"`
	TrechoID       string  `json:"trecho_id"`
	Quantidade     int     `json:"quantidade"`
	Origem         string  `json:"origem,omitempty"`
	Destino        string  `json:"destino,omitempty"`
	Data           string  `json:"data,omitempty"`
	HorarioSaida   int     `json:"horario_saida,omitempty"`
	HorarioChegada int     `json:"horario_chegada,omitempty"`
	Valor          float64 `json:"valor,omitempty"`
	MotoristaID    string  `json:"motorista_id,omitempty"`
}