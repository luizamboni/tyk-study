import http from "node:http";

const port = Number(process.env.PORT || 3000);
const tokenUrl = process.env.TOKEN_URL;
const clientId = process.env.CLIENT_ID;
const clientSecret = process.env.CLIENT_SECRET;
const upstreamUrl = process.env.UPSTREAM_URL;
let cachedToken;

const json = (res, status, body) => {
  res.writeHead(status, { "content-type": "application/json" });
  res.end(JSON.stringify(body, null, 2));
};

async function accessToken() {
  if (cachedToken && Date.now() < cachedToken.expiresAt - 5000) {
    return { value: cachedToken.value, source: "cache" };
  }
  const credentials = Buffer.from(`${clientId}:${clientSecret}`).toString("base64");
  const response = await fetch(tokenUrl, {
    method: "POST",
    headers: {
      authorization: `Basic ${credentials}`,
      "content-type": "application/x-www-form-urlencoded"
    },
    body: "grant_type=client_credentials"
  });
  const body = await response.json();
  if (!response.ok) throw new Error(`token endpoint HTTP ${response.status}: ${JSON.stringify(body)}`);
  cachedToken = {
    value: body.access_token,
    expiresAt: Date.now() + body.expires_in * 1000
  };
  return { value: cachedToken.value, source: "keycloak" };
}

http.createServer(async (req, res) => {
  if (req.url === "/health") return json(res, 200, { status: "ok" });
  try {
    const token = await accessToken();
    const upstream = await fetch(`${upstreamUrl}${req.url}`, {
      method: req.method,
      headers: { authorization: `Bearer ${token.value}` }
    });
    const body = await upstream.text();
    res.writeHead(upstream.status, {
      "content-type": upstream.headers.get("content-type") || "application/json",
      "x-token-source": token.source
    });
    res.end(body);
  } catch (error) {
    json(res, 502, { error: "token_broker_error", detail: error.message });
  }
}).listen(port, "0.0.0.0", () => console.log(`Token broker listening on ${port}`));
