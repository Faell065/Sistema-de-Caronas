# VaiJunto — Consórcio Distribuído de Caronas

Sistema acadêmico de caronas desenvolvido em Go para o Problema 2 da disciplina TEC502. O projeto mantém uma instância independente para cada empresa e permite que passageiros reservem itinerários combinando caronas publicadas em diferentes servidores.

O sistema anterior, centralizado e desenvolvido no PBL 1, está documentado em [`PBL1.md`](PBL1.md).

## Funcionalidades

- Três ou mais empresas configuráveis por arquivos JSON.
- Cada empresa mantém seus próprios usuários, caronas, reservas e dados persistentes.
- Clientes motorista e passageiro continuam acessando exclusivamente sua empresa por TCP.
- Comunicação entre servidores por API REST.
- Busca de itinerários locais e compostos por diferentes empresas.
- Recursos identificados pelo par `empresa_id/trecho_id`.
- Holds temporários persistidos durante transações distribuídas.
- Máquinas de estados locais para `PREPARE`, `COMMIT`, `ABORT` e `CANCELAR`.
- Repetição segura de mensagens já processadas pelo mesmo `tx_id`.
- HMAC opcional entre servidores configurado por chave compartilhada.
- Persistência em JSON e recuperação básica após reinício.
- Docker Compose para emulação local das três empresas.

## Arquitetura

- `cmd/cliente`: cliente de terminal para motorista e passageiro.
- `cmd/servidor`: inicialização configurada das instâncias.
- `cmd/teste_carga` e `cmd/teste_integridade`: simuladores de carga e integridade legados em correção.
- `configuracao`: identidade e configuração das empresas e pares.
- `dominio`: usuários, trechos, rotas, reservas, transações e persistência.
- `protocolo`: envelope das mensagens TCP.
- `servidor`: gerenciamento TCP, REST, busca e coordenação.

## Pré-requisitos

- Go 1.27 ou versão compatível declarada em `go.mod`.
- Docker e Docker Compose para executar o ambiente com três empresas.
- GCC apenas para executações com o detector de corrida Go (`-race`), se disponível no sistema.

## Instalação

Clone ou obtenha o projeto e entre na raiz:

```bash
cd Sistema-de-Caronas
go mod download
```

Compile todos os pacotes:

```bash
go build ./...
```

## Configuração das empresas

A instância recebe um arquivo JSON por meio de `-config`.

Exemplos locais:

- `config/empresa_a.json`
- `config/empresa_b.json`
- `config/empresa_c.json`

Exemplos Docker:

- `config/docker/empresa_a.json`
- `config/docker/empresa_b.json`
- `config/docker/empresa_c.json`

Cada arquivo define identidade, URLs, portas TCP/HTTP, diretório de dados e lista de pares. Os IDs das empresas devem ser exclusivos.

## Como executar localmente

Abra um terminal para cada instância:

```bash
go run ./cmd/servidor -config config/empresa_a.json
go run ./cmd/servidor -config config/empresa_b.json
go run ./cmd/servidor -config config/empresa_c.json
```

Em outro terminal, conecte o cliente à própria empresa:

```bash
go run ./cmd/cliente 127.0.0.1:8080
```

Clientes podem apontar para A, B ou C pelas respectivas portas TCP configuradas.

## Como executar com Docker

A emulação local das três empresas usa:

```bash
make docker-up
make docker-logs
make docker-down
```

Os dados ficam isolados em `dados_empresas/empresa_a`, `dados_empresas/empresa_b` e `dados_empresas/empresa_c`. Não migre, apague ou sobrescreva os dados antigos em `dados_persistentados` sem autorização.

Para o laboratório com computadores distintos, copie o projeto para cada máquina, ajuste listeners para `0.0.0.0`, publique TCP e HTTP e substitua URLs e IPs dos pares pelos endereços acessíveis.

## Testes

Execute os testes da suíte atual:

```bash
CGO_ENABLED=0 go test ./...
```

Os testes usam diretórios temporários ou servidores HTTP em memória quando podem modificar estado.

## Comunicação REST

A comunicação entre empresas inclui recursos de identidade, busca local, consulta de trecho e operações de transação. O formato completo dos recursos, métodos, códigos e exemplos será documentado em um relatório técnico; o README mantém apenas uma visão operacional.

## Tecnologias utilizadas

- Go.
- TCP nativo com JSON delimitado por quebra de linha.
- HTTP REST.
- HTTP standard library.
- Goroutines, mutexes e estruturas concorrentes.
- Arquivos JSON como persistência.
- Docker e Docker Compose.
- HMAC-SHA256 opcional para mensagens entre empresas.

## Limites conhecidos

- A solução não foi validada em computadores físicos separados.
- Timeout não é tratado como autorização automática de commit ou abort.
- Transações preparadas podem aguardar quando coordenador ou participante estiver indisponível.
- Não há consenso, replicação distribuída ou tolerância a perda permanente de disco.
- A autenticação HMAC protege integridade entre servidores conhecidos, mas não substitui autorização, gerenciamento seguro de certificados ou confiança do participante.

## Documentação histórica

A implementação anterior centralizada está registrada em [`PBL1.md`](PBL1.md).
