import { clientId, clientSecret, tokenUrl } from "./config.mjs";

const expirationMarginMs = 5000;
let cachedToken;

function validCachedToken() {
  if (!cachedToken) {
    return undefined;
  }
  if (Date.now() >= cachedToken.expiresAt - expirationMarginMs) {
    return undefined;
  }
  return cachedToken;
}

function basicCredentials() {
  return Buffer.from(`${clientId}:${clientSecret}`).toString("base64");
}

async function requestToken() {
  const response = await fetch(tokenUrl, {
    method: "POST",
    headers: {
      authorization: `Basic ${basicCredentials()}`,
      "content-type": "application/x-www-form-urlencoded"
    },
    body: "grant_type=client_credentials"
  });
  const body = await response.json();

  if (!response.ok) {
    throw new Error(`token endpoint HTTP ${response.status}: ${JSON.stringify(body)}`);
  }

  cachedToken = {
    value: body.access_token,
    expiresAt: Date.now() + body.expires_in * 1000
  };

  return cachedToken;
}

export async function accessToken() {
  const token = validCachedToken();
  if (token) {
    return { ...token, source: "cache" };
  }

  const newToken = await requestToken();
  return { ...newToken, source: "keycloak" };
}
