#!/usr/bin/env node
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const scanPath = join(repoRoot, "site", "public", "demo", "nightward-sample-scan.json");
const report = JSON.parse(readFileSync(scanPath, "utf8"));
let failures = 0;

for (const item of report.items || []) {
  const expected = stableId(["item", item.tool || "", item.path || ""]);
  if (item.id !== expected) {
    console.error(`item id mismatch for ${item.path}: expected ${expected}, got ${item.id}`);
    failures += 1;
  }
}

for (const finding of report.findings || []) {
  const expected = `${finding.rule}-${stableId([
    finding.rule || "",
    finding.tool || "",
    finding.path || "",
    finding.server || "",
    finding.evidence || "",
  ])}`;
  if (finding.id !== expected) {
    console.error(`finding id mismatch for ${finding.rule}: expected ${expected}, got ${finding.id}`);
    failures += 1;
  }
}

if (failures > 0) {
  process.exit(1);
}
console.log("demo sample ids match scrubbed paths.");

function stableId(parts) {
  const hash = createHash("sha256");
  for (const part of parts) {
    hash.update(String(part));
    hash.update("\0");
  }
  return hash.digest("hex").slice(0, 12);
}
