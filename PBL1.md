# VaiJunto — PBL 1

Versão original e centralizada do VaiJunto, desenvolvida para o primeiro problema da disciplina.

> **Contexto histórico:** esta documentação descreve o PBL 1. Para a versão distribuída atual com várias empresas, consulte o [`README.md`](README.md) da raiz.

## Descrição

O VaiJunto é um sistema de caronas composto por clientes de terminal e um servidor central. Motoristas cadastram rotas com paradas intermediárias; passageiros podem pesquisar trechos e reservar itinerários combinando uma ou mais partes de uma viagem.

## Funcionalidades

- Cadastro e login de motoristas e passageiros.
- Cadastro de rotas com várias paradas.
- Decomposição automática da rota em trechos.
- Busca de itinerários diretos ou compostos por DFS.
- Capacidade independentemente controlada por trecho.
- Reserva e cancelamento locais.
- Persistência dos usuários, trechos e reservas em arquivos JSON.
- Comunicação concorrente com clientes por TCP.

## Arquitetura PBL 1

- Cliente: interface de terminal em Go.
- Servidor central: aceita conexões TCP e cria uma goroutine por cliente.
- Gerenciador: centraliza usuários, trechos, buscas e reservas.
- Persistência: arquivos JSON em disco.
- Protocolo: requisições e respostas JSON delimitadas por `\n`.

## Pré-requisitos

- Go compatível com o projeto.
- Terminal Linux, macOS ou Windows.
- Docker e Docker Compose, apenas para execução em contêineres.

## Execução local

Inicie o servidor na porta configurada:

```bash
make run-servidor
```

Em outro terminal, execute o cliente:

```bash
make run-cliente
```

Também é possível informar o endereço manualmente:

```bash
go run ./cmd/cliente 127.0.0.1:8080
```

## Execução com Docker

```bash
docker compose -f docker-compose.yml up --build -d
docker compose -f docker-compose.yml logs -f
docker compose -f docker-compose.yml down
```

O `docker-compose.yml` desta versão é destinado ao ambiente único do PBL 1. O ambiente PBL 2 possui configuração separada em `docker-compose.pbl2.yml`.

## Fluxo principal

1. Motorista ou passageiro se cadastra.
2. Motorista cria uma rota com cidades intermediárias.
3. O servidor divide a rota em trechos independentes.
4. Passageiro pesquisa origem, destino e data.
5. O servidor encontra itinerários locais por DFS.
6. Passageiro escolhe um itinerário e efetua uma reserva.

## Tecnologias utilizadas

- Go.
- Sockets TCP nativos.
- JSON.
- Goroutines e mutexes.
- Arquivos JSON como persistência.
- Docker e Docker Compose.
