# PBL 2 — etapa 1: identidade e configuração

O binário é o mesmo em todas as empresas. O arquivo JSON define identidade, endereço TCP dos clientes, endereço HTTP, URL anunciada, pasta de dados e uma lista de pares de tamanho variável. A lista é carregada na inicialização: esta etapa não implementa descoberta automática nem entrada de membros durante uma transação.

## Execução local

Na raiz do projeto, abra um terminal para cada comando:

```bash
go run ./cmd/servidor -config config/empresa_a.json
go run ./cmd/servidor -config config/empresa_b.json
go run ./cmd/servidor -config config/empresa_c.json
```

| Empresa | TCP para os clientes | HTTP |
|---|---|---|
| A | 127.0.0.1:8080 | http://127.0.0.1:8081 |
| B | 127.0.0.1:8090 | http://127.0.0.1:8091 |
| C | 127.0.0.1:8100 | http://127.0.0.1:8101 |

```bash
go run ./cmd/cliente 127.0.0.1:8090
curl -i http://127.0.0.1:8091/api/empresa
```

`GET /api/empresa` retorna HTTP 200 com `id`, `nome` e `url` em JSON. Outro método retorna 405 e `Allow: GET`. Recurso inexistente retorna 404. Os pares ainda não são consultados automaticamente.

## Configuração

```json
{
  "empresa": {"id": "empresa_a", "nome": "VaiJunto", "url": "http://127.0.0.1:8081"},
  "endereco_tcp": "127.0.0.1:8080",
  "endereco_http": "127.0.0.1:8081",
  "diretorio_dados": "../dados_empresas/empresa_a",
  "pares": [
    {"id": "empresa_b", "nome": "Empresa B", "url": "http://127.0.0.1:8091"}
  ]
}
```

`endereco_http` é onde esta instância escuta; `empresa.url` é onde as outras conseguem acessá-la. `0.0.0.0` serve para escutar em todas as interfaces, mas não deve ser anunciado como endereço de um par. A pasta de dados relativa é resolvida a partir da localização do JSON, não do diretório do terminal.

Para adicionar D: crie outro JSON, dê um ID exclusivo, escolha portas disponíveis e uma pasta própria, e inclua D nas listas dos outros membros. Reinicie as instâncias para recarregar as listas. Não há limite de três no código. Esta versão não verifica a reciprocidade das listas nem detecta automaticamente configurações conflitantes entre computadores.

Não execute duas instâncias sobre a mesma pasta de dados. Não reutilize o ID de uma empresa para representar outra. As instâncias de exemplo começam vazias: os arquivos antigos em dados_persistentados não são migrados nem apagados.

## Docker

```bash
docker compose -f docker-compose.pbl2.yml up --build -d
docker compose -f docker-compose.pbl2.yml logs -f
docker compose -f docker-compose.pbl2.yml down
```

Esse Compose cria três serviços no mesmo computador para desenvolvimento, com volumes separados. Os nomes empresa_a, empresa_b e empresa_c são resolvidos pela rede Docker desse Compose. Não execute o Compose antigo junto nas mesmas portas. O Compose padrão agora executa apenas a empresa A com seu próprio diretório novo.

Para o laboratório, copie o projeto para cada computador, configure listeners em 0.0.0.0:8080 e 0.0.0.0:8081, publique as duas portas e substitua as URLs próprias e dos pares pelos IPs acessíveis dos computadores. O Compose local não substitui a demonstração em máquinas distintas exigida pelo enunciado.

## Escopo e limites

Implementado: configuração validada, quantidade variável de pares, diretório separado por instância, porta TCP configurável, endpoint HTTP de identidade e exemplos Docker.

Não implementado nesta etapa: busca entre empresas, reservas distribuídas, Lamport, worker pool e recuperação de transações. O mutex global e a persistência antiga permanecem; ela ainda não é um diário transacional resistente a falhas. O registro de pares não é autenticação entre empresas. A API de exemplo usa HTTP sem TLS para o laboratório.

## Validação

```bash
go test -race ./configuracao ./dominio ./servidor
go build ./...
```

Os testes de configuração cobrem lista com 20 pares, IDs/URLs repetidos, campos desconhecidos e resolução de caminhos. Os demais verificam HTTP 200/405 e persistência isolada. Os executáveis antigos de teste de carga/integridade não foram corrigidos nesta etapa.

Validação executada nesta entrega: testes com `-race` dos três pacotes acima, compilação de todos os pacotes e integração com três processos, consultando identidades por HTTP e cadastrando usuário por TCP exclusivamente em A. Docker não foi executado neste ambiente.
