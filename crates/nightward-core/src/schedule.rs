use crate::inventory::load_report_summary;
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
        let Ok(file_type) = entry.file_type() else {
            continue;
        };
        if !file_type.is_file() || file_type.is_symlink() {
            continue;
        }
        let Ok(meta) = entry.metadata() else {
            continue;
        };
        let Ok(modified) = meta.modified() else {
            continue;
        };
        let Ok(findings) = load_report_summary(&path) else {
            continue;
        };
        out.push(ReportHistoryEntry {
            report_name: entry.file_name().to_string_lossy().to_string(),
            path: path.display().to_string(),
            findings,
            mod_time: DateTime::<Utc>::from(modified),
        });
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn history_skips_oversized_reports_before_counting_findings() {
        let home = tempfile::tempdir().expect("temp home");
        let dir = report_dir(home.path());
        fs::create_dir_all(&dir).expect("report dir");
        fs::write(
            dir.join("small.json"),
            r#"{"summary":{"total_findings":3}}"#,
        )
        .expect("small report");
        let huge = fs::File::create(dir.join("huge.json")).expect("huge report");
        huge.set_len(17 * 1024 * 1024).expect("set len");

        let status = status(home.path());

        assert_eq!(status.history.len(), 1);
        assert_eq!(status.history[0].report_name, "small.json");
        assert_eq!(status.history[0].findings, 3);
    }

    #[cfg(unix)]
    #[test]
    fn history_skips_json_fifos() {
        use std::os::unix::fs::FileTypeExt;
        use std::process::Command;

        let home = tempfile::tempdir().expect("temp home");
        let dir = report_dir(home.path());
        fs::create_dir_all(&dir).expect("report dir");
        let fifo = dir.join("pipe.json");
        let mkfifo_status = Command::new("mkfifo").arg(&fifo).status().expect("mkfifo");
        assert!(mkfifo_status.success());
        assert!(fs::symlink_metadata(&fifo)
            .expect("fifo metadata")
            .file_type()
            .is_fifo());

        let status = status(home.path());

        assert!(status.history.is_empty());
    }
}
