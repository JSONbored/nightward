import assert from "node:assert/strict";
import test from "node:test";
import {
  findingMarkdown,
  maxSeverity,
  redactText,
  sortedFindings,
} from "../src/format";
import type { Finding } from "../src/types";

test("redacts obvious secret assignments and long token-like values", () => {
  const keyName = "API_" + "KEY";
  const token = "sk-" + "1234567890abcdef";
  const longValue = "abcdefghijklmno" + "pqrstuvwxyz123456";
  const path =
    "/Users/example/Library/Application Support/Claude/claude_desktop_config.json";
  const output = redactText(
    `${keyName}=${token} secret: ${longValue} path: ${path}`,
  );
  assert.match(output, /API_KEY=\[redacted\]/);
  assert.match(output, /secret: \[redacted\]/);
  assert.match(
    output,
    /Library\/Application Support\/Claude\/claude_desktop_config\.json/,
  );
  assert.doesNotMatch(output, /1234567890abcdef/);
  assert.doesNotMatch(output, /abcdefghijklmnopqrstuvwxyz123456/);
});

test("finding markdown keeps guidance while avoiding secret values", () => {
  const keyName = "API_" + "KEY";
  const token = "sk-" + "1234567890abcdef";
  const finding: Finding = {
    id: "mcp_secret_env-123",
    tool: "Codex",
    path: "/tmp/config.toml",
    severity: "critical",
    rule: "mcp_secret_env",
    message: `Inline credential ${keyName}=${token}`,
    evidence: "env_key=API_KEY",
    recommended_action: "Move API_KEY into an environment variable.",
    impact: "Credential material can leak.",
    why_this_matters: "Agents bridge local files and remote models.",
    fix_available: true,
    fix_kind: "externalize-secret",
    confidence: "high",
    risk: "high",
    requires_review: true,
    fix_summary: "Move API_KEY out of this config.",
    fix_steps: [`Remove ${keyName}=${token} from the MCP config.`],
  };

  const markdown = findingMarkdown(finding);
  assert.match(markdown, /API_KEY/);
  assert.match(markdown, /\[redacted\]/);
  assert.doesNotMatch(markdown, /1234567890abcdef/);
});

test("findings sort by severity then stable identity", () => {
  const findings: Finding[] = [
    baseFinding("b", "low", "Cursor"),
    baseFinding("a", "critical", "Codex"),
    baseFinding("c", "high", "Claude"),
  ];

  assert.equal(maxSeverity(findings), "critical");
  assert.deepEqual(
    sortedFindings(findings).map((finding) => finding.id),
    ["a", "c", "b"],
  );
});

function baseFinding(
  id: string,
  severity: Finding["severity"],
  tool: string,
): Finding {
  return {
    id,
    tool,
    path: `/tmp/${id}`,
    severity,
    rule: "mcp_review",
    message: "Review finding",
    recommended_action: "Review manually.",
    fix_available: false,
    requires_review: true,
  };
}
