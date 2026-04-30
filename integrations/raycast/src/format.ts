import type {
  AdapterStatus,
  AnalysisReport,
  AnalysisSignal,
  Classification,
  Finding,
  FixPlan,
  RiskLevel,
  ScanReport,
} from "./types";

const secretAssignmentPattern =
  /((?:token|secret|password|passwd|api[_-]?key|auth|credential|private[_-]?key)[\w.-]*\s*[:=]\s*)(["']?)[^"',\s}]+/gi;
const longSecretLikePattern = /\bsk-[A-Za-z0-9_-]{12,}\b/g;

const riskRank: Record<RiskLevel, number> = {
  info: 0,
  low: 1,
  medium: 2,
  high: 3,
  critical: 4,
};

export function redactText(value: string | undefined): string {
  if (!value) return "";
  return value
    .replace(secretAssignmentPattern, "$1$2[redacted]")
    .replace(longSecretLikePattern, "[redacted]");
}

export function severityColor(severity: RiskLevel): string {
  switch (severity) {
    case "critical":
      return "#ff4d4f";
    case "high":
      return "#ff8c42";
    case "medium":
      return "#f5c542";
    case "low":
      return "#7dd3fc";
    default:
      return "#a1a1aa";
  }
}

export function classificationColor(classification: Classification): string {
  switch (classification) {
    case "portable":
      return "#8bd450";
    case "machine-local":
      return "#f5c542";
    case "secret-auth":
      return "#ff4d4f";
    case "runtime-cache":
      return "#a1a1aa";
    case "app-owned":
      return "#94a3b8";
    default:
      return "#c4b5fd";
  }
}

export function maxSeverity(findings: Finding[]): RiskLevel {
  return findings.reduce<RiskLevel>((max, finding) => {
    return riskRank[finding.severity] > riskRank[max] ? finding.severity : max;
  }, "info");
}

export function sortedFindings(findings: Finding[]): Finding[] {
  return [...findings].sort((a, b) => {
    const riskDelta = riskRank[b.severity] - riskRank[a.severity];
    if (riskDelta !== 0) return riskDelta;
    if (a.tool !== b.tool) return a.tool.localeCompare(b.tool);
    if (a.rule !== b.rule) return a.rule.localeCompare(b.rule);
    return a.id.localeCompare(b.id);
  });
}

export function findingTitle(finding: Finding): string {
  return `${finding.severity.toUpperCase()} ${finding.rule}`;
}

export function findingSubtitle(finding: Finding): string {
  const parts = [finding.tool];
  if (finding.server) parts.push(finding.server);
  parts.push(basename(finding.path));
  return parts.join(" - ");
}

export function findingKeywords(finding: Finding): string[] {
  return [
    finding.id,
    finding.tool,
    finding.path,
    finding.server ?? "",
    finding.rule,
    finding.severity,
    finding.fix_kind ?? "",
    finding.confidence ?? "",
  ].filter(Boolean);
}

export function findingMarkdown(finding: Finding): string {
  const lines = [
    `# ${escapeMarkdown(finding.rule)}`,
    "",
    redactText(finding.message),
    "",
    "## Evidence",
    finding.evidence
      ? markdownCode(redactText(finding.evidence))
      : "No redacted evidence was emitted for this finding.",
    "",
    "## Impact",
    redactText(finding.impact) ||
      "Nightward did not emit a specific impact statement for this finding.",
    "",
    "## Recommended Action",
    redactText(finding.recommended_action),
  ];

  if (finding.fix_available) {
    lines.push("", "## Suggested Fix");
    lines.push(`Kind: \`${finding.fix_kind ?? "manual-review"}\``);
    lines.push(`Confidence: \`${finding.confidence ?? "unknown"}\``);
    lines.push(`Risk: \`${finding.risk ?? "unknown"}\``);
    lines.push(`Requires review: \`${String(finding.requires_review)}\``);
    if (finding.fix_summary) lines.push("", redactText(finding.fix_summary));
    if (finding.fix_steps && finding.fix_steps.length > 0) {
      lines.push("");
      for (const [index, step] of finding.fix_steps.entries()) {
        lines.push(`${index + 1}. ${redactText(step)}`);
      }
    }
  }

  if (finding.why_this_matters) {
    lines.push("", "## Why This Matters", redactText(finding.why_this_matters));
  }

  return lines.join("\n");
}

export function sortedSignals(signals: AnalysisSignal[]): AnalysisSignal[] {
  return [...signals].sort((a, b) => {
    const riskDelta = riskRank[b.severity] - riskRank[a.severity];
    if (riskDelta !== 0) return riskDelta;
    if (a.provider !== b.provider) return a.provider.localeCompare(b.provider);
    if (a.rule !== b.rule) return a.rule.localeCompare(b.rule);
    return a.id.localeCompare(b.id);
  });
}

export function signalMarkdown(signal: AnalysisSignal): string {
  return [
    `# ${escapeMarkdown(signal.rule)}`,
    "",
    redactText(signal.message),
    "",
    "## Evidence",
    signal.evidence
      ? markdownCode(redactText(signal.evidence))
      : "No redacted evidence was emitted for this signal.",
    "",
    "## Recommended Action",
    redactText(signal.recommended_action),
    "",
    "## Metadata",
    `Provider: \`${signal.provider}\``,
    `Category: \`${signal.category}\``,
    `Severity: \`${signal.severity}\``,
    `Confidence: \`${signal.confidence}\``,
    signal.path ? `Path: ${markdownCode(signal.path)}` : "",
    signal.why_this_matters ? "" : "",
    signal.why_this_matters ? "## Why This Matters" : "",
    signal.why_this_matters ? redactText(signal.why_this_matters) : "",
  ]
    .filter((line) => line !== "")
    .join("\n");
}

export function analysisMarkdown(report: AnalysisReport): string {
  const lines = [
    "# Nightward Analysis",
    "",
    `Generated: \`${report.generated_at}\``,
    `Mode: \`${report.mode}\``,
    report.workspace ? `Workspace: \`${report.workspace}\`` : "",
    "",
    "## Summary",
    `Signals: \`${report.summary.total_signals}\``,
    `Subjects: \`${report.summary.total_subjects}\``,
    `Highest severity: \`${report.summary.highest_severity || "info"}\``,
    `Provider warnings: \`${report.summary.provider_warnings}\``,
    "",
    "Nightward analysis is offline by default and does not claim a package or server is safe.",
  ].filter(Boolean);
  if (report.signals.length > 0) {
    lines.push("", "## Top Signals");
    for (const signal of sortedSignals(report.signals).slice(0, 8)) {
      lines.push(
        `- \`${signal.severity}\` ${signal.rule}: ${redactText(signal.message)}`,
      );
    }
  }
  return lines.join("\n");
}

export function dashboardMarkdown(
  report: ScanReport,
  doctor: {
    schedule?: {
      installed?: boolean;
      last_report?: string;
      last_findings?: number;
      report_dir?: string;
    };
  },
): string {
  const max = maxSeverity(report.findings);
  const lines = [
    "# Nightward Dashboard",
    "",
    `Generated: \`${report.generated_at}\``,
    `Host: \`${report.hostname}\``,
    `Home: \`${report.home}\``,
    "",
    "## Scan",
    `Items: \`${report.summary.total_items}\``,
    `Findings: \`${report.summary.total_findings}\``,
    `Max severity: \`${max}\``,
    `Critical findings: \`${report.summary.findings_by_severity.critical ?? 0}\``,
    `High findings: \`${report.summary.findings_by_severity.high ?? 0}\``,
    "",
    "## Schedule",
    `Installed: \`${doctor.schedule?.installed ? "yes" : "no"}\``,
    `Report dir: \`${doctor.schedule?.report_dir ?? ""}\``,
  ];
  if (doctor.schedule?.last_report)
    lines.push(`Last report: \`${doctor.schedule.last_report}\``);
  if (doctor.schedule?.last_findings !== undefined)
    lines.push(`Last findings: \`${doctor.schedule.last_findings}\``);
  return lines.join("\n");
}

export function adapterSummary(adapters: AdapterStatus[]): string {
  const found = adapters.filter((adapter) => adapter.available).length;
  return `${found}/${adapters.length} adapters found`;
}

export function fixPlanSummary(plan: FixPlan): string {
  return `Safe ${plan.summary.safe} - Review ${plan.summary.review} - Blocked ${plan.summary.blocked}`;
}

export function basename(path: string): string {
  const parts = path.split("/");
  return parts[parts.length - 1] || path;
}

export function truncate(value: string, maxLength: number): string {
  if (value.length <= maxLength) return value;
  if (maxLength <= 3) return value.slice(0, maxLength);
  return `${value.slice(0, maxLength - 3)}...`;
}

function markdownCode(value: string): string {
  const longestRun = maxBacktickRun(value);
  const delimiter = "`".repeat(longestRun + 1);
  const padded =
    value.startsWith("`") || value.endsWith("`") ? ` ${value} ` : value;
  return `${delimiter}${padded}${delimiter}`;
}

function escapeMarkdown(value: string): string {
  const escaped: string[] = [];
  for (const char of value) {
    if (markdownSpecialChars.has(char)) {
      escaped.push("\\");
    }
    escaped.push(char);
  }
  return escaped.join("");
}

function maxBacktickRun(value: string): number {
  let max = 0;
  let current = 0;
  for (const char of value) {
    if (char === "`") {
      current += 1;
      if (current > max) max = current;
    } else {
      current = 0;
    }
  }
  return max;
}

const markdownSpecialChars = new Set([
  "\\",
  "`",
  "*",
  "_",
  "{",
  "}",
  "[",
  "]",
  "(",
  ")",
  "#",
  "+",
  "-",
  ".",
  "!",
  "|",
  ">",
  "~",
]);
