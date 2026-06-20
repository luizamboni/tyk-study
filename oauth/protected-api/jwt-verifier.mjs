import crypto from "node:crypto";

import { expectedClient, issuer } from "./config.mjs";

let jwksCache;

function decodeJson(value) {
  const decoded = Buffer.from(value, "base64url").toString();
  return JSON.parse(decoded);
}

async function getJwks() {
  if (jwksCache) {
    return jwksCache;
  }

  const response = await fetch(`${issuer}/protocol/openid-connect/certs`);
  if (!response.ok) {
    throw new Error(`JWKS indisponível: HTTP ${response.status}`);
  }

  jwksCache = await response.json();
  return jwksCache;
}

function validateClaims(payload) {
  if (payload.iss !== issuer) {
    throw new Error("issuer inválido");
  }
  if (payload.exp * 1000 <= Date.now()) {
    throw new Error("token expirado");
  }
  if (payload.azp !== expectedClient) {
    throw new Error("cliente OAuth inválido");
  }
}

export async function verifyJwt(token) {
  const parts = token.split(".");
  if (parts.length !== 3) {
    throw new Error("token não é um JWT");
  }

  const [encodedHeader, encodedPayload, signature] = parts;
  const header = decodeJson(encodedHeader);
  const payload = decodeJson(encodedPayload);
  const jwks = await getJwks();
  const jwk = jwks.keys.find((key) => key.kid === header.kid);

  if (!jwk) {
    throw new Error("chave de assinatura desconhecida");
  }

  const publicKey = crypto.createPublicKey({ key: jwk, format: "jwk" });
  const signedContent = Buffer.from(`${encodedHeader}.${encodedPayload}`);
  const decodedSignature = Buffer.from(signature, "base64url");
  const valid = crypto.verify("RSA-SHA256", signedContent, publicKey, decodedSignature);

  if (!valid) {
    throw new Error("assinatura inválida");
  }

  validateClaims(payload);
  return payload;
}
