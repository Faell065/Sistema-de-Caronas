# VaiJunto — estado atual do PBL 2

## Arquitetura implementada

- Clientes permanecem em TCP nativo, com JSON delimitado por quebra de linha.
- Cada instância mantém usuários, rotas, trechos, reservas e transações em diretórios separados.
- Motorista publica somente na empresa do servidor local.
- Servidores cooperam por HTTP REST configurável para pares conhecidos.
- Não há servidor central nem descoberta automática de membros.

## APIs REST atuais

- `GET /api/empresa`
- `GET /api/v1/empresa`
- `GET /api/v1/trechos/{id}`
- `POST /api/v1/itinerarios/buscar`
- `POST /api/v1/itinerarios/buscar-distribuido`
- `POST /api/v1/transacoes/{tx_id}/prepare`
- `POST /api/v1/transacoes/{tx_id}/commit`
- `POST /api/v1/transacoes/{tx_id}/abort`
- `POST /api/v1/transacoes/{tx_id}/cancelar`
- `GET /api/v1/transacoes/{tx_id}`

## Reserva distribuída

A reserva usa estado persistido por transação, holds locais e coordenador por transação. Itens são identificados pelo par `empresa_id/trecho_id`. A ordem das preparações é determinada pelos participantes; timeout não decide sozinho commit ou abort.

Precedência é atribuída por uma sequência monotônica persistida quando o participante proprietário recebe pela primeira vez a preparação. Retransmissões usam o mesmo `tx_id` e não recebem nova sequência. Relógios físicos não são usados como critério.

## Recuperação

- Transações locais preparadas são identificadas ao reiniciar.
- Transações coordenadoras persistidas são reclassificadas para retomada ou decisão.
- `COMMIT` e `ABORT` repetidos são idempotentes.
- Estados ambíguos não são convertidos em `ABORTED` por timeout.
- Transações em `PREPARING` esperam decisão persistida ou recuperação externa.

## Limites

O sistema não foi validado em computadores distintos nem garante recuperação irrestrita sob perda permanente de dados, coordenador indefinidamente indisponível ou partição permanente. Trânsito HTTP usa HMAC opcional configurado por chave compartilhada.

## Execução local

```bash
go run ./cmd/servidor -config config/empresa_a.json
go run ./cmd/servidor -config config/empresa_b.json
go run ./cmd/servidor -config config/empresa_c.json
```

Docker local:

```bash
make docker-up
make docker-logs
make docker-down
```

## Validação implementada

```bash
CGO_ENABLED=0 go test ./... -vet=off
```

Há testes temporários para concorrência local, última vaga, HMAC, rollback local, cancelamento, persistência e indisponibilidade remota simulada.
