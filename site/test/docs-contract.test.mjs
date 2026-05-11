import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { test } from "vitest";

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "../..");

test("public docs do not contain stale release placeholders", () => {
  const stalePatterns = [
    /After the first tagged release/i,
    /First signed `?v0\.1\.0`? release/i,
    /Trusted npm publishing/i,
    /uses:\s*JSONbored\/nightward@v0\.1\.0/i,
    /trunk .*v0\.1\.0/i,
    /v0\.1\.(?:[0-9]|10)\b/i,
    /semantic_version:\s*0\.1\.0/i,
    /MCP is read-only/i,
    /MCP cannot apply local writes/i,
    /read-only action list\/preview/i,
    /Static HTML report export before any self-hosted dashboard/i,
    /Broader provider execution beyond the first explicit local/i,
    /Rules list\/explain commands and contributor fixture templates/i,
  ];
  const files = gitTrackedDocs();
  const failures = [];

  for (const file of files) {
    const text = readFileSync(join(repoRoot, file), "utf8");
    for (const pattern of stalePatterns) {
      if (pattern.test(text)) {
        failures.push(`${file}: ${pattern}`);
      }
    }
  }

  assert.deepEqual(failures, []);
});

test("demo sample IDs match scrubbed fixture paths", () => {
  const scanPath = join(repoRoot, "site/public/demo/nightward-sample-scan.json");
  const report = JSON.parse(readFileSync(scanPath, "utf8"));
  const failures = [];

  for (const item of report.items || []) {
    const expected = stableId(["item", item.tool || "", item.path || ""]);
    if (item.id !== expected) {
      failures.push(`item id mismatch for ${item.path}: expected ${expected}, got ${item.id}`);
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
      failures.push(
        `finding id mismatch for ${finding.rule}: expected ${expected}, got ${finding.id}`,
      );
    }
  }

  assert.deepEqual(failures, []);
});

test("MCP docs list every runtime tool, resource, and prompt", { timeout: 60000 }, () => {
  const home = mkdtempSync(join(tmpdir(), "nightward-mcp-docs-home-"));
  try {
    const input = [
      request(1, "initialize", { protocolVersion: "2025-11-25" }),
      request(2, "tools/list"),
      request(3, "resources/list"),
      request(4, "prompts/list"),
    ].join("\n");
    const output = execFileSync(
      "cargo",
      ["run", "--quiet", "--bin", "nw", "--", "mcp", "serve"],
      {
        cwd: repoRoot,
        encoding: "utf8",
        input: `${input}\n`,
        env: {
          ...process.env,
          PATH: `${process.env.HOME}/.cargo/bin:/opt/homebrew/bin:${process.env.PATH || ""}`,
          NIGHTWARD_HOME: home,
        },
        stdio: ["pipe", "pipe", "pipe"],
      },
    );
    const responses = output
      .trim()
      .split("\n")
      .filter(Boolean)
      .map((line) => JSON.parse(line));
    const byId = new Map(responses.map((response) => [response.id, response]));
    const tools = byId.get(2)?.result?.tools?.map((tool) => tool.name) || [];
    const resources = byId.get(3)?.result?.resources?.map((resource) => resource.uri) || [];
    const prompts = byId.get(4)?.result?.prompts?.map((prompt) => prompt.name) || [];
    const docs = [
      readFileSync(join(repoRoot, "docs/mcp-server.md"), "utf8"),
      readFileSync(join(repoRoot, "site/integrations/mcp-server.md"), "utf8"),
    ].join("\n");

    const missing = [...tools, ...resources, ...prompts].filter((value) => !docs.includes(value));
    assert.deepEqual(missing, []);
    assert.equal(tools.length, 17);
    assert.equal(resources.length, 9);
    assert.equal(prompts.length, 5);
  } finally {
    rmSync(home, { recursive: true, force: true });
  }
});

test("generated CLI reference includes approval commands", () => {
  const cliReference = readFileSync(join(repoRoot, "site/reference/cli.md"), "utf8");
  assert.match(cliReference, /nightward approvals list --json/);
  assert.match(cliReference, /nightward approvals approve <approval-id>/);
  assert.match(cliReference, /nightward approvals apply <approval-id>/);
});

function gitTrackedDocs() {
  return execFileSync(
    "git",
    [
      "ls-files",
      "README.md",
      "docs/*.md",
      "docs/**/*.md",
      "site/*.md",
      "site/**/*.md",
      ".nightward*.yml",
    ],
    { cwd: repoRoot, encoding: "utf8" },
  )
    .trim()
    .split("\n")
    .filter(Boolean);
}

function request(id, method, params = {}) {
  return JSON.stringify({ jsonrpc: "2.0", id, method, params });
}

function stableId(parts) {
  const hash = createHash("sha256");
  for (const part of parts) {
    hash.update(String(part));
    hash.update("\0");
  }
  return hash.digest("hex").slice(0, 12);
}
