import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { test } from "vitest";

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "../../..");

function readJSON(path) {
  return JSON.parse(readFileSync(path, "utf8"));
}

test("MCP Registry metadata targets the npm launcher", () => {
  const server = readJSON(join(repoRoot, "server.json"));
  const npmPackage = readJSON(join(repoRoot, "packages/npm/package.json"));

  assert.equal(server.name, npmPackage.mcpName);
  assert.ok(server.name.startsWith("io.github.jsonbored/"));
  assert.equal(server.version, npmPackage.version);
  assert.equal(server.packages.length, 1);

  const target = server.packages[0];
  assert.equal(target.registryType, "npm");
  assert.equal(target.identifier, npmPackage.name);
  assert.equal(target.version, npmPackage.version);
  assert.equal(target.transport?.type, "stdio");
  assert.equal(
    target.environmentVariables?.some((entry) => entry.isRequired) ?? false,
    false,
  );
});
