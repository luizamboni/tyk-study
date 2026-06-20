# Laboratorio Tyk: hub de integracoes

Ambiente local para explorar o Tyk Gateway OSS como um hub de
integracoes, onde cada integracao tem seu propio upstream e
metodo de autenticacao, roteados pelo plugin Go.

```
Cliente -> chave Tyk -> Tyk + plugin Go -> upstream (httpbin)
                                         -> token broker -> Keycloak -> protected-api
                                         -> upstream (httpbin) com Basic Auth
```

## Pre-requisitos

- Docker com Docker Compose
- `make` e `curl`

## Uso rapido

```sh
make up
make integration-httpbin
make oauth-upstream
make integration-echo
make down
```

## Rotas

Todas as requisicoes passam por uma unica API definition no Tyk:

```
/api/integration/{integracao}/operation/{operacao}
```

| Integracao | Operacao | Autenticacao | Upstream |
|---|---|---|---|
| `httpbin` | `get` | API key (`X-Api-Key`) | httpbin |
| `oauth` | `resource` | Bearer token (broker -> Keycloak) | protected-api |
| `echo` | `headers` | Basic Auth | httpbin |

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
| `make integration-httpbin` | Chama httpbin com API key injetada pelo plugin |
| `make oauth-upstream` | Fluxo OAuth2 completo (broker -> Keycloak -> JWT) |
| `make integration-echo` | Chama httpbin com Basic Auth injetado pelo plugin |
| `make oauth-token-cache` | Mostra o cache do access token no plugin Go |
| `make plugin-denied` | Mostra o plugin bloqueando operacao nao catalogada (403) |
| `make oauth-direct-denied` | Mostra que a API protegida rejeita chamadas sem token |

### Administracao do Tyk

| Comando | Descricao |
|---|---|
| `make reload` | Recarrega definicoes sem reiniciar o container |
| `make list-apis` | Lista as APIs carregadas |

## Arquitetura

O plugin Go intercepta requisicoes em `/api/integration/`, extrai o nome
da integracao e a operacao do path, consulta um catalogo interno que
define o upstream alvo e o tipo de autenticacao:

- **api-key**: le a chave de uma variavel de ambiente e injeta no header
- **oauth**: consulta o token broker (`POST /internal/v1/tokens/resolve`),
  que obtem um access token do Keycloak. O token e cacheado em memoria
  no plugin (30s, mesmo TTL do JWT). A API externa valida o JWT pelo JWKS.
- **basic**: le usuario/senha de variaveis de ambiente e injeta
  `Authorization: Basic ...`

Integracoes com autenticacao simples (api-key, basic) sao chamadas
diretamente pelo plugin via HTTP. A integracao OAuth modifica os headers
da requisicao original e deixa o Tyk fazer o proxy para o upstream.

O console administrativo do Keycloak esta em `http://localhost:8081`,
usuario e senha `admin`.

## Personalizacao

```sh
make integration-httpbin DEMO_KEY=outra-chave
```

As credenciais de cada integracao estao no `compose.yaml` como variaveis
de ambiente do container `tyk`. Este ambiente e didatico: nao reutilize
as credenciais em producao.
