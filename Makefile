SHELL := /bin/sh
.DEFAULT_GOAL := help

TYK_URL ?= http://localhost:8080
TYK_SECRET ?= development-secret
DEMO_KEY ?= demo-client-key
CLIENT_KEY := demo-org$(DEMO_KEY)
COMPOSE := docker compose

.PHONY: help check plugin-check plugin-build up down reset logs ps health create-key oauth-ready oauth-upstream oauth-direct-denied oauth-token-cache plugin-denied reload list-apis integration-httpbin integration-echo

help: ## Lista os comandos disponiveis
	@awk 'BEGIN {FS = ":.*## "; printf "Uso: make <alvo>\n\n"} /^[a-zA-Z_-]+:.*## / {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

check: ## Valida a configuracao do Docker Compose
	@command -v docker >/dev/null || { echo "docker nao encontrado"; exit 1; }
	@$(COMPOSE) version >/dev/null
	@$(COMPOSE) config --quiet
	@echo "Configuracao valida."

plugin-check: ## Verifica se o plugin Go esta compilado
	@test -f tyk/middleware/IntegrationBroker_v5.8.5_linux_arm64.so || $(MAKE) --no-print-directory plugin-build

plugin-build: ## Compila o plugin Go para Tyk 5.8.5 linux/arm64
	@echo "Compilando plugin Go com o compilador oficial do Tyk..."
	@rm -f plugin/IntegrationBroker_v5.8.5_linux_arm64.so
	@docker run --rm --platform=linux/amd64 \
	  --volume "$(CURDIR)/plugin:/plugin-source" \
	  tykio/tyk-plugin-compiler:v5.8.5 \
	  IntegrationBroker.so build-$$(date +%s) linux arm64
	@mkdir -p tyk/middleware
	@mv plugin/IntegrationBroker_v5.8.5_linux_arm64.so tyk/middleware/
	@echo "Plugin criado em tyk/middleware/IntegrationBroker_v5.8.5_linux_arm64.so"

up: check plugin-check ## Inicia Redis, Keycloak, token-broker e Tyk
	@$(COMPOSE) up -d --wait
	@attempt=1; until curl --fail --silent "$(TYK_URL)/hello" >/dev/null; do \
	  [ $$attempt -ge 20 ] && { echo "Tyk nao ficou disponivel a tempo"; $(COMPOSE) logs tyk; exit 1; }; \
	  attempt=$$((attempt + 1)); sleep 1; \
	done
	@echo "Tyk disponivel em $(TYK_URL)"

down: ## Encerra os containers
	@$(COMPOSE) down

reset: ## Encerra e remove tambem os dados do Redis
	@$(COMPOSE) down --volumes --remove-orphans

logs: ## Acompanha os logs do Tyk
	@$(COMPOSE) logs -f tyk

ps: ## Exibe o estado dos servicos
	@$(COMPOSE) ps

health: ## Verifica a saude do Gateway
	@echo "=== Saude do Tyk Gateway ==="
	@echo "Requiscao: GET $(TYK_URL)/hello"
	@echo "A resposta abaixo identifica a versao e confirma que o Gateway esta ativo:"
	@echo
	@curl --fail --silent --show-error "$(TYK_URL)/hello" | jq '.'; echo

create-key: ## Cria uma chave de acesso para as APIs de integracao
	@echo "=== Criacao de chave de acesso ==="
	@echo "A API administrativa do Tyk registrara a chave '$(CLIENT_KEY)'."
	@echo
	@echo "Resposta da API administrativa:"
	@curl --fail --silent --show-error \
	  --header "x-tyk-authorization: $(TYK_SECRET)" \
	  --header "Content-Type: application/json" \
	  --request POST "$(TYK_URL)/tyk/keys/$(DEMO_KEY)" \
	  --data '{"alias":"Cliente demo","org_id":"demo-org","rate":100,"per":60,"quota_max":-1,"meta_data":{"tenant_id":"tenant-a"},"access_rights":{"oauth-broker-api":{"api_id":"oauth-broker-api","api_name":"Integracoes via plugin Go","versions":["Default"]}}}' | jq '.' && echo

oauth-direct-denied: ## Mostra que a API externa rejeita chamadas sem OAuth
	@echo "=== Protecao real da API externa ==="
	@echo "Chamada direta sem Bearer token: http://localhost:3001/resource"
	@status=$$(curl --silent --output /dev/null --write-out '%{http_code}' http://localhost:3001/resource); \
	 test "$$status" = "401" && echo "Resultado: HTTP $$status — bloqueada como esperado." || { echo "Esperado 401, recebido $$status"; exit 1; }

oauth-ready: ## Aguarda o Keycloak concluir a importacao do realm
	@echo "Aguardando o realm 'tyk-demo' ficar disponivel no Keycloak..."
	@attempt=1; until curl --fail --silent http://localhost:8081/realms/tyk-demo >/dev/null; do \
	  [ $$attempt -ge 30 ] && { echo "Keycloak nao ficou disponivel a tempo"; exit 1; }; \
	  attempt=$$((attempt + 1)); sleep 1; \
	done
	@echo "Keycloak pronto."

oauth-upstream: create-key oauth-direct-denied ## Demonstra Tyk, broker, Keycloak e API OAuth
	@$(MAKE) --no-print-directory oauth-ready
	@$(MAKE) --no-print-directory reload
	@echo
	@echo "=== API externa protegida por OAuth2 Client Credentials ==="
	@echo "URL: /api/integration/oauth/operation/resource"
	@echo "Autenticacao: Bearer token obtido via broker -> Keycloak"
	@echo
	@curl --fail --silent --show-error \
	  --header "Authorization: $(CLIENT_KEY)" \
	  "$(TYK_URL)/api/integration/oauth/operation/resource" | jq '.' && echo

oauth-token-cache: create-key ## Mostra obtencao e reutilizacao do access token
	@$(MAKE) --no-print-directory oauth-ready
	@$(MAKE) --no-print-directory reload
	@$(COMPOSE) restart tyk >/dev/null
	@sleep 2
	@echo "=== Cache do access token no plugin Go ==="
	@echo "A primeira chamada consulta o broker; a segunda usa a memoria do plugin."
	@echo
	@i=1; while [ $$i -le 2 ]; do \
	  echo "Chamada $$i:"; \
	  curl --fail --silent --show-error \
	    --header "Authorization: $(CLIENT_KEY)" "$(TYK_URL)/api/integration/oauth/operation/resource" \
	    | jq '{ message, received_headers: { x_broker_source: .received_headers["x-broker-source"] } }'; \
	  echo; \
	  i=$$((i + 1)); \
	done

plugin-denied: create-key ## Mostra o broker rejeitando uma acao nao catalogada
	@echo "=== Acao nao catalogada ==="
	@http_code=$$(curl --silent --output /dev/null --write-out '%{http_code}' \
	  --header "Authorization: $(CLIENT_KEY)" "$(TYK_URL)/api/integration/oauth/operation/delete-all"); \
	 test "$$http_code" = "502" && echo "Resultado: HTTP 502 — broker rejeitou (credential_not_found)." || { echo "Esperado 502, recebido $$http_code"; exit 1; }

integration-httpbin: create-key ## Testa integracao httpbin com API key
	@echo "=== Integracao httpbin (API key) ==="
	@echo "Autenticacao: X-Api-Key injetado pelo plugin Go"
	@echo
	@curl --fail --silent --show-error \
	  --header "Authorization: $(CLIENT_KEY)" \
	  "$(TYK_URL)/api/integration/httpbin/operation/get" | jq '.url, .headers."X-Api-Key"' && echo

integration-echo: create-key ## Testa integracao echo com Basic Auth
	@echo "=== Integracao echo (Basic Auth) ==="
	@echo "Autenticacao: Basic Auth injetado pelo plugin Go"
	@echo
	@curl --fail --silent --show-error \
	  --header "Authorization: $(CLIENT_KEY)" \
	  "$(TYK_URL)/api/integration/echo/operation/headers" | jq '.headers.Authorization' && echo

reload: ## Recarrega as APIs sem reiniciar o Gateway
	@echo "=== Hot reload das APIs ==="
	@curl --fail --silent --show-error \
	  --header "x-tyk-authorization: $(TYK_SECRET)" \
	  "$(TYK_URL)/tyk/reload/group" | jq '.' && echo
	@sleep 2
	@echo "Reload aplicado; as novas rotas ja podem receber trafego."

list-apis: ## Lista as APIs conhecidas pelo Gateway
	@echo "=== APIs carregadas no Gateway ==="
	@curl --fail --silent --show-error \
	  --header "x-tyk-authorization: $(TYK_SECRET)" \
	  "$(TYK_URL)/tyk/apis" | jq '.' && echo
