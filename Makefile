SHELL := /bin/sh
.DEFAULT_GOAL := help

TYK_URL ?= http://localhost:8080
TYK_SECRET ?= development-secret
DEMO_KEY ?= demo-client-key
CLIENT_KEY := demo-org$(DEMO_KEY)
COMPOSE := docker compose

.PHONY: help check up down reset logs ps health proxy auth-denied create-key auth rate-limit upstream-credentials oauth-ready oauth-upstream oauth-direct-denied oauth-token-cache reload list-apis deploy-dynamic update-dynamic call-dynamic delete-dynamic hot-reload-demo demo

help: ## Lista os comandos disponíveis
	@awk 'BEGIN {FS = ":.*## "; printf "Uso: make <alvo>\n\n"} /^[a-zA-Z_-]+:.*## / {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

check: ## Valida a configuração do Docker Compose
	@command -v docker >/dev/null || { echo "docker não encontrado"; exit 1; }
	@$(COMPOSE) version >/dev/null
	@$(COMPOSE) config --quiet
	@echo "Configuração válida."

up: check ## Inicia Redis, upstream e Tyk
	@$(COMPOSE) up -d --wait
	@attempt=1; until curl --fail --silent "$(TYK_URL)/hello" >/dev/null; do \
	  [ $$attempt -ge 20 ] && { echo "Tyk não ficou disponível a tempo"; $(COMPOSE) logs tyk; exit 1; }; \
	  attempt=$$((attempt + 1)); sleep 1; \
	done
	@echo "Tyk disponível em $(TYK_URL)"

down: ## Encerra os containers
	@$(COMPOSE) down

reset: ## Encerra e remove também os dados do Redis
	@$(COMPOSE) down --volumes --remove-orphans

logs: ## Acompanha os logs do Tyk
	@$(COMPOSE) logs -f tyk

ps: ## Exibe o estado dos serviços
	@$(COMPOSE) ps

health: ## Verifica a saúde do Gateway
	@echo "=== Saúde do Tyk Gateway ==="
	@echo "Requisição: GET $(TYK_URL)/hello"
	@echo "A resposta abaixo identifica a versão e confirma que o Gateway está ativo:"
	@echo
	@curl --fail --silent --show-error "$(TYK_URL)/hello"; echo

proxy: ## Demonstra proxy reverso sem autenticação
	@echo "=== Exemplo 1: proxy reverso sem autenticação ==="
	@echo "Cliente chama:  GET $(TYK_URL)/public/get?example=proxy"
	@echo "Tyk encaminha: GET http://upstream/get?example=proxy"
	@echo "Transformação: remove o prefixo /public/ antes de chamar o upstream."
	@echo "Resultado esperado: HTTP 200 com o JSON produzido pelo HTTPBin."
	@echo
	@echo "Resposta do upstream recebida através do Tyk:"
	@curl --fail --silent --show-error "$(TYK_URL)/public/get?example=proxy"; echo

auth-denied: ## Demonstra a rejeição de uma chamada sem token
	@echo "=== Exemplo 2: acesso sem token ==="
	@echo "Cliente chama: GET $(TYK_URL)/protected/get"
	@echo "Cabeçalho Authorization: ausente"
	@echo "Resultado esperado: o Tyk bloqueia a chamada com HTTP 401 antes do upstream."
	@echo
	@status=$$(curl --silent --output /dev/null --write-out '%{http_code}' "$(TYK_URL)/protected/get"); \
	 test "$$status" = "401" && echo "Resultado: HTTP $$status — acesso bloqueado como esperado." || { echo "Falha: esperado HTTP 401, recebido $$status."; exit 1; }

create-key: ## Cria uma chave de acesso para a API protegida
	@echo "=== Preparação: criação de uma chave de acesso ==="
	@echo "A API administrativa do Tyk registrará a chave '$(CLIENT_KEY)'."
	@echo "Ela terá acesso às APIs protegidas deste laboratório."
	@echo
	@echo "Resposta da API administrativa:"
	@curl --fail --silent --show-error \
	  --header "x-tyk-authorization: $(TYK_SECRET)" \
	  --header "Content-Type: application/json" \
	  --request POST "$(TYK_URL)/tyk/keys/$(DEMO_KEY)" \
	  --data '{"alias":"Cliente demo","org_id":"demo-org","rate":100,"per":60,"quota_max":-1,"access_rights":{"protected-api":{"api_id":"protected-api","api_name":"API protegida por token","versions":["Default"]},"upstream-query-api":{"api_id":"upstream-query-api","api_name":"Upstream autenticado por query string","versions":["Default"]},"upstream-header-api":{"api_id":"upstream-header-api","api_name":"Upstream autenticado por cabecalho","versions":["Default"]},"oauth-broker-api":{"api_id":"oauth-broker-api","api_name":"API externa OAuth via token broker","versions":["Default"]}}}' && echo

auth: create-key ## Chama a API protegida usando a chave criada
	@echo
	@echo "=== Exemplo 3: acesso autenticado ==="
	@echo "Cliente chama: GET $(TYK_URL)/protected/get?example=auth"
	@echo "Credencial: cabeçalho Authorization com a chave criada acima."
	@echo "Resultado esperado: o Tyk valida a chave e encaminha a chamada ao HTTPBin."
	@echo
	@echo "Resposta do upstream recebida através do Tyk:"
	@curl --fail --silent --show-error --header "Authorization: $(CLIENT_KEY)" "$(TYK_URL)/protected/get?example=auth" && echo

rate-limit: ## Faz 5 chamadas; após 3, o Gateway deve responder 429
	@echo "=== Exemplo 4: limite global de requisições ==="
	@echo "Rota: GET $(TYK_URL)/limited/get"
	@echo "Regra: no máximo 3 requisições a cada 10 segundos."
	@echo "Serão feitas 5 chamadas consecutivas."
	@echo "Resultado esperado: 3 respostas HTTP 200 e depois 2 respostas HTTP 429."
	@echo
	@i=1; while [ $$i -le 5 ]; do \
	  status=$$(curl --silent --output /dev/null --write-out '%{http_code}' "$(TYK_URL)/limited/get"); \
	  if [ "$$status" = "200" ]; then meaning="permitida pelo Tyk"; \
	  elif [ "$$status" = "429" ]; then meaning="bloqueada pelo rate limit"; \
	  else meaning="resposta inesperada"; fi; \
	  echo "Requisição $$i: HTTP $$status — $$meaning"; i=$$((i + 1)); \
	done

upstream-credentials: create-key ## Usa uma chave Tyk em APIs com credenciais upstream distintas
	@echo
	@echo "=== Exemplo 5: uma chave do cliente, duas credenciais de upstream ==="
	@echo "O cliente usará sempre: Authorization: $(CLIENT_KEY)"
	@echo
	@echo "API A: $(TYK_URL)/services/query-auth/anything"
	@echo "O Tyk remove a chave do cliente e inclui api_key na query string do upstream."
	@echo "O campo args da resposta permite observar a credencial recebida pelo HTTPBin:"
	@curl --fail --silent --show-error --header "Authorization: $(CLIENT_KEY)" \
	  "$(TYK_URL)/services/query-auth/anything" && echo
	@echo
	@echo "API B: $(TYK_URL)/services/header-auth/anything"
	@echo "O Tyk remove a chave do cliente e inclui X-Upstream-Api-Key no upstream."
	@echo "O campo headers permite observar a credencial recebida pelo HTTPBin:"
	@curl --fail --silent --show-error --header "Authorization: $(CLIENT_KEY)" \
	  "$(TYK_URL)/services/header-auth/anything" && echo
	@echo
	@echo "Conclusão: o cliente usou a mesma chave nas duas chamadas; cada API aplicou sua própria credencial."

oauth-direct-denied: ## Mostra que a API externa rejeita chamadas sem OAuth
	@echo "=== Proteção real da API externa ==="
	@echo "Chamada direta sem Bearer token: http://localhost:3001/resource"
	@status=$$(curl --silent --output /dev/null --write-out '%{http_code}' http://localhost:3001/resource); \
	 test "$$status" = "401" && echo "Resultado: HTTP $$status — bloqueada como esperado." || { echo "Esperado 401, recebido $$status"; exit 1; }

oauth-ready: ## Aguarda o Keycloak concluir a importação do realm
	@echo "Aguardando o realm 'tyk-demo' ficar disponível no Keycloak..."
	@attempt=1; until curl --fail --silent http://localhost:8081/realms/tyk-demo >/dev/null; do \
	  [ $$attempt -ge 30 ] && { echo "Keycloak não ficou disponível a tempo"; exit 1; }; \
	  attempt=$$((attempt + 1)); sleep 1; \
	done
	@echo "Keycloak pronto."

oauth-upstream: create-key oauth-direct-denied ## Demonstra Tyk, broker, Keycloak e API OAuth
	@$(MAKE) --no-print-directory oauth-ready
	@$(MAKE) --no-print-directory reload
	@echo
	@echo "=== API externa protegida por OAuth2 Client Credentials ==="
	@echo "1. Cliente autentica no Tyk com a chave única."
	@echo "2. Tyk remove essa chave e chama o token broker."
	@echo "3. Broker obtém um access token no Keycloak."
	@echo "4. Broker chama a API externa com Authorization: Bearer <token>."
	@echo "5. API valida assinatura, issuer, expiração e client_id pelo JWKS."
	@echo
	@curl --fail --silent --show-error --include \
	  --header "Authorization: $(CLIENT_KEY)" \
	  "$(TYK_URL)/services/oauth/resource" && echo

oauth-token-cache: create-key ## Mostra obtenção e reutilização do access token
	@$(MAKE) --no-print-directory oauth-ready
	@$(MAKE) --no-print-directory reload
	@$(COMPOSE) restart token-broker >/dev/null
	@sleep 1
	@echo "=== Cache do access token no broker ==="
	@echo "X-Token-Source=keycloak indica token novo; cache indica reutilização."
	@echo
	@i=1; while [ $$i -le 2 ]; do \
	  echo "Chamada $$i:"; \
	  curl --fail --silent --show-error --dump-header - --output /dev/null \
	    --header "Authorization: $(CLIENT_KEY)" "$(TYK_URL)/services/oauth/resource" \
	    | grep -i '^x-token-source:'; \
	  i=$$((i + 1)); \
	done

reload: ## Recarrega as APIs sem reiniciar o Gateway
	@echo "=== Hot reload das APIs ==="
	@echo "Requisição administrativa: GET $(TYK_URL)/tyk/reload/group"
	@echo "Efeito: todos os Gateways do grupo recarregam as definições sem derrubar o processo."
	@echo "O endpoint é assíncrono; este comando aguarda o ciclo de aplicação terminar."
	@echo
	@curl --fail --silent --show-error \
	  --header "x-tyk-authorization: $(TYK_SECRET)" \
	  "$(TYK_URL)/tyk/reload/group" && echo
	@sleep 2
	@echo "Reload aplicado; as novas rotas já podem receber tráfego."

list-apis: ## Lista as APIs conhecidas pelo Gateway
	@echo "=== APIs carregadas no Gateway ==="
	@curl --fail --silent --show-error \
	  --header "x-tyk-authorization: $(TYK_SECRET)" \
	  "$(TYK_URL)/tyk/apis" && echo

deploy-dynamic: ## Cria uma API via REST e aplica hot reload
	@echo "=== Criação dinâmica: versão v1 ==="
	@echo "A definição será enviada à Gateway API, sem reiniciar o container."
	@curl --fail --silent --show-error \
	  --header "x-tyk-authorization: $(TYK_SECRET)" \
	  --header "Content-Type: application/json" \
	  --request POST "$(TYK_URL)/tyk/apis/dynamic-api" \
	  --data @examples/dynamic-api-v1.json && echo
	@$(MAKE) --no-print-directory reload

update-dynamic: ## Atualiza a API via REST para a versão v2
	@echo "=== Atualização dinâmica: v1 → v2 ==="
	@echo "Mudanças: X-Config-Version=v2 e adição de X-New-Behavior=enabled."
	@curl --fail --silent --show-error \
	  --header "x-tyk-authorization: $(TYK_SECRET)" \
	  --header "Content-Type: application/json" \
	  --request POST "$(TYK_URL)/tyk/apis/dynamic-api" \
	  --data @examples/dynamic-api-v2.json && echo
	@$(MAKE) --no-print-directory reload

call-dynamic: ## Chama a API dinâmica e mostra a configuração aplicada
	@echo "=== Chamada à API dinâmica ==="
	@echo "Observe X-Config-Version e X-New-Behavior nos headers recebidos pelo HTTPBin:"
	@curl --fail --silent --show-error "$(TYK_URL)/dynamic/anything" && echo

delete-dynamic: ## Exclui a API dinâmica e aplica hot reload
	@echo "=== Exclusão dinâmica ==="
	@curl --fail --silent --show-error \
	  --header "x-tyk-authorization: $(TYK_SECRET)" \
	  --request DELETE "$(TYK_URL)/tyk/apis/dynamic-api" && echo
	@$(MAKE) --no-print-directory reload

hot-reload-demo: ## Demonstra criação e atualização on the fly
	@echo "Demonstração: controle de APIs em tempo de execução"
	@echo
	@$(MAKE) --no-print-directory deploy-dynamic
	@echo
	@echo "Estado inicial:"
	@$(MAKE) --no-print-directory call-dynamic
	@echo
	@$(MAKE) --no-print-directory update-dynamic
	@echo
	@echo "Estado após a atualização, sem reiniciar o Tyk:"
	@$(MAKE) --no-print-directory call-dynamic

demo: ## Executa todos os exemplos
	@echo "Laboratório Tyk: proxy, autenticação e rate limit"
	@echo
	@$(MAKE) --no-print-directory proxy
	@echo
	@$(MAKE) --no-print-directory auth-denied
	@echo
	@$(MAKE) --no-print-directory auth
	@echo
	@$(MAKE) --no-print-directory rate-limit
	@echo
	@$(MAKE) --no-print-directory upstream-credentials
