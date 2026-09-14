package dominio

//apenas os modelos de dados

type Trecho struct {
	ID string
	RotaID string
	MotoristaID string
	Origem string 
	Destino string
	AssentosTotais int
	AssentosOcupados int
	HorarioSaida int 
	HorarioChegada int
	

}

// Rota é a viagem completa de um motorista, composta por vários trechos
type Rota struct {
	ID string
	Trechos []Trecho // lista dinamica 
	MotoristaID string 
}

// Itinerario representa o caminho encontrado para o passageiro
type Itinerario struct {
	Trechos []Trecho
}

