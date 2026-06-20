import { sendJson } from "./http-response.mjs";
import { accessToken } from "./oauth-client.mjs";
import { readJson } from "./request-body.mjs";
import { getCatalog, reloadCatalog } from "./catalog-loader.mjs";

const resolvePath = "/internal/v1/credentials/resolve";
const reloadPath = "/internal/v1/reload";

function findOperation(tenant, integrationName, operationName, method) {
  const catalog = getCatalog();
  const tenantConfig = catalog.tenants?.[tenant];
  if (!tenantConfig) return null;
  const integration = tenantConfig.integrations?.[integrationName];
  if (!integration) return null;
  const operation = integration.operations?.[operationName];
  if (!operation) return null;
  if (operation.method && operation.method !== method) return null;
  return {
    target_url: integration.target_url,
    operation_path: operation.path || "/",
    auth_type: integration.auth?.type,
    ...integration.auth,
    ...operation
  };
}

async function resolveOAuth(found) {
  const catalog = getCatalog();
  const provider = catalog.oauth_providers?.[found.oauth_provider];
  if (!provider) {
    throw new Error(`oauth_provider '${found.oauth_provider}' nao encontrado no catalog`);
  }
  const token = await accessToken(provider.token_url, provider.client_id, provider.client_secret);
  return {
    auth_type: "bearer",
    credentials: { access_token: token.value },
    expires_at: new Date(token.expiresAt).toISOString(),
    source: token.source
  };
}

function resolveApiKey(config) {
  return {
    auth_type: "api-key",
    credentials: { header_name: config.header_name, header_value: config.header_value },
    source: "static"
  };
}

function resolveBasic(config) {
  return {
    auth_type: "basic",
    credentials: { username: config.username, password: config.password },
    source: "static"
  };
}

export async function handleRequest(request, response) {
  if (request.url === "/health") {
    sendJson(response, 200, { status: "ok" });
    return;
  }

  if (request.url === reloadPath && request.method === "POST") {
    const result = reloadCatalog();
    const status = result.status === "ok" ? 200 : 500;
    sendJson(response, status, result);
    return;
  }

  if (request.url !== resolvePath || request.method !== "POST") {
    sendJson(response, 404, { error: "not_found" });
    return;
  }

  try {
    const input = await readJson(request);
    const found = findOperation(input.tenant_id, input.integration, input.operation, input.method);
    if (!found) {
      sendJson(response, 404, { error: "credential_not_found", detail: `${input.integration}/${input.operation}` });
      return;
    }

    let result;
    switch (found.auth_type) {
      case "bearer":
        result = await resolveOAuth(found);
        break;
      case "api-key":
        result = resolveApiKey(found);
        break;
      case "basic":
        result = resolveBasic(found);
        break;
      default:
        sendJson(response, 400, { error: "unknown_auth_type", detail: found.auth_type });
        return;
    }

    sendJson(response, 200, {
      target_url: found.target_url,
      operation_path: found.operation_path,
      auth_type: result.auth_type,
      credentials: result.credentials,
      expires_at: result.expires_at || null,
      source: result.source
    });
  } catch (error) {
    sendJson(response, 502, {
      error: "credential_broker_error",
      detail: error.message
    });
  }
}
