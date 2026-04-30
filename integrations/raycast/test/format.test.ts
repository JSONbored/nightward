import assert from "node:assert/strict";
import test from "node:test";
import {
  analysisMarkdown,
  findingMarkdown,
  maxSeverity,
  redactText,
  signalMarkdown,
  sortedFindings,
  sortedSignals,
} from "../src/format";
import type { AnalysisReport, AnalysisSignal, Finding } from "../src/types";

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

test("analysis markdown redacts signal evidence", () => {
  const keyName = "API_" + "KEY";
  const token = "sk-" + "1234567890abcdef";
  const signal: AnalysisSignal = {
    id: "signal-a",
    provider: "nightward",
    rule: "nightward/mcp_secret_env",
    category: "secrets-exposure",
    subject_id: "finding-a",
    subject_type: "finding",
    severity: "critical",
    confidence: "high",
    message: `${keyName}=${token}`,
    evidence: "env_key=API_KEY",
    recommended_action: `Remove ${keyName}=${token}.`,
  };
  const report: AnalysisReport = {
    generated_at: "2026-04-30T00:00:00Z",
    mode: "home",
    summary: {
      total_subjects: 1,
      total_signals: 1,
      signals_by_severity: { critical: 1 },
      signals_by_category: { "secrets-exposure": 1 },
      signals_by_provider: { nightward: 1 },
      highest_severity: "critical",
      provider_warnings: 0,
      no_known_risk_signals: false,
    },
    providers: [],
    subjects: [],
    signals: [signal],
  };

  assert.equal(sortedSignals([baseSignal("b", "low"), signal])[0]?.id, "signal-a");
  assert.doesNotMatch(signalMarkdown(signal), /1234567890abcdef/);
  assert.doesNotMatch(analysisMarkdown(report), /1234567890abcdef/);
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

function baseSignal(id: string, severity: AnalysisSignal["severity"]): AnalysisSignal {
  return {
    id,
    provider: "nightward",
    rule: "nightward/review",
    category: "execution-risk",
    subject_id: id,
    subject_type: "finding",
    severity,
    confidence: "medium",
    message: "Review signal",
    recommended_action: "Review manually.",
  };
}
