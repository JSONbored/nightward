export type Risk = "critical" | "high" | "medium" | "low" | "info" | string;

export interface NightwardFinding {
  id?: string;
  tool?: string;
  path?: string;
  server?: string;
  severity?: Risk;
  rule?: string;
  message?: string;
  evidence?: string;
  impact?: string;
  recommendation?: string;
  recommended_action?: string;
  fix_available?: boolean;
  fix_kind?: string;
  fix_summary?: string;
  fix_steps?: string[];
}

export interface NightwardSignal {
  id?: string;
  provider?: string;
  rule?: string;
  category?: string;
  severity?: Risk;
  confidence?: string;
  message?: string;
  evidence?: string;
  recommended_action?: string;
  why_this_matters?: string;
}

export interface NightwardAnalysis {
  summary?: {
    total_signals?: number;
    total_subjects?: number;
    provider_warnings?: number;
    highest_severity?: Risk;
    signals_by_severity?: Record<string, number>;
  };
  signals?: NightwardSignal[];
}

export interface NightwardFix {
  finding_id?: string;
  rule?: string;
  severity?: Risk;
  status?: string;
  summary?: string;
  steps?: string[];
  fix_kind?: string;
}

export interface NightwardFixPlan {
  summary?: {
    total?: number;
    safe?: number;
    review?: number;
    blocked?: number;
  };
  fixes?: NightwardFix[];
}

export interface NightwardBackupEntry {
  source?: string;
  target?: string;
  tool?: string;
  classification?: string;
  risk?: Risk;
  action?: string;
  reason?: string;
  recommended_action?: string;
}

export interface NightwardBackupPlan {
  target_root?: string;
  summary?: {
    included?: number;
    review?: number;
    excluded?: number;
  };
  entries?: NightwardBackupEntry[];
}

export interface NightwardItem {
  tool?: string;
  path?: string;
  classification?: string;
  risk?: Risk;
  reason?: string;
}

export interface NightwardReport {
  generated_at?: string;
  hostname?: string;
  home?: string;
  scan_mode?: string;
  workspace?: string;
  summary?: {
    total_items?: number;
    total_findings?: number;
    findings_by_severity?: Record<string, number>;
    items_by_classification?: Record<string, number>;
  };
  findings?: NightwardFinding[];
  items?: NightwardItem[];
}

export interface NightwardBundle {
  scan: NightwardReport;
  analysis?: NightwardAnalysis;
  fix_plan?: NightwardFixPlan;
  backup_plan?: NightwardBackupPlan;
}

export interface TuiState {
  tab: number;
  cursor: number;
  severity: string;
  search: string;
  searchMode: boolean;
  status: string;
}

export const tabs = ["Overview", "Findings", "Analysis", "Fix Plan", "Inventory", "Backup", "Help"] as const;
export const severityOrder = ["critical", "high", "medium", "low", "info"] as const;

const secretAssignmentPattern =
  /((?:token|secret|password|passwd|api[_-]?key|auth|credential|private[_-]?key)[\w.-]*\s*[:=]\s*)(["']?)[^"',\s}]+/gi;
const longSecretPattern = /\bsk-[A-Za-z0-9_-]{12,}\b/g;

export function redact(value: string | undefined): string {
  if (!value) {
    return "";
  }
  return value.replace(secretAssignmentPattern, "$1$2[redacted]").replace(longSecretPattern, "[redacted]");
}

export function severityRank(severity: Risk | undefined): number {
  switch ((severity || "info").toLowerCase()) {
    case "critical":
      return 5;
    case "high":
      return 4;
    case "medium":
      return 3;
    case "low":
      return 2;
    default:
      return 1;
  }
}

export function severityCounts(report: NightwardReport): Record<string, number> {
  const existing = report.summary?.findings_by_severity || {};
  const counts: Record<string, number> = {};
  for (const severity of severityOrder) {
    counts[severity] = Number(existing[severity] || 0);
  }
  for (const finding of report.findings || []) {
    const severity = (finding.severity || "info").toLowerCase();
    if (existing[severity] === undefined) {
      counts[severity] = (counts[severity] || 0) + 1;
    }
  }
  return counts;
}

export function totalFindings(report: NightwardReport): number {
  return report.summary?.total_findings ?? report.findings?.length ?? 0;
}

export function highestSeverity(report: NightwardReport): string {
  let highest = "info";
  for (const finding of report.findings || []) {
    if (severityRank(finding.severity) > severityRank(highest)) {
      highest = (finding.severity || "info").toLowerCase();
    }
  }
  return highest;
}

export function filteredFindings(report: NightwardReport, state: Pick<TuiState, "severity" | "search">): NightwardFinding[] {
  const query = state.search.trim().toLowerCase();
  return (report.findings || []).filter((finding) => {
    if (state.severity && (finding.severity || "").toLowerCase() !== state.severity) {
      return false;
    }
    if (!query) {
      return true;
    }
    const haystack = [
      finding.id,
      finding.tool,
      finding.path,
      finding.server,
      finding.rule,
      finding.message,
      finding.evidence,
      finding.recommendation,
      finding.recommended_action,
      finding.fix_summary
    ]
      .filter(Boolean)
      .join("\n")
      .toLowerCase();
    return haystack.includes(query);
  });
}

export function nextSeverity(current: string): string {
  if (!current) {
    return "critical";
  }
  const index = severityOrder.indexOf(current as (typeof severityOrder)[number]);
  if (index < 0 || index === severityOrder.length - 1) {
    return "";
  }
  return severityOrder[index + 1];
}

export function truncate(value: string | undefined, width: number): string {
  const text = redact(value || "").replace(/\s+/g, " ").trim();
  if (text.length <= width) {
    return text;
  }
  if (width <= 3) {
    return text.slice(0, width);
  }
  return text.slice(0, width - 3) + "...";
}

export function selectedFinding(report: NightwardReport, state: Pick<TuiState, "severity" | "search" | "cursor">): NightwardFinding | undefined {
  const findings = filteredFindings(report, state);
  if (findings.length === 0) {
    return undefined;
  }
  return findings[Math.min(Math.max(state.cursor, 0), findings.length - 1)];
}

export function normalizeBundle(value: NightwardReport | NightwardBundle): NightwardBundle {
  if ("scan" in value && value.scan) {
    return value;
  }
  return { scan: value };
}
