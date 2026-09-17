//Sistema-de-Caronas/dominio/gerenciador.go
package dominio
import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)



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
	Usuarios map[string]*Usuario // Novo mapa para gerenciar usuários em memória
	Reservas map[string]*Reserva // Novo mapa para gerenciar reservas em memória
}

// função para eu criar a instancia do GerenciadorDeRotas, que é um mapa de trechos, onde a chave é uma string (ID do trecho) e o valor é um ponteiro para o trecho correspondente.
func NovoGerenciador() *GerenciadorDeRotas {
	g := &GerenciadorDeRotas{
		Trechos:  make(map[string]*Trecho),
		Usuarios: make(map[string]*Usuario),
		Reservas: make(map[string]*Reserva),
	}
	g.CarregarDados() // Restaura dados ao ligar o servidor
	return g
}

// METODOS GERENCIADOR DE ROTAS 

// USUARIO
// CadastrarUsuario gera um ID único (nome + números aleatórios) e salva o usuário
func (g *GerenciadorDeRotas) CadastrarUsuario(nome, senha, tipo string) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Normaliza o nome para o formato base do ID (ex: "João Silva" -> "joao_silva")
	baseNome := strings.ToLower(strings.ReplaceAll(nome, " ", "_"))
	
	// Gera um número aleatório de 4 dígitos para garantir unicidade
	rand.Seed(time.Now().UnixNano())
	sufixo := rand.Intn(8999) + 1000
	idGerado := fmt.Sprintf("%s_%d", baseNome, sufixo)

	// Verifica se por coincidência o ID já existe (raro, mas seguro)
	if _, existe := g.Usuarios[idGerado]; existe {
		idGerado = fmt.Sprintf("%s_%s", idGerado, strconv.FormatInt(time.Now().UnixNano()%100, 10))
	}

	g.Usuarios[idGerado] = &Usuario{
	ID:    idGerado,
	Nome:  nome,
	Senha: senha,
	Tipo:  tipo,
	}

	g.SalvarDados() // Grava alteração no disco
	return idGerado, nil
}
// USUARIO

// Autenticar valida se o ID, a senha e o TIPO conferem
func (g *GerenciadorDeRotas) Autenticar(idUsuario, senha, tipo string) (bool, *Usuario) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	usuario, existe := g.Usuarios[idUsuario]
	if !existe {
		return false, nil
	}

	// Bloqueia se a senha estiver errada ou se o tipo de usuário for diferente do menu acessado
	if usuario.Senha != senha || usuario.Tipo != tipo {
		return false, nil
	}

	return true, usuario
}

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

// BuscarItinerarios encontra caminhos (diretos ou múltiplos trechos) entre origem e destino para uma data
func (g *GerenciadorDeRotas) BuscarItinerarios(origem, destino, data string) []Itinerario {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Filtra apenas trechos da data correta
	var trechosValidos []Trecho
	for _, t := range g.Trechos {
		if t.Data == data && t.AssentosOcupados < t.AssentosTotais {
			trechosValidos = append(trechosValidos, *t)
		}
	}

	var itinerariosEncontrados []Itinerario

	// Função recursiva interna para buscar caminhos (DFS)
	var dfs func(cidadeAtual string, caminhoAtual []Trecho, visitados map[string]bool)
	
	dfs = func(cidadeAtual string, caminhoAtual []Trecho, visitados map[string]bool) {
		// Se chegamos ao destino desejado, salvamos o itinerário válido
		if cidadeAtual == destino && len(caminhoAtual) > 0 {
			var valorTotal float64
			minAssentos := 999999

			for _, t := range caminhoAtual {
				valorTotal += t.Valor
				vagasRestantes := t.AssentosTotais - t.AssentosOcupados
				if vagasRestantes < minAssentos {
					minAssentos = vagasRestantes
				}
			}

			// Gera um ID único para o itinerário composto
			itinerarioID := fmt.Sprintf("itin_%d", time.Now().UnixNano())
			
			itinerariosEncontrados = append(itinerariosEncontrados, Itinerario{
				ID:                 itinerarioID,
				Trechos:            caminhoAtual,
				ValorTotal:         valorTotal,
				AssentosDisponivel: minAssentos,
			})
			return
		}

		// Procura o próximo trecho a partir da cidade atual
		for _, t := range trechosValidos {
			if t.Origem == cidadeAtual && !visitados[t.ID] {
				// Evita ciclos no grafo
				visitados[t.ID] = true
				
				// O horário de saída do próximo trecho deve ser posterior ou igual à chegada do trecho anterior (se houver)
				if len(caminhoAtual) > 0 {
					trechoAnterior := caminhoAtual[len(caminhoAtual)-1]
					if t.HorarioSaida < trechoAnterior.HorarioChegada {
						visitados[t.ID] = false
						continue
					}
				}

				dfs(t.Destino, append(caminhoAtual, t), visitados)
				
				// Backtrack
				visitados[t.ID] = false
			}
		}
	}

	// Inicia a busca recursiva a partir da origem informada
	visitadosMap := make(map[string]bool)
	dfs(origem, []Trecho{}, visitadosMap)

	return itinerariosEncontrados
}


//SUBSTITUIDO POR RESERVARCOMID
// ReservarItinerario tenta reservar assentos em todos os trechos de um itinerário de forma atômica
func (g *GerenciadorDeRotas) ReservarItinerario(idsTrechos []string) (bool, string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// 1. Validação prévia: Verifica se TODOS os trechos ainda possuem vagas
	for _, idTrecho := range idsTrechos {
		trecho, existe := g.Trechos[idTrecho]
		if !existe {
			return false, fmt.Sprintf("Trecho %s não existe mais.", idTrecho)
		}
		if trecho.AssentosOcupados >= trecho.AssentosTotais {
			return false, fmt.Sprintf("O trecho de %s para %s esgotou as vagas!", trecho.Origem, trecho.Destino)
		}
	}

	// 2. Efetivação: Como todos têm vagas, incrementa os assentos ocupados de todos os trechos atomicamente
	for _, idTrecho := range idsTrechos {
		g.Trechos[idTrecho].AssentosOcupados++
	}

	return true, "Itinerario reservado com sucesso!"
}

// ITNERARIO
// ITNERARIO
// ITNERARIO




// ParadaRota representa cada cidade no caminho e as condições daquele trecho específico que se inicia nela
type ParadaRota struct {
	Cidade          string  `json:"cidade"`
	HorarioSaida    int     `json:"horario_saida"`
	AssentosTotais  int     `json:"assentos_totais"` // Assentos específicos para o trecho que sai daqui
	Valor           float64 `json:"valor"`           // Valor para o trecho que sai daqui
}

// CadastrarRotaCompleta processa uma rota contínua e a fragmenta em trechos atômicos
func (g *GerenciadorDeRotas) CadastrarRotaCompleta(
	motoristaID string,
	data string,
	paradas []ParadaRota, // Sequência de cidades e os dados de seus respectivos trechos de saída
) []string {
	g.mu.Lock()
	defer g.mu.Unlock()

	rotaID := fmt.Sprintf("rota_%d", time.Now().UnixNano())
	var idsTrechosGerados []string

	// Precisamos de pelo menos uma origem e um destino (2 paradas)
	for i := 0; i < len(paradas)-1; i++ {
		origemAtual := paradas[i]
		destinoSeguinte := paradas[i+1]

		trechoID := fmt.Sprintf("t_%d_%d", time.Now().UnixNano(), i)

		trecho := &Trecho{
			ID:               trechoID,
			RotaID:           rotaID,
			MotoristaID:      motoristaID,
			Origem:           origemAtual.Cidade,
			Destino:          destinoSeguinte.Cidade,
			Data:             data,
			HorarioSaida:     origemAtual.HorarioSaida,
			HorarioChegada:   destinoSeguinte.HorarioSaida, // Usamos o horário de saída da próxima parada como chegada estimada
			AssentosTotais:   origemAtual.AssentosTotais,   // Assentos independentes deste trecho
			AssentosOcupados: 0,
			Valor:            origemAtual.Valor,            // Valor independente deste trecho
		}

		g.Trechos[trechoID] = trecho
		idsTrechosGerados = append(idsTrechosGerados, trechoID)
	}
	g.SalvarDados() // Grava alteração no disco
	return idsTrechosGerados
}



// --- MÉTODOS DO MOTORISTA ---

// ListarRotasMotorista retorna todas as rotas ativas agrupadas criadas por um motorista
func (g *GerenciadorDeRotas) ListarRotasMotorista(motoristaID string) map[string][]Trecho {
	g.mu.RLock()
	defer g.mu.RUnlock()

	rotas := make(map[string][]Trecho)
	for _, t := range g.Trechos {
		if t.MotoristaID == motoristaID {
			rotas[t.RotaID] = append(rotas[t.RotaID], *t)
		}
	}
	return rotas
}

// CancelarRotaMotorista remove todos os trechos associados a uma rota
func (g *GerenciadorDeRotas) CancelarRotaMotorista(motoristaID, rotaID string) (bool, string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	encontrou := false
	for id, t := range g.Trechos {
		if t.RotaID == rotaID && t.MotoristaID == motoristaID {
			delete(g.Trechos, id)
			encontrou = true
		}
	}

	if !encontrou {
		return false, "Rota não encontrada ou não pertence a este motorista."
	}
	g.SalvarDados() // Grava alteração no disco
	return true, "Rota cancelada e removida com sucesso!"
}

// --- MÉTODOS DO PASSAGEIRO ---

// ReservarItinerario atômico com registro de ID da Reserva
func (g *GerenciadorDeRotas) ReservarItinerarioComID(passageiroID string, idsTrechos []string) (bool, string, string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	for _, idTrecho := range idsTrechos {
		trecho, existe := g.Trechos[idTrecho]
		if !existe {
			return false, fmt.Sprintf("Trecho %s não existe mais.", idTrecho), ""
		}
		if trecho.AssentosOcupados >= trecho.AssentosTotais {
			return false, fmt.Sprintf("O trecho de %s para %s esgotou as vagas!", trecho.Origem, trecho.Destino), ""
		}
	}

	for _, idTrecho := range idsTrechos {
		g.Trechos[idTrecho].AssentosOcupados++
	}

	reservaID := fmt.Sprintf("res_%d", time.Now().UnixNano())
	g.Reservas[reservaID] = &Reserva{
		ID:           reservaID,
		PassageiroID: passageiroID,
		IDsTrechos:   idsTrechos,
		Data:         time.Now().Format("02/01/2006"),
	}

	g.SalvarDados() // Grava alteração no disco
	return true, "Itinerário reservado com sucesso!", reservaID
}

// ListarReservasPassageiro busca todas as reservas ativas do passageiro
func (g *GerenciadorDeRotas) ListarReservasPassageiro(passageiroID string) []map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var lista []map[string]interface{}
	for _, res := range g.Reservas {
		if res.PassageiroID == passageiroID {
			var trechosReserva []Trecho
			var valorTotal float64
			for _, idT := range res.IDsTrechos {
				if t, ok := g.Trechos[idT]; ok {
					trechosReserva = append(trechosReserva, *t)
					valorTotal += t.Valor
				}
			}

			lista = append(lista, map[string]interface{}{
				"id_reserva":  res.ID,
				"trechos":     trechosReserva,
				"valor_total": valorTotal,
			})
		}
	}
	return lista
}

// CancelarReservaPassageiro libera os assentos de volta no sistema e deleta a reserva
func (g *GerenciadorDeRotas) CancelarReservaPassageiro(passageiroID, reservaID string) (bool, string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	res, existe := g.Reservas[reservaID]
	if !existe || res.PassageiroID != passageiroID {
		return false, "Reserva não encontrada ou não pertence a este passageiro."
	}

	// Devolve os assentos ocupados nos trechos correspondentes
	for _, idTrecho := range res.IDsTrechos {
		if trecho, ok := g.Trechos[idTrecho]; ok {
			if trecho.AssentosOcupados > 0 {
				trecho.AssentosOcupados--
			}
		}
	}

	delete(g.Reservas, reservaID)
	g.SalvarDados() // Grava alteração no disco
	return true, "Reserva cancelada com sucesso! As vagas foram liberadas no sistema."
}