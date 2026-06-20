import { createServer } from "http";

const port = Number(process.env.PORT || 3000);

function gerarPdfBase64(dados, tamanhoMB = 0) {
  const streamContent =
    "BT /F1 10 Tf\n" +
    "50 780 Td (SECRETARIA MUNICIPAL DE SAUDE) Tj\n" +
    "0 -20 Td (Protocolo: " + dados.protocolo + ") Tj\n" +
    "0 -20 Td (Paciente: " + dados.paciente + ") Tj\n" +
    "0 -20 Td (Medico: " + dados.medico + ") Tj\n" +
    "0 -20 Td (Data: " + dados.data + ") Tj\n" +
    "0 -20 Td (Status: " + dados.status + ") Tj\n" +
    "ET";

  const streamBytes = Buffer.from(streamContent, "latin1");
  const header = Buffer.from("%PDF-1.4\n", "latin1");
  const obj1 = Buffer.from("1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n", "latin1");
  const obj2 = Buffer.from("2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n", "latin1");
  const obj3 = Buffer.from("3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 612 792]/Contents 4 0 R/Resources<</Font<</F1 5 0 R>>>>>>endobj\n", "latin1");
  const obj4Head = Buffer.from("4 0 obj<</Length " + streamBytes.length + ">>stream\n", "latin1");
  const obj4Tail = Buffer.from("\nendstream\nendobj\n", "latin1");
  const obj5 = Buffer.from("5 0 obj<</Type/Font/Subtype/Type1/BaseFont/Helvetica>>endobj\n", "latin1");

  const parts = [header, obj1, obj2, obj3, obj4Head, streamBytes, obj4Tail, obj5];
  let numObjects = 6;

  if (tamanhoMB > 0) {
    const targetBytes = tamanhoMB * 1024 * 1024;
    const baseSize = parts.reduce((s, p) => s + p.length, 0);
    const xrefOverhead = 120;
    const padNeeded = Math.max(0, targetBytes - baseSize - xrefOverhead);
    if (padNeeded > 50) {
      const padStream = Buffer.alloc(padNeeded, 48); // '0' bytes
      const obj6 = Buffer.concat([
        Buffer.from("6 0 obj<</Length " + padNeeded + ">>stream\n", "latin1"),
        padStream,
        Buffer.from("\nendstream\nendobj\n", "latin1"),
      ]);
      parts.push(obj6);
      numObjects = 7;
    }
  }

  const offsets = [];
  let pos = 0;
  for (const p of parts) {
    offsets.push(pos);
    pos += p.length;
  }
  const xrefOffset = pos;

  const xrefEntries = [
    "0000000000 65535 f \n",
    String(offsets[1]).padStart(10, "0") + " 00000 n \n",
    String(offsets[2]).padStart(10, "0") + " 00000 n \n",
    String(offsets[3]).padStart(10, "0") + " 00000 n \n",
    String(offsets[4]).padStart(10, "0") + " 00000 n \n",
    String(offsets[7]).padStart(10, "0") + " 00000 n \n",
  ];
  if (numObjects > 6) {
    xrefEntries.push(String(offsets[8]).padStart(10, "0") + " 00000 n \n");
  }

  const xref =
    "xref\n0 " + numObjects + "\n" +
    xrefEntries.join("") +
    "trailer\n<</Size " + numObjects + "/Root 1 0 R>>\n" +
    "startxref\n" + xrefOffset + "\n%%EOF";

  const buf = Buffer.concat([...parts, Buffer.from(xref, "latin1")]);
  return buf.toString("base64");
}

function consultaResponse(tamanhoMB = 0) {
  const dados = {
    protocolo: "CNS-20260620-0042",
    paciente: "Joao Santos",
    medico: "Dra. Ana Costa (CRM 12345)",
    data: "25/06/2026 as 14:30",
    status: "Confirmada"
  };
  const suffixo = tamanhoMB > 0 ? "-" + tamanhoMB + "mb" : "";
  return {
    filename: "consulta" + suffixo + ".pdf",
    content_type: "application/pdf",
    content: gerarPdfBase64(dados, tamanhoMB),
    metadata: dados
  };
}

function certidaoResponse() {
  return `<?xml version="1.0" encoding="UTF-8"?>
<certidao>
  <numero>2024/123</numero>
  <imovel>
    <matricula>MAT-45921</matricula>
    <endereco>Rua Aurora, 456 - Centro</endereco>
    <area_m2>850.00</area_m2>
    <proprietario>Construtora Aurora Ltda</proprietario>
  </imovel>
  <situacao>regular</situacao>
  <emissao>2026-06-20</emissao>
  <validade>2027-06-20</validade>
</certidao>`;
}

const server = createServer((request, response) => {
  const url = new URL(request.url, `http://${request.headers.host || "localhost"}`);
  const path = url.pathname;
  const tamanho = Math.max(0, parseInt(url.searchParams.get("tamanho") || "0", 10));

  if (path.startsWith("/api/construcao/")) {
    response.writeHead(200, { "content-type": "application/xml" });
    response.end(certidaoResponse());
    return;
  }

  if (path.startsWith("/api/saude/")) {
    response.writeHead(200, { "content-type": "application/json" });
    response.end(JSON.stringify(consultaResponse(tamanho)));
    return;
  }

  response.writeHead(404, { "content-type": "text/plain" });
  response.end("not found");
});

server.listen(port, () => {
  console.log(`Domain server listening on ${port}`);
});
