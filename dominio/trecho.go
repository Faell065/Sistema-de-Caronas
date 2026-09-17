//Sistema-de-Caronas/dominio/trecho.go
package dominio

//apenas os modelos de dados

type Trecho struct {
	ID               string  `json:"id"`
	RotaID           string  `json:"rota_id"`
	MotoristaID      string  `json:"motorista_id"`
	Origem           string  `json:"origem"`
	Destino          string  `json:"destino"`
	Data             string  `json:"data"`              // Nova: Data da viagem (ex: "2026-06-10")
	HorarioSaida     int     `json:"horario_saida"`     // Ex: 800
	HorarioChegada   int     `json:"horario_chegada"`   // Ex: 930
	AssentosTotais   int     `json:"assentos_totais"`
	AssentosOcupados int     `json:"assentos_ocupados"`
	Valor            float64 `json:"valor"`             // Novo: Preço específico deste trecho
}

// Rota é a viagem completa de um motorista, composta por vários trechos
type Rota struct {
	ID string
	Trechos []Trecho // lista dinamica 
	MotoristaID string 
}

// Itinerario representa o caminho encontrado para o passageiro
type Itinerario struct {
	ID                string   `json:"id"`
	Trechos           []Trecho `json:"trechos"`
	ValorTotal        float64  `json:"valor_total"`
	AssentosDisponivel int     `json:"assentos_disponivel"` // Mínimo de vagas entre os trechos do itinerario
}



// Usuario representa qualquer pessoa cadastrada (motorista ou passageiro)
type Usuario struct {
	ID    string  `json:"id"`    // ID único gerado automaticamente (ex: joao_4892)
	Nome  string  `json:"nome"`  // Nome informado pelo usuário
	Senha string  `json:"senha"` // Senha para login futuro
	Tipo  string  `json:"tipo"`  // "motorista" ou "passageiro"
}

type Reserva struct {
	ID        string    `json:"id"`
	PassageiroID string `json:"passageiro_id"`
	IDsTrechos []string `json:"ids_trechos"`
	Data      string    `json:"data"`
}
