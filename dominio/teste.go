package dominio
import ("fmt"; "encoding/json"; "projeto_redes/protocolo")

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

func TestRequisicao(){
	req := protocolo.Requisicao{
		TipoAcao:    "BUSCAR_ITINERARIO",
		TipoUsuario: "passageiro",
		IDUsuario:   "maria_123",
		Origem:      "Feira",
		Destino:     "Salvador",
		HorarioMin:  800,
	}

	// 2. Transformando a Struct em JSON (Marshal)
	dadosBytes, err := json.Marshal(req)
	if err != nil {
		fmt.Println("Erro ao gerar JSON:", err)
		return
	}

	fmt.Println("--- Mensagem enviada pelo Socket (JSON Puro) ---")
	fmt.Println(string(dadosBytes))

	// 3. Simulando a recepção no Servidor: convertendo de volta para Struct (Unmarshal)
	var reqRecebida protocolo.Requisicao
	json.Unmarshal(dadosBytes, &reqRecebida)

	fmt.Println("\n--- Servidor leu a Struct convertida ---")
	fmt.Printf("Ação solicitada: %s por %s\n", reqRecebida.TipoAcao, reqRecebida.IDUsuario)
	fmt.Printf("De: %s -> Para: %s\n", reqRecebida.Origem, reqRecebida.Destino)

}
