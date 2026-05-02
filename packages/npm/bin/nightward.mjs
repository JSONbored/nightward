#!/usr/bin/env node
import { createHash } from "node:crypto";
import { createWriteStream, existsSync, realpathSync } from "node:fs";
import { chmod, mkdir, readFile, rm } from "node:fs/promises";
import { get } from "node:https";
import { homedir, platform as osPlatform, tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { spawn, spawnSync } from "node:child_process";

const modulePath = fileURLToPath(import.meta.url);
const packageRoot = path.resolve(path.dirname(modulePath), "..");
const packageJSON = JSON.parse(await readFile(path.join(packageRoot, "package.json"), "utf8"));

export function targetFor(platform = osPlatform(), arch = process.arch) {
  const os = {
    darwin: "darwin",
    linux: "linux",
    win32: "windows"
  }[platform];
  const cpu = {
    arm64: "arm64",
    x64: "amd64"
  }[arch];
  if (!os || !cpu) {
    throw new Error(`unsupported platform: ${platform}/${arch}`);
  }
  if (os === "windows" && cpu === "arm64") {
    throw new Error(`unsupported platform: ${platform}/${arch}`);
  }
  return { os, arch: cpu, extension: os === "windows" ? ".zip" : ".tar.gz" };
}

export function assetName(version, target = targetFor()) {
  return `nightward_${version}_${target.os}_${target.arch}${target.extension}`;
}

export function parseChecksums(text) {
  const checksums = new Map();
  for (const line of text.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed) {
      continue;
    }
    const [hash, file] = trimmed.split(/\s+/, 2);
    if (/^[a-f0-9]{64}$/i.test(hash) && file) {
      checksums.set(path.basename(file), hash.toLowerCase());
    }
  }
  return checksums;
}

export async function sha256(file) {
  const bytes = await readFile(file);
  return createHash("sha256").update(bytes).digest("hex");
}

export async function verifyArchiveChecksum(file, expected) {
  const actual = await sha256(file);
  if (actual !== expected.toLowerCase()) {
    throw new Error(`checksum mismatch for ${path.basename(file)}: expected ${expected}, got ${actual}`);
  }
}

export function cacheRoot() {
  if (process.env.NIGHTWARD_NPM_CACHE) {
    return process.env.NIGHTWARD_NPM_CACHE;
  }
  return path.join(homedir(), ".cache", "nightward", "npm");
}

export function releaseVersion() {
  return process.env.NIGHTWARD_NPM_VERSION || packageJSON.version;
}

export function releaseBaseURL(version = releaseVersion()) {
  if (process.env.NIGHTWARD_NPM_DOWNLOAD_BASE) {
    return process.env.NIGHTWARD_NPM_DOWNLOAD_BASE.replace(/\/$/, "");
  }
  return `https://github.com/JSONbored/nightward/releases/download/v${version}`;
}

export function commandName(argv1 = process.argv[1]) {
  const invoked = path.basename(argv1 || "nightward").toLowerCase();
  return invoked.startsWith("nw") ? "nw" : "nightward";
}

export function cachedBinaryPath(command = commandName(), version = releaseVersion(), target = targetFor()) {
  const binary = target.os === "windows" ? `${command}.exe` : command;
  return path.join(cacheRoot(), version, `${target.os}-${target.arch}`, binary);
}

async function download(url, destination, redirects = 0) {
  if (redirects > 5) {
    throw new Error(`too many redirects while downloading ${url}`);
  }
  await mkdir(path.dirname(destination), { recursive: true });
  await new Promise((resolve, reject) => {
    const request = get(url, (response) => {
      if ([301, 302, 303, 307, 308].includes(response.statusCode || 0) && response.headers.location) {
        response.resume();
        download(new URL(response.headers.location, url).toString(), destination, redirects + 1).then(resolve, reject);
        return;
      }
      if (response.statusCode !== 200) {
        response.resume();
        reject(new Error(`download failed for ${url}: HTTP ${response.statusCode}`));
        return;
      }
      const file = createWriteStream(destination, { mode: 0o600 });
      response.pipe(file);
      file.on("finish", () => file.close(resolve));
      file.on("error", reject);
    });
    request.on("error", reject);
  });
}

async function downloadText(url) {
  const file = path.join(tmpdir(), `nightward-checksums-${process.pid}-${Date.now()}.txt`);
  try {
    await download(url, file);
    return await readFile(file, "utf8");
  } finally {
    await rm(file, { force: true });
  }
}

async function extractArchive(archive, destination, target = targetFor()) {
  await mkdir(destination, { recursive: true });
  const result = target.os === "windows"
    ? spawnSync("powershell", [
      "-NoProfile",
      "-ExecutionPolicy",
      "Bypass",
      "-Command",
      "Expand-Archive",
      "-LiteralPath",
      archive,
      "-DestinationPath",
      destination,
      "-Force"
    ], { stdio: "inherit" })
    : spawnSync("tar", ["-xzf", archive, "-C", destination], { stdio: "inherit" });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error(`failed to extract ${path.basename(archive)}`);
  }
}

export async function ensureBinary(command = commandName()) {
  if (process.env.NIGHTWARD_BIN) {
    return process.env.NIGHTWARD_BIN;
  }

  const version = releaseVersion();
  if (version.includes("development")) {
    throw new Error("the development npm package cannot download a release binary; set NIGHTWARD_BIN for local testing");
  }

  const target = targetFor();
  const binary = cachedBinaryPath(command, version, target);
  if (existsSync(binary)) {
    return binary;
  }

  const asset = assetName(version, target);
  const baseURL = releaseBaseURL(version);
  const archive = path.join(cacheRoot(), version, `${target.os}-${target.arch}`, asset);
  const installDir = path.dirname(binary);
  const checksums = parseChecksums(await downloadText(`${baseURL}/checksums.txt`));
  const expected = checksums.get(asset);
  if (!expected) {
    throw new Error(`checksums.txt does not include ${asset}`);
  }

  await download(`${baseURL}/${asset}`, archive);
  await verifyArchiveChecksum(archive, expected);
  await extractArchive(archive, installDir, target);
  await chmod(binary, 0o755);
  return binary;
}

async function main() {
  const command = commandName();
  const binary = await ensureBinary(command);
  const result = await new Promise((resolve, reject) => {
    const child = spawn(binary, process.argv.slice(2), { stdio: "inherit" });
    child.on("error", reject);
    child.on("exit", (code, signal) => resolve({ code: code ?? 1, signal }));
  });

  if (result.signal) {
    process.kill(process.pid, result.signal);
    return;
  }
  process.exit(result.code);
}

function isMainModule() {
  if (!process.argv[1]) {
    return false;
  }
  try {
    return realpathSync(modulePath) === realpathSync(process.argv[1]);
  } catch {
    return import.meta.url === pathToFileURL(process.argv[1]).href;
  }
}

if (isMainModule()) {
  main().catch((error) => {
    console.error(`nightward launcher failed: ${error.message}`);
    process.exit(1);
  });
}
