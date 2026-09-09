package main
import (
	"fmt"; "sync" //"os"; "bufio"; "strings"
)

func main() {
	paradasViagem := []string{"Feira de Santana", "Salvador", "Ilheus", "Porto Seguro"}
	minhaRota := CriarRota("Rota1", "Motorista1", paradasViagem, 5)
	
	fmt.Printf("Rota ID: %s\n (Motorista: %s)\n", minhaRota.ID, minhaRota.MotoristaID)
	fmt.Println("Trechos gerados:")
	for index, trecho:= range minhaRota.Trechos{
		fmt.Printf("Trecho %d: %s -> %s | Assentos Totais: %d | Assentos Ocupados: %d\n", 
		index+1, 
		trecho.Origem, 
		trecho.Destino, 
		trecho.AssentosTotais, 
		trecho.AssentosOcupados)
	}
}

// CLASSES 
// CLASSES 
// CLASSES 
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

type Rota struct {
	ID string
	Trechos []Trecho // lista dinamica 
	MotoristaID string 	
}
// CLASSES 
// CLASSES 
// CLASSES 

func CriarRota(id string, motoristaID string, paradas []string, assentos int) Rota {
	var trechos []Trecho

	for i := 0; i < len(paradas) - 1; i++ {
		novoTrecho := Trecho{
			Origem: paradas[i],
			Destino: paradas[i+1],
			AssentosTotais: assentos,
			AssentosOcupados: 0,
	}
	trechos = append(trechos, novoTrecho)
}
	return Rota{
		ID: id,
		Trechos: trechos,
		MotoristaID: motoristaID,
	}
}

type GerenciadorDeRotas struct {
	mu sync.RWMutex // minha variavel de controle de concorrencia, usada para acesso seguro dados compartilhados entre goroutines, um mutex de leitura e escrita, para que eu possa ler e escrever de forma segura 
	Trechos map[string]*Trecho
}


