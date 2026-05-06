import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { test } from "vitest";

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const docsFixtureHome = join(repoRoot, "testdata/homes/docs-fixture");
const previousReport = join(
  docsFixtureHome,
  ".local/state/nightward/reports/previous.json",
);
const currentReport = join(
  docsFixtureHome,
  ".local/state/nightward/reports/current.json",
);

test("public docs do not contain stale release placeholders", () => {
  const stalePatterns = [
    { pattern: /After the first tagged release/i, reason: "release placeholder" },
    { pattern: /First signed `?v0\.1\.0`? release/i, reason: "release placeholder" },
    { pattern: /uses:\s*JSONbored\/nightward@v0\.1\.0/i, reason: "old action tag" },
    { pattern: /trunk .*v0\.1\.0/i, reason: "old trunk tag" },
    { pattern: /semantic_version:\s*0\.1\.0/i, reason: "old plugin version" },
    {
      pattern: /Static HTML report export before any self-hosted dashboard/i,
      reason: "shipped report surface still described as roadmap",
    },
    {
      pattern: /Broader provider execution beyond the first explicit local/i,
      reason: "shipped provider surface still described as roadmap",
    },
    {
      pattern: /Rules list\/explain commands and contributor fixture templates/i,
      reason: "shipped rules surface still described as roadmap",
    },
    {
      pattern: /Homebrew tap\s*\|\s*Shipped/i,
      reason: "tap publication must not be described as shipped until a tap exists",
    },
    {
      pattern: /brew install\s+JSONbored\/nightward\/nightward/i,
      reason: "public brew install command must wait for a published tap",
    },
    {
      pattern: /nw report latest --json/i,
      reason: "report latest prints the latest path, not a JSON object",
    },
    {
      pattern: /\b(?:added_findings|removed_findings|changed_findings)\b/i,
      reason: "report diff JSON uses added, removed, and changed arrays",
    },
  ];
  const files = gitTrackedDocs();
  const failures = [];

  for (const file of files) {
    const text = readFileSync(join(repoRoot, file), "utf8");
    for (const { pattern, reason } of stalePatterns) {
      if (pattern.test(text)) {
        failures.push(`${file}: ${reason}: ${pattern}`);
      }
    }
  }

  assert.deepEqual(failures, []);
});

test("public docs command snippets match fixture-backed CLI behavior", { timeout: 120000 }, () => {
  assertFencedSnippet(
    "site/reference/json-output.md",
    "```sh\nnw scan --json\n```",
  );
  assertFencedSnippet(
    "site/reference/json-output.md",
    "```sh\nnw findings list --json\nnw findings explain <finding-id> --json\n```",
  );
  assertFencedSnippet(
    "site/reference/json-output.md",
    "```sh\nnw report diff --from previous.json --to current.json --json\nnw report history --json\nnw report latest\n```",
  );
  assertFencedSnippet("README.md", "```sh\nnw scan --json\n```");

  const scan = runNightwardJSON(["scan", "--json"]);
  assert.equal(scan.summary.total_findings, 4);
  assert.equal(scan.summary.findings_by_rule.mcp_secret_env, 1);
  assert.equal(scan.summary.findings_by_rule.mcp_unpinned_package, 1);
  assert.doesNotMatch(JSON.stringify(scan), /docs-fixture-secret/);

  const findingId = scan.findings.find(
    (finding) => finding.rule === "mcp_secret_env",
  )?.id;
  assert.ok(findingId);
  const findings = runNightwardJSON(["findings", "list", "--json"]);
  assert.equal(findings.length, scan.summary.total_findings);
  const finding = runNightwardJSON([
    "findings",
    "explain",
    "--json",
    findingId,
  ]);
  assert.equal(finding.rule, "mcp_secret_env");
  assert.equal(finding.severity, "critical");

  const diff = runNightwardJSON([
    "report",
    "diff",
    "--from",
    previousReport,
    "--to",
    currentReport,
    "--json",
  ]);
  assert.deepEqual(diff.summary, {
    added: 1,
    removed: 1,
    changed: 1,
    max_added_severity: "critical",
  });
  assert.equal(diff.added[0]?.id, "fixture-added");
  assert.equal(diff.removed[0]?.id, "fixture-old");
  assert.equal(diff.changed[0]?.id, "fixture-review");

  const history = runNightwardJSON(["report", "history", "--json"]);
  assert.deepEqual(
    history.map((record) => record.report_name).sort(),
    ["current.json", "previous.json"],
  );
  assert.ok(history.every((record) => record.path.startsWith(docsFixtureHome)));

  const latest = runNightward(["report", "latest"]).trim();
  assert.match(
    latest,
    /testdata\/homes\/docs-fixture\/\.local\/state\/nightward\/reports\/.+\.json$/,
  );
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
    assert.equal(tools.length, 14);
    assert.equal(resources.length, 8);
    assert.equal(prompts.length, 5);
  } finally {
    rmSync(home, { recursive: true, force: true });
  }
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

function assertFencedSnippet(file, snippet) {
  const text = readFileSync(join(repoRoot, file), "utf8");
  assert.ok(text.includes(snippet), `${file} is missing stable snippet:\n${snippet}`);
}

function runNightwardJSON(args) {
  return JSON.parse(runNightward(args));
}

function runNightward(args) {
  assertReadOnlyArgs(args);
  return execFileSync("cargo", ["run", "--quiet", "--bin", "nw", "--", ...args], {
    cwd: repoRoot,
    encoding: "utf8",
    env: {
      ...process.env,
      PATH: `${process.env.HOME}/.cargo/bin:/opt/homebrew/bin:${process.env.PATH || ""}`,
      NIGHTWARD_HOME: docsFixtureHome,
    },
    stdio: ["ignore", "pipe", "pipe"],
  });
}

function assertReadOnlyArgs(args) {
  const command = ["nw", ...args].join(" ");
  const writefulPatterns = [
    /\bnpx\b/,
    /\bnpm\s+(?:install|publish|exec|run)\b/,
    /--output(?!\s+-)/,
    /\breport\s+html\b/,
    /\bpolicy\s+init\b/,
    /\bactions\s+apply\b/,
    /\bschedule\s+(?:install|remove)\b/,
    /\bbackup\s+(?:create|snapshot)\b/,
    /\bmcp\s+serve\b/,
  ];
  for (const pattern of writefulPatterns) {
    assert.doesNotMatch(command, pattern, `docs command contract is not read-only: ${command}`);
  }
}

function stableId(parts) {
  const hash = createHash("sha256");
  for (const part of parts) {
    hash.update(String(part));
    hash.update("\0");
  }
  return hash.digest("hex").slice(0, 12);
}
