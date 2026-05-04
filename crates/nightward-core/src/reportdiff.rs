use crate::{Finding, Report as ScanReport, RiskLevel};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DiffReport {
    pub schema_version: u32,
    pub generated_at: DateTime<Utc>,
    pub base: String,
    pub head: String,
    pub summary: DiffSummary,
    pub added: Vec<Finding>,
    pub removed: Vec<Finding>,
    pub changed: Vec<ChangedFinding>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct DiffSummary {
    pub added: usize,
    pub removed: usize,
    pub changed: usize,
    pub max_added_severity: RiskLevel,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChangedFinding {
    pub id: String,
    pub before: Finding,
    pub after: Finding,
}

pub fn diff(
    base_name: String,
    base: &ScanReport,
    head_name: String,
    head: &ScanReport,
) -> DiffReport {
    let base_map: BTreeMap<_, _> = base
        .findings
        .iter()
        .map(|finding| (finding_key(finding), finding.clone()))
        .collect();
    let head_map: BTreeMap<_, _> = head
        .findings
        .iter()
        .map(|finding| (finding_key(finding), finding.clone()))
        .collect();
    let mut added = Vec::new();
    let mut removed = Vec::new();
    let mut changed = Vec::new();
    for (key, finding) in &head_map {
        match base_map.get(key) {
            None => added.push(finding.clone()),
            Some(before)
                if before.message != finding.message || before.severity != finding.severity =>
            {
                changed.push(ChangedFinding {
                    id: key.clone(),
                    before: before.clone(),
                    after: finding.clone(),
                });
            }
            _ => {}
        }
    }
    for (key, finding) in &base_map {
        if !head_map.contains_key(key) {
            removed.push(finding.clone());
        }
    }
    let max_added_severity = added
        .iter()
        .map(|finding| finding.severity)
        .max_by_key(|risk| risk.rank())
        .unwrap_or(RiskLevel::Info);
    let summary = DiffSummary {
        added: added.len(),
        removed: removed.len(),
        changed: changed.len(),
        max_added_severity,
    };
    DiffReport {
        schema_version: 1,
        generated_at: Utc::now(),
        base: base_name,
        head: head_name,
        summary,
        added,
        removed,
        changed,
    }
}

fn finding_key(finding: &Finding) -> String {
    if !finding.id.is_empty() {
        finding.id.clone()
    } else {
        format!(
            "{}:{}:{}:{}",
            finding.tool, finding.path, finding.server, finding.rule
        )
    }
}
