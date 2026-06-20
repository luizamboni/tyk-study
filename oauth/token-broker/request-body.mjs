export async function readJson(request) {
  const chunks = [];

  for await (const chunk of request) {
    chunks.push(chunk);
  }

  const content = Buffer.concat(chunks).toString() || "{}";
  return JSON.parse(content);
}
