use crate::analysis::Report as AnalysisReport;
use crate::{Finding, Report as ScanReport, RiskLevel};
use anyhow::{Context, Result};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::json;
use std::collections::BTreeSet;
use std::fs;
use std::path::Path;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PolicyConfig {
    #[serde(default = "default_threshold")]
    pub severity_threshold: RiskLevel,
    #[serde(default)]
    pub ignore_findings: Vec<IgnoreEntry>,
    #[serde(default)]
    pub ignore_rules: Vec<IgnoreEntry>,
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
pub struct IgnoreEntry {
    pub id: String,
    #[serde(default)]
    pub reason: String,
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
    pub max_severity: RiskLevel,
    pub findings: Vec<Finding>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub analysis: Option<AnalysisSummary>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AnalysisSummary {
    pub signal_count: usize,
    pub provider_warnings: usize,
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
    Ok(serde_yaml::from_str(&text)?)
}

pub fn init_file(path: impl AsRef<Path>) -> Result<()> {
    if path.as_ref().exists() {
        return Ok(());
    }
    if let Some(parent) = path.as_ref().parent() {
        fs::create_dir_all(parent)?;
    }
    fs::write(
        path.as_ref(),
        "severity_threshold: high\nignore_findings: []\nignore_rules: []\ninclude_analysis: false\nanalysis_threshold: high\nanalysis_providers: []\nallow_online_providers: false\n",
    )?;
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
        .map(|entry| entry.id.as_str())
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
    let analysis_summary = analysis.map(|report| {
        if report.summary.highest_severity.rank() > max_severity.rank() {
            max_severity = report.summary.highest_severity;
        }
        AnalysisSummary {
            signal_count: report.summary.total_signals,
            provider_warnings: report.summary.provider_warnings,
            max_severity: report.summary.highest_severity,
        }
    });
    PolicyReport {
        schema_version: 1,
        generated_at: Utc::now(),
        passed: blocking.is_empty(),
        threshold: config.severity_threshold,
        finding_count: scan.summary.total_findings,
        blocking_count: blocking.len(),
        ignored_count,
        max_severity,
        findings: blocking,
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
        sarif,
    }
}

pub fn sarif(scan: &ScanReport, policy: Option<&PolicyReport>) -> serde_json::Value {
    let findings = policy
        .map(|report| report.findings.as_slice())
        .unwrap_or(scan.findings.as_slice());
    let results: Vec<_> = findings
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
    let rules: Vec<_> = scan
        .summary
        .findings_by_rule
        .keys()
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
