package servidor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"projeto_redes/configuracao"
	"sort"
	"strings"
	"time"
)

type CaminhoDistribuido struct {
	ID                  string       `json:"id"`
	Trechos             []TrechoRest `json:"trechos"`
	ValorTotal          float64      `json:"valor_total"`
	AssentosDisponiveis int          `json:"assentos_disponiveis"`
}

type RespostaBuscaDistribuida struct {
	EmpresaID string               `json:"empresa_id"`
	Caminhos  []CaminhoDistribuido `json:"caminhos"`
	Falhas    []string             `json:"falhas,omitempty"`
}

// BuscarDistribuido compõe o Grafo Centralizado Descentralizado:
// A empresa "A" aciona "B" e "C" (os Pares) via API REST para recolher todos os trechos
// em um "catalogo" temporário. Em seguida roda um DFS (Busca em Profundidade) local 
// unindo os fragmentos e traçando o caminho multi-empresa.
func (s *Servidor) BuscarDistribuido(origem, destino, data string) RespostaBuscaDistribuida {
	if s.Configuracao == nil {
		return RespostaBuscaDistribuida{}
	}
	resposta := RespostaBuscaDistribuida{EmpresaID: s.Configuracao.Empresa.ID}
	cliente := NovoClienteREST(s.Configuracao.Empresa.ID, s.Configuracao.ChaveHMAC, 1500*time.Millisecond)

	catalogo := make([]TrechoRest, 0)
	
	// Passo 1: Insere as rotas locais da própria Empresa A no catálogo
	locais := s.Gerenciador.BuscarItinerarios(origem, destino, data)
	for _, local := range locais {
		for _, trecho := range local.Trechos {
			catalogo = append(catalogo, trechoRest(s.Configuracao.Empresa, trecho))
		}
	}

	// Passo 2: Puxa RESTs remotas dos peers na mesma data
	pares := append([]configuracao.Empresa(nil), s.Configuracao.Pares...)
	sort.Slice(pares, func(i, j int) bool { return pares[i].ID < pares[j].ID })
	for _, par := range pares {
		status, body, err := cliente.ChamarComRepeticao(http.MethodPost,
			strings.TrimRight(par.URL, "/")+"/api/v1/itinerarios/buscar",
			"busca-"+par.ID, RequisicaoBuscaRest{Origem: origem, Destino: destino, Data: data}, 2)
		if err != nil || status != http.StatusOK {
			resposta.Falhas = append(resposta.Falhas, par.ID)
			continue
		}
		var remota RespostaBuscaRest
		if err := json.Unmarshal(body, &remota); err != nil {
			resposta.Falhas = append(resposta.Falhas, par.ID)
			continue
		}
		for _, itinerario := range remota.Itinerarios {
			catalogo = append(catalogo, itinerario.Trechos...)
		}
	}

	visitados := make(map[string]bool)
	var dfs func(string, []TrechoRest, float64, int)
	
	// Passo 3: Executa DFS misturando peças recebidas de A, B e C.
	dfs = func(cidade string, caminho []TrechoRest, valor float64, minimo int) {
		if cidade == destino && len(caminho) > 0 {
			resposta.Caminhos = append(resposta.Caminhos, CaminhoDistribuido{
				ID:         fmt.Sprintf("dist_%d", time.Now().UnixNano()),
				Trechos:    append([]TrechoRest(nil), caminho...),
				ValorTotal: valor, AssentosDisponiveis: minimo,
			})
			return
		}
		// Limite imposto (profundidade máx 4) para evitar saturação e timeout na rede.
		if len(caminho) >= 4 {
			return
		}
		for _, trecho := range catalogo {
			chave := trecho.Recurso
			if trecho.Origem != cidade || visitados[chave] {
				continue
			}
			// Verifica ordem cronológica de viagens multi-hop
			if len(caminho) > 0 && trecho.HorarioSaida < caminho[len(caminho)-1].HorarioChegada {
				continue
			}
			visitados[chave] = true
			novoValor := valor + trecho.Valor
			novoMinimo := trecho.AssentosDisponiveis
			if len(caminho) > 0 && minimo < novoMinimo {
				novoMinimo = minimo
			}
			dfs(trecho.Destino, append(caminho, trecho), novoValor, novoMinimo)
			delete(visitados, chave)
		}
	}
	dfs(origem, nil, 0, 0)
	return resposta
}