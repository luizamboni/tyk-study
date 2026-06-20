# Laboratório básico do Tyk Gateway

Projeto mínimo para explorar o Tyk Gateway OSS localmente. O ambiente contém o
Gateway, Redis e um `httpbin` usado como API upstream.

## Pré-requisitos

- Docker com o plugin Docker Compose
- `make` e `curl`

## Uso

```sh
make up
make demo
make down
```

Use `make help` para ver cada exemplo separadamente. As rotas demonstram:

- `/public/`: proxy reverso sem autenticação;
- `/protected/`: autenticação por token e criação de chave pela API administrativa;
- `/limited/`: limite global de 3 requisições a cada 10 segundos.
- `/services/query-auth/`: troca a chave do cliente por uma API key na query string;
- `/services/header-auth/`: troca a mesma chave do cliente por outra API key em cabeçalho.

O exemplo de credenciais de upstream pode ser executado separadamente:

```sh
make upstream-credentials
```

## Atualização on the fly

A Gateway API permite criar, atualizar, listar e excluir APIs em tempo de
execução. O exemplo abaixo muda uma API de `v1` para `v2` e executa hot reload
sem reiniciar o container do Tyk:

```sh
make hot-reload-demo
```

Os comandos administrativos também podem ser executados separadamente:

```sh
make deploy-dynamic
make call-dynamic
make update-dynamic
make reload
make list-apis
make delete-dynamic
```

## API externa protegida por OAuth2

O laboratório inclui um cenário completo de Client Credentials:

```text
Cliente → chave Tyk → Tyk + plugin Go → API protegida
                              │
                              └→ Token Broker → Keycloak
```

O Keycloak emite access tokens de 30 segundos. O broker guarda as credenciais
OAuth e emite tokens para o plugin. O plugin mantém cache local, resolve
`tenant + serviço + ação`, injeta o Bearer token e chama diretamente a API
externa. A API externa valida criptograficamente o JWT pelo JWKS.

```sh
make oauth-upstream
make oauth-token-cache
make plugin-denied
```

O plugin é compilado para a versão e arquitetura exatas do Gateway:

```sh
make plugin-build
```

O console administrativo do Keycloak fica em `http://localhost:8081`, com
usuário e senha `admin`, exclusivamente para este laboratório.

O diretório `tyk/apps` é gravável pelo container porque, no modo OSS baseado em
arquivos, a Gateway API persiste ali as definições recebidas. A Gateway API deve
ficar restrita à rede administrativa em ambientes reais.

As credenciais didáticas dos upstreams são fornecidas ao container por variáveis
de ambiente e referenciadas nas definições com `$secret_env`. Para alterá-las:

```sh
UPSTREAM_QUERY_API_KEY=segredo-a \
UPSTREAM_HEADER_API_KEY=segredo-b \
docker compose up -d --force-recreate tyk
```

As definições ficam em `tyk/apps/`. O segredo administrativo e a chave são
deliberadamente locais e podem ser sobrescritos, por exemplo:

```sh
make auth DEMO_KEY=outra-chave
```

Em produção, use um cofre de segredos, como Vault ou Consul, em vez de valores
default no Compose. Este ambiente é didático: não reutilize os segredos, portas
expostas ou a configuração permissiva em produção.
