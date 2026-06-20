import { createServer } from "http";

const port = Number(process.env.PORT || 3000);

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

function consultaResponse() {
  return `Protocolo: CNS-20260620-0042
Paciente: Joao Santos
Medico: Dra. Ana Costa (CRM 12345)
Data: 25/06/2026 as 14:30
Unidade: UBS Jardim das Flores
Status: Confirmada`;
}

const server = createServer((request, response) => {
  const path = new URL(request.url, `http://${request.headers.host}`).pathname;

  if (path.startsWith("/api/construcao/")) {
    response.writeHead(200, { "content-type": "application/xml" });
    response.end(certidaoResponse());
    return;
  }

  if (path.startsWith("/api/saude/")) {
    response.writeHead(200, { "content-type": "text/plain" });
    response.end(consultaResponse());
    return;
  }

  response.writeHead(404, { "content-type": "text/plain" });
  response.end("not found");
});

server.listen(port, () => {
  console.log(`Domain server listening on ${port}`);
});
