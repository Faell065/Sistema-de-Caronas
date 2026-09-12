package protocolo

// Requisicao representa o envelope padrão que o cliente envia para o servidor
type Requisicao struct {
	TipoAcao     string `json:"tipo_acao"`     // Ex: "BUSCAR_ITINERARIO", "RESERVAR_TRECHO"
	TipoUsuario  string `json:"tipo_usuario"`  // Ex: "passageiro", "motorista"
	IDUsuario    string `json:"id_usuario"`    // Identificador único de quem envia
	Origem       string `json:"origem,omitempty"`       // Usado na busca
	Destino      string `json:"destino,omitempty"`      // Usado na busca
	HorarioMin   int    `json:"horario_min,omitempty"`  // Usado na busca
	IDsTrechos   []string `json:"ids_trechos,omitempty"`// Usado na reserva
}

// Resposta representa o envelope padrão que o servidor devolve para o cliente
type Resposta struct {
	Sucesso   bool   `json:"sucesso"`
	Mensagem  string `json:"mensagem"`
	DadosJSON string `json:"dados_json,omitempty"` // Opcional: para carregar dados complexos se necessário
}