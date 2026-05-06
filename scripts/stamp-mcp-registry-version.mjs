#!/usr/bin/env node
import { readFileSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const version = process.argv[2];

if (!version || !/^[0-9]+\.[0-9]+\.[0-9]+$/.test(version)) {
  throw new Error("usage: stamp-mcp-registry-version.mjs <x.y.z>");
}

const scriptDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(process.env.NIGHTWARD_REPO_ROOT ?? join(scriptDir, ".."));
const serverPath = join(repoRoot, "server.json");
const npmPackagePath = join(repoRoot, "packages", "npm", "package.json");

function readJSON(path) {
  return JSON.parse(readFileSync(path, "utf8"));
}

const server = readJSON(serverPath);
const npmPackage = readJSON(npmPackagePath);

if (npmPackage.version !== version) {
  throw new Error(
    `packages/npm/package.json is ${npmPackage.version}, expected ${version}`,
  );
}

if (server.name !== npmPackage.mcpName) {
  throw new Error(`server.json name ${server.name} does not match npm mcpName`);
}

const targets = server.packages?.filter(
  (entry) =>
    entry?.registryType === "npm" && entry?.identifier === npmPackage.name,
);

if (!targets || targets.length !== 1) {
  throw new Error(
    `expected one npm package target for ${npmPackage.name} in server.json`,
  );
}

server.version = version;
targets[0].version = version;

writeFileSync(serverPath, `${JSON.stringify(server, null, 2)}\n`);
console.log(`stamped MCP registry metadata to ${version}`);
