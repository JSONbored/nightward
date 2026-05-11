use crate::inventory::redact_text;
use crate::{backupplan, policy, providers, schedule, state};
use anyhow::{anyhow, Context, Result};
use chrono::Utc;
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::{Component, Path, PathBuf};
use std::process::Command;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActionSpec {
    pub id: String,
    pub title: String,
    pub description: String,
    pub category: String,
    pub risk: String,
    pub available: bool,
    pub requires_confirmation: bool,
    pub requires_online: bool,
    pub reversible: bool,
    pub writes: Vec<String>,
    pub command: Vec<String>,
    pub blocked_reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActionPreview {
    pub schema_version: u32,
    pub action: ActionSpec,
    pub steps: Vec<String>,
    pub warnings: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActionResult {
    pub schema_version: u32,
    pub action_id: String,
    pub status: String,
    pub message: String,
    pub writes: Vec<String>,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub audit_path: String,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct ApplyOptions {
    pub confirm: bool,
    pub executable: String,
    pub policy_path: String,
    pub finding_id: String,
    pub rule: String,
    pub reason: String,
}

#[derive(Debug, Serialize)]
struct AuditEvent {
    schema_version: u32,
    generated_at: chrono::DateTime<Utc>,
    action_id: String,
    status: String,
    message: String,
    writes: Vec<String>,
}

pub fn list(home: impl AsRef<Path>) -> Vec<ActionSpec> {
    let home = home.as_ref();
    let settings = state::load_settings(home).unwrap_or_default();
    let disclosure_accepted = state::disclosure_status(home).accepted;
    let mut actions = Vec::new();
    if !disclosure_accepted {
        actions.push(ActionSpec {
            id: "disclosure.accept".to_string(),
            title: "Accept responsibility disclosure".to_string(),
            description: "Required before write-capable TUI actions run.".to_string(),
            category: "setup".to_string(),
            risk: "info".to_string(),
            available: true,
            requires_confirmation: true,
            requires_online: false,
            reversible: false,
            writes: vec![state::settings_path(home).display().to_string()],
            command: Vec::new(),
            blocked_reason: String::new(),
        });
    }

    let schedule_status = schedule::status(home);
    if schedule_status.installed {
        actions.push(ActionSpec {
            id: "schedule.remove".to_string(),
            title: "Disable scheduled scans".to_string(),
            description:
                "Remove the user-level Nightward scheduled scan job. Reports are left in place."
                    .to_string(),
            category: "schedule".to_string(),
            risk: "medium".to_string(),
            available: disclosure_accepted,
            requires_confirmation: true,
            requires_online: false,
            reversible: true,
            writes: schedule::plan(home, false, "").writes,
            command: schedule::plan(home, false, "").command,
            blocked_reason: disclosure_gate_reason(disclosure_accepted, ""),
        });
    } else {
        let schedule_available = schedule::supports_install();
        let schedule_blocked_reason = if schedule_available {
            String::new()
        } else {
            "schedule install is only implemented for macOS launchd and Linux systemd user timers"
                .to_string()
        };
        actions.push(ActionSpec {
            id: "schedule.install".to_string(),
            title: "Enable scheduled scans".to_string(),
            description: "Install a user-level nightly scan job that writes redacted reports under Nightward state.".to_string(),
            category: "schedule".to_string(),
            risk: "medium".to_string(),
            available: disclosure_accepted && schedule_available,
            requires_confirmation: true,
            requires_online: false,
            reversible: true,
            writes: schedule::plan(home, true, "").writes,
            command: schedule::plan(home, true, "").command,
            blocked_reason: disclosure_gate_reason(disclosure_accepted, schedule_blocked_reason),
        });
    }

    actions.push(ActionSpec {
        id: "backup.snapshot".to_string(),
        title: "Create portable config backup".to_string(),
        description: "Copy portable Nightward backup candidates into a timestamped local snapshot."
            .to_string(),
        category: "backup".to_string(),
        risk: "medium".to_string(),
        available: disclosure_accepted,
        requires_confirmation: true,
        requires_online: false,
        reversible: false,
        writes: vec![snapshot_root(home).display().to_string()],
        command: Vec::new(),
        blocked_reason: disclosure_gate_reason(disclosure_accepted, ""),
    });

    actions.push(ActionSpec {
        id: "reports.cleanup".to_string(),
        title: "Clean saved reports and logs".to_string(),
        description:
            "Remove Nightward-owned scheduled report and log files without touching schedules."
                .to_string(),
        category: "cleanup".to_string(),
        risk: "medium".to_string(),
        available: disclosure_accepted,
        requires_confirmation: true,
        requires_online: false,
        reversible: false,
        writes: report_cleanup_targets(home)
            .into_iter()
            .map(|path| path.display().to_string())
            .collect(),
        command: Vec::new(),
        blocked_reason: disclosure_gate_reason(disclosure_accepted, ""),
    });

    actions.push(ActionSpec {
        id: "cache.cleanup".to_string(),
        title: "Clean Nightward caches".to_string(),
        description:
            "Remove Nightward-owned cache directories while leaving reports and audit logs."
                .to_string(),
        category: "cleanup".to_string(),
        risk: "medium".to_string(),
        available: disclosure_accepted,
        requires_confirmation: true,
        requires_online: false,
        reversible: false,
        writes: cache_cleanup_targets(home)
            .into_iter()
            .map(|path| path.display().to_string())
            .collect(),
        command: Vec::new(),
        blocked_reason: disclosure_gate_reason(disclosure_accepted, ""),
    });

    let default_policy = default_policy_path(home);
    actions.push(ActionSpec {
        id: "policy.init".to_string(),
        title: "Initialize Nightward policy".to_string(),
        description: "Write a default Nightward policy file if it does not already exist."
            .to_string(),
        category: "policy".to_string(),
        risk: "low".to_string(),
        available: disclosure_accepted,
        requires_confirmation: true,
        requires_online: false,
        reversible: false,
        writes: vec![default_policy.display().to_string()],
        command: Vec::new(),
        blocked_reason: disclosure_gate_reason(disclosure_accepted, ""),
    });

    actions.push(ActionSpec {
        id: "policy.ignore".to_string(),
        title: "Add policy ignore with reason".to_string(),
        description:
            "Append one reviewed finding or rule ignore to a bounded Nightward policy file."
                .to_string(),
        category: "policy".to_string(),
        risk: "medium".to_string(),
        available: disclosure_accepted,
        requires_confirmation: true,
        requires_online: false,
        reversible: true,
        writes: vec![default_policy.display().to_string()],
        command: Vec::new(),
        blocked_reason: disclosure_gate_reason(disclosure_accepted, ""),
    });

    actions.push(ActionSpec {
        id: if settings.allow_online_providers {
            "providers.online.disable".to_string()
        } else {
            "providers.online.enable".to_string()
        },
        title: if settings.allow_online_providers {
            "Block online-capable providers by default".to_string()
        } else {
            "Allow online-capable providers".to_string()
        },
        description: "Toggle whether configured online-capable providers may run without passing --online each time.".to_string(),
        category: "providers".to_string(),
        risk: "high".to_string(),
        available: disclosure_accepted,
        requires_confirmation: true,
        requires_online: false,
        reversible: true,
        writes: vec![state::settings_path(home).display().to_string()],
        command: Vec::new(),
        blocked_reason: disclosure_gate_reason(disclosure_accepted, ""),
    });

    for provider in providers::providers()
        .into_iter()
        .filter(|provider| !provider.default)
    {
        let selected = settings.selected_providers.contains(&provider.name);
        actions.push(ActionSpec {
            id: format!(
                "provider.{}.{}",
                if selected { "disable" } else { "enable" },
                provider.name
            ),
            title: format!(
                "{} {}",
                if selected { "Disable" } else { "Enable" },
                provider.name
            ),
            description: format!(
                "{} for Nightward analysis runs.",
                if selected {
                    "Remove this provider from the default selected set"
                } else {
                    "Add this provider to the default selected set"
                }
            ),
            category: "providers".to_string(),
            risk: if provider.online { "high" } else { "medium" }.to_string(),
            available: disclosure_accepted,
            requires_confirmation: true,
            requires_online: false,
            reversible: true,
            writes: vec![state::settings_path(home).display().to_string()],
            command: Vec::new(),
            blocked_reason: disclosure_gate_reason(disclosure_accepted, ""),
        });
        let provider_available = which::which(&provider.name).is_ok();
        if !provider_available {
            if let Some(install) = providers::install_command(&provider.name) {
                let package_manager_available = which::which(&install.program).is_ok();
                let package_manager_blocked_reason = if package_manager_available {
                    String::new()
                } else {
                    format!("{} is not available on PATH", install.program)
                };
                actions.push(ActionSpec {
                    id: format!("provider.install.{}", provider.name),
                    title: format!("Install {}", provider.name),
                    description: format!(
                        "Install the {} provider CLI using a known package-manager command.",
                        provider.name
                    ),
                    category: "providers".to_string(),
                    risk: "high".to_string(),
                    available: disclosure_accepted && package_manager_available,
                    requires_confirmation: true,
                    requires_online: true,
                    reversible: false,
                    writes: vec![format!("package manager state via {}", install.program)],
                    command: install.command(),
                    blocked_reason: disclosure_gate_reason(
                        disclosure_accepted,
                        package_manager_blocked_reason,
                    ),
                });
            }
        }
    }
    actions
}

fn disclosure_gate_reason(accepted: bool, fallback: impl Into<String>) -> String {
    if accepted {
        fallback.into()
    } else {
        "accept the Nightward beta responsibility disclosure before applying write-capable actions"
            .to_string()
    }
}

pub fn preview(home: impl AsRef<Path>, id: &str) -> Result<ActionPreview> {
    let action = find_action(home, id)?;
    let mut warnings = Vec::new();
    if action.requires_online {
        warnings.push(
            "This action can use the network through a package manager or third-party provider."
                .to_string(),
        );
    }
    if action.risk == "high" {
        warnings.push(
            "Review the command, provider behavior, and rollback path before applying.".to_string(),
        );
    }
    Ok(ActionPreview {
        schema_version: 1,
        action,
        steps: preview_steps(id),
        warnings,
    })
}

pub fn apply(home: impl AsRef<Path>, id: &str, options: ApplyOptions) -> Result<ActionResult> {
    let home = home.as_ref();
    let action = find_action(home, id)?;
    if !action.available {
        return Err(anyhow!("{}", action.blocked_reason));
    }
    if action.requires_confirmation && !options.confirm {
        return Err(anyhow!("refusing to apply {id} without --confirm"));
    }
    if id != "disclosure.accept" && !state::disclosure_status(home).accepted {
        return Err(anyhow!(
            "accept the Nightward beta responsibility disclosure before applying write-capable actions"
        ));
    }

    let mut writes = action.writes.clone();
    let message = match id {
        "disclosure.accept" => {
            state::accept_disclosure(home)?;
            "responsibility disclosure accepted".to_string()
        }
        "schedule.install" => {
            let executable = if options.executable.trim().is_empty() {
                "nightward".to_string()
            } else {
                options.executable
            };
            let status = schedule::install(home, &executable)?;
            writes = schedule::plan(home, true, &executable).writes;
            format!("scheduled scans enabled for {}", status.platform)
        }
        "schedule.remove" => {
            schedule::remove(home)?;
            writes = schedule::plan(home, false, "").writes;
            "scheduled scans disabled".to_string()
        }
        "backup.snapshot" => {
            let result = create_backup_snapshot(home)?;
            writes = result.writes;
            result.message
        }
        "reports.cleanup" => {
            let result = cleanup_owned_dirs(&report_cleanup_targets(home), "report/log")?;
            writes = result.writes;
            result.message
        }
        "cache.cleanup" => {
            let result = cleanup_owned_dirs(&cache_cleanup_targets(home), "cache")?;
            writes = result.writes;
            result.message
        }
        "policy.init" => {
            let path = bounded_policy_path(home, &options.policy_path)?;
            init_policy_file(&path)?;
            writes = vec![path.display().to_string()];
            format!("policy initialized at {}", path.display())
        }
        "policy.ignore" => {
            let result = add_policy_ignore(home, &options)?;
            writes = result.writes;
            result.message
        }
        "providers.online.enable" => {
            state::set_online_providers_allowed(home, true)?;
            "online-capable providers are allowed for configured default runs".to_string()
        }
        "providers.online.disable" => {
            state::set_online_providers_allowed(home, false)?;
            "online-capable providers are blocked by default".to_string()
        }
        value if value.starts_with("provider.enable.") => {
            let provider = value.trim_start_matches("provider.enable.");
            state::set_provider_selected(home, provider, true)?;
            format!("{provider} enabled for default analysis runs")
        }
        value if value.starts_with("provider.disable.") => {
            let provider = value.trim_start_matches("provider.disable.");
            state::set_provider_selected(home, provider, false)?;
            format!("{provider} disabled for default analysis runs")
        }
        value if value.starts_with("provider.install.") => {
            let provider = value.trim_start_matches("provider.install.");
            install_provider(provider)?
        }
        _ => return Err(anyhow!("unknown action {id}")),
    };

    let audit = AuditEvent {
        schema_version: 1,
        generated_at: Utc::now(),
        action_id: id.to_string(),
        status: "applied".to_string(),
        message: message.clone(),
        writes: writes.clone(),
    };
    let audit_path = state::append_audit(home, &audit)?;
    Ok(ActionResult {
        schema_version: 1,
        action_id: id.to_string(),
        status: "applied".to_string(),
        message,
        writes,
        audit_path: audit_path.display().to_string(),
    })
}

fn find_action(home: impl AsRef<Path>, id: &str) -> Result<ActionSpec> {
    list(home)
        .into_iter()
        .find(|action| action.id == id)
        .ok_or_else(|| anyhow!("unknown action {id}"))
}

fn preview_steps(id: &str) -> Vec<String> {
    match id {
        "schedule.install" => vec![
            "Create Nightward report and log directories.".to_string(),
            "Write a local scheduled-scan runner script.".to_string(),
            "Install a user-level launchd agent or systemd user timer.".to_string(),
        ],
        "schedule.remove" => vec![
            "Disable the user-level scheduled scan job.".to_string(),
            "Remove Nightward schedule files.".to_string(),
            "Leave existing reports and audit logs in place.".to_string(),
        ],
        "backup.snapshot" => vec![
            "Build the backup plan from portable candidates.".to_string(),
            "Copy existing portable files into a timestamped snapshot.".to_string(),
            "Write a manifest describing copied and skipped paths.".to_string(),
        ],
        "reports.cleanup" => vec![
            "Inspect Nightward-owned scheduled report and log directories.".to_string(),
            "Remove existing files and child directories inside those directories.".to_string(),
            "Leave schedule, settings, policy, backup snapshots, and audit logs in place."
                .to_string(),
        ],
        "cache.cleanup" => vec![
            "Inspect Nightward-owned cache directories.".to_string(),
            "Remove existing files and child directories inside those cache directories."
                .to_string(),
            "Leave reports, schedules, settings, policy, snapshots, and audit logs in place."
                .to_string(),
        ],
        "policy.init" => vec![
            "Resolve the bounded policy path under NIGHTWARD_HOME.".to_string(),
            "Write the default policy file only if it is missing.".to_string(),
        ],
        "policy.ignore" => vec![
            "Resolve the bounded policy path under NIGHTWARD_HOME.".to_string(),
            "Require a finding ID or rule plus a non-empty reason.".to_string(),
            "Append the ignore entry and preserve the rest of the policy.".to_string(),
        ],
        value if value.starts_with("provider.install.") => vec![
            "Run the displayed package-manager command.".to_string(),
            "Refresh provider doctor status after installation.".to_string(),
        ],
        value if value.starts_with("provider.") => vec![
            "Update Nightward local settings.".to_string(),
            "Use the setting for future analysis runs when --with is omitted.".to_string(),
        ],
        _ => vec!["Apply the selected local Nightward action.".to_string()],
    }
}

fn install_provider(provider: &str) -> Result<String> {
    let install = providers::install_command(provider)
        .ok_or_else(|| anyhow!("no install command is known for {provider}"))?;
    let output = Command::new(&install.program)
        .args(&install.args)
        .output()
        .with_context(|| format!("spawn {}", install.program))?;
    if !output.status.success() {
        return Err(anyhow!(
            "{} failed: {}",
            install.command().join(" "),
            redact_text(&String::from_utf8_lossy(&output.stderr))
        ));
    }
    let stdout = redact_text(&String::from_utf8_lossy(&output.stdout));
    Ok(if stdout.trim().is_empty() {
        format!("{provider} installed")
    } else {
        format!(
            "{provider} installed: {}",
            stdout.lines().next().unwrap_or("").trim()
        )
    })
}

#[derive(Debug)]
struct SnapshotResult {
    message: String,
    writes: Vec<String>,
}

fn snapshot_root(home: &Path) -> PathBuf {
    state::state_dir(home).join("snapshots")
}

fn report_cleanup_targets(home: &Path) -> Vec<PathBuf> {
    vec![schedule::report_dir(home), schedule::log_dir(home)]
}

fn cache_cleanup_targets(home: &Path) -> Vec<PathBuf> {
    vec![state::state_dir(home).join("cache"), state::cache_dir(home)]
}

fn default_policy_path(home: &Path) -> PathBuf {
    state::config_dir(home).join("nightward-policy.yml")
}

fn bounded_policy_path(home: &Path, requested: &str) -> Result<PathBuf> {
    let requested = requested.trim();
    if requested.is_empty() {
        return Ok(default_policy_path(home));
    }
    let path = Path::new(requested);
    if path.is_absolute() {
        return Err(anyhow!("policy path must be relative to NIGHTWARD_HOME"));
    }
    let parts = normal_relative_components(path).with_context(|| {
        format!("policy path must be a clean relative path under NIGHTWARD_HOME: {requested}")
    })?;
    if parts.is_empty() {
        return Err(anyhow!("policy path cannot be empty"));
    }
    let file_name = path
        .file_name()
        .and_then(|name| name.to_str())
        .unwrap_or_default();
    let extension = path.extension().and_then(|ext| ext.to_str()).unwrap_or("");
    if !matches!(extension, "yml" | "yaml") {
        return Err(anyhow!("policy path must end in .yml or .yaml"));
    }
    let in_nightward_config = parts.len() >= 3 && parts[0] == ".config" && parts[1] == "nightward";
    let in_project_policy_dir = parts.iter().any(|part| part == ".nightward");
    let named_policy_file = matches!(
        file_name,
        "nightward-policy.yml" | "nightward-policy.yaml" | ".nightward.yml" | ".nightward.yaml"
    );
    if !(in_nightward_config || in_project_policy_dir || named_policy_file) {
        return Err(anyhow!(
            "policy path must be a Nightward policy file or live under .nightward/"
        ));
    }
    Ok(home.join(path))
}

fn add_policy_ignore(home: &Path, options: &ApplyOptions) -> Result<SnapshotResult> {
    let path = bounded_policy_path(home, &options.policy_path)?;
    let reason = options.reason.trim();
    if reason.is_empty() {
        return Err(anyhow!("policy.ignore requires a non-empty reason"));
    }
    let finding_id = options.finding_id.trim();
    let rule = options.rule.trim();
    if finding_id.is_empty() && rule.is_empty() {
        return Err(anyhow!("policy.ignore requires finding_id or rule"));
    }
    if !path.exists() {
        init_policy_file(&path)?;
    }
    state::ensure_regular_file_or_missing(&path)?;
    let mut config = policy::load(&path)?;
    if !finding_id.is_empty() {
        config
            .ignore_findings
            .retain(|entry| entry.id != finding_id);
        config.ignore_findings.push(policy::IgnoreFindingEntry {
            id: finding_id.to_string(),
            reason: reason.to_string(),
        });
    } else {
        config
            .ignore_rules
            .retain(|entry| entry.rule != rule && entry.id != rule);
        config.ignore_rules.push(policy::IgnoreRuleEntry {
            rule: rule.to_string(),
            id: String::new(),
            reason: reason.to_string(),
        });
    }
    state::write_private_file(&path, serde_yaml::to_string(&config)?)
        .with_context(|| format!("write {}", path.display()))?;
    Ok(SnapshotResult {
        message: format!("policy ignore added at {}", path.display()),
        writes: vec![path.display().to_string()],
    })
}

fn init_policy_file(path: &Path) -> Result<()> {
    state::ensure_regular_file_or_missing(path)?;
    if path.exists() {
        return Ok(());
    }
    state::write_private_file(path, policy::DEFAULT_POLICY)
}

fn normal_relative_components(path: &Path) -> Result<Vec<String>> {
    let mut parts = Vec::new();
    for component in path.components() {
        match component {
            Component::Normal(part) => parts.push(part.to_string_lossy().to_string()),
            Component::ParentDir => {
                return Err(anyhow!("path cannot contain parent directory components"))
            }
            Component::CurDir => {
                return Err(anyhow!("path cannot contain current directory components"))
            }
            Component::RootDir | Component::Prefix(_) => {
                return Err(anyhow!("path must be relative"))
            }
        }
    }
    Ok(parts)
}

fn safe_snapshot_relative_path(rel: &str) -> Result<PathBuf> {
    let rel = rel.trim();
    if rel.is_empty() {
        return Err(anyhow!("snapshot path cannot be empty"));
    }
    let path = Path::new(rel);
    let parts = normal_relative_components(path)?;
    if parts.is_empty() {
        return Err(anyhow!("snapshot path cannot be empty"));
    }
    let mut scoped = PathBuf::new();
    for part in parts {
        scoped.push(part);
    }
    Ok(scoped)
}

fn cleanup_owned_dirs(targets: &[PathBuf], label: &str) -> Result<SnapshotResult> {
    let mut removed = 0usize;
    for target in targets {
        removed += cleanup_owned_dir(target)?;
    }
    Ok(SnapshotResult {
        message: format!("removed {removed} Nightward-owned {label} entries"),
        writes: targets
            .iter()
            .map(|path| path.display().to_string())
            .collect(),
    })
}

fn cleanup_owned_dir(target: &Path) -> Result<usize> {
    if !target.exists() {
        return Ok(0);
    }
    let metadata =
        fs::symlink_metadata(target).with_context(|| format!("inspect {}", target.display()))?;
    if metadata.file_type().is_symlink() {
        return Err(anyhow!(
            "refusing to clean symlinked Nightward directory {}",
            target.display()
        ));
    }
    if !metadata.is_dir() {
        return Err(anyhow!(
            "refusing to clean non-directory Nightward path {}",
            target.display()
        ));
    }
    let mut removed = 0usize;
    for entry in fs::read_dir(target).with_context(|| format!("read {}", target.display()))? {
        let entry = entry.with_context(|| format!("read entry in {}", target.display()))?;
        let child = entry.path();
        let file_type = entry
            .file_type()
            .with_context(|| format!("inspect {}", child.display()))?;
        if file_type.is_dir() {
            fs::remove_dir_all(&child).with_context(|| format!("remove {}", child.display()))?;
            removed += 1;
        } else if file_type.is_file() || file_type.is_symlink() {
            fs::remove_file(&child).with_context(|| format!("remove {}", child.display()))?;
            removed += 1;
        }
    }
    Ok(removed)
}

fn create_backup_snapshot(home: &Path) -> Result<SnapshotResult> {
    let plan = backupplan::plan(home);
    let snapshot = snapshot_root(home).join(Utc::now().format("%Y%m%dT%H%M%SZ").to_string());
    state::create_private_dir(&snapshot)?;
    let mut copied = Vec::new();
    let mut skipped = Vec::new();
    for rel in &plan.include {
        let rel_path = match safe_snapshot_relative_path(rel) {
            Ok(path) => path,
            Err(error) => {
                skipped.push(format!("{rel}: {}", error));
                continue;
            }
        };
        let source = home.join(&rel_path);
        let metadata = match fs::symlink_metadata(&source) {
            Ok(metadata) => metadata,
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
                skipped.push(format!("{rel}: missing"));
                continue;
            }
            Err(error) => {
                return Err(error).with_context(|| format!("inspect {}", source.display()))
            }
        };
        if metadata.file_type().is_symlink() {
            skipped.push(format!("{rel}: symlink"));
            continue;
        }
        if !metadata.is_file() {
            skipped.push(format!("{rel}: not a regular file"));
            continue;
        }
        let destination = snapshot.join(&rel_path);
        if let Some(parent) = destination.parent() {
            state::create_private_dir(parent)?;
        }
        fs::copy(&source, &destination)
            .with_context(|| format!("copy {} to {}", source.display(), destination.display()))?;
        state::set_private_file_permissions(&destination)?;
        copied.push(rel.clone());
    }
    let manifest = serde_json::json!({
        "schema_version": 1,
        "generated_at": Utc::now(),
        "root": plan.root,
        "snapshot": snapshot.display().to_string(),
        "copied": copied,
        "skipped": skipped,
        "excluded": plan.exclude,
        "notes": plan.notes,
    });
    let manifest_path = snapshot.join("manifest.json");
    state::write_private_file(
        &manifest_path,
        format!("{}\n", serde_json::to_string_pretty(&manifest)?),
    )
    .with_context(|| format!("write {}", manifest_path.display()))?;
    state::set_private_file_permissions(&manifest_path)?;
    Ok(SnapshotResult {
        message: format!("backup snapshot created at {}", snapshot.display()),
        writes: vec![snapshot.display().to_string()],
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn backup_snapshot_action_copies_portable_candidates_and_audits() {
        let home = tempfile::tempdir().expect("temp home");
        let codex = home.path().join(".codex");
        std::fs::create_dir_all(&codex).expect("codex dir");
        std::fs::write(codex.join("config.toml"), "model = \"test\"\n").expect("config");
        apply(
            home.path(),
            "disclosure.accept",
            ApplyOptions {
                confirm: true,
                executable: "nightward".to_string(),
                ..Default::default()
            },
        )
        .expect("accept disclosure");

        let result = apply(
            home.path(),
            "backup.snapshot",
            ApplyOptions {
                confirm: true,
                executable: "nightward".to_string(),
                ..Default::default()
            },
        )
        .expect("snapshot action");

        assert_eq!(result.status, "applied");
        let snapshot = PathBuf::from(result.writes[0].clone());
        assert!(snapshot.join(".codex/config.toml").is_file());
        assert!(snapshot.join("manifest.json").is_file());
        assert!(state::audit_path(home.path()).is_file());
    }

    #[cfg(unix)]
    #[test]
    fn backup_snapshot_skips_symlinked_candidates_without_copying_targets() {
        use std::os::unix::fs::symlink;

        let home = tempfile::tempdir().expect("temp home");
        let outside = tempfile::tempdir().expect("outside");
        let secret = outside.path().join("secret.toml");
        std::fs::write(&secret, "token = \"SECRET_VALUE\"\n").expect("secret");
        let codex = home.path().join(".codex");
        std::fs::create_dir_all(&codex).expect("codex dir");
        symlink(&secret, codex.join("config.toml")).expect("symlink config");
        apply(
            home.path(),
            "disclosure.accept",
            ApplyOptions {
                confirm: true,
                executable: "nightward".to_string(),
                ..Default::default()
            },
        )
        .expect("accept disclosure");

        let result = apply(
            home.path(),
            "backup.snapshot",
            ApplyOptions {
                confirm: true,
                executable: "nightward".to_string(),
                ..Default::default()
            },
        )
        .expect("snapshot action");

        let snapshot = PathBuf::from(result.writes[0].clone());
        assert!(!snapshot.join(".codex/config.toml").exists());
        let manifest = std::fs::read_to_string(snapshot.join("manifest.json")).expect("manifest");
        assert!(manifest.contains(".codex/config.toml: symlink"));
        assert!(!manifest.contains("SECRET_VALUE"));
    }

    #[cfg(unix)]
    #[test]
    fn disclosure_accept_rejects_symlinked_nightward_settings_dir() {
        use std::os::unix::fs::symlink;

        let home = tempfile::tempdir().expect("temp home");
        let outside = tempfile::tempdir().expect("outside");
        std::fs::create_dir_all(home.path().join(".config")).expect("config dir");
        symlink(outside.path(), home.path().join(".config/nightward")).expect("settings symlink");

        let error = apply(
            home.path(),
            "disclosure.accept",
            ApplyOptions {
                confirm: true,
                executable: "nightward".to_string(),
                ..Default::default()
            },
        )
        .expect_err("symlinked settings dir rejected");

        assert!(error.to_string().contains("symlinked Nightward directory"));
    }

    #[test]
    fn apply_refuses_confirmation_gated_actions_without_confirm() {
        let home = tempfile::tempdir().expect("temp home");
        apply(
            home.path(),
            "disclosure.accept",
            ApplyOptions {
                confirm: true,
                executable: "nightward".to_string(),
                ..Default::default()
            },
        )
        .expect("accept disclosure");

        let error = apply(
            home.path(),
            "backup.snapshot",
            ApplyOptions {
                confirm: false,
                executable: "nightward".to_string(),
                ..Default::default()
            },
        )
        .expect_err("confirmation required");

        assert!(error.to_string().contains("without --confirm"));
    }

    #[test]
    fn policy_actions_initialize_and_append_reasoned_ignores() {
        let home = tempfile::tempdir().expect("temp home");
        apply(
            home.path(),
            "disclosure.accept",
            ApplyOptions {
                confirm: true,
                executable: "nightward".to_string(),
                ..Default::default()
            },
        )
        .expect("accept disclosure");

        let init = apply(
            home.path(),
            "policy.init",
            ApplyOptions {
                confirm: true,
                policy_path: "project/.nightward.yml".to_string(),
                ..Default::default()
            },
        )
        .expect("policy init");
        let path = PathBuf::from(&init.writes[0]);
        assert!(path.is_file());

        apply(
            home.path(),
            "policy.ignore",
            ApplyOptions {
                confirm: true,
                policy_path: "project/.nightward.yml".to_string(),
                finding_id: "finding-123".to_string(),
                reason: "reviewed local-only fixture".to_string(),
                ..Default::default()
            },
        )
        .expect("policy ignore");

        let config = policy::load(&path).expect("policy config");
        assert!(config.ignore_findings.iter().any(
            |entry| entry.id == "finding-123" && entry.reason == "reviewed local-only fixture"
        ));

        let error = apply(
            home.path(),
            "policy.ignore",
            ApplyOptions {
                confirm: true,
                finding_id: "finding-456".to_string(),
                ..Default::default()
            },
        )
        .expect_err("reason required");
        assert!(error.to_string().contains("non-empty reason"));

        let error = apply(
            home.path(),
            "policy.init",
            ApplyOptions {
                confirm: true,
                policy_path: "../outside.yml".to_string(),
                ..Default::default()
            },
        )
        .expect_err("bounded path");
        assert!(error.to_string().contains("clean relative path"));

        let error = apply(
            home.path(),
            "policy.init",
            ApplyOptions {
                confirm: true,
                policy_path: "project/not-nightward-policy.yml".to_string(),
                ..Default::default()
            },
        )
        .expect_err("policy file name bounded");
        assert!(error.to_string().contains("Nightward policy file"));
    }

    #[cfg(unix)]
    #[test]
    fn policy_actions_reject_symlinked_policy_files() {
        use std::os::unix::fs::symlink;

        let home = tempfile::tempdir().expect("temp home");
        apply(
            home.path(),
            "disclosure.accept",
            ApplyOptions {
                confirm: true,
                executable: "nightward".to_string(),
                ..Default::default()
            },
        )
        .expect("accept disclosure");
        let outside = tempfile::tempdir().expect("outside");
        let target = outside.path().join("policy.yml");
        std::fs::write(&target, "severity_threshold: low\n").expect("target policy");
        let project = home.path().join("project");
        std::fs::create_dir_all(&project).expect("project dir");
        symlink(&target, project.join(".nightward.yml")).expect("policy symlink");

        let error = apply(
            home.path(),
            "policy.ignore",
            ApplyOptions {
                confirm: true,
                policy_path: "project/.nightward.yml".to_string(),
                finding_id: "finding-123".to_string(),
                reason: "reviewed".to_string(),
                ..Default::default()
            },
        )
        .expect_err("symlinked policy rejected");

        assert!(error.to_string().contains("symlinked Nightward path"));
    }

    #[test]
    fn cleanup_actions_remove_only_owned_report_log_and_cache_entries() {
        let home = tempfile::tempdir().expect("temp home");
        apply(
            home.path(),
            "disclosure.accept",
            ApplyOptions {
                confirm: true,
                executable: "nightward".to_string(),
                ..Default::default()
            },
        )
        .expect("accept disclosure");

        let report_dir = schedule::report_dir(home.path());
        let log_dir = schedule::log_dir(home.path());
        let state_cache_dir = state::state_dir(home.path()).join("cache");
        let user_cache_dir = state::cache_dir(home.path());
        std::fs::create_dir_all(&report_dir).expect("report dir");
        std::fs::create_dir_all(&log_dir).expect("log dir");
        std::fs::create_dir_all(state_cache_dir.join("nested")).expect("state cache dir");
        std::fs::create_dir_all(&user_cache_dir).expect("user cache dir");
        std::fs::write(report_dir.join("scan.json"), "{}\n").expect("report");
        std::fs::write(log_dir.join("nightward.log"), "ok\n").expect("log");
        std::fs::write(state_cache_dir.join("nested/cache.bin"), "cache\n").expect("state cache");
        std::fs::write(user_cache_dir.join("cache.bin"), "cache\n").expect("user cache");

        let reports = apply(
            home.path(),
            "reports.cleanup",
            ApplyOptions {
                confirm: true,
                ..Default::default()
            },
        )
        .expect("reports cleanup");
        assert_eq!(reports.status, "applied");
        assert!(report_dir.is_dir());
        assert!(log_dir.is_dir());
        assert!(!report_dir.join("scan.json").exists());
        assert!(!log_dir.join("nightward.log").exists());
        assert!(state_cache_dir.join("nested/cache.bin").exists());

        let cache = apply(
            home.path(),
            "cache.cleanup",
            ApplyOptions {
                confirm: true,
                ..Default::default()
            },
        )
        .expect("cache cleanup");
        assert_eq!(cache.status, "applied");
        assert!(state_cache_dir.is_dir());
        assert!(user_cache_dir.is_dir());
        assert!(!state_cache_dir.join("nested").exists());
        assert!(!user_cache_dir.join("cache.bin").exists());
        assert!(state::audit_path(home.path()).is_file());
    }
}
