use crate::analysis::{Report as AnalysisReport, Signal as AnalysisSignal};
use crate::{Finding, Report as ScanReport, RiskLevel};
use anyhow::{anyhow, Context, Result};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::json;
use std::collections::BTreeSet;
use std::fs;
use std::path::Path;

pub const DEFAULT_POLICY: &str = "severity_threshold: high\nignore_findings: []\nignore_rules: []\ninclude_analysis: false\nanalysis_threshold: high\nanalysis_providers: []\nallow_online_providers: false\n";

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PolicyConfig {
    #[serde(default = "default_threshold")]
    pub severity_threshold: RiskLevel,
    #[serde(default)]
    pub ignore_findings: Vec<IgnoreFindingEntry>,
    #[serde(default)]
    pub ignore_rules: Vec<IgnoreRuleEntry>,
    #[serde(default)]
    pub include_analysis: bool,
    #[serde(default = "default_analysis_threshold")]
    pub analysis_threshold: RiskLevel,
    #[serde(default)]
    pub analysis_providers: Vec<String>,
    #[serde(default)]
    pub allow_online_providers: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IgnoreFindingEntry {
    pub id: String,
    #[serde(default)]
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IgnoreRuleEntry {
    #[serde(default)]
    pub rule: String,
    #[serde(default)]
    pub id: String,
    #[serde(default)]
    pub reason: String,
}

impl IgnoreRuleEntry {
    fn rule_id(&self) -> &str {
        if self.rule.trim().is_empty() {
            self.id.trim()
        } else {
            self.rule.trim()
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PolicyReport {
    pub schema_version: u32,
    pub generated_at: DateTime<Utc>,
    pub passed: bool,
    pub threshold: RiskLevel,
    pub finding_count: usize,
    pub blocking_count: usize,
    pub ignored_count: usize,
    pub analysis_violation_count: usize,
    pub max_severity: RiskLevel,
    pub findings: Vec<Finding>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub analysis_violations: Vec<AnalysisSignal>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub analysis: Option<AnalysisSummary>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AnalysisSummary {
    pub signal_count: usize,
    pub violation_count: usize,
    pub provider_warnings: usize,
    pub threshold: RiskLevel,
    pub max_severity: RiskLevel,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Badge {
    pub schema_version: u32,
    pub generated_at: DateTime<Utc>,
    pub status: String,
    pub passed: bool,
    pub threshold: RiskLevel,
    pub finding_count: usize,
    pub blocking_count: usize,
    pub ignored_count: usize,
    pub analysis_signal_count: usize,
    pub analysis_violation_count: usize,
    pub provider_warnings: usize,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub sarif: String,
}

impl Default for PolicyConfig {
    fn default() -> Self {
        Self {
            severity_threshold: default_threshold(),
            ignore_findings: Vec::new(),
            ignore_rules: Vec::new(),
            include_analysis: false,
            analysis_threshold: default_analysis_threshold(),
            analysis_providers: Vec::new(),
            allow_online_providers: false,
        }
    }
}

pub fn load(path: impl AsRef<Path>) -> Result<PolicyConfig> {
    let text = fs::read_to_string(path.as_ref())
        .with_context(|| format!("read {}", path.as_ref().display()))?;
    let config: PolicyConfig = serde_yaml::from_str(&text)?;
    validate(&config)?;
    Ok(config)
}

pub fn init_file(path: impl AsRef<Path>) -> Result<()> {
    if path.as_ref().exists() {
        return Ok(());
    }
    if let Some(parent) = path.as_ref().parent() {
        fs::create_dir_all(parent)?;
    }
    fs::write(path.as_ref(), DEFAULT_POLICY)?;
    Ok(())
}

pub fn check(
    scan: &ScanReport,
    config: &PolicyConfig,
    analysis: Option<&AnalysisReport>,
) -> PolicyReport {
    let ignored_findings: BTreeSet<_> = config
        .ignore_findings
        .iter()
        .map(|entry| entry.id.as_str())
        .collect();
    let ignored_rules: BTreeSet<_> = config
        .ignore_rules
        .iter()
        .map(IgnoreRuleEntry::rule_id)
        .filter(|rule| !rule.is_empty())
        .collect();
    let mut blocking = Vec::new();
    let mut ignored_count = 0;
    for finding in &scan.findings {
        if ignored_findings.contains(finding.id.as_str())
            || ignored_rules.contains(finding.rule.as_str())
        {
            ignored_count += 1;
            continue;
        }
        if finding.severity.rank() >= config.severity_threshold.rank() {
            blocking.push(finding.clone());
        }
    }
    let mut max_severity = blocking
        .iter()
        .map(|finding| finding.severity)
        .max_by_key(|risk| risk.rank())
        .unwrap_or(RiskLevel::Info);
    let analysis_violations: Vec<_> = analysis
        .map(|report| {
            report
                .signals
                .iter()
                .filter(|signal| signal.severity.rank() >= config.analysis_threshold.rank())
                .cloned()
                .collect()
        })
        .unwrap_or_default();
    let analysis_summary = analysis.map(|report| {
        if report.summary.highest_severity.rank() > max_severity.rank() {
            max_severity = report.summary.highest_severity;
        }
        AnalysisSummary {
            signal_count: report.summary.total_signals,
            violation_count: analysis_violations.len(),
            provider_warnings: report.summary.provider_warnings,
            threshold: config.analysis_threshold,
            max_severity: report.summary.highest_severity,
        }
    });
    let analysis_violation_count = analysis_violations.len();
    PolicyReport {
        schema_version: 1,
        generated_at: Utc::now(),
        passed: blocking.is_empty() && analysis_violations.is_empty(),
        threshold: config.severity_threshold,
        finding_count: scan.summary.total_findings,
        blocking_count: blocking.len(),
        ignored_count,
        analysis_violation_count,
        max_severity,
        findings: blocking,
        analysis_violations,
        analysis: analysis_summary,
    }
}

pub fn badge(report: &PolicyReport, sarif: String) -> Badge {
    Badge {
        schema_version: 1,
        generated_at: Utc::now(),
        status: if report.passed {
            "passing".to_string()
        } else {
            "failing".to_string()
        },
        passed: report.passed,
        threshold: report.threshold,
        finding_count: report.finding_count,
        blocking_count: report.blocking_count,
        ignored_count: report.ignored_count,
        analysis_signal_count: report
            .analysis
            .as_ref()
            .map(|analysis| analysis.signal_count)
            .unwrap_or(0),
        analysis_violation_count: report.analysis_violation_count,
        provider_warnings: report
            .analysis
            .as_ref()
            .map(|analysis| analysis.provider_warnings)
            .unwrap_or(0),
        sarif,
    }
}

pub fn sarif(scan: &ScanReport, policy: Option<&PolicyReport>) -> serde_json::Value {
    let findings = policy
        .map(|report| report.findings.as_slice())
        .unwrap_or(scan.findings.as_slice());
    let mut results: Vec<_> = findings
        .iter()
        .map(|finding| {
            json!({
                "ruleId": finding.rule,
                "level": sarif_level(finding.severity),
                "message": { "text": finding.message },
                "locations": [{
                    "physicalLocation": {
                        "artifactLocation": { "uri": finding.path },
                        "region": { "startLine": 1 }
                    }
                }],
                "properties": {
                    "nightward_id": finding.id,
                    "severity": finding.severity,
                    "tool": finding.tool,
                    "server": finding.server
                }
            })
        })
        .collect();
    if let Some(report) = policy {
        results.extend(report.analysis_violations.iter().map(|signal| {
            json!({
                "ruleId": signal.rule,
                "level": sarif_level(signal.severity),
                "message": { "text": signal.message },
                "locations": [{
                    "physicalLocation": {
                        "artifactLocation": {
                            "uri": if signal.path.is_empty() { "nightward-analysis" } else { signal.path.as_str() }
                        },
                        "region": { "startLine": 1 }
                    }
                }],
                "properties": {
                    "nightward_signal_id": signal.id,
                    "severity": signal.severity,
                    "provider": signal.provider,
                    "category": signal.category
                }
            })
        }));
    }
    let mut rule_ids: BTreeSet<String> = scan.summary.findings_by_rule.keys().cloned().collect();
    if let Some(report) = policy {
        rule_ids.extend(
            report
                .analysis_violations
                .iter()
                .map(|signal| signal.rule.clone()),
        );
    }
    let rules: Vec<_> = rule_ids
        .into_iter()
        .map(|rule| {
            json!({
                "id": rule,
                "name": rule,
                "shortDescription": { "text": rule },
            })
        })
        .collect();
    json!({
        "version": "2.1.0",
        "$schema": "https://json.schemastore.org/sarif-2.1.0.json",
        "runs": [{
            "tool": {
                "driver": {
                    "name": "Nightward",
                    "informationUri": "https://github.com/JSONbored/nightward",
                    "rules": rules
                }
            },
            "results": results
        }]
    })
}

fn validate(config: &PolicyConfig) -> Result<()> {
    for (index, entry) in config.ignore_findings.iter().enumerate() {
        if entry.id.trim().is_empty() {
            return Err(anyhow!("ignore_findings[{index}] requires an id"));
        }
        if entry.reason.trim().is_empty() {
            return Err(anyhow!(
                "ignore_findings[{index}] requires a non-empty reason"
            ));
        }
    }
    for (index, entry) in config.ignore_rules.iter().enumerate() {
        if entry.rule_id().is_empty() {
            return Err(anyhow!("ignore_rules[{index}] requires a rule"));
        }
        if entry.reason.trim().is_empty() {
            return Err(anyhow!("ignore_rules[{index}] requires a non-empty reason"));
        }
    }
    Ok(())
}

fn default_threshold() -> RiskLevel {
    RiskLevel::High
}

fn default_analysis_threshold() -> RiskLevel {
    RiskLevel::High
}

fn sarif_level(level: RiskLevel) -> &'static str {
    match level {
        RiskLevel::Critical | RiskLevel::High => "error",
        RiskLevel::Medium => "warning",
        _ => "note",
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::analysis::{Signal, SignalCategory, SubjectType, Summary as AnalysisReportSummary};
    use crate::{Finding, FixKind, Report as ScanReport};
    use std::collections::BTreeMap;

    #[test]
    fn load_requires_reasons_for_ignores() {
        let dir = tempfile::tempdir().expect("temp dir");
        let path = dir.path().join("policy.yml");
        fs::write(
            &path,
            "severity_threshold: high\nignore_findings:\n  - id: finding-1\nignore_rules: []\n",
        )
        .expect("write policy");

        let error = load(&path).expect_err("reasonless ignore should fail");

        assert!(error
            .to_string()
            .contains("ignore_findings[0] requires a non-empty reason"));
    }

    #[test]
    fn load_accepts_rule_key_and_legacy_id_key_for_ignored_rules() {
        let dir = tempfile::tempdir().expect("temp dir");
        let modern = dir.path().join("modern.yml");
        let legacy = dir.path().join("legacy.yml");
        fs::write(
            &modern,
            "ignore_rules:\n  - rule: mcp_server_review\n    reason: reviewed locally\n",
        )
        .expect("write modern policy");
        fs::write(
            &legacy,
            "ignore_rules:\n  - id: mcp_server_review\n    reason: reviewed locally\n",
        )
        .expect("write legacy policy");

        assert_eq!(
            load(&modern).unwrap().ignore_rules[0].rule_id(),
            "mcp_server_review"
        );
        assert_eq!(
            load(&legacy).unwrap().ignore_rules[0].rule_id(),
            "mcp_server_review"
        );
    }

    #[test]
    fn check_honors_analysis_threshold() {
        let scan = ScanReport::empty("home".to_string(), String::new(), "home".to_string());
        let analysis = analysis_report_with_signal(RiskLevel::High);
        let mut config = PolicyConfig {
            analysis_threshold: RiskLevel::High,
            ..PolicyConfig::default()
        };

        let report = check(&scan, &config, Some(&analysis));

        assert!(!report.passed);
        assert_eq!(report.analysis_violation_count, 1);
        assert_eq!(report.analysis.as_ref().unwrap().violation_count, 1);

        config.analysis_threshold = RiskLevel::Critical;
        let report = check(&scan, &config, Some(&analysis));

        assert!(report.passed);
        assert_eq!(report.analysis_violation_count, 0);
    }

    #[test]
    fn sarif_preserves_provider_warning_analysis_violations() {
        let scan = ScanReport::empty("home".to_string(), String::new(), "home".to_string());
        let mut analysis = analysis_report_with_signal(RiskLevel::Low);
        analysis.summary.provider_warnings = 1;
        analysis.signals[0].provider = "gitleaks".to_string();
        analysis.signals[0].rule = "gitleaks/provider_execution_failed".to_string();
        analysis.signals[0].category = SignalCategory::Unknown;
        analysis.signals[0].message = "gitleaks provider execution failed.".to_string();
        let config = PolicyConfig {
            analysis_threshold: RiskLevel::Low,
            ..PolicyConfig::default()
        };

        let report = check(&scan, &config, Some(&analysis));
        let sarif = sarif(&scan, Some(&report));
        let sarif_text = serde_json::to_string(&sarif).expect("sarif json");

        assert_eq!(report.analysis.as_ref().unwrap().provider_warnings, 1);
        assert!(sarif_text.contains("gitleaks/provider_execution_failed"));
        assert!(sarif_text.contains("\"provider\":\"gitleaks\""));
    }

    #[test]
    fn default_policy_blocks_high_provider_execution_failures() {
        let scan = ScanReport::empty("home".to_string(), String::new(), "home".to_string());
        let mut analysis = analysis_report_with_signal(RiskLevel::High);
        analysis.summary.provider_warnings = 1;
        analysis.signals[0].provider = "gitleaks".to_string();
        analysis.signals[0].rule = "gitleaks/provider_execution_failed".to_string();
        let config = PolicyConfig::default();

        let report = check(&scan, &config, Some(&analysis));

        assert!(!report.passed);
        assert_eq!(report.analysis_violation_count, 1);
    }

    #[test]
    fn check_ignores_rules_by_rule_key() {
        let mut scan = ScanReport::empty("home".to_string(), String::new(), "home".to_string());
        scan.findings
            .push(finding("finding-1", "mcp_server_review", RiskLevel::High));
        scan.recompute_summary();
        let config = PolicyConfig {
            ignore_rules: vec![IgnoreRuleEntry {
                rule: "mcp_server_review".to_string(),
                id: String::new(),
                reason: "reviewed locally".to_string(),
            }],
            ..PolicyConfig::default()
        };

        let report = check(&scan, &config, None);

        assert!(report.passed);
        assert_eq!(report.ignored_count, 1);
        assert_eq!(report.blocking_count, 0);
    }

    fn analysis_report_with_signal(severity: RiskLevel) -> AnalysisReport {
        AnalysisReport {
            schema_version: 1,
            generated_at: Utc::now(),
            mode: "home".to_string(),
            workspace: String::new(),
            summary: AnalysisReportSummary {
                total_subjects: 1,
                total_signals: 1,
                signals_by_severity: BTreeMap::from([(severity, 1)]),
                signals_by_category: BTreeMap::from([(SignalCategory::ExecutionRisk, 1)]),
                signals_by_provider: BTreeMap::from([("nightward".to_string(), 1)]),
                highest_severity: severity,
                provider_warnings: 0,
                no_known_risk_signals: false,
            },
            providers: Vec::new(),
            subjects: Vec::new(),
            signals: vec![Signal {
                id: "signal-1".to_string(),
                provider: "nightward".to_string(),
                rule: "analysis_review".to_string(),
                category: SignalCategory::ExecutionRisk,
                subject_id: "subject-1".to_string(),
                subject_type: SubjectType::Finding,
                path: "config.toml".to_string(),
                severity,
                confidence: "medium".to_string(),
                message: "review analysis signal".to_string(),
                evidence: "fixture".to_string(),
                recommended_action: "review".to_string(),
                why: String::new(),
            }],
        }
    }

    fn finding(id: &str, rule: &str, severity: RiskLevel) -> Finding {
        Finding {
            id: id.to_string(),
            tool: "Codex".to_string(),
            path: "config.toml".to_string(),
            server: String::new(),
            severity,
            rule: rule.to_string(),
            message: "review finding".to_string(),
            evidence: "fixture".to_string(),
            recommended_action: "review".to_string(),
            impact: String::new(),
            why: String::new(),
            docs_url: String::new(),
            fix_available: false,
            fix_kind: Some(FixKind::ManualReview),
            confidence: "medium".to_string(),
            risk: None,
            requires_review: true,
            fix_summary: String::new(),
            fix_steps: Vec::new(),
            patch_hint: None,
        }
    }
}
