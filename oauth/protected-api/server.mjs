import http from "node:http";
import crypto from "node:crypto";

const port = Number(process.env.PORT || 3000);
const issuer = process.env.OAUTH_ISSUER;
const expectedClient = process.env.OAUTH_CLIENT_ID;
let jwksCache;

const json = (res, status, body) => {
  res.writeHead(status, { "content-type": "application/json" });
  res.end(JSON.stringify(body, null, 2));
};

const decode = (value) => JSON.parse(Buffer.from(value, "base64url").toString());

async function verifyJwt(token) {
  const parts = token.split(".");
  if (parts.length !== 3) throw new Error("token não é um JWT");
  const [encodedHeader, encodedPayload, signature] = parts;
  const header = decode(encodedHeader);
  const payload = decode(encodedPayload);

  if (!jwksCache) {
    const response = await fetch(`${issuer}/protocol/openid-connect/certs`);
    if (!response.ok) throw new Error(`JWKS indisponível: HTTP ${response.status}`);
    jwksCache = await response.json();
  }

  const jwk = jwksCache.keys.find((key) => key.kid === header.kid);
  if (!jwk) throw new Error("chave de assinatura desconhecida");
  const valid = crypto.verify(
    "RSA-SHA256",
    Buffer.from(`${encodedHeader}.${encodedPayload}`),
    crypto.createPublicKey({ key: jwk, format: "jwk" }),
    Buffer.from(signature, "base64url")
  );
  if (!valid) throw new Error("assinatura inválida");
  if (payload.iss !== issuer) throw new Error("issuer inválido");
  if (payload.exp * 1000 <= Date.now()) throw new Error("token expirado");
  if (payload.azp !== expectedClient) throw new Error("cliente OAuth inválido");
  return payload;
}

http.createServer(async (req, res) => {
  if (req.url === "/health") return json(res, 200, { status: "ok" });
  const authorization = req.headers.authorization || "";
  if (!authorization.startsWith("Bearer ")) {
    return json(res, 401, { error: "missing_bearer_token" });
  }
  try {
    const claims = await verifyJwt(authorization.slice(7));
    return json(res, 200, {
      message: "A API externa aceitou o token emitido pelo Keycloak.",
      path: req.url,
      validated_claims: {
        issuer: claims.iss,
        client: claims.azp,
        subject: claims.sub,
        expires_at: new Date(claims.exp * 1000).toISOString()
      }
    });
  } catch (error) {
    return json(res, 401, { error: "invalid_token", detail: error.message });
  }
}).listen(port, "0.0.0.0", () => console.log(`Protected API listening on ${port}`));
