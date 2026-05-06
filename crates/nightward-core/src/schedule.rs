use crate::inventory::load_report_summary;
use crate::state;
use anyhow::{anyhow, Context, Result};
use chrono::{DateTime, Utc};
use serde::Serialize;
use std::fs;
use std::path::{Path, PathBuf};
use std::process::Command;

#[derive(Debug, Clone, Serialize)]
pub struct ScheduleStatus {
    pub preset: String,
    pub installed: bool,
    pub platform: String,
    pub report_dir: String,
    pub log_dir: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_report: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_run: Option<String>,
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
    pub preset: String,
    pub platform: String,
    pub command: Vec<String>,
    pub writes: Vec<String>,
    pub notes: Vec<String>,
}

pub fn report_dir(home: impl AsRef<Path>) -> PathBuf {
    home.as_ref().join(".local/state/nightward/reports")
}

pub fn log_dir(home: impl AsRef<Path>) -> PathBuf {
    home.as_ref().join(".local/state/nightward/logs")
}

pub fn runner_path(home: impl AsRef<Path>) -> PathBuf {
    home.as_ref()
        .join(".local/state/nightward/bin/nightward-scheduled-scan")
}

pub fn status(home: impl AsRef<Path>) -> ScheduleStatus {
    let home = home.as_ref();
    let dir = report_dir(home);
    let mut history = history(&dir);
    history.sort_by(|a, b| b.mod_time.cmp(&a.mod_time));
    let last = history.first().cloned();
    ScheduleStatus {
        preset: "nightly".to_string(),
        installed: schedule_files(home).iter().any(|path| path.exists()),
        platform: std::env::consts::OS.to_string(),
        report_dir: dir.display().to_string(),
        log_dir: log_dir(home).display().to_string(),
        last_report: last.as_ref().map(|entry| entry.path.clone()),
        last_run: last.as_ref().map(|entry| entry.mod_time.to_rfc3339()),
        last_findings: last.as_ref().map(|entry| entry.findings),
        history,
    }
}

pub fn supports_install() -> bool {
    matches!(std::env::consts::OS, "macos" | "linux")
}

pub fn plan(home: impl AsRef<Path>, install: bool, executable: &str) -> SchedulePlan {
    let home = home.as_ref();
    let executable = if executable.trim().is_empty() {
        "nightward"
    } else {
        executable
    };
    SchedulePlan {
        schema_version: 1,
        mode: if install {
            "install-preview"
        } else {
            "remove-preview"
        }
        .to_string(),
        install,
        preset: "nightly".to_string(),
        platform: std::env::consts::OS.to_string(),
        command: vec![
            executable.to_string(),
            "scan".to_string(),
            "--json".to_string(),
        ],
        writes: schedule_files(home)
            .into_iter()
            .map(|path| path.display().to_string())
            .collect(),
        notes: vec![
            "Schedule actions install user-level jobs only; they do not install root daemons."
                .to_string(),
            "Scheduled scans write redacted JSON reports and logs under ~/.local/state/nightward."
                .to_string(),
        ],
    }
}

pub fn install(home: impl AsRef<Path>, executable: &str) -> Result<ScheduleStatus> {
    let home = home.as_ref();
    if !supports_install() {
        return Err(anyhow!(
            "schedule install is not implemented for {}",
            std::env::consts::OS
        ));
    }
    state::create_private_dir(&report_dir(home))?;
    state::create_private_dir(&log_dir(home))?;
    write_runner(home, executable)?;
    match std::env::consts::OS {
        "macos" => install_launchd(home)?,
        "linux" => install_systemd(home)?,
        _ => unreachable!(),
    }
    Ok(status(home))
}

pub fn remove(home: impl AsRef<Path>) -> Result<ScheduleStatus> {
    let home = home.as_ref();
    match std::env::consts::OS {
        "macos" => remove_launchd(home)?,
        "linux" => remove_systemd(home)?,
        _ => {}
    }
    for path in schedule_files(home) {
        if path.exists() {
            fs::remove_file(&path).with_context(|| format!("remove {}", path.display()))?;
        }
    }
    Ok(status(home))
}

fn schedule_files(home: &Path) -> Vec<PathBuf> {
    let mut files = vec![runner_path(home)];
    match std::env::consts::OS {
        "macos" => files.push(launchd_plist(home)),
        "linux" => {
            files.push(systemd_service(home));
            files.push(systemd_timer(home));
        }
        _ => {}
    }
    files
}

fn write_runner(home: &Path, executable: &str) -> Result<()> {
    let path = runner_path(home);
    if let Some(parent) = path.parent() {
        state::create_private_dir(parent)?;
    }
    let report_dir = report_dir(home);
    let log_dir = log_dir(home);
    let body = format!(
        "#!/bin/sh\nset -eu\nmkdir -p '{}' '{}'\nTS=$(date -u +%Y%m%dT%H%M%SZ)\nNIGHTWARD_HOME='{}' '{}' scan --json --output \"{}/scan-$TS.json\" >> '{}/nightward-scheduled-scan.log' 2>&1\n",
        shell_quote_path(&report_dir),
        shell_quote_path(&log_dir),
        shell_quote_path(home),
        shell_quote(executable),
        shell_quote_path(&report_dir),
        shell_quote_path(&log_dir)
    );
    state::write_private_file(&path, body).with_context(|| format!("write {}", path.display()))?;
    set_executable(&path)
}

#[cfg(unix)]
fn set_executable(path: &Path) -> Result<()> {
    use std::os::unix::fs::PermissionsExt;
    fs::set_permissions(path, fs::Permissions::from_mode(0o700))
        .with_context(|| format!("chmod 700 {}", path.display()))
}

#[cfg(not(unix))]
fn set_executable(_path: &Path) -> Result<()> {
    Ok(())
}

fn shell_quote_path(path: &Path) -> String {
    shell_quote(&path.display().to_string())
}

fn shell_quote(value: &str) -> String {
    value.replace('\'', "'\\''")
}

#[cfg(target_os = "macos")]
fn launchd_plist(home: &Path) -> PathBuf {
    home.join("Library/LaunchAgents/dev.aethereal.nightward.plist")
}

#[cfg(not(target_os = "macos"))]
fn launchd_plist(home: &Path) -> PathBuf {
    home.join("Library/LaunchAgents/dev.aethereal.nightward.plist")
}

fn install_launchd(home: &Path) -> Result<()> {
    let plist = launchd_plist(home);
    if let Some(parent) = plist.parent() {
        state::create_private_dir(parent)?;
    }
    let runner = runner_path(home);
    let log = log_dir(home).join("launchd.log");
    let err = log_dir(home).join("launchd.err.log");
    let body = format!(
        r#"<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>dev.aethereal.nightward</string>
  <key>ProgramArguments</key>
  <array><string>{}</string></array>
  <key>StartCalendarInterval</key>
  <dict><key>Hour</key><integer>2</integer><key>Minute</key><integer>0</integer></dict>
  <key>StandardOutPath</key><string>{}</string>
  <key>StandardErrorPath</key><string>{}</string>
</dict>
</plist>
"#,
        xml_escape(&runner.display().to_string()),
        xml_escape(&log.display().to_string()),
        xml_escape(&err.display().to_string())
    );
    state::write_private_file(&plist, body)
        .with_context(|| format!("write {}", plist.display()))?;
    let domain = launchd_domain()?;
    let _ = Command::new("launchctl")
        .args(["bootout", &domain, plist.to_string_lossy().as_ref()])
        .output();
    run_command(
        "launchctl",
        &["bootstrap", &domain, plist.to_string_lossy().as_ref()],
    )?;
    run_command(
        "launchctl",
        &["enable", &format!("{domain}/dev.aethereal.nightward")],
    )
}

fn remove_launchd(home: &Path) -> Result<()> {
    let plist = launchd_plist(home);
    let domain = launchd_domain()?;
    let _ = Command::new("launchctl")
        .args(["bootout", &domain, plist.to_string_lossy().as_ref()])
        .output();
    Ok(())
}

fn launchd_domain() -> Result<String> {
    let output = Command::new("id").arg("-u").output().context("run id -u")?;
    if !output.status.success() {
        return Err(anyhow!("id -u failed"));
    }
    Ok(format!(
        "gui/{}",
        String::from_utf8_lossy(&output.stdout).trim()
    ))
}

fn systemd_dir(home: &Path) -> PathBuf {
    home.join(".config/systemd/user")
}

fn systemd_service(home: &Path) -> PathBuf {
    systemd_dir(home).join("nightward-scheduled-scan.service")
}

fn systemd_timer(home: &Path) -> PathBuf {
    systemd_dir(home).join("nightward-scheduled-scan.timer")
}

fn install_systemd(home: &Path) -> Result<()> {
    state::create_private_dir(&systemd_dir(home))?;
    state::write_private_file(
        &systemd_service(home),
        format!(
            "[Unit]\nDescription=Nightward scheduled scan\n\n[Service]\nType=oneshot\nExecStart={}\n",
            runner_path(home).display()
        ),
    )?;
    state::write_private_file(
        &systemd_timer(home),
        "[Unit]\nDescription=Run Nightward scheduled scan nightly\n\n[Timer]\nOnCalendar=*-*-* 02:00:00\nPersistent=true\n\n[Install]\nWantedBy=timers.target\n",
    )?;
    run_command("systemctl", &["--user", "daemon-reload"])?;
    run_command(
        "systemctl",
        &[
            "--user",
            "enable",
            "--now",
            "nightward-scheduled-scan.timer",
        ],
    )
}

fn remove_systemd(_home: &Path) -> Result<()> {
    let _ = Command::new("systemctl")
        .args([
            "--user",
            "disable",
            "--now",
            "nightward-scheduled-scan.timer",
        ])
        .output();
    let _ = Command::new("systemctl")
        .args(["--user", "daemon-reload"])
        .output();
    Ok(())
}

fn run_command(program: &str, args: &[&str]) -> Result<()> {
    let output = Command::new(program)
        .args(args)
        .output()
        .with_context(|| format!("spawn {program}"))?;
    if output.status.success() {
        return Ok(());
    }
    Err(anyhow!(
        "{} {} failed: {}",
        program,
        args.join(" "),
        String::from_utf8_lossy(&output.stderr).trim()
    ))
}

fn xml_escape(value: &str) -> String {
    value
        .replace('&', "&amp;")
        .replace('<', "&lt;")
        .replace('>', "&gt;")
        .replace('"', "&quot;")
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
