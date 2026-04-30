import { mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import test from "node:test";
import assert from "node:assert/strict";

import {
  assetName,
  cachedBinaryPath,
  commandName,
  parseChecksums,
  releaseBaseURL,
  targetFor,
  verifyArchiveChecksum
} from "../bin/nightward.mjs";

test("maps supported platforms to GoReleaser asset names", () => {
  assert.equal(assetName("0.1.0", targetFor("darwin", "arm64")), "nightward_0.1.0_darwin_arm64.tar.gz");
  assert.equal(assetName("0.1.0", targetFor("linux", "x64")), "nightward_0.1.0_linux_amd64.tar.gz");
  assert.equal(assetName("0.1.0", targetFor("win32", "x64")), "nightward_0.1.0_windows_amd64.zip");
});

test("rejects unsupported npm launcher platforms", () => {
  assert.throws(() => targetFor("freebsd", "x64"), /unsupported platform/);
  assert.throws(() => targetFor("linux", "ia32"), /unsupported platform/);
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
