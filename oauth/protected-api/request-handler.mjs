import { sendJson } from "./http-response.mjs";
import { verifyJwt } from "./jwt-verifier.mjs";

function receivedHeaders(request) {
  const hdrs = {};
  for (const [key, value] of Object.entries(request.headers)) {
    if (key.startsWith("x-") || key.startsWith("x-integration-") || key.startsWith("x-plugin-") || key.startsWith("x-broker-")) {
      hdrs[key] = value;
    }
  }
  hdrs["x-ratelimit-limit"] = request.headers["x-ratelimit-limit"];
  hdrs["x-ratelimit-remaining"] = request.headers["x-ratelimit-remaining"];
  return hdrs;
}

function successBody(request, claims) {
  return {
    message: "A API externa aceitou o token emitido pelo Keycloak.",
    path: request.url,
    validated_claims: {
      issuer: claims.iss,
      client: claims.azp,
      subject: claims.sub,
      expires_at: new Date(claims.exp * 1000).toISOString()
    },
    routing: {
      tenant: request.headers["x-tenant-id"],
      service: request.headers["x-integration-service"],
      action: request.headers["x-integration-action"],
      plugin_token_source: request.headers["x-plugin-token-source"],
      broker_token_source: request.headers["x-broker-token-source"]
    },
    received_headers: receivedHeaders(request)
  };
}

export async function handleRequest(request, response) {
  if (request.url === "/health") {
    sendJson(response, 200, { status: "ok" });
    return;
  }

  const authorization = request.headers.authorization || "";
  if (!authorization.startsWith("Bearer ")) {
    sendJson(response, 401, { error: "missing_bearer_token" });
    return;
  }

  try {
    const token = authorization.slice(7);
    const claims = await verifyJwt(token);
    sendJson(response, 200, successBody(request, claims));
  } catch (error) {
    sendJson(response, 401, {
      error: "invalid_token",
      detail: error.message
    });
  }
}
