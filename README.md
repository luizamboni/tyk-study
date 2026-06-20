# Laboratorio Tyk: hub de integracoes

Ambiente local para explorar o Tyk Gateway OSS como um hub de
integracoes, onde cada integracao tem seu propio upstream e
metodo de autenticacao, com resolucao centralizada de credenciais.

```
Cliente -> chave Tyk -> Tyk + plugin Go -> credential broker -> upstream (echo server)
                                                              -> Keycloak -> protected-api
                                                              -> upstream (echo server)
```

O **credential broker** é o ponto único de resolucao de autenticacao:
plugin pergunta "o que preciso para chamar X?", broker responde com
o tipo de auth + credenciais + target URL.

## Pre-requisitos

- Docker com Docker Compose
- `make` e `curl`

## Uso rapido

```sh
make up
make integration-construcao
make integration-educacao
make integration-saude
make down
```

## Rotas

Todas as requisicoes passam por uma unica API definition no Tyk:

```
/api/integration/{integracao}/operation/{operacao}
```

| Integracao | Operacao | Autenticacao | Rota upstream |
|---|---|---|---|
| `construcao` | `consultar-certidao` | API key (`X-Api-Key`) | `GET /api/construcao/v1/certidoes/123` |
| `educacao` | `emitir-historico` | Bearer token (broker -> Keycloak) | `GET /api/educacao/v1/historico` |
| `saude` | `agendar-consulta` | Basic Auth | `POST /api/saude/v1/consultas` |

## Comandos

### Infraestrutura

| Comando | Descricao |
|---|---|
| `make up` | Sobe todos os containers |
| `make down` | Encerra os containers |
| `make reset` | Encerra e remove dados do Redis |
| `make logs` | Logs do Tyk |
| `make health` | Verifica se o Gateway esta ativo |

### Plugin Go

| Comando | Descricao |
|---|---|
| `make plugin-build` | Compila o plugin Go para a versao exata do Gateway |
| `make plugin-check` | Verifica se o .so existe; compila se necessario |

### Integracoes

| Comando | Exemplo pratico |
|---|---|
| `make create-key` | Cria chave de acesso no Tyk |
| `make integration-construcao` | `GET /api/construcao/v1/certidoes/123` via API key |
| `make integration-educacao` | `GET /api/educacao/v1/historico` via OAuth2 |
| `make integration-saude` | `POST /api/saude/v1/consultas` via Basic Auth |
| `make cache-educacao` | Mostra o cache do access token (educacao) |
| `make denied-educacao` | Mostra o broker rejeitando operacao inexistente |
| `make oauth-direct-denied` | Mostra que a API protegida rejeita chamadas sem token |

### Administracao do Tyk

| Comando | Descricao |
|---|---|
| `make reload` | Recarrega definicoes sem reiniciar o container |
| `make list-apis` | Lista as APIs carregadas |

## Arquitetura

### Fluxo de requisicao

1. Cliente envia requisicao para `/api/integration/{nome}/operation/{op}`
   com uma chave Tyk que contem `tenant_id` nos metadados
2. Plugin Go extrai `{nome}`, `{op}` e o metodo HTTP do path
3. Plugin consulta o **credential broker** via
   `POST /internal/v1/credentials/resolve` com `{tenant, integration, operation, method}`
4. Broker consulta `catalog.json`, monta a resposta com `target_url`,
   `operation_path`, `auth_type` e as credenciais adequadas
5. Plugin aplica a autenticacao e encaminha:
   - **api-key / basic**: plugin faz a chamada HTTP diretamente ao upstream
   - **bearer**: plugin modifica headers e URL, Tyk faz o proxy

### Credential broker

Servico Node.js que centraliza toda configuracao de integracoes:

- `catalog.json` externo (montado como volume): tenants, integracoes,
  operacoes, credenciais e provedores OAuth
- `POST /internal/v1/reload` recarrega o catalog.json sem restart
- Cache de access tokens OAuth em memoria (expira junto com o JWT)

### Catalog.json

```json
{
  "oauth_providers": {
    "keycloak-tyk-demo": {
      "token_url": "http://keycloak:8080/realms/tyk-demo/...",
      "client_id": "tyk-broker",
      "client_secret": "broker-secret"
    }
  },
  "tenants": {
    "tenant-a": {
      "integrations": {
        "construcao": {
          "target_url": "http://upstream:3000",
          "auth": {
            "type": "api-key",
            "header_name": "X-Api-Key",
            "header_value": "demo-httpbin-key"
          },
          "operations": {
            "consultar-certidao": {
              "path": "/api/construcao/v1/certidoes/123",
              "method": "GET"
            }
          }
        }
      }
    }
  }
}
```

### Plugin Go

Nao possui catalogo interno nem le variaveis de ambiente de
credenciais. Delega toda resolucao de autenticacao ao broker.
Cacheia apenas access tokens OAuth (que expiram).

## Personalizacao

```sh
make integration-httpbin DEMO_KEY=outra-chave
```

Para alterar credenciais, edite `oauth/token-broker/catalog.json` e
recarregue sem restart:

```sh
curl -s -X POST http://localhost:3002/internal/v1/reload | jq '.'
```

Este ambiente e didatico: nao reutilize as credenciais em producao.
