# Laboratorio Tyk: integracoes OAuth2 com plugin Go

Ambiente local para explorar o Tyk Gateway OSS com autenticacao OAuth2
Client Credentials via plugin Go, token broker e Keycloak.

```
Cliente -> chave Tyk -> Tyk + plugin Go -> API protegida
                              |
                              -> Token Broker -> Keycloak
```

## Pre-requisitos

- Docker com Docker Compose
- `make` e `curl`

## Uso rapido

```sh
make up
make oauth-upstream
make down
```

## O que cada comando faz

### Infraestrutura

| Comando | Descricao |
|---|---|
| `make up` | Sobe todos os containers (Tyk, Redis, Keycloak, etc.) |
| `make down` | Encerra os containers |
| `make reset` | Encerra e remove dados do Redis |
| `make logs` | Logs do Tyk |
| `make health` | Verifica se o Gateway esta ativo |

### Plugin Go

| Comando | Descricao |
|---|---|
| `make plugin-build` | Compila o plugin Go para a versao exata do Gateway |
| `make plugin-check` | Verifica se o .so existe; compila se necessario |

### Fluxo OAuth2

| Comando | Descricao |
|---|---|
| `make create-key` | Cria chave de acesso no Tyk |
| `make oauth-direct-denied` | Mostra que a API externa rejeita chamadas sem token |
| `make oauth-upstream` | Fluxo completo: Tyk -> broker -> Keycloak -> API |
| `make oauth-token-cache` | Mostra o cache do access token no plugin Go |
| `make plugin-denied` | Mostra o plugin bloqueando acao nao catalogada (HTTP 403) |

### Administracao do Tyk

| Comando | Descricao |
|---|---|
| `make reload` | Recarrega definicoes sem reiniciar o container |
| `make list-apis` | Lista as APIs carregadas |

## Arquitetura

O plugin Go intercepta requisicoes em `/integrations/`, extrai `tenant`,
`servico` e `acao` do path e headers, consulta o token broker via
`POST /internal/v1/tokens/resolve`, e chama a API externa com o
Bearer token obtido do Keycloak. O token e cacheado em memoria no
plugin (30s, mesmo TTL do JWT).

A API externa valida criptograficamente o JWT pelo JWKS do Keycloak.

O console administrativo do Keycloak esta em `http://localhost:8081`,
usuario e senha `admin`.

## Personalizacao

A chave de exemplo pode ser sobrescrita via ambiente:

```sh
make oauth-upstream DEMO_KEY=outra-chave
```

Os segredos dos servicos internos estao no `compose.yaml`. Este ambiente
e didatico: nao reutilize as credenciais em producao.
