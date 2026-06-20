import { readFileSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const catalogPath = join(__dirname, "catalog.json");

let catalog;

export function loadCatalog() {
  const raw = readFileSync(catalogPath, "utf-8");
  catalog = JSON.parse(raw);
  return catalog;
}

export function getCatalog() {
  if (!catalog) loadCatalog();
  return catalog;
}

export function reloadCatalog() {
  try {
    loadCatalog();
    return { status: "ok", detail: "catalog reloaded" };
  } catch (error) {
    return { status: "error", detail: error.message };
  }
}

loadCatalog();
