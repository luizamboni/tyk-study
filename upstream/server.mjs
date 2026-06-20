import { createServer } from "http";

const port = Number(process.env.PORT || 3000);

function echoBody(request) {
  const hdrs = {};
  for (const [k, v] of Object.entries(request.headers)) {
    hdrs[k] = v;
  }
  return {
    method: request.method,
    url: request.url,
    path: new URL(request.url, `http://${request.headers.host}`).pathname,
    headers: hdrs
  };
}

const server = createServer((request, response) => {
  const body = JSON.stringify(echoBody(request), null, 2);
  response.writeHead(200, { "content-type": "application/json" });
  response.end(body);
});

server.listen(port, () => {
  console.log(`Echo server listening on ${port}`);
});
