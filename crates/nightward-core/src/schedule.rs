use chrono::{DateTime, Utc};
use serde::Serialize;
use std::fs;
use std::path::{Path, PathBuf};

#[derive(Debug, Clone, Serialize)]
pub struct ScheduleStatus {
    pub installed: bool,
    pub platform: String,
    pub report_dir: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_report: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_findings: Option<usize>,
    pub history: Vec<ReportHistoryEntry>,
}

#[derive(Debug, Clone, Serialize)]
pub struct ReportHistoryEntry {
    pub report_name: String,
    pub path: String,
    pub findings: usize,
    pub mod_time: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize)]
pub struct SchedulePlan {
    pub schema_version: u32,
    pub mode: String,
    pub install: bool,
    pub platform: String,
    pub command: String,
    pub notes: Vec<String>,
}

pub fn report_dir(home: impl AsRef<Path>) -> PathBuf {
    home.as_ref().join(".local/state/nightward/reports")
}

pub fn status(home: impl AsRef<Path>) -> ScheduleStatus {
    let dir = report_dir(home);
    let mut history = history(&dir);
    history.sort_by(|a, b| b.mod_time.cmp(&a.mod_time));
    let last = history.first().cloned();
    ScheduleStatus {
        installed: false,
        platform: std::env::consts::OS.to_string(),
        report_dir: dir.display().to_string(),
        last_report: last.as_ref().map(|entry| entry.path.clone()),
        last_findings: last.as_ref().map(|entry| entry.findings),
        history,
    }
}

pub fn plan(install: bool) -> SchedulePlan {
    SchedulePlan {
        schema_version: 1,
        mode: "plan-only".to_string(),
        install,
        platform: std::env::consts::OS.to_string(),
        command: "nightward scan --json".to_string(),
        notes: vec![
            "Schedule changes are explicit command paths, not automatic mutation.".to_string(),
            "Review platform-specific launchd/systemd/cron output before installing.".to_string(),
        ],
    }
}

fn history(dir: &Path) -> Vec<ReportHistoryEntry> {
    let mut out = Vec::new();
    let Ok(entries) = fs::read_dir(dir) else {
        return out;
    };
    for entry in entries.flatten() {
        let path = entry.path();
        if path.extension().and_then(|ext| ext.to_str()) != Some("json") {
            continue;
        }
        let Ok(meta) = entry.metadata() else {
            continue;
        };
        let Ok(modified) = meta.modified() else {
            continue;
        };
        let findings = fs::read_to_string(&path)
            .ok()
            .and_then(|text| serde_json::from_str::<serde_json::Value>(&text).ok())
            .and_then(|value| {
                value
                    .get("summary")
                    .and_then(|summary| summary.get("total_findings"))
                    .and_then(serde_json::Value::as_u64)
            })
            .unwrap_or(0) as usize;
        out.push(ReportHistoryEntry {
            report_name: entry.file_name().to_string_lossy().to_string(),
            path: path.display().to_string(),
            findings,
            mod_time: DateTime::<Utc>::from(modified),
        });
    }
    out
}
