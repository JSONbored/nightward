#!/usr/bin/env node
import { createHash } from "node:crypto";
import { createWriteStream, existsSync, realpathSync } from "node:fs";
import { chmod, copyFile, lstat, mkdir, readFile, rm, writeFile } from "node:fs/promises";
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

async function verifiedArchive(archive, url, expected) {
  if (existsSync(archive)) {
    try {
      await verifyArchiveChecksum(archive, expected);
      return archive;
    } catch {
      await rm(archive, { force: true });
    }
  }
  await download(url, archive);
  await verifyArchiveChecksum(archive, expected);
  return archive;
}

async function verifyChecksumsSigstore(baseURL, checksumsText) {
  if (process.env.NIGHTWARD_NPM_REQUIRE_SIGSTORE !== "1") {
    return;
  }
  requireCosignAvailable();
  const dir = path.join(tmpdir(), `nightward-sigstore-${process.pid}-${Date.now()}`);
  const checksumsPath = path.join(dir, "checksums.txt");
  const bundlePath = path.join(dir, "checksums.txt.sigstore.json");
  try {
    await mkdir(dir, { recursive: true });
    await writeFile(checksumsPath, checksumsText, { mode: 0o600 });
    await download(`${baseURL}/checksums.txt.sigstore.json`, bundlePath);
    const result = spawnSync("cosign", [
      "verify-blob",
      "--bundle",
      bundlePath,
      "--certificate-identity-regexp",
      "https://github.com/JSONbored/nightward/.github/workflows/release.yml@refs/tags/v.*",
      "--certificate-oidc-issuer",
      "https://token.actions.githubusercontent.com",
      checksumsPath
    ], { encoding: "utf8" });
    if (result.error) {
      throw result.error;
    }
    if (result.status !== 0) {
      throw new Error(`cosign verification failed: ${(result.stderr || result.stdout).trim()}`);
    }
  } finally {
    await rm(dir, { recursive: true, force: true });
  }
}

function requireCosignAvailable() {
  const cosign = spawnSync("cosign", ["version"], { stdio: "ignore" });
  if (cosign.error || cosign.status !== 0) {
    throw new Error("NIGHTWARD_NPM_REQUIRE_SIGSTORE=1 requires cosign on PATH");
  }
}

async function assertRegularBinary(binary) {
  const info = await lstat(binary);
  if (!info.isFile() || info.isSymbolicLink()) {
    throw new Error(`cached binary is not a regular file: ${binary}`);
  }
}

export function expectedArchiveEntries(target = targetFor()) {
  return target.os === "windows"
    ? ["nightward.exe", "nw.exe"]
    : ["nightward", "nw"];
}

export function validateArchiveEntryName(entry) {
  const trimmed = entry.trim();
  if (!trimmed || trimmed.endsWith("/")) {
    throw new Error(`release archive contains unexpected directory entry: ${entry}`);
  }
  if (trimmed.includes("\\") || path.isAbsolute(trimmed)) {
    throw new Error(`release archive contains unsafe entry: ${entry}`);
  }
  const normalized = trimmed.replace(/^\.\/+/, "");
  if (
    !normalized ||
    normalized.includes("/") ||
    normalized.split("/").some((part) => part === "." || part === "..")
  ) {
    throw new Error(`release archive contains unsafe entry: ${entry}`);
  }
  return normalized;
}

export function validateArchiveEntries(archive, target = targetFor()) {
  const entries = listArchiveEntries(archive, target);
  const normalized = entries.map(validateArchiveEntryName);
  const expected = expectedArchiveEntries(target);
  const unexpected = normalized.filter((entry) => !expected.includes(entry));
  const missing = expected.filter((entry) => !normalized.includes(entry));
  if (unexpected.length > 0) {
    throw new Error(`release archive contains unexpected entries: ${unexpected.join(", ")}`);
  }
  if (missing.length > 0) {
    throw new Error(`release archive is missing expected entries: ${missing.join(", ")}`);
  }
  if (new Set(normalized).size !== normalized.length) {
    throw new Error("release archive contains duplicate binary entries");
  }
  rejectTarSymlinks(archive, target);
  return normalized;
}

function listArchiveEntries(archive, target) {
  const result = target.os === "windows"
    ? listZipEntries(archive)
    : spawnSync("tar", ["-tzf", archive], { encoding: "utf8" });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error(`failed to list ${path.basename(archive)}: ${(result.stderr || "").trim()}`);
  }
  return result.stdout.split(/\r?\n/).map((line) => line.trim()).filter(Boolean);
}

function listZipEntries(archive) {
  const powershell = spawnSync("powershell", [
    "-NoProfile",
    "-ExecutionPolicy",
    "Bypass",
    "-Command",
    "Add-Type -AssemblyName System.IO.Compression.FileSystem; [IO.Compression.ZipFile]::OpenRead($args[0]).Entries | ForEach-Object { $_.FullName }",
    archive
  ], { encoding: "utf8" });
  if (!powershell.error && powershell.status === 0) {
    return powershell;
  }
  return spawnSync("unzip", ["-Z1", archive], { encoding: "utf8" });
}

function rejectTarSymlinks(archive, target) {
  if (target.os === "windows") {
    return;
  }
  const result = spawnSync("tar", ["-tvzf", archive], { encoding: "utf8" });
  if (result.error || result.status !== 0) {
    return;
  }
  const symlinks = result.stdout
    .split(/\r?\n/)
    .filter((line) => line.startsWith("l"))
    .map((line) => line.trim());
  if (symlinks.length > 0) {
    throw new Error("release archive contains symlink entries");
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
  if (url.startsWith("file://")) {
    await mkdir(path.dirname(destination), { recursive: true });
    await copyFile(fileURLToPath(url), destination);
    return;
  }
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
  validateArchiveEntries(archive, target);
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
  for (const entry of expectedArchiveEntries(target)) {
    await assertRegularBinary(path.join(destination, entry));
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
  const asset = assetName(version, target);
  const baseURL = releaseBaseURL(version);
  const archive = path.join(cacheRoot(), version, `${target.os}-${target.arch}`, asset);
  const installDir = path.dirname(binary);
  if (process.env.NIGHTWARD_NPM_REQUIRE_SIGSTORE === "1") {
    requireCosignAvailable();
  }
  const checksumsText = await downloadText(`${baseURL}/checksums.txt`);
  await verifyChecksumsSigstore(baseURL, checksumsText);
  const checksums = parseChecksums(checksumsText);
  const expected = checksums.get(asset);
  if (!expected) {
    throw new Error(`checksums.txt does not include ${asset}`);
  }

  await verifiedArchive(archive, `${baseURL}/${asset}`, expected);
  await rm(binary, { force: true });
  await extractArchive(archive, installDir, target);
  await assertRegularBinary(binary);
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
