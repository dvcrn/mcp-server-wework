import { readFile, writeFile } from "node:fs/promises";

async function readJSON(path) {
  return JSON.parse(await readFile(path, "utf8"));
}

async function writeJSON(path, value) {
  await writeFile(path, `${JSON.stringify(value, null, 2)}\n`);
}

const npmPackage = await readJSON("npm/package.json");
const version = npmPackage.version;
if (typeof version !== "string" || version.trim() === "") {
  throw new Error("npm/package.json is missing a version");
}

// Re-write to normalise formatting
await writeJSON("npm/package.json", npmPackage);

console.log(`Version is ${version}`);
