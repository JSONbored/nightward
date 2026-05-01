#!/usr/bin/env node
import { readFileSync } from "node:fs";
import { execFileSync } from "node:child_process";

const stalePatterns = [
  /After the first tagged release/i,
  /First signed `?v0\.1\.0`? release/i,
  /Trusted npm publishing/i,
  /uses:\s*JSONbored\/nightward@v0\.1\.0/i,
  /trunk .*v0\.1\.0/i,
  /semantic_version:\s*0\.1\.0/i,
];

const files = execFileSync("git", [
  "ls-files",
  "README.md",
  "docs/*.md",
  "docs/**/*.md",
  "site/*.md",
  "site/**/*.md",
  ".nightward*.yml",
], { encoding: "utf8" })
  .trim()
  .split("\n")
  .filter(Boolean);

const failures = [];
for (const file of files) {
  const text = readFileSync(file, "utf8");
  for (const pattern of stalePatterns) {
    if (pattern.test(text)) {
      failures.push(`${file}: ${pattern}`);
    }
  }
}

if (failures.length > 0) {
  console.error("Stale Nightward docs copy found:");
  for (const failure of failures) console.error(`- ${failure}`);
  process.exit(1);
}
