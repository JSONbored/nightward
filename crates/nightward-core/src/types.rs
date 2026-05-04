use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

pub const REPORT_SCHEMA_VERSION: u32 = 1;

#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum Classification {
    Portable,
    MachineLocal,
    SecretAuth,
    RuntimeCache,
    AppOwned,
    Unknown,
}

impl Default for Classification {
    fn default() -> Self {
        Self::Unknown
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum RiskLevel {
    Info,
    Low,
    Medium,
    High,
    Critical,
}

impl Default for RiskLevel {
    fn default() -> Self {
        Self::Info
    }
}

impl RiskLevel {
    pub fn rank(self) -> u8 {
        match self {
            Self::Info => 0,
            Self::Low => 1,
            Self::Medium => 2,
            Self::High => 3,
            Self::Critical => 4,
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum FixKind {
    PinPackage,
    ExternalizeSecret,
    ReplaceShellWrapper,
    NarrowFilesystem,
    ManualReview,
    IgnoreWithReason,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Item {
    pub id: String,
    pub tool: String,
    pub path: String,
    pub kind: String,
    pub classification: Classification,
    pub risk: RiskLevel,
    pub reason: String,
    pub recommended_action: String,
    pub exists: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub size_bytes: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mod_time: Option<DateTime<Utc>>,
    #[serde(default, skip_serializing_if = "BTreeMap::is_empty")]
    pub metadata: BTreeMap<String, String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Finding {
    pub id: String,
    pub tool: String,
    pub path: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub server: String,
    pub severity: RiskLevel,
    pub rule: String,
    pub message: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub evidence: String,
    pub recommended_action: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub impact: String,
    #[serde(
        rename = "why_this_matters",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub why: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub docs_url: String,
    pub fix_available: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub fix_kind: Option<FixKind>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub confidence: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub risk: Option<RiskLevel>,
    pub requires_review: bool,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub fix_summary: String,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub fix_steps: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub patch_hint: Option<PatchHint>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PatchHint {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub kind: Option<FixKind>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub package: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub env_key: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub header_key: String,
    #[serde(default, skip_serializing_if = "is_false")]
    pub inline_secret: bool,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub direct_command: String,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub direct_args: Vec<String>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub replacement: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdapterStatus {
    pub name: String,
    pub description: String,
    pub available: bool,
    pub checked: Vec<String>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub found: Vec<String>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct Summary {
    pub total_items: usize,
    pub total_findings: usize,
    pub items_by_classification: BTreeMap<Classification, usize>,
    pub items_by_risk: BTreeMap<RiskLevel, usize>,
    pub items_by_tool: BTreeMap<String, usize>,
    pub findings_by_severity: BTreeMap<RiskLevel, usize>,
    pub findings_by_rule: BTreeMap<String, usize>,
    pub findings_by_tool: BTreeMap<String, usize>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Report {
    pub schema_version: u32,
    pub generated_at: DateTime<Utc>,
    pub hostname: String,
    pub home: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub workspace: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub scan_mode: String,
    pub summary: Summary,
    pub items: Vec<Item>,
    pub findings: Vec<Finding>,
    pub adapters: Vec<AdapterStatus>,
}

impl Report {
    pub fn empty(home: String, workspace: String, scan_mode: String) -> Self {
        Self {
            schema_version: REPORT_SCHEMA_VERSION,
            generated_at: Utc::now(),
            hostname: std::env::var("HOSTNAME")
                .ok()
                .filter(|value| !value.is_empty())
                .unwrap_or_else(|| "unknown".to_string()),
            home,
            workspace,
            scan_mode,
            summary: Summary::default(),
            items: Vec::new(),
            findings: Vec::new(),
            adapters: Vec::new(),
        }
    }

    pub fn recompute_summary(&mut self) {
        let mut summary = Summary {
            total_items: self.items.len(),
            total_findings: self.findings.len(),
            ..Summary::default()
        };
        for item in &self.items {
            *summary
                .items_by_classification
                .entry(item.classification)
                .or_default() += 1;
            *summary.items_by_risk.entry(item.risk).or_default() += 1;
            *summary.items_by_tool.entry(item.tool.clone()).or_default() += 1;
        }
        for finding in &self.findings {
            *summary
                .findings_by_severity
                .entry(finding.severity)
                .or_default() += 1;
            *summary
                .findings_by_rule
                .entry(finding.rule.clone())
                .or_default() += 1;
            *summary
                .findings_by_tool
                .entry(finding.tool.clone())
                .or_default() += 1;
        }
        self.summary = summary;
    }
}

pub fn max_risk(findings: &[Finding]) -> RiskLevel {
    findings
        .iter()
        .map(|finding| finding.severity)
        .max_by_key(|risk| risk.rank())
        .unwrap_or(RiskLevel::Info)
}

fn is_false(value: &bool) -> bool {
    !*value
}
