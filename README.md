# Sistema-de-Caronas (VaiJunto)
O **VaiJunto** é um sistema distribuído projetado para gerenciamento e reserva de caronas com múltiplas paradas. O projeto divide-se estruturalmente em componentes de Cliente (passagerios e motoristas) e Servidor (gerenciador e banco de dados JSON), se comunicando nativamente por meio de Sockets TCP.

Este README serve como manual de instrução, execução via Makefile e documentação de integração de rede.

## 1) Visão Rápida da Conexão

- **Cliente -> Servidor**: Comunicação direta via `TCP`, configurada para rodar na porta `8080`.



- **Protocolo de Comunicação**: Camada de aplicação própria utilizando serialização em formato `JSON`. As mensagens trafegam encapsuladas através de delimitadores padrão de quebra de linha (`\n`).




Fluxo básico:

1. O `servidor` é instanciado em plano de fundo ou local, restaurando dados anteriores em disco e escutando por conexões TCP ativas.



2. Os `clientes` (Motoristas ou Passageiros) abrem uma sessão de socket.



3. O servidor roteia as solicitações concorrentes através de blocos isolados com Mutex (`sync.RWMutex`), lidando atomicamente com o agendamento de assentos.



4. O servidor devolve uma notificação assíncrona (`Sucesso=true/false`) confirmando as operações.




## 2) Manual de Execução (Makefile)

O projeto acompanha um `Makefile` configurado para automatizar os builds e os processos de emulação e testes de concorrência.

### 2.1 Servidor e Ambiente Docker

A hospedagem principal lida com o roteamento e com a lógica de caminhos e itinerários baseada em DFS.

Comandos principais para emulação via *containers*:

Bash

```
make docker-up      # Inicializa e recria o servidor VaiJunto em background via Compose
make docker-down    # Encerra os containers ativos e remove as redes criadas
make docker-logs    # Exibe os logs do servidor em tempo real

```

Para executar o servidor nativamente no terminal:

Bash

```
make run-servidor   # Inicializa o listener na porta 8080 em modo local

```

### 2.2 Aplicação do Cliente

A interface do cliente possui menus adaptativos que se transformam de acordo com a escolha entre *Motorista* ou *Passageiro*.

Bash

```
make run-cliente    # Executa a aplicação do cliente. Por padrão, aponta para 127.0.0.1:8080

```

*(Nota: Para apontar o cliente a um IP de rede local externo, basta utilizar* *`go run cmd/cliente/main.go [SEU_IP]:8080`**)*.

### 2.3 Ferramentas de Manutenção e Estresse

Bash

```
make run-teste-concorrencia # Inicializa um teste que dispara 10 clientes disputando 2 vagas simultaneamente 
make clean-data             # Formata o banco JSON e deleta todo o estado persistido das caronas

```

## 3) Como usar o Menu do Cliente

Ao conectar, você deverá informar se é Motorista ou Passageiro.

Após criar uma conta e anotar o seu **ID de Acesso**, faça o Login.

### Modo Motorista

**1) Adicionar Rota Completa:** Crie a viagem adicionando iterativamente a cidade de Origem e as respectivas cidades subsequentes. Para cada trecho criado, o sistema solicitará o horário de saída daquela cidade, valor fracionado do trecho e total de assentos.

**2) Minhas Rotas Publicadas:** Exibe os trechos independentes que você possui no sistema e o status atual da ocupação (`0/4`).

**3) Cancelar Rota Publicada:** Encerra a viagem informando o `ID` gerado na publicação.

### Modo Passageiro

**1) Buscar e Reservar Itinerário:** Digite Origem, Destino e Data. O sistema executará uma busca profunda e lhe entregará as rotas diretas, ou, caso não exista caminho direto, conectará motoristas diferentes apresentando um roteiro segmentado para você. Escolha o número desejado para efetivar. O sistema garante que a transação inteira ocorra de modo atômico.

**2) Minhas Reservas:** Visualize seu histórico de embarque com chaves de ID e somatório dos valores dos trechos adquiridos.

**3) Cancelar Reserva:** Informando o ID da Reserva, você cancela a ida e, automaticamente, suas vagas ocupadas são devolvidas ao banco do servidor para que outras pessoas comprem.

## Robustez e Concorrência

- A aplicação impede matematicamente inconsistências temporais (um passageiro tentar buscar um trecho cujo horário de chegada é anterior ao da saída).



- **Bloqueio de Overbooking:** Travas de exclusão (Mutexes) garantem que em caso de acesso em massa, o assento fique para a requisição que fechar o TCP primeiro. Nenhuma reserva parcial (conseguir apenas metade de uma viagem mista) ocorre dentro do fluxo do `GerenciadorDeRotas`.
