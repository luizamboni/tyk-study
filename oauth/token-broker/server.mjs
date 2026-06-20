import http from "node:http";

import { port } from "./config.mjs";
import { handleRequest } from "./request-handler.mjs";

const server = http.createServer(handleRequest);

server.listen(port, "0.0.0.0", () => {
  console.log(`Token broker listening on ${port}`);
});
