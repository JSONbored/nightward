import type {
  AdapterStatus,
  AnalysisReport,
  AnalysisSignal,
  Classification,
  DoctorReport,
  Finding,
  FixPlan,
  RiskLevel,
  ScanReport,
} from "./types";

const secretAssignmentPattern =
  /((?:token|secret|password|passwd|api[_-]?key|auth|credential|private[_-]?key)[\w.-]*\s*[:=]\s*)(["']?)(?:\$\{[A-Za-z_][A-Za-z0-9_]*\}|[^"',\s}]+)/gi;
const providerTokenPattern =
  /\b(?:sk-[A-Za-z0-9_-]{12,}|gh[pousr]_[A-Za-z0-9_]{20,}|glpat-[A-Za-z0-9_-]{20,}|npm_[A-Za-z0-9]{20,}|xox[abprs]-[A-Za-z0-9-]{20,}|eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,})\b/g;

const riskRank: Record<RiskLevel, number> = {
  info: 0,
  low: 1,
  medium: 2,
  high: 3,
  critical: 4,
};

export function redactText(value: string | undefined): string {
  if (!value) return "";
  const redacted = value
    .replace(secretAssignmentPattern, "$1$2[redacted]")
    .replace(providerTokenPattern, "[redacted]");
  return redacted
    .split(/(\s+)/)
    .map((part) => (looksOpaqueProviderToken(part) ? "[redacted]" : part))
    .join("");
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
  return humanizeIdentifier(finding.rule);
}

export function findingFixLabel(finding: Finding): string {
  if (!finding.fix_available) return "review";
  if (!finding.fix_kind) return "fix";
  switch (finding.fix_kind) {
    case "pin-package":
      return "pin";
    case "externalize-secret":
      return "secret";
    case "replace-shell-wrapper":
      return "shell";
    case "narrow-filesystem":
      return "scope";
    case "ignore-with-reason":
      return "ignore";
    case "manual-review":
      return "review";
  }
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
    `# ${escapeMarkdown(findingTitle(finding))}`,
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
    `# ${escapeMarkdown(humanizeIdentifier(signal.rule))}`,
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
    signal.why_this_matters ? "" : "",
    signal.why_this_matters ? "## Why This Matters" : "",
    signal.why_this_matters ? redactText(signal.why_this_matters) : "",
  ]
    .filter((line) => line !== "")
    .join("\n");
}

export function signalTitle(signal: AnalysisSignal): string {
  return humanizeIdentifier(signal.rule);
}

export function signalSubtitle(signal: AnalysisSignal): string {
  return signal.provider;
}

export function analysisMarkdown(report: AnalysisReport): string {
  const lines = [
    "# Nightward Analysis",
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
      history?: Array<{
        findings: number;
        report_name: string;
        mod_time: string;
      }>;
    };
  },
  plan?: FixPlan,
  analysis?: AnalysisReport,
): string {
  const max = maxSeverity(report.findings);
  const critical = report.summary.findings_by_severity.critical ?? 0;
  const high = report.summary.findings_by_severity.high ?? 0;
  const medium = report.summary.findings_by_severity.medium ?? 0;
  const planTotal = plan ? fixPlanTotal(plan) : undefined;
  const signals = analysis?.summary.total_signals;
  const delta = reportHistoryDelta(doctor.schedule?.history);
  const lines = [
    "# Nightward Review",
    "",
    `**${humanizeIdentifier(max)} posture** across \`${report.summary.total_findings}\` findings from \`${report.summary.total_items}\` scanned item${report.summary.total_items === 1 ? "" : "s"}.`,
    "",
    "| Critical | High | Medium | Signals | Fix Plan |",
    "| ---: | ---: | ---: | ---: | ---: |",
    `| ${critical} | ${high} | ${medium} | ${signals ?? "n/a"} | ${planTotal ?? "n/a"} |`,
    "",
    "## Next Action",
    nextActionForReport(report),
    "",
    "## Review Queue",
  ];
  const topFindings = sortedFindings(report.findings).slice(0, 5);
  if (topFindings.length === 0) {
    lines.push("No findings were emitted for this scan.");
  } else {
    for (const finding of topFindings) {
      lines.push(
        `- \`${finding.severity}\` ${finding.rule}: ${redactText(finding.message)}`,
      );
    }
  }
  lines.push(
    "",
    "## Safety",
    "Raycast actions stay read-only: view, copy, export, and open local reports. Nightward does not mutate MCP config, dotfiles, schedules, or secrets from this dashboard.",
  );
  if (doctor.schedule?.installed || doctor.schedule?.last_report || delta) {
    lines.push("", "## Scheduled Reports");
    lines.push(
      `Status: \`${doctor.schedule?.installed ? "installed" : "off"}\``,
    );
    if (doctor.schedule?.last_report)
      lines.push(`Latest report: \`${doctor.schedule.last_report}\``);
    if (doctor.schedule?.last_findings !== undefined)
      lines.push(`Latest findings: \`${doctor.schedule.last_findings}\``);
    if (delta) lines.push(`Change: \`${delta}\``);
  }
  return lines.join("\n");
}

function nextActionForReport(report: ScanReport): string {
  const critical = report.summary.findings_by_severity.critical ?? 0;
  const high = report.summary.findings_by_severity.high ?? 0;
  const medium = report.summary.findings_by_severity.medium ?? 0;
  if (critical > 0) {
    return "Externalize inline secrets first, then rerun the scan before reviewing lower-severity items.";
  }
  if (high > 0) {
    return "Pin package executors and review remote MCP wrappers before syncing this configuration.";
  }
  if (medium > 0) {
    return "Review filesystem scope and local endpoint assumptions before treating this config as portable.";
  }
  if (report.summary.total_findings > 0) {
    return "Review accepted informational findings and add policy ignores only with clear reasons.";
  }
  return "No findings in this scan. Keep scheduled reports on if this machine changes frequently.";
}

export function adapterSummary(adapters: AdapterStatus[]): string {
  const found = adapters.filter((adapter) => adapter.available).length;
  return `${found}/${adapters.length} adapters found`;
}

export function fixPlanSummary(plan: FixPlan): string {
  return `Total ${fixPlanTotal(plan)} - Safe ${plan.summary.safe} - Review ${plan.summary.review} - Blocked ${plan.summary.blocked}`;
}

export function fixPlanTotal(plan: FixPlan): number {
  if (typeof plan.summary.total === "number") return plan.summary.total;
  const summaryTotal =
    (plan.summary.safe ?? 0) +
    (plan.summary.review ?? 0) +
    (plan.summary.blocked ?? 0);
  if (summaryTotal > 0) return summaryTotal;
  return plan.actions?.length ?? plan.fixes?.length ?? 0;
}

export type MenuBarStatus = {
  title: string;
  tooltip: string;
  risk: RiskLevel;
  findings: number;
  critical: number;
  high: number;
  medium: number;
  signals: number;
  providerWarnings: number;
  scheduled: boolean;
  lastFindings?: number;
  lastReport?: string;
  historyDelta?: string;
};

export function menuBarStatus(
  report: ScanReport,
  doctor: DoctorReport,
  analysis: AnalysisReport,
): MenuBarStatus {
  const risk = maxSeverity(report.findings);
  const critical = report.summary.findings_by_severity.critical ?? 0;
  const high = report.summary.findings_by_severity.high ?? 0;
  const medium = report.summary.findings_by_severity.medium ?? 0;
  const low = report.summary.findings_by_severity.low ?? 0;
  const info = report.summary.findings_by_severity.info ?? 0;
  const findings = report.summary.total_findings;
  const signals = analysis.summary.total_signals;
  const providerWarnings = analysis.summary.provider_warnings;
  const historyDelta = reportHistoryDelta(doctor.schedule.history);
  const issueCount = findings + providerWarnings;
  const title =
    issueCount === 0
      ? "OK"
      : critical > 0
        ? `${critical}C`
        : high > 0
          ? `${high}H`
          : medium > 0
            ? `${medium}M`
            : String(issueCount);
  const tooltip = [
    `Nightward: ${critical} critical, ${high} high, ${findings} total`,
    low > 0 || info > 0 ? `${medium} medium, ${low} low, ${info} info` : "",
    `${signals} signals, ${providerWarnings} provider warnings`,
    historyDelta ? `scheduled delta: ${historyDelta}` : "",
    doctor.schedule.installed ? "scheduled scan on" : "scheduled scan off",
  ]
    .filter(Boolean)
    .join(" • ");

  return {
    title,
    tooltip,
    risk,
    findings,
    critical,
    high,
    medium,
    signals,
    providerWarnings,
    scheduled: doctor.schedule.installed,
    lastFindings: doctor.schedule.last_findings,
    lastReport: doctor.schedule.last_report,
    historyDelta,
  };
}

export function policyIgnoreSnippet(
  finding: Finding,
  reason = "reviewed locally",
): string {
  return [
    "ignore_findings:",
    `  - id: ${JSON.stringify(finding.id)}`,
    `    reason: ${JSON.stringify(reason)}`,
  ].join("\n");
}

export function menuBarStatusMarkdown(status: MenuBarStatus): string {
  return [
    "# Nightward Status",
    "",
    `Findings: \`${status.findings}\``,
    `Critical: \`${status.critical}\``,
    `High: \`${status.high}\``,
    `Medium: \`${status.medium}\``,
    `Analysis signals: \`${status.signals}\``,
    `Provider warnings: \`${status.providerWarnings}\``,
    `Scheduled: \`${status.scheduled ? "yes" : "no"}\``,
    status.lastFindings !== undefined
      ? `Last scheduled findings: \`${status.lastFindings}\``
      : "",
    status.historyDelta
      ? `Change since previous scheduled scan: \`${status.historyDelta}\``
      : "",
    status.lastReport ? `Last report: \`${status.lastReport}\`` : "",
  ]
    .filter(Boolean)
    .join("\n");
}

export function reportHistoryDelta(
  history?: Array<{ findings: number }>,
): string | undefined {
  if (!history || history.length < 2) return undefined;
  const delta = history[0].findings - history[1].findings;
  if (delta === 0) return "no change";
  return delta > 0 ? `+${delta} findings` : `${delta} findings`;
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

function humanizeIdentifier(value: string): string {
  const raw = value.split("/").pop() ?? value;
  return raw
    .replace(/[_-]+/g, " ")
    .split(" ")
    .filter(Boolean)
    .map((word) => {
      const lower = word.toLowerCase();
      if (uppercaseTokens.has(lower)) return lower.toUpperCase();
      return `${lower.charAt(0).toUpperCase()}${lower.slice(1)}`;
    })
    .join(" ");
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

function looksOpaqueProviderToken(value: string): boolean {
  const trimmed = value.replace(/^["'`,]+|["'`,.]+$/g, "");
  if (
    trimmed.length < 36 ||
    trimmed.includes("/") ||
    trimmed.includes("\\") ||
    trimmed.includes("@") ||
    trimmed.includes(".") ||
    !/^[A-Za-z0-9_-]+$/.test(trimmed)
  ) {
    return false;
  }
  return /\d/.test(trimmed) && /[A-Za-z]/.test(trimmed);
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

const uppercaseTokens = new Set([
  "ai",
  "api",
  "ci",
  "id",
  "json",
  "mcp",
  "sarif",
  "toml",
  "url",
  "yaml",
]);
