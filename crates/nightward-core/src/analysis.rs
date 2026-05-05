use crate::inventory::{redact_text, stable_id};
use crate::providers::{run_selected, statuses, ProviderStatus};
use crate::{max_risk, Classification, Finding, Item, Report as ScanReport, RiskLevel};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;
use std::path::Path;

#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum SubjectType {
    Finding,
    Item,
    Package,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum SignalCategory {
    SupplyChain,
    SecretsExposure,
    FilesystemScope,
    NetworkExposure,
    ExecutionRisk,
    MachineLocality,
    AppState,
    Unknown,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Options {
    #[serde(default)]
    pub mode: String,
    #[serde(default)]
    pub workspace: String,
    #[serde(default)]
    pub with: Vec<String>,
    #[serde(default)]
    pub online: bool,
    #[serde(default)]
    pub package: String,
    #[serde(default)]
    pub finding_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Report {
    pub schema_version: u32,
    pub generated_at: DateTime<Utc>,
    pub mode: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub workspace: String,
    pub summary: Summary,
    pub providers: Vec<ProviderStatus>,
    pub subjects: Vec<Subject>,
    pub signals: Vec<Signal>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct Summary {
    pub total_subjects: usize,
    pub total_signals: usize,
    pub signals_by_severity: BTreeMap<RiskLevel, usize>,
    pub signals_by_category: BTreeMap<SignalCategory, usize>,
    pub signals_by_provider: BTreeMap<String, usize>,
    pub highest_severity: RiskLevel,
    pub provider_warnings: usize,
    pub no_known_risk_signals: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Subject {
    pub id: String,
    #[serde(rename = "type")]
    pub subject_type: SubjectType,
    pub name: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub tool: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub path: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub rule: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub package: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub evidence: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Signal {
    pub id: String,
    pub provider: String,
    pub rule: String,
    pub category: SignalCategory,
    pub subject_id: String,
    pub subject_type: SubjectType,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub path: String,
    pub severity: RiskLevel,
    pub confidence: String,
    pub message: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub evidence: String,
    pub recommended_action: String,
    #[serde(
        rename = "why_this_matters",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub why: String,
}

#[derive(Debug, Clone)]
pub struct ProviderFinding {
    pub rule: String,
    pub path: String,
    pub message: String,
    pub evidence: String,
    pub severity: RiskLevel,
    pub category: SignalCategory,
}

pub fn run(scan: &ScanReport, options: Options) -> Report {
    let mode = if options.mode.is_empty() {
        scan.scan_mode.clone()
    } else {
        options.mode.clone()
    };
    let mut out = Report {
        schema_version: 1,
        generated_at: scan.generated_at,
        mode,
        workspace: options.workspace.clone(),
        summary: Summary::default(),
        providers: statuses(&options.with, options.online),
        subjects: Vec::new(),
        signals: Vec::new(),
    };

    if !options.package.is_empty() {
        add_package_signal(&mut out, &options.package);
    }

    for finding in &scan.findings {
        if !options.finding_id.is_empty()
            && finding.id != options.finding_id
            && !finding.id.starts_with(&options.finding_id)
        {
            continue;
        }
        let subject = subject_from_finding(finding);
        out.signals.push(signal_from_finding(&subject, finding));
        out.subjects.push(subject);
    }

    if options.finding_id.is_empty() && options.package.is_empty() {
        for item in &scan.items {
            if let Some((subject, signal)) = signal_from_item(item) {
                out.subjects.push(subject);
                out.signals.push(signal);
            }
        }
        append_provider_signals(&mut out, scan, &options);
    }

    out.subjects.sort_by(|a, b| {
        a.subject_type
            .cmp(&b.subject_type)
            .then_with(|| a.name.cmp(&b.name))
    });
    out.signals.sort_by(|a, b| {
        b.severity
            .rank()
            .cmp(&a.severity.rank())
            .then_with(|| a.id.cmp(&b.id))
    });
    finalize(&mut out);
    out
}

pub fn explain(report: &Report, id_or_prefix: &str) -> Option<Signal> {
    let exact = report
        .signals
        .iter()
        .find(|signal| signal.id == id_or_prefix)
        .cloned();
    exact.or_else(|| {
        let matches: Vec<_> = report
            .signals
            .iter()
            .filter(|signal| signal.id.starts_with(id_or_prefix))
            .cloned()
            .collect();
        (matches.len() == 1).then(|| matches[0].clone())
    })
}

fn add_package_signal(out: &mut Report, package: &str) {
    let subject = Subject {
        id: stable_id(&["package", package]),
        subject_type: SubjectType::Package,
        name: package.to_string(),
        tool: String::new(),
        path: String::new(),
        rule: String::new(),
        package: package.to_string(),
        evidence: String::new(),
    };
    out.signals.push(Signal {
        id: stable_id(&["signal", "nightward", "package_review", package]),
        provider: "nightward".to_string(),
        rule: "package_review".to_string(),
        category: SignalCategory::SupplyChain,
        subject_id: subject.id.clone(),
        subject_type: subject.subject_type,
        path: String::new(),
        severity: RiskLevel::Info,
        confidence: "low".to_string(),
        message: "Package analysis is structural only without an explicit provider.".to_string(),
        evidence: format!("package={package}"),
        recommended_action:
            "Run with an explicit provider after reviewing provider privacy behavior.".to_string(),
        why: "Nightward avoids making package safety claims without explicit provider evidence."
            .to_string(),
    });
    out.subjects.push(subject);
}

fn subject_from_finding(finding: &Finding) -> Subject {
    Subject {
        id: stable_id(&["finding", &finding.id]),
        subject_type: SubjectType::Finding,
        name: finding.id.clone(),
        tool: finding.tool.clone(),
        path: finding.path.clone(),
        rule: finding.rule.clone(),
        package: String::new(),
        evidence: finding.evidence.clone(),
    }
}

fn signal_from_finding(subject: &Subject, finding: &Finding) -> Signal {
    Signal {
        id: stable_id(&["signal", "nightward", &finding.rule, &finding.id]),
        provider: "nightward".to_string(),
        rule: finding.rule.clone(),
        category: category_for_rule(&finding.rule),
        subject_id: subject.id.clone(),
        subject_type: subject.subject_type,
        path: finding.path.clone(),
        severity: finding.severity,
        confidence: if finding.confidence.is_empty() {
            "medium".to_string()
        } else {
            finding.confidence.clone()
        },
        message: finding.message.clone(),
        evidence: redact_text(&finding.evidence),
        recommended_action: finding.recommended_action.clone(),
        why: finding.why.clone(),
    }
}

fn signal_from_item(item: &Item) -> Option<(Subject, Signal)> {
    if !matches!(
        item.classification,
        Classification::SecretAuth | Classification::MachineLocal
    ) {
        return None;
    }
    let subject = Subject {
        id: stable_id(&["item", &item.id]),
        subject_type: SubjectType::Item,
        name: item.id.clone(),
        tool: item.tool.clone(),
        path: item.path.clone(),
        rule: String::new(),
        package: String::new(),
        evidence: format!("{:?}", item.classification),
    };
    let signal = Signal {
        id: stable_id(&["signal", "nightward", "item_classification", &item.id]),
        provider: "nightward".to_string(),
        rule: "item_classification".to_string(),
        category: match item.classification {
            Classification::SecretAuth => SignalCategory::SecretsExposure,
            Classification::MachineLocal => SignalCategory::MachineLocality,
            _ => SignalCategory::Unknown,
        },
        subject_id: subject.id.clone(),
        subject_type: subject.subject_type,
        path: item.path.clone(),
        severity: item.risk,
        confidence: "medium".to_string(),
        message: item.reason.clone(),
        evidence: redact_text(&item.path),
        recommended_action: item.recommended_action.clone(),
        why: "Classification signals help decide what is safe to sync or back up.".to_string(),
    };
    Some((subject, signal))
}

fn append_provider_signals(out: &mut Report, scan: &ScanReport, options: &Options) {
    let root = if !scan.workspace.is_empty() {
        Path::new(&scan.workspace)
    } else if !options.workspace.is_empty() {
        Path::new(&options.workspace)
    } else {
        Path::new(&scan.home)
    };
    for (provider, result) in run_selected(root, &options.with, options.online) {
        match result {
            Ok(findings) => {
                for finding in findings {
                    append_provider_signal(out, &provider, finding);
                }
            }
            Err(error) => {
                append_provider_signal(
                    out,
                    &provider,
                    ProviderFinding {
                        rule: "provider_execution_failed".to_string(),
                        path: root.display().to_string(),
                        message: format!("{provider} provider execution failed."),
                        evidence: redact_text(&error.to_string()),
                        severity: RiskLevel::High,
                        category: SignalCategory::Unknown,
                    },
                );
            }
        }
    }
}

fn append_provider_signal(out: &mut Report, provider: &str, finding: ProviderFinding) {
    let subject_id = stable_id(&[
        "provider-subject",
        provider,
        &finding.rule,
        &finding.path,
        &finding.evidence,
    ]);
    out.subjects.push(Subject {
        id: subject_id.clone(),
        subject_type: SubjectType::Item,
        name: format!("{provider}/{}", finding.rule),
        tool: provider.to_string(),
        path: finding.path.clone(),
        rule: format!("{provider}/{}", finding.rule),
        package: String::new(),
        evidence: finding.evidence.clone(),
    });
    out.signals.push(Signal {
        id: stable_id(&[
            "signal",
            provider,
            &finding.rule,
            &finding.path,
            &finding.evidence,
        ]),
        provider: provider.to_string(),
        rule: format!("{provider}/{}", finding.rule),
        category: finding.category,
        subject_id,
        subject_type: SubjectType::Item,
        path: finding.path,
        severity: finding.severity,
        confidence: "medium".to_string(),
        message: finding.message,
        evidence: redact_text(&finding.evidence),
        recommended_action: provider_recommendation(provider, finding.category),
        why: "Provider execution was explicitly requested, so Nightward preserves only redacted finding metadata for review.".to_string(),
    });
}

fn provider_recommendation(provider: &str, category: SignalCategory) -> String {
    match category {
        SignalCategory::SecretsExposure => {
            format!("Review the {provider} secret signal, rotate exposed values if real, and keep secrets out of synced config.")
        }
        SignalCategory::SupplyChain => {
            format!(
                "Review the {provider} dependency signal and pin, patch, or remove risky packages."
            )
        }
        _ => format!("Review the {provider} signal before trusting this workspace or config."),
    }
}

fn category_for_rule(rule: &str) -> SignalCategory {
    if rule.contains("secret") || rule.contains("token") {
        SignalCategory::SecretsExposure
    } else if rule.contains("package") {
        SignalCategory::SupplyChain
    } else if rule.contains("filesystem") {
        SignalCategory::FilesystemScope
    } else if rule.contains("endpoint") {
        SignalCategory::NetworkExposure
    } else if rule.contains("shell") || rule.contains("command") {
        SignalCategory::ExecutionRisk
    } else {
        SignalCategory::Unknown
    }
}

fn finalize(report: &mut Report) {
    report.summary.total_subjects = report.subjects.len();
    report.summary.total_signals = report.signals.len();
    report.summary.highest_severity = max_risk_from_signals(&report.signals);
    report.summary.no_known_risk_signals = report.signals.is_empty()
        || report
            .signals
            .iter()
            .all(|signal| signal.severity == RiskLevel::Info);
    report.summary.provider_warnings = report
        .signals
        .iter()
        .filter(|signal| signal.rule.ends_with("provider_execution_failed"))
        .count();
    for signal in &report.signals {
        *report
            .summary
            .signals_by_severity
            .entry(signal.severity)
            .or_default() += 1;
        *report
            .summary
            .signals_by_category
            .entry(signal.category)
            .or_default() += 1;
        *report
            .summary
            .signals_by_provider
            .entry(signal.provider.clone())
            .or_default() += 1;
    }
}

fn max_risk_from_signals(signals: &[Signal]) -> RiskLevel {
    signals
        .iter()
        .map(|signal| signal.severity)
        .max_by_key(|risk| risk.rank())
        .unwrap_or(RiskLevel::Info)
}

#[allow(dead_code)]
fn _max_risk_from_findings(findings: &[Finding]) -> RiskLevel {
    max_risk(findings)
}
