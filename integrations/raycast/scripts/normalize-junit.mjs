import { readFileSync, writeFileSync } from "node:fs";

const [, , inputPath, outputPath] = process.argv;

if (!inputPath || !outputPath) {
  console.error("usage: normalize-junit.mjs <input> <output>");
  process.exit(2);
}

const raw = readFileSync(inputPath, "utf8");
const body = raw
  .replace(/^<\?xml[^>]*>\s*/u, "")
  .replace(/^<testsuites>\s*/u, "")
  .replace(/\s*<\/testsuites>\s*$/u, "")
  .trim();

const tests = count(raw, /<testcase\b/gu);
const failures = count(raw, /<failure\b/gu);
const errors = count(raw, /<error\b/gu);
const skipped = count(raw, /<skipped\b/gu);
const timestamp = new Date().toISOString();

const output = `<?xml version="1.0" encoding="utf-8"?>
<testsuites tests="${tests}" failures="${failures}" errors="${errors}" skipped="${skipped}" timestamp="${timestamp}">
  <testsuite name="raycast" tests="${tests}" failures="${failures}" errors="${errors}" skipped="${skipped}" timestamp="${timestamp}">
${indent(body, "    ")}
  </testsuite>
</testsuites>
`;

writeFileSync(outputPath, output);

function count(value, pattern) {
  return [...value.matchAll(pattern)].length;
}

function indent(value, prefix) {
  return value
    .split(/\r?\n/u)
    .map((line) => `${prefix}${line}`)
    .join("\n");
}
