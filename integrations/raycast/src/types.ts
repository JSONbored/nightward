export type Classification =
  | "portable"
  | "machine-local"
  | "secret-auth"
  | "runtime-cache"
  | "app-owned"
  | "unknown";

export type RiskLevel = "info" | "low" | "medium" | "high" | "critical";

export type FixKind =
  | "pin-package"
  | "externalize-secret"
  | "replace-shell-wrapper"
  | "narrow-filesystem"
  | "manual-review"
  | "ignore-with-reason";

export type Summary = {
  total_items: number;
  total_findings: number;
  items_by_classification: Partial<Record<Classification, number>>;
  items_by_risk: Partial<Record<RiskLevel, number>>;
  items_by_tool: Record<string, number>;
  findings_by_severity: Partial<Record<RiskLevel, number>>;
  findings_by_rule: Record<string, number>;
  findings_by_tool: Record<string, number>;
};

export type InventoryItem = {
  id: string;
  tool: string;
  path: string;
  kind: string;
  classification: Classification;
  risk: RiskLevel;
  reason: string;
  recommended_action: string;
  exists: boolean;
  size_bytes?: number;
  mod_time?: string;
  metadata?: Record<string, string>;
};

export type Finding = {
  id: string;
  tool: string;
  path: string;
  server?: string;
  severity: RiskLevel;
  rule: string;
  message: string;
  evidence?: string;
  recommended_action: string;
  impact?: string;
  why_this_matters?: string;
  docs_url?: string;
  fix_available: boolean;
  fix_kind?: FixKind;
  confidence?: string;
  risk?: RiskLevel;
  requires_review: boolean;
  fix_summary?: string;
  fix_steps?: string[];
};

export type AdapterStatus = {
  name: string;
  description: string;
  available: boolean;
  checked: string[];
  found?: string[];
};

export type ScanReport = {
  schema_version?: number;
  generated_at: string;
  hostname: string;
  home: string;
  workspace?: string;
  scan_mode?: string;
  summary: Summary;
  items: InventoryItem[];
  findings: Finding[];
  adapters: AdapterStatus[];
};

export type SignalCategory =
  | "supply-chain"
  | "secrets-exposure"
  | "filesystem-scope"
  | "network-exposure"
  | "execution-risk"
  | "machine-locality"
  | "app-state"
  | "unknown";

export type AnalysisSignal = {
  id: string;
  provider: string;
  rule: string;
  category: SignalCategory;
  subject_id: string;
  subject_type: "finding" | "item" | "package";
  path?: string;
  severity: RiskLevel;
  confidence: string;
  message: string;
  evidence?: string;
  recommended_action: string;
  why_this_matters?: string;
};

export type ProviderStatus = {
  name: string;
  kind: string;
  command?: string;
  online: boolean;
  default: boolean;
  privacy: string;
  capabilities: string;
  enabled: boolean;
  available: boolean;
  status: string;
  detail?: string;
};

export type AnalysisReport = {
  schema_version?: number;
  generated_at: string;
  mode: string;
  workspace?: string;
  summary: {
    total_subjects: number;
    total_signals: number;
    signals_by_severity: Partial<Record<RiskLevel, number>>;
    signals_by_category: Partial<Record<SignalCategory, number>>;
    signals_by_provider: Record<string, number>;
    highest_severity: RiskLevel;
    provider_warnings: number;
    no_known_risk_signals: boolean;
  };
  providers: ProviderStatus[];
  subjects: Array<{
    id: string;
    type: "finding" | "item" | "package";
    name: string;
    tool?: string;
    path?: string;
    rule?: string;
    package?: string;
    evidence?: string;
  }>;
  signals: AnalysisSignal[];
};

export type DoctorCheck = {
  id: string;
  status: "ok" | "warn" | "info" | "error" | string;
  message: string;
  detail?: string;
};

export type SchedulePlan = {
  schema_version?: number;
  preset: string;
  platform: string;
  report_dir: string;
  log_dir: string;
  command?: string[];
  installed: boolean;
  last_report?: string;
  last_run?: string;
  last_findings?: number;
  history?: ReportRecord[];
};

export type ReportRecord = {
  path: string;
  mod_time: string;
  findings: number;
  highest_severity?: RiskLevel;
  findings_by_severity?: Partial<Record<RiskLevel, number>>;
  size_bytes: number;
  report_name: string;
};

export type DoctorReport = {
  schema_version?: number;
  generated_at: string;
  version: string;
  home: string;
  executable: string;
  checks: DoctorCheck[];
  schedule: SchedulePlan;
  adapters: AdapterStatus[];
};

export type FixPlan = {
  schema_version?: number;
  generated_at: string;
  summary: {
    total: number;
    safe: number;
    review: number;
    blocked: number;
  };
  fixes: Array<{
    finding_id: string;
    tool: string;
    path: string;
    severity: RiskLevel;
    rule: string;
    fix_available: boolean;
    fix_kind?: FixKind;
    confidence?: string;
    risk?: RiskLevel;
    requires_review: boolean;
    status: "safe" | "review" | "blocked";
    summary: string;
    steps?: string[];
    evidence?: string;
    impact?: string;
    why_this_matters?: string;
  }>;
};
