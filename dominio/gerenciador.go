// Sistema-de-Caronas/dominio/gerenciador.go
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

	for i := 0; i < len(paradas)-1; i++ {
		novoTrecho := Trecho{
			Origem:           paradas[i],
			Destino:          paradas[i+1],
			AssentosTotais:   assentos,
			AssentosOcupados: 0,
		}
		trechos = append(trechos, novoTrecho)
	}
	return Rota{
		ID:          id,
		Trechos:     trechos,
		MotoristaID: motoristaID,
	}
}

type GerenciadorDeRotas struct {
	diretorioDados string

	muUsuarios   sync.RWMutex
	muTrechos    sync.RWMutex
	muReservas   sync.RWMutex
	muTransacoes sync.RWMutex

	// Operações locais que alteram trechos e reservas juntas usam esta trava.
	// Ela não deve ser mantida enquanto houver espera de rede em etapas futuras.
	muOperacoes sync.Mutex

	Trechos               map[string]*Trecho
	Usuarios              map[string]*Usuario
	Reservas              map[string]*Reserva
	Transacoes            map[string]*TransacaoLocal
	TransacoesCoordenador map[string]*TransacaoCoordenador
}

func NovoGerenciadorComDiretorio(diretorio string) *GerenciadorDeRotas {
	g := &GerenciadorDeRotas{
		diretorioDados:        diretorio,
		Trechos:               make(map[string]*Trecho),
		Usuarios:              make(map[string]*Usuario),
		Reservas:              make(map[string]*Reserva),
		Transacoes:            make(map[string]*TransacaoLocal),
		TransacoesCoordenador: make(map[string]*TransacaoCoordenador),
	}
	g.CarregarDados()
	return g
}

func NovoGerenciador() *GerenciadorDeRotas {
	return NovoGerenciadorComDiretorio("dados")
}

// METODOS GERENCIADOR DE ROTAS

// USUARIO
// CadastrarUsuario gera um ID único (nome + números aleatórios) e salva o usuário
func (g *GerenciadorDeRotas) CadastrarUsuario(nome, senha, tipo string) (string, error) {
	g.muUsuarios.Lock()

	baseNome := strings.ToLower(strings.ReplaceAll(nome, " ", "_"))
	rand.Seed(time.Now().UnixNano())
	sufixo := rand.Intn(8999) + 1000
	idGerado := fmt.Sprintf("%s_%d", baseNome, sufixo)

	if _, existe := g.Usuarios[idGerado]; existe {
		idGerado = fmt.Sprintf("%s_%s", idGerado, strconv.FormatInt(time.Now().UnixNano()%100, 10))
	}

	g.Usuarios[idGerado] = &Usuario{
		ID:    idGerado,
		Nome:  nome,
		Senha: senha,
		Tipo:  tipo,
	}
	g.muUsuarios.Unlock()

	if err := g.SalvarDados(); err != nil {
		return "", err
	}
	return idGerado, nil
}

// USUARIO

// Autenticar valida se o ID, a senha e o TIPO conferem
func (g *GerenciadorDeRotas) Autenticar(idUsuario, senha, tipo string) (bool, *Usuario) {
	g.muUsuarios.RLock()
	defer g.muUsuarios.RUnlock()

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
	g.muTrechos.Lock() // trava o mutex para escrita, garantindo que nenhuma outra goroutine possa acessar os dados enquanto estou escrevendo
	defer g.muTrechos.Unlock()

	for _, t := range novosTrechos {
		trechoCopy := t               // cria uma cópia do trecho, para o mapa não guardar o mesmo endereço de memoria
		g.Trechos[t.ID] = &trechoCopy // adiciona o trecho ao mapa, usando o ID como chave e o ponteiro para a cópia como valor
	}
}

func (g *GerenciadorDeRotas) ObterTrecho(id string) (*Trecho, bool) {
	g.muTrechos.RLock()
	defer g.muTrechos.RUnlock()

	trecho, existe := g.Trechos[id]
	if !existe {
		return nil, false
	}
	copiado := *trecho
	return &copiado, true
}

// ITNERARIO
// ITNERARIO
// ITNERARIO
// Itinerario é uma combinação de 1 ou mais trechos que levam da Origem ao Destino

// BuscarItinerarios encontra caminhos (diretos ou múltiplos trechos) entre origem e destino para uma data
func (g *GerenciadorDeRotas) BuscarItinerarios(origem, destino, data string) []Itinerario {
	g.muTrechos.RLock()
	defer g.muTrechos.RUnlock()

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

// SUBSTITUIDO POR RESERVARCOMID
// ReservarItinerario tenta reservar assentos em todos os trechos de um itinerário de forma atômica
func (g *GerenciadorDeRotas) ReservarItinerario(idsTrechos []string) (bool, string) {
	g.muOperacoes.Lock()
	defer g.muOperacoes.Unlock()

	g.muTrechos.Lock()
	defer g.muTrechos.Unlock()

	for _, idTrecho := range idsTrechos {
		trecho, existe := g.Trechos[idTrecho]
		if !existe {
			return false, fmt.Sprintf("Trecho %s não existe mais.", idTrecho)
		}
		if trecho.AssentosOcupados >= trecho.AssentosTotais {
			return false, fmt.Sprintf("O trecho de %s para %s esgotou as vagas!", trecho.Origem, trecho.Destino)
		}
	}
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
	Cidade         string  `json:"cidade"`
	HorarioSaida   int     `json:"horario_saida"`
	AssentosTotais int     `json:"assentos_totais"` // Assentos específicos para o trecho que sai daqui
	Valor          float64 `json:"valor"`           // Valor para o trecho que sai daqui
}

// CadastrarRotaCompleta processa uma rota contínua e a fragmenta em trechos atômicos
func (g *GerenciadorDeRotas) CadastrarRotaCompleta(
	motoristaID string,
	data string,
	paradas []ParadaRota, // Sequência de cidades e os dados de seus respectivos trechos de saída
) []string {
	g.muOperacoes.Lock()
	var idsTrechosGerados []string
	rotaID := fmt.Sprintf("rota_%d", time.Now().UnixNano())

	g.muTrechos.Lock()
	for i := 0; i < len(paradas)-1; i++ {
		origemAtual := paradas[i]
		destinoSeguinte := paradas[i+1]
		trechoID := fmt.Sprintf("t_%d_%d", time.Now().UnixNano(), i)
		trecho := &Trecho{
			ID: trechoID, RotaID: rotaID, MotoristaID: motoristaID,
			Origem: origemAtual.Cidade, Destino: destinoSeguinte.Cidade,
			Data: data, HorarioSaida: origemAtual.HorarioSaida,
			HorarioChegada: destinoSeguinte.HorarioSaida,
			AssentosTotais: origemAtual.AssentosTotais, AssentosOcupados: 0,
			Valor: origemAtual.Valor,
		}
		g.Trechos[trechoID] = trecho
		idsTrechosGerados = append(idsTrechosGerados, trechoID)
	}
	g.muTrechos.Unlock()
	g.muOperacoes.Unlock()

	if len(idsTrechosGerados) > 0 {
		_ = g.SalvarDados()
	}
	return idsTrechosGerados
}

// --- MÉTODOS DO MOTORISTA ---

// ListarRotasMotorista retorna todas as rotas ativas agrupadas criadas por um motorista
func (g *GerenciadorDeRotas) ListarRotasMotorista(motoristaID string) map[string][]Trecho {
	g.muTrechos.RLock()
	defer g.muTrechos.RUnlock()

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
	g.muOperacoes.Lock()

	g.muTrechos.Lock()
	encontrou := false
	for id, t := range g.Trechos {
		if t.RotaID == rotaID && t.MotoristaID == motoristaID {
			delete(g.Trechos, id)
			encontrou = true
		}
	}
	g.muTrechos.Unlock()
	g.muOperacoes.Unlock()

	if !encontrou {
		return false, "Rota não encontrada ou não pertence a este motorista."
	}
	_ = g.SalvarDados()
	return true, "Rota cancelada e removida com sucesso!"
}

// --- MÉTODOS DO PASSAGEIRO ---

// ReservarItinerario atômico com registro de ID da Reserva
func (g *GerenciadorDeRotas) ReservarItinerarioComID(passageiroID string, idsTrechos []string) (bool, string, string) {
	g.muOperacoes.Lock()

	g.muTrechos.Lock()
	for _, idTrecho := range idsTrechos {
		trecho, existe := g.Trechos[idTrecho]
		if !existe {
			g.muTrechos.Unlock()
			g.muOperacoes.Unlock()
			return false, fmt.Sprintf("Trecho %s não existe mais.", idTrecho), ""
		}
		if trecho.AssentosOcupados >= trecho.AssentosTotais {
			g.muTrechos.Unlock()
			g.muOperacoes.Unlock()
			return false, fmt.Sprintf("O trecho de %s para %s esgotou as vagas!", trecho.Origem, trecho.Destino), ""
		}
	}
	for _, idTrecho := range idsTrechos {
		g.Trechos[idTrecho].AssentosOcupados++
	}
	g.muTrechos.Unlock()

	reservaID := fmt.Sprintf("res_%d", time.Now().UnixNano())
	g.muReservas.Lock()
	g.Reservas[reservaID] = &Reserva{
		ID: reservaID, PassageiroID: passageiroID,
		IDsTrechos: append([]string(nil), idsTrechos...),
		Data:       time.Now().Format("02/01/2006"),
	}
	g.muReservas.Unlock()
	g.muOperacoes.Unlock()

	_ = g.SalvarDados()
	return true, "Itinerário reservado com sucesso!", reservaID
}

// ListarReservasPassageiro busca todas as reservas ativas do passageiro
func (g *GerenciadorDeRotas) ListarReservasPassageiro(passageiroID string) []map[string]interface{} {
	g.muReservas.RLock()
	reservas := cloneReservas(g.Reservas)
	g.muReservas.RUnlock()

	g.muTrechos.RLock()
	defer g.muTrechos.RUnlock()

	var lista []map[string]interface{}
	for _, res := range reservas {
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
	g.muOperacoes.Lock()

	g.muReservas.Lock()
	res, existe := g.Reservas[reservaID]
	if !existe || res.PassageiroID != passageiroID {
		g.muReservas.Unlock()
		g.muOperacoes.Unlock()
		return false, "Reserva não encontrada ou não pertence a este passageiro."
	}
	idsTrechos := append([]string(nil), res.IDsTrechos...)
	delete(g.Reservas, reservaID)
	g.muReservas.Unlock()

	g.muTrechos.Lock()
	for _, idTrecho := range idsTrechos {
		if trecho, ok := g.Trechos[idTrecho]; ok && trecho.AssentosOcupados > 0 {
			trecho.AssentosOcupados--
		}
	}
	g.muTrechos.Unlock()
	g.muOperacoes.Unlock()

	_ = g.SalvarDados()
	return true, "Reserva cancelada com sucesso! As vagas foram liberadas no sistema."
}
