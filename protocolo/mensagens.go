//Sistema-de-Caronas/protocolo/mensagens.go
package protocolo

// Requisicao representa o envelope padrão que o cliente envia para o servidor
type Requisicao struct {
	TipoAcao     string `json:"tipo_acao"`     // Ex: "BUSCAR_ITINERARIO", "RESERVAR_TRECHO"
	TipoUsuario  string `json:"tipo_usuario"`  // Ex: "passageiro", "motorista"
	IDUsuario    string `json:"id_usuario"`    // Identificador único de quem envia
	Nome           string   `json:"nome,omitempty"`            // Nome informado no cadastro
	Senha          string   `json:"senha,omitempty"`           // Senha para cadastro e login
	Tipo           string   `json:"tipo,omitempty"`            // "motorista" ou "passageiro"
	Origem       string `json:"origem,omitempty"`       // Usado na busca
	Destino      string `json:"destino,omitempty"`      // Usado na busca
	Data           string             `json:"data,omitempty"`             // Nova: Data da viagem
	Paradas        []ParadaRequisicao `json:"paradas,omitempty"`          // Nova: Sequência de paradas/trechos
	HorarioMin   int    `json:"horario_min,omitempty"`  // Usado na busca
	HorarioSaida     int      `json:"horario_saida,omitempty"`     // Add para o cadastro
	HorarioChegada   int      `json:"horario_chegada,omitempty"`   // Add para o cadastro
	AssentosTotais   int      `json:"assentos_totais,omitempty"`   // Add para o cadastro
	IDsTrechos   []string `json:"ids_trechos,omitempty"`// Usado na reserva
}
// observação: os campos com `omitempty` são opcionais e só aparecem no JSON se tiverem valor
// Resposta representa o envelope padrão que o servidor devolve para o cliente
type Resposta struct {
	Sucesso   bool   `json:"sucesso"`
	Mensagem  string `json:"mensagem"`
	DadosJSON string `json:"dados_json,omitempty"` // Opcional: para carregar dados complexos se necessário
}

// ParadaRequisicao espelha os dados de cada parada fornecida pelo motorista
type ParadaRequisicao struct {
	Cidade         string  `json:"cidade"`
	HorarioSaida   int     `json:"horario_saida"`
	AssentosTotais int     `json:"assentos_totais"`
	Valor          float64 `json:"valor"`
}