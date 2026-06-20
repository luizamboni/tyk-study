import { internalToken } from "./config.mjs";
import { sendJson } from "./http-response.mjs";
import { accessToken } from "./oauth-client.mjs";
import { readJson } from "./request-body.mjs";

const resolveTokenPath = "/internal/v1/tokens/resolve";

function validCredential(request) {
  return request.headers.authorization === `Bearer ${internalToken}`;
}

function knownIntegration(input) {
  return input.tenant_id === "tenant-a" && input.service === "oauth-demo";
}

export async function handleRequest(request, response) {
  if (request.url === "/health") {
    sendJson(response, 200, { status: "ok" });
    return;
  }

  if (request.url !== resolveTokenPath || request.method !== "POST") {
    sendJson(response, 404, { error: "not_found" });
    return;
  }

  if (!validCredential(request)) {
    sendJson(response, 401, { error: "invalid_internal_credential" });
    return;
  }

  try {
    const input = await readJson(request);
    if (!knownIntegration(input)) {
      sendJson(response, 404, { error: "credential_not_found" });
      return;
    }

    const token = await accessToken();
    sendJson(response, 200, {
      token_type: "Bearer",
      access_token: token.value,
      expires_at: new Date(token.expiresAt).toISOString(),
      source: token.source
    });
  } catch (error) {
    sendJson(response, 502, {
      error: "token_broker_error",
      detail: error.message
    });
  }
}
