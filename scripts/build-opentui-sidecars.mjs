#!/usr/bin/env node

import { existsSync } from "node:fs";
import { mkdir, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const packageRoot = path.join(repoRoot, "packages", "tui");
const outputRoot = process.env.NIGHTWARD_OPENTUI_OUT_DIR
  ? path.resolve(repoRoot, process.env.NIGHTWARD_OPENTUI_OUT_DIR)
  : path.join(repoRoot, "build", "opentui");
const packageJSON = JSON.parse(
  readFileSync(path.join(packageRoot, "package.json"), "utf8"),
);
const openTUIVersion = packageJSON.dependencies?.["@opentui/core"];

if (!openTUIVersion) {
  throw new Error("packages/tui/package.json is missing @opentui/core");
}

const targets = [
  {
    goos: "darwin",
    goarch: "amd64",
    bun: "bun-darwin-x64",
    opentui: "darwin-x64",
    binary: "nightward-tui",
  },
  {
    goos: "darwin",
    goarch: "arm64",
    bun: "bun-darwin-arm64",
    opentui: "darwin-arm64",
    binary: "nightward-tui",
  },
  {
    goos: "linux",
    goarch: "amd64",
    bun: "bun-linux-x64",
    opentui: "linux-x64",
    binary: "nightward-tui",
  },
  {
    goos: "linux",
    goarch: "arm64",
    bun: "bun-linux-arm64",
    opentui: "linux-arm64",
    binary: "nightward-tui",
  },
  {
    goos: "windows",
    goarch: "amd64",
    bun: "bun-windows-x64",
    opentui: "win32-x64",
    binary: "nightward-tui.exe",
  },
];

const selected = selectTargets(process.argv.slice(2));

for (const target of selected) {
  await ensureOpenTUIPackage(target.opentui);
  const outDir = path.join(outputRoot, `${target.goos}_${target.goarch}`);
  await mkdir(outDir, { recursive: true });
  run("bun", [
    "build",
    "src/main.ts",
    "--compile",
    `--target=${target.bun}`,
    "--outfile",
    path.join(outDir, target.binary),
  ]);
}

function selectTargets(args) {
  const only = args.find((arg) => arg.startsWith("--target="));
  if (!only) return targets;
  const value = only.slice("--target=".length);
  const matched = targets.filter(
    (target) =>
      value === `${target.goos}_${target.goarch}` ||
      value === `${target.goos}/${target.goarch}` ||
      value === target.bun,
  );
  if (matched.length === 0) {
    throw new Error(`unknown OpenTUI sidecar target: ${value}`);
  }
  return matched;
}

async function ensureOpenTUIPackage(platformArch) {
  const packageName = `@opentui/core-${platformArch}`;
  const packageDir = path.join(
    packageRoot,
    "node_modules",
    "@opentui",
    `core-${platformArch}`,
  );
  if (existsSync(path.join(packageDir, "package.json"))) {
    return;
  }

  await mkdir(packageDir, { recursive: true });
  const tempDir = await mkdirTemp("nightward-opentui-");
  try {
    const tarball = run("npm", [
      "pack",
      `${packageName}@${openTUIVersion}`,
      "--silent",
    ], tempDir)
      .trim()
      .split(/\r?\n/)
      .pop();
    if (!tarball) {
      throw new Error(`npm pack did not return a tarball for ${packageName}`);
    }
    execFileSync("tar", [
      "-xzf",
      path.join(tempDir, tarball),
      "-C",
      packageDir,
      "--strip-components=1",
    ], { stdio: "inherit" });
  } finally {
    await rm(tempDir, { recursive: true, force: true });
  }
}

async function mkdirTemp(prefix) {
  const dir = path.join(tmpdir(), `${prefix}${Date.now()}-${Math.random().toString(16).slice(2)}`);
  await mkdir(dir, { recursive: true });
  return dir;
}

function run(command, args, cwd = packageRoot) {
  return execFileSync(command, args, {
    cwd,
    encoding: "utf8",
    stdio: ["ignore", "pipe", "inherit"],
  });
}
