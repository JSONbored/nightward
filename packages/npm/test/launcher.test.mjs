import { chmod, mkdir, mkdtemp, readFile, symlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath, pathToFileURL } from "node:url";
import assert from "node:assert/strict";
import { test } from "vitest";

import {
  assetName,
  cachedBinaryPath,
  commandName,
  ensureBinary,
  parseChecksums,
  releaseBaseURL,
  sha256,
  targetFor,
  validateArchiveEntries,
  validateArchiveEntryName,
  verifyArchiveChecksum
} from "../bin/nightward.mjs";

test("maps supported platforms to Rust release archive names", () => {
  assert.equal(assetName("0.1.0", targetFor("darwin", "arm64")), "nightward_0.1.0_darwin_arm64.tar.gz");
  assert.equal(assetName("0.1.0", targetFor("darwin", "x64")), "nightward_0.1.0_darwin_amd64.tar.gz");
  assert.equal(assetName("0.1.0", targetFor("linux", "x64")), "nightward_0.1.0_linux_amd64.tar.gz");
  assert.equal(assetName("0.1.0", targetFor("linux", "arm64")), "nightward_0.1.0_linux_arm64.tar.gz");
  assert.equal(assetName("0.1.0", targetFor("win32", "x64")), "nightward_0.1.0_windows_amd64.zip");
});

test("rejects unsupported npm launcher platforms", () => {
  assert.throws(() => targetFor("freebsd", "x64"), /unsupported platform/);
  assert.throws(() => targetFor("linux", "ia32"), /unsupported platform/);
  assert.throws(() => targetFor("win32", "arm64"), /unsupported platform/);
});

test("parses checksums by archive basename", () => {
  const checksums = parseChecksums(`
9d5e3f5a13b86f661d9c61ef081bb9680186c02356b54e56058aca2c6f5393b6  nightward_0.1.0_linux_amd64.tar.gz
invalid ignored
  `);
  assert.equal(checksums.get("nightward_0.1.0_linux_amd64.tar.gz"), "9d5e3f5a13b86f661d9c61ef081bb9680186c02356b54e56058aca2c6f5393b6");
});

test("verifies archive sha256 before extraction", async () => {
  const dir = await mkdtemp(path.join(tmpdir(), "nightward-npm-"));
  const archive = path.join(dir, "archive.tar.gz");
  await writeFile(archive, "nightward");
  await verifyArchiveChecksum(archive, "3fc82c839197a667cf521e474cfdecb275ecc536fa0c66b2d9a3fbc98bc29a21");
  await assert.rejects(
    () => verifyArchiveChecksum(archive, "0".repeat(64)),
    /checksum mismatch/
  );
});

test("validates release archive entries before extraction", async () => {
  const dir = await mkdtemp(path.join(tmpdir(), "nightward-npm-archive-"));
  const packageDir = path.join(dir, "package");
  const target = targetFor(process.platform, process.arch);
  await mkdir(packageDir, { recursive: true });
  const names = target.os === "windows"
    ? ["nightward.exe", "nw.exe"]
    : ["nightward", "nw"];
  for (const name of names) {
    await writeFile(path.join(packageDir, name), "binary\n");
  }

  const archive = path.join(dir, target.os === "windows" ? "valid.zip" : "valid.tar.gz");
  const archiveResult = target.os === "windows"
    ? spawnSync("powershell", [
      "-NoProfile",
      "-ExecutionPolicy",
      "Bypass",
      "-Command",
      "Compress-Archive -LiteralPath $args[0],$args[1] -DestinationPath $args[2] -Force",
      path.join(packageDir, names[0]),
      path.join(packageDir, names[1]),
      archive
    ], { encoding: "utf8" })
    : spawnSync("tar", ["-czf", archive, "-C", packageDir, ...names], {
      encoding: "utf8"
    });
  assert.equal(archiveResult.status, 0, archiveResult.stderr);

  assert.deepEqual(validateArchiveEntries(archive, target).sort(), names.sort());
  assert.throws(() => validateArchiveEntryName("../nightward"), /unsafe entry/);

  if (target.os !== "windows") {
    const outside = path.join(dir, "outside");
    await mkdir(outside, { recursive: true });
    await writeFile(path.join(outside, "nightward"), "evil\n");
    const malicious = path.join(dir, "malicious.tar.gz");
    const maliciousResult = spawnSync("tar", [
      "-czf",
      malicious,
      "-C",
      packageDir,
      "nw",
      "-C",
      packageDir,
      "nightward",
      "-C",
      packageDir,
      "../outside/nightward"
    ], { encoding: "utf8" });
    assert.equal(maliciousResult.status, 0, maliciousResult.stderr);
    assert.throws(() => validateArchiveEntries(malicious, target), /unsafe entry|unexpected entries/);
  }
});

test("strict sigstore mode requires cosign before trusting checksums", async () => {
  const dir = await mkdtemp(path.join(tmpdir(), "nightward-npm-sigstore-"));
  const launcher = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../bin/nightward.mjs");
  const result = spawnSync(process.execPath, [launcher, "--version"], {
    env: {
      ...process.env,
      NIGHTWARD_NPM_REQUIRE_SIGSTORE: "1",
      NIGHTWARD_NPM_VERSION: "9.9.9",
      NIGHTWARD_NPM_CACHE: path.join(dir, "cache"),
      NIGHTWARD_NPM_DOWNLOAD_BASE: pathToFileURL(dir).href,
      PATH: dir
    },
    encoding: "utf8"
  });

  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /requires cosign on PATH|checksums/);
});

test("uses invocation name and environment overrides", () => {
  assert.equal(commandName("/usr/local/bin/nw"), "nw");
  assert.equal(commandName("/usr/local/bin/nightward"), "nightward");

  process.env.NIGHTWARD_NPM_CACHE = "/tmp/nightward-cache";
  process.env.NIGHTWARD_NPM_DOWNLOAD_BASE = "https://example.test/releases/";
  try {
    assert.equal(cachedBinaryPath("nw", "0.1.0", targetFor("linux", "x64")), "/tmp/nightward-cache/0.1.0/linux-amd64/nw");
    assert.equal(releaseBaseURL("0.1.0"), "https://example.test/releases");
  } finally {
    delete process.env.NIGHTWARD_NPM_CACHE;
    delete process.env.NIGHTWARD_NPM_DOWNLOAD_BASE;
  }
});

test("runs through npm bin symlink and waits for child output", async () => {
  const dir = await mkdtemp(path.join(tmpdir(), "nightward-npm-bin-"));
  const fakeBinary = path.join(dir, "fake-nightward.mjs");
  const launcherLink = path.join(dir, "nightward");
  const packageRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

  await writeFile(fakeBinary, `#!/usr/bin/env node
console.log("fake-nightward " + process.argv.slice(2).join(" "));
`, "utf8");
  await chmod(fakeBinary, 0o755);
  await symlink(path.join(packageRoot, "bin/nightward.mjs"), launcherLink);

  const result = spawnSync(process.execPath, [launcherLink, "--version"], {
    cwd: packageRoot,
    env: {
      ...process.env,
      NIGHTWARD_BIN: fakeBinary
    },
    encoding: "utf8"
  });

  assert.equal(result.status, 0, result.stderr);
  assert.equal(result.stdout.trim(), "fake-nightward --version");
});

test("cache hits are re-extracted from a verified archive before execution", async () => {
  const dir = await mkdtemp(path.join(tmpdir(), "nightward-npm-cache-"));
  const release = path.join(dir, "release");
  const packageDir = path.join(dir, "package");
  const cache = path.join(dir, "cache");
  const version = "9.9.9";
  const target = targetFor(process.platform, process.arch);
  const asset = assetName(version, target);
  await mkdir(release, { recursive: true });
  await mkdir(packageDir, { recursive: true });
  const targetNames = target.os === "windows"
    ? ["nightward.exe", "nw.exe"]
    : ["nightward", "nw"];
  for (const name of targetNames) {
    const binary = path.join(packageDir, name);
    await writeFile(binary, "#!/usr/bin/env node\nconsole.log('GOOD_CACHE_BINARY');\n");
    await chmod(binary, 0o755);
  }
  const archive = path.join(release, asset);
  const archiveResult = process.platform === "win32"
    ? spawnSync("powershell", [
      "-NoProfile",
      "-ExecutionPolicy",
      "Bypass",
      "-Command",
      "Compress-Archive -LiteralPath $args[0],$args[1] -DestinationPath $args[2] -Force",
      path.join(packageDir, targetNames[0]),
      path.join(packageDir, targetNames[1]),
      archive
    ], { encoding: "utf8" })
    : spawnSync("tar", ["-czf", archive, "-C", packageDir, ...targetNames], {
      encoding: "utf8"
    });
  assert.equal(archiveResult.status, 0, archiveResult.stderr);
  await writeFile(
    path.join(release, "checksums.txt"),
    `${await sha256(archive)}  ${asset}\n`
  );

  const previous = {
    cache: process.env.NIGHTWARD_NPM_CACHE,
    version: process.env.NIGHTWARD_NPM_VERSION,
    base: process.env.NIGHTWARD_NPM_DOWNLOAD_BASE
  };
  process.env.NIGHTWARD_NPM_CACHE = cache;
  process.env.NIGHTWARD_NPM_VERSION = version;
  process.env.NIGHTWARD_NPM_DOWNLOAD_BASE = pathToFileURL(release).href;
  try {
    const cached = await ensureBinary("nightward");
    await writeFile(cached, "#!/usr/bin/env node\nconsole.log('POISONED_CACHE_BINARY');\n");
    await chmod(cached, 0o755);

    const repaired = await ensureBinary("nightward");
    const contents = await readFile(repaired, "utf8");

    assert.equal(repaired, cached);
    assert.match(contents, /GOOD_CACHE_BINARY/);
    assert.doesNotMatch(contents, /POISONED_CACHE_BINARY/);
  } finally {
    restoreEnv("NIGHTWARD_NPM_CACHE", previous.cache);
    restoreEnv("NIGHTWARD_NPM_VERSION", previous.version);
    restoreEnv("NIGHTWARD_NPM_DOWNLOAD_BASE", previous.base);
  }
});

function restoreEnv(key, value) {
  if (value === undefined) {
    delete process.env[key];
  } else {
    process.env[key] = value;
  }
}
