#!/usr/bin/env node
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname } from "node:path";

const targets = [
  { os: "darwin", arch: "arm64", platform: "macos", cpu: "arm" },
  { os: "darwin", arch: "amd64", platform: "macos", cpu: "intel" },
  { os: "linux", arch: "arm64", platform: "linux", cpu: "arm" },
  { os: "linux", arch: "amd64", platform: "linux", cpu: "intel" },
];

const options = parseArgs(process.argv.slice(2));
const repo = options.repo || process.env.GITHUB_REPOSITORY || "JSONbored/nightward";
const version = normalizeVersion(options.version || process.env.VERSION || "");
const checksumsPath = options.checksums || "dist/checksums.txt";
const urlBase =
  options.urlBase || `https://github.com/${repo}/releases/download/v${version}`;
const output = options.output || "dist/homebrew/nightward.rb";
const checksums = parseChecksums(readFileSync(checksumsPath, "utf8"));

const formula = renderFormula({ repo, version, urlBase, checksums });
if (output === "-") {
  process.stdout.write(formula);
} else {
  mkdirSync(dirname(output), { recursive: true });
  writeFileSync(output, formula);
  console.log(output);
}

function parseArgs(args) {
  const out = {};
  for (let index = 0; index < args.length; index += 1) {
    const arg = args[index];
    if (!arg.startsWith("--")) {
      throw new Error(`unexpected argument: ${arg}`);
    }
    const key = arg.slice(2);
    const value = args[index + 1];
    if (!value || value.startsWith("--")) {
      throw new Error(`missing value for ${arg}`);
    }
    out[key] = value;
    index += 1;
  }
  return out;
}

function normalizeVersion(value) {
  const version = value.startsWith("v") ? value.slice(1) : value;
  if (!/^[0-9]+\.[0-9]+\.[0-9]+$/.test(version)) {
    throw new Error("version must be strict SemVer, for example 0.1.6");
  }
  return version;
}

function parseChecksums(text) {
  const checksums = new Map();
  for (const line of text.split(/\r?\n/)) {
    const match = line.match(/^([a-f0-9]{64})\s+\*?(.+)$/i);
    if (match) checksums.set(match[2].trim(), match[1].toLowerCase());
  }
  return checksums;
}

function renderFormula({ repo, version, urlBase, checksums }) {
  const platformBlocks = ["macos", "linux"].map((platform) => {
    const cpuBlocks = targets
      .filter((target) => target.platform === platform)
      .map((target) => {
        const asset = `nightward_${version}_${target.os}_${target.arch}.tar.gz`;
        const sha256 = checksums.get(asset);
        if (!sha256) {
          throw new Error(`missing checksum for ${asset}`);
        }
        return [
          `    on_${target.cpu} do`,
          `      url "${urlBase}/${asset}"`,
          `      sha256 "${sha256}"`,
          "    end",
        ].join("\n");
      })
      .join("\n");
    return [`  on_${platform} do`, cpuBlocks, "  end"].join("\n");
  });

  return `${[
    "class Nightward < Formula",
    '  desc "Local-first AI agent, MCP, and dotfiles risk scanner"',
    `  homepage "https://github.com/${repo}"`,
    `  version "${version}"`,
    '  license "MIT"',
    "",
    ...platformBlocks,
    "",
    "  def install",
    '    bin.install "nightward", "nw"',
    "  end",
    "",
    "  test do",
    '    assert_match version.to_s, shell_output("#{bin}/nightward --version")',
    '    assert_match version.to_s, shell_output("#{bin}/nw --version")',
    "  end",
    "end",
  ].join("\n")}\n`;
}
