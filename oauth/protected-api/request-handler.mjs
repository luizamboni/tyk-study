import { sendJson } from "./http-response.mjs";
import { verifyJwt } from "./jwt-verifier.mjs";

function successBody(request, claims) {
  return {
    aluno: {
      matricula: "20240101",
      nome: "Maria Silva",
      cpf: "123.456.789-00"
    },
    curso: {
      codigo: "ENG-CIV",
      nome: "Engenharia Civil",
      instituicao: "Universidade Federal do Brasil",
      periodo_ingresso: "2024.1"
    },
    historico: [
      { codigo: "MAT101", disciplina: "Calculo I", carga_horaria: 80, nota: 8.5, frequencia: "92%", status: "aprovado" },
      { codigo: "FIS101", disciplina: "Fisica I", carga_horaria: 80, nota: 7.0, frequencia: "88%", status: "aprovado" },
      { codigo: "QUI101", disciplina: "Quimica Geral", carga_horaria: 60, nota: 6.5, frequencia: "75%", status: "aprovado" },
      { codigo: "DES101", disciplina: "Desenho Tecnico", carga_horaria: 40, nota: 9.0, frequencia: "95%", status: "aprovado" },
      { codigo: "INT101", disciplina: "Introducao a Engenharia", carga_horaria: 40, nota: 7.5, frequencia: "90%", status: "aprovado" },
      { codigo: "MAT102", disciplina: "Calculo II", carga_horaria: 80, nota: 4.0, frequencia: "60%", status: "reprovado" }
    ],
    cr: 7.08,
    creditos_cursados: 300,
    creditos_integrais: 3200,
    jwt_validated: {
      issuer: claims.iss,
      client: claims.azp,
      expires_at: new Date(claims.exp * 1000).toISOString()
    }
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
