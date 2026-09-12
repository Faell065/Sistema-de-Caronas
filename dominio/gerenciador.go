package dominio
import ("fmt"; "sync")



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
	Trechos map[string] *Trecho
}

// função para eu criar a instancia do GerenciadorDeRotas, que é um mapa de trechos, onde a chave é uma string (ID do trecho) e o valor é um ponteiro para o trecho correspondente.
func NovoGerenciador() *GerenciadorDeRotas {
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


func (g *GerenciadorDeRotas) ObterTrecho(id string) (*Trecho, bool) {
	g.mu.RLock() // trava o mutex para leitura, permitindo que outras goroutines leiam os dados ao mesmo tempo, mas bloqueando a escrita
	defer g.mu.RUnlock()

	trecho, existe := g.Trechos[id] // verifica se o trecho existe no mapa
	return trecho, existe // retorna o ponteiro para o trecho e um booleano indicando se ele existe
}		

// ITNERARIO
// ITNERARIO
// ITNERARIO
// Itinerario é uma combinação de 1 ou mais trechos que levam da Origem ao Destino

func (g *GerenciadorDeRotas) BuscarItinerarios(origem, destino string, horarioMinimo int) []Itinerario {
	g.mu.RLock()         // Trava apenas para LEITURA (múltiplas leituras simultâneas são permitidas)
	defer g.mu.RUnlock()

	var resultados []Itinerario
	var caminhoAtual []Trecho

	// Função recursiva interna para busca em profundidade (DFS)
	var buscar func(pontoAtual string, horaAtual int)
	
	buscar = func(pontoAtual string, horaAtual int) {
		// Caso base: chegamos ao destino final desejado!
		if pontoAtual == destino {
			// Copiamos o caminho atual para os resultados
			caminhoCopia := make([]Trecho, len(caminhoAtual))
			copy(caminhoCopia, caminhoAtual)
			resultados = append(resultados, Itinerario{Trechos: caminhoCopia})
			return
		}

		// Varre todos os trechos do gerenciador procurando conexões válidas
		for _, t := range g.Trechos {
			// Regras de Validação da Aresta:
			// 1. Origem bate com onde estamos
			// 2. Horário de saída é DEPOIS do horário em que chegamos no ponto atual
			// 3. Tem pelo menos 1 assento vago
			if t.Origem == pontoAtual && t.HorarioSaida >= horaAtual && (t.AssentosTotais - t.AssentosOcupados) > 0 {
				
				// Avança no grafo
				caminhoAtual = append(caminhoAtual, *t)
				
				// Recursão: próximo ponto é o destino deste trecho, e a nova hora limite é a chegada
				buscar(t.Destino, t.HorarioChegada)
				
				// Backtracking: remove o último trecho para testar outros caminhos
				caminhoAtual = caminhoAtual[:len(caminhoAtual)-1]
			}
		}
	}

	// Inicia a busca a partir da origem do passageiro
	buscar(origem, horarioMinimo)
	return resultados
}
// ITNERARIO
// ITNERARIO
// ITNERARIO


// ReservarItinerario tenta reservar 1 assento em todos os trechos da lista de forma atômica
func (g *GerenciadorDeRotas) ReservarItinerario(idsTrechos []string) error {
	// 1. Adquire a trava EXCLUSIVA de escrita (Lock)
	// Nenhuma outra goroutine pode ler ou alterar o gerenciador enquanto reservamos
	g.mu.Lock()
	defer g.mu.Unlock() // Destrava automaticamente ao final da função

	// 2. FASE 1: Validação (Verificar se todos os trechos existem e têm vaga)
	var trechosParaReservar []*Trecho

	for _, id := range idsTrechos {
		trecho, existe := g.Trechos[id]
		
		// Validação A: Trecho existe no mapa?
		if !existe {
			return fmt.Errorf("trecho com ID '%s' não foi encontrado", id)
		}

		// Validação B: Tem assento livre?
		if trecho.AssentosOcupados >= trecho.AssentosTotais {
			return fmt.Errorf("trecho '%s' (%s -> %s) não possui vagas disponíveis", 
				id, trecho.Origem, trecho.Destino)
		}

		// Salva o ponteiro do trecho aprovado na lista temporária
		trechosParaReservar = append(trechosParaReservar, trecho)
	}

	// 3. FASE 2: Efetivação da Reserva (Atomicidade)
	// Se chegou aqui, TODOS os trechos têm vaga garantida!
	for _, trecho := range trechosParaReservar {
		trecho.AssentosOcupados++
	}

	return nil // Reserva realizada com sucesso!
}

// GERENCIADOR DE ROTAS
// GERENCIADOR DE ROTAS
// GERENCIADOR DE ROTAS



// TESTES
// TESTES
func TestBuscarItinerarios() {
	gerenciador := NovoGerenciador()

	// Motorista 1 (João): Feira (08:00) -> Santo Amaro (09:00)
	t1 := Trecho{
		ID: "t1", RotaID: "r1", MotoristaID: "joao",
		Origem: "Feira de Santana", Destino: "Santo Amaro",
		HorarioSaida: 800, HorarioChegada: 900,
		AssentosTotais: 4, AssentosOcupados: 0,
	}

	// Motorista 2 (Maria): Santo Amaro (09:30) -> Salvador (10:30)
	t2 := Trecho{
		ID: "t2", RotaID: "r2", MotoristaID: "maria",
		Origem: "Santo Amaro", Destino: "Salvador",
		HorarioSaida: 930, HorarioChegada: 1030,
		AssentosTotais: 2, AssentosOcupados: 0,
	}

	// Adiciona os trechos soltos no gerenciador
	gerenciador.AdicionarTrechos([]Trecho{t1, t2})

	// Passageiro busca carona de Feira para Salvador a partir das 07:00
	itinerariosEncontrados := gerenciador.BuscarItinerarios("Feira de Santana", "Salvador", 700)

	fmt.Printf("Encontrados %d itinerário(s):\n", len(itinerariosEncontrados))
	for i, itin := range itinerariosEncontrados {
		fmt.Printf("\n--- Opção %d ---\n", i+1)
		for _, trecho := range itin.Trechos {
			fmt.Printf("  * Motorista %s: %s (%d:00) -> %s (%d:00)\n",
				trecho.MotoristaID, trecho.Origem, trecho.HorarioSaida/100, trecho.Destino, trecho.HorarioChegada/100)
		}
	}

}

func TestCriarRota() {
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

func TestReservarItinerario() {
	gerenciador := NovoGerenciador()

	// Adiciona trechos de exemplo
	t1 := Trecho{ID: "t1", Origem: "Feira de Santana", Destino: "Santo Amaro", HorarioSaida: 800, HorarioChegada: 900, AssentosTotais: 4, AssentosOcupados: 0}
	t2 := Trecho{ID: "t2", Origem: "Santo Amaro", Destino: "Salvador", HorarioSaida: 930, HorarioChegada: 1030, AssentosTotais: 3, AssentosOcupados: 1}
	gerenciador.AdicionarTrechos([]Trecho{t1, t2})

	// Tenta reservar um itinerário válido
	err := gerenciador.ReservarItinerario([]string{"t1", "t2"})
	if err != nil {
		fmt.Println("Erro ao reservar itinerário:", err)
	} else {
		fmt.Println("Reserva realizada com sucesso!")
	}

	// Tenta reservar novamente o mesmo itinerário (deve falhar se não houver assentos)
	err = gerenciador.ReservarItinerario([]string{"t1", "t2"})
	if err != nil {
		fmt.Println("Erro ao reservar itinerário novamente:", err)
	} else {
		fmt.Println("Reserva realizada com sucesso novamente!")
	}
	
}
// TESTES
// TESTES

