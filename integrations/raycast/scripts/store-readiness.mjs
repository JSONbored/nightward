#!/usr/bin/env node
import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const strict = process.argv.includes("--strict");
const warnings = [];
const errors = [];

function requireField(object, field, label = field) {
  if (!object[field]) errors.push(`missing ${label}`);
}

function warn(message) {
  warnings.push(message);
}

function fail(message) {
  errors.push(message);
}

function fileExists(path) {
  return existsSync(path) && statSync(path).isFile();
}

function pngDimensions(path) {
  const data = readFileSync(path);
  if (data.length < 24 || data.toString("ascii", 1, 4) !== "PNG") return null;
  return { width: data.readUInt32BE(16), height: data.readUInt32BE(20) };
}

const pkg = JSON.parse(readFileSync(join(root, "package.json"), "utf8"));
for (const field of ["name", "title", "description", "author", "license", "platforms", "categories", "commands", "icon"]) {
  requireField(pkg, field);
}
if (pkg.license !== "MIT") fail("license must be MIT for Raycast Store submission");
if (!Array.isArray(pkg.platforms) || !pkg.platforms.includes("macOS")) fail("platforms must include macOS");
if (!fileExists(join(root, "package-lock.json"))) fail("package-lock.json is required");
if (!fileExists(join(root, "README.md"))) fail("README.md is required");
if (!fileExists(join(root, "CHANGELOG.md"))) fail("CHANGELOG.md is required");

const iconPath = join(root, pkg.icon || "");
if (!fileExists(iconPath)) {
  fail(`manifest icon not found: ${pkg.icon}`);
} else {
  const dimensions = pngDimensions(iconPath);
  if (!dimensions || dimensions.width !== 512 || dimensions.height !== 512) {
    fail("manifest icon must be a 512x512 PNG");
  }
}

const commands = Array.isArray(pkg.commands) ? pkg.commands : [];
for (const command of commands) {
  for (const field of ["name", "title", "description", "mode"]) {
    requireField(command, field, `command ${command.name || "(unnamed)"} ${field}`);
  }
  const sourceCandidates = [
    join(root, "src", `${command.name}.tsx`),
    join(root, "src", `${command.name}.ts`),
  ];
  if (!sourceCandidates.some(fileExists)) {
    fail(`command ${command.name} has no matching src/${command.name}.tsx or .ts`);
  }
}

const metadataDir = join(root, "metadata");
const screenshots = existsSync(metadataDir)
  ? readdirSync(metadataDir).filter((name) => /\.(png|jpg|jpeg)$/i.test(name))
  : [];
if (screenshots.length < 3) {
  warn(`Raycast Store recommends at least 3 metadata screenshots; found ${screenshots.length}`);
}

const readme = readFileSync(join(root, "README.md"), "utf8");
if (!/read-only/i.test(readme)) warn("README should state the read-only boundary");
if (!/Home Override/i.test(readme)) warn("README should document fixture Home Override preference");
if (!/Online Providers/i.test(readme)) warn("README should document online provider gating");

const result = {
  package: pkg.name,
  commands: commands.length,
  screenshots: screenshots.length,
  strict,
  errors,
  warnings,
  ready: errors.length === 0 && (!strict || warnings.length === 0),
};

console.log(JSON.stringify(result, null, 2));

if (errors.length > 0 || (strict && warnings.length > 0)) {
  process.exit(1);
}
