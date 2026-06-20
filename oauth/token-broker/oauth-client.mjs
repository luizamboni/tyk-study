const expirationMarginMs = 5000;
let cachedToken;

function validCachedToken(clientId) {
  if (!cachedToken || cachedToken.clientId !== clientId) {
    return undefined;
  }
  if (Date.now() >= cachedToken.expiresAt - expirationMarginMs) {
    return undefined;
  }
  return cachedToken;
}

function basicCredentials(clientId, clientSecret) {
  return Buffer.from(`${clientId}:${clientSecret}`).toString("base64");
}

async function requestToken(tokenUrl, clientId, clientSecret) {
  const response = await fetch(tokenUrl, {
    method: "POST",
    headers: {
      authorization: `Basic ${basicCredentials(clientId, clientSecret)}`,
      "content-type": "application/x-www-form-urlencoded"
    },
    body: "grant_type=client_credentials"
  });
  const body = await response.json();

  if (!response.ok) {
    throw new Error(`token endpoint HTTP ${response.status}: ${JSON.stringify(body)}`);
  }

  cachedToken = {
    clientId,
    value: body.access_token,
    expiresAt: Date.now() + body.expires_in * 1000
  };

  return cachedToken;
}

export async function accessToken(tokenUrl, clientId, clientSecret) {
  const token = validCachedToken(clientId);
  if (token) {
    return { ...token, source: "cache" };
  }

  const newToken = await requestToken(tokenUrl, clientId, clientSecret);
  return { ...newToken, source: "keycloak" };
}
