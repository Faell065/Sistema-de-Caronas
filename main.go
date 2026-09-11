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


// FUNÇÃO PARA ROTA
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

// GERENCIADOR DE ROTAS
// GERENCIADOR DE ROTAS
// GERENCIADOR DE ROTAS
type GerenciadorDeRotas struct {
	mu sync.RWMutex // minha variavel de controle de concorrencia, usada para acesso seguro dados compartilhados entre goroutines, um mutex de leitura e escrita, para que eu possa ler e escrever de forma segura 
	Trechos map[string] *Trecho
}
// função para eu criar a instancia do GerenciadorDeRotas, que é um mapa de trechos, onde a chave é uma string (ID do trecho) e o valor é um ponteiro para o trecho correspondente.
func NovoGerenciadorDeRotas() *GerenciadorDeRotas {
	return &GerenciadorDeRotas{ Trechos: make(map[string]*Trecho)}
}
// METODOS GERENCIADOR DE ROTAS 
func (g *GerenciadorDeRotas) AdicionarTrechos(novosTrechos []Trecho) {
	g.mu.Lock() // trava o mutex para escrita, garantindo que nenhuma outra goroutine possa acessar os dados enquanto estou escrevendo
	defer g.mu.Unlock()

	for _, t := range novosTrechos {
		trechoCopy := t // cria uma cópia do trecho, para o mapa não guardar o mesmo endereço de memoria
		g.Trechos[t.ID] = &trechoCopy // adiciona o trecho ao mapa, usando o ID como chave e o ponteiro para a cópia como valor
	}
}

func (g *GerenciadorDeRotas) 

func (g *GerenciadorDeRotas) ObterTrecho(id string) (*Trecho, bool) {
	g.mu.RLock() // trava o mutex para leitura, permitindo que outras goroutines leiam os dados ao mesmo tempo, mas bloqueando a escrita
	defer g.mu.RUnlock()

	trecho, existe := g.Trechos[id] // verifica se o trecho existe no mapa
	return trecho, existe // retorna o ponteiro para o trecho e um booleano indicando se ele existe
}		

// GERENCIADOR DE ROTAS
// GERENCIADOR DE ROTAS
// GERENCIADOR DE ROTAS
