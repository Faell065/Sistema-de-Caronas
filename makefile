.PHONY: build run-servidor run-cliente run-teste-concorrencia docker-up docker-down docker-logs clean-data
.PHONY: help
.DEFAULT_GOAL := help

help:
	@echo "Comandos disponíveis:"
	@awk 'BEGIN {FS = ":.*##"; printf "\033[36m%-25s\033[0m %s\n", "Alvo", "Descrição"} /^[a-zA-Z_-]+:.*?##/ { printf "\033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Aplicações em Go

run-servidor: ## Executa o servidor Go localmente
	go run cmd/servidor/main.go

run-cliente: ## Executa o cliente Go apontando para o localhost
	go run cmd/cliente/main.go 127.0.0.1:8080

run-teste-concorrencia: ## Executa testes automatizados de concorrência e carga
	go run cmd/teste_carga/main.go $(SERVER_IP)

run-teste-integridade: ## Testa a integridade do sistema em cenarios complexos de requisições
	go run cmd/teste_integridade/main.go $(SERVER_IP)

##@ Docker

docker-up: ## Inicializa e recria os containers em segundo plano (background)
	docker compose up --build -d

docker-down: ## Para e remove os containers ativos
	docker compose down

docker-logs: ## Exibe e acompanha os logs dos containers em tempo real
	docker compose logs -f

##@ Manutenção

clean-data: ## Remove todos os arquivos JSON de dados gravados no disco
	rm -rf dados_persistentados/*.json
