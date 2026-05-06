use crate::actions::{self, ActionPreview, ActionResult, ApplyOptions};
use crate::{inventory::redact_text, state};
use anyhow::{anyhow, Context, Result};
use chrono::{DateTime, Duration, Utc};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::fs;
use std::path::{Path, PathBuf};

const APPROVAL_SCHEMA_VERSION: u32 = 1;
const DEFAULT_TTL_SECONDS: i64 = 15 * 60;
const MAX_APPROVAL_FILES: usize = 64;

#[derive(Debug, Clone, Default, Serialize, Deserialize, PartialEq, Eq)]
pub struct ApprovalActionOptions {
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub executable: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub policy_path: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub finding_id: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub rule: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub reason: String,
}

impl ApprovalActionOptions {
    pub fn from_apply_options(options: ApplyOptions) -> Self {
        Self {
            executable: options.executable,
            policy_path: options.policy_path,
            finding_id: options.finding_id,
            rule: options.rule,
            reason: options.reason,
        }
    }

    fn into_apply_options(self) -> ApplyOptions {
        ApplyOptions {
            confirm: true,
            executable: self.executable,
            policy_path: self.policy_path,
            finding_id: self.finding_id,
            rule: self.rule,
            reason: self.reason,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApprovalRequestOptions {
    pub action_id: String,
    #[serde(default)]
    pub action_options: ApprovalActionOptions,
    #[serde(default = "default_requested_by")]
    pub requested_by: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActionApproval {
    pub schema_version: u32,
    pub approval_id: String,
    pub status: String,
    pub action_id: String,
    #[serde(default)]
    pub action_options: ApprovalActionOptions,
    pub preview_digest: String,
    pub preview: ActionPreview,
    pub requested_by: String,
    pub requested_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub decision_reason: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub decided_at: Option<DateTime<Utc>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub applied_at: Option<DateTime<Utc>>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub result_message: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub action_audit_path: String,
}

#[derive(Debug, Clone, Serialize)]
pub struct ApprovalList {
    pub schema_version: u32,
    pub approvals: Vec<ActionApproval>,
}

#[derive(Debug, Clone, Serialize)]
pub struct ApprovedActionResult {
    pub schema_version: u32,
    pub approval: ActionApproval,
    pub action_result: ActionResult,
}

#[derive(Debug, Serialize)]
struct ApprovalAuditEvent {
    schema_version: u32,
    generated_at: DateTime<Utc>,
    event: String,
    approval_id: String,
    action_id: String,
    status: String,
    message: String,
}

#[derive(Serialize)]
struct DigestMaterial<'a> {
    schema_version: u32,
    action_id: &'a str,
    action_options: &'a ApprovalActionOptions,
    preview: &'a ActionPreview,
}

pub fn request(home: impl AsRef<Path>, options: ApprovalRequestOptions) -> Result<ActionApproval> {
    request_with_ttl(home, options, DEFAULT_TTL_SECONDS)
}

fn request_with_ttl(
    home: impl AsRef<Path>,
    options: ApprovalRequestOptions,
    ttl_seconds: i64,
) -> Result<ActionApproval> {
    let home = home.as_ref();
    cleanup_terminal(home)?;
    enforce_queue_limit(home)?;
    let action_id = options.action_id.trim();
    if action_id.is_empty() {
        return Err(anyhow!("action_id is required"));
    }
    if action_id == "disclosure.accept" {
        return Err(anyhow!(
            "MCP approvals cannot accept the Nightward beta responsibility disclosure; accept it in the Nightward CLI, TUI, or Raycast extension"
        ));
    }
    if !state::disclosure_status(home).accepted {
        return Err(anyhow!(
            "accept the Nightward beta responsibility disclosure in the Nightward CLI, TUI, or Raycast extension before requesting write-capable MCP actions"
        ));
    }
    validate_options_for_action(action_id, &options.action_options)?;
    let preview = actions::preview(home, action_id)?;
    if !preview.action.available {
        return Err(anyhow!("{}", preview.action.blocked_reason));
    }
    let digest = preview_digest(action_id, &options.action_options, &preview)?;
    let now = Utc::now();
    let record = ActionApproval {
        schema_version: APPROVAL_SCHEMA_VERSION,
        approval_id: approval_id(action_id, &digest, now),
        status: "pending".to_string(),
        action_id: action_id.to_string(),
        action_options: options.action_options,
        preview_digest: digest,
        preview,
        requested_by: safe_requested_by(&options.requested_by),
        requested_at: now,
        expires_at: now + Duration::seconds(ttl_seconds.max(1)),
        decision_reason: String::new(),
        decided_at: None,
        applied_at: None,
        result_message: String::new(),
        action_audit_path: String::new(),
    };
    save_record(home, &record)?;
    append_approval_audit(home, &record, "requested", "approval requested")?;
    Ok(record)
}

pub fn list(home: impl AsRef<Path>) -> Result<ApprovalList> {
    let home = home.as_ref();
    cleanup_terminal(home)?;
    let mut approvals = load_records(home)?;
    approvals.sort_by(|left, right| right.requested_at.cmp(&left.requested_at));
    Ok(ApprovalList {
        schema_version: APPROVAL_SCHEMA_VERSION,
        approvals: approvals.into_iter().map(mark_expired).collect(),
    })
}

pub fn status(home: impl AsRef<Path>, approval_id: &str) -> Result<ActionApproval> {
    let home = home.as_ref();
    let mut record = load_record(home, approval_id)?;
    if is_expired(&record) {
        record.status = "expired".to_string();
        save_record(home, &record)?;
        append_approval_audit(home, &record, "expired", "approval expired")?;
    }
    Ok(record)
}

pub fn approve(
    home: impl AsRef<Path>,
    approval_id: &str,
    decision_reason: impl Into<String>,
) -> Result<ActionApproval> {
    let home = home.as_ref();
    let mut record = load_record(home, approval_id)?;
    ensure_pending(&record)?;
    record.status = "approved".to_string();
    record.decided_at = Some(Utc::now());
    record.decision_reason = redact_text(&decision_reason.into());
    save_record(home, &record)?;
    append_approval_audit(home, &record, "approved", "approval granted locally")?;
    Ok(record)
}

pub fn deny(
    home: impl AsRef<Path>,
    approval_id: &str,
    decision_reason: impl Into<String>,
) -> Result<ActionApproval> {
    let home = home.as_ref();
    let mut record = load_record(home, approval_id)?;
    ensure_pending(&record)?;
    record.status = "denied".to_string();
    record.decided_at = Some(Utc::now());
    record.decision_reason = redact_text(&decision_reason.into());
    save_record(home, &record)?;
    append_approval_audit(home, &record, "denied", "approval denied locally")?;
    Ok(record)
}

pub fn apply_approved(home: impl AsRef<Path>, approval_id: &str) -> Result<ApprovedActionResult> {
    let home = home.as_ref();
    let mut record = load_record(home, approval_id)?;
    if is_expired(&record) {
        record.status = "expired".to_string();
        save_record(home, &record)?;
        append_approval_audit(home, &record, "expired", "approval expired before apply")?;
        return Err(anyhow!("approval {approval_id} expired"));
    }
    if record.status != "approved" {
        return Err(anyhow!(
            "approval {approval_id} is {}, not approved",
            record.status
        ));
    }

    let current_preview = actions::preview(home, &record.action_id)?;
    if !current_preview.action.available {
        record.status = "invalidated".to_string();
        save_record(home, &record)?;
        append_approval_audit(
            home,
            &record,
            "invalidated",
            "approved action is no longer available",
        )?;
        return Err(anyhow!("approved action is no longer available"));
    }
    let current_digest =
        preview_digest(&record.action_id, &record.action_options, &current_preview)?;
    if current_digest != record.preview_digest {
        record.status = "invalidated".to_string();
        save_record(home, &record)?;
        append_approval_audit(
            home,
            &record,
            "invalidated",
            "approved action no longer matches current preview",
        )?;
        return Err(anyhow!("approved action no longer matches current preview"));
    }

    match actions::apply(
        home,
        &record.action_id,
        record.action_options.clone().into_apply_options(),
    ) {
        Ok(result) => {
            record.status = "applied".to_string();
            record.applied_at = Some(Utc::now());
            record.result_message = redact_text(&result.message);
            record.action_audit_path = result.audit_path.clone();
            save_record(home, &record)?;
            append_approval_audit(home, &record, "applied", "approved action applied")?;
            Ok(ApprovedActionResult {
                schema_version: APPROVAL_SCHEMA_VERSION,
                approval: record,
                action_result: result,
            })
        }
        Err(error) => {
            record.status = "failed".to_string();
            record.applied_at = Some(Utc::now());
            record.result_message = redact_text(&error.to_string());
            save_record(home, &record)?;
            append_approval_audit(home, &record, "failed", "approved action failed")?;
            Err(error)
        }
    }
}

pub fn cleanup(home: impl AsRef<Path>) -> Result<ApprovalList> {
    cleanup_terminal(home.as_ref())?;
    list(home)
}

fn approval_dir(home: &Path) -> PathBuf {
    state::state_dir(home).join("action-approvals")
}

fn approval_path(home: &Path, approval_id: &str) -> Result<PathBuf> {
    if approval_id.is_empty()
        || approval_id.len() > 96
        || !approval_id
            .chars()
            .all(|ch| ch.is_ascii_alphanumeric() || matches!(ch, '-' | '_'))
    {
        return Err(anyhow!("invalid approval_id"));
    }
    Ok(approval_dir(home).join(format!("{approval_id}.json")))
}

fn load_records(home: &Path) -> Result<Vec<ActionApproval>> {
    let dir = approval_dir(home);
    match fs::symlink_metadata(&dir) {
        Ok(metadata) => {
            if metadata.file_type().is_symlink() {
                return Err(anyhow!(
                    "refusing to read symlinked approval directory {}",
                    dir.display()
                ));
            }
            if !metadata.is_dir() {
                return Err(anyhow!(
                    "approval path is not a directory: {}",
                    dir.display()
                ));
            }
        }
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => return Ok(Vec::new()),
        Err(error) => return Err(error).with_context(|| format!("inspect {}", dir.display())),
    }
    match fs::read_dir(&dir) {
        Ok(entries) => entries
            .map(|entry| -> Result<Option<ActionApproval>> {
                let entry = entry.with_context(|| format!("read {}", dir.display()))?;
                let path = entry.path();
                let metadata = fs::symlink_metadata(&path)
                    .with_context(|| format!("inspect {}", path.display()))?;
                if metadata.file_type().is_symlink() {
                    return Err(anyhow!(
                        "refusing to read symlinked approval file {}",
                        path.display()
                    ));
                }
                if !metadata.is_file()
                    || path.extension().and_then(|ext| ext.to_str()) != Some("json")
                {
                    return Ok(None);
                }
                Ok(Some(load_record_path(&path)?))
            })
            .filter_map(|result| result.transpose())
            .collect(),
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => Ok(Vec::new()),
        Err(error) => Err(error).with_context(|| format!("read {}", dir.display())),
    }
}

fn load_record(home: &Path, approval_id: &str) -> Result<ActionApproval> {
    load_record_path(&approval_path(home, approval_id)?)
}

fn load_record_path(path: &Path) -> Result<ActionApproval> {
    let metadata =
        fs::symlink_metadata(path).with_context(|| format!("inspect {}", path.display()))?;
    if metadata.file_type().is_symlink() {
        return Err(anyhow!(
            "refusing to read symlinked approval file {}",
            path.display()
        ));
    }
    if !metadata.is_file() {
        return Err(anyhow!("approval path is not a regular file"));
    }
    let text = fs::read_to_string(path).with_context(|| format!("read {}", path.display()))?;
    serde_json::from_str(&text).with_context(|| format!("parse {}", path.display()))
}

fn save_record(home: &Path, record: &ActionApproval) -> Result<()> {
    let path = approval_path(home, &record.approval_id)?;
    state::write_private_file(
        &path,
        format!("{}\n", serde_json::to_string_pretty(record)?),
    )
}

fn cleanup_terminal(home: &Path) -> Result<()> {
    let now = Utc::now();
    let dir = approval_dir(home);
    let records = load_records(home)?;
    for mut record in records {
        if is_expired(&record) {
            record.status = "expired".to_string();
            save_record(home, &record)?;
            append_approval_audit(home, &record, "expired", "approval expired")?;
        }
        let terminal = matches!(
            record.status.as_str(),
            "denied" | "expired" | "applied" | "failed" | "invalidated"
        );
        if terminal && record.expires_at + Duration::hours(24) < now {
            let path = approval_path(home, &record.approval_id)?;
            match fs::remove_file(&path) {
                Ok(()) => {}
                Err(error) if error.kind() == std::io::ErrorKind::NotFound => {}
                Err(error) => {
                    return Err(error).with_context(|| format!("remove {}", path.display()))
                }
            }
        }
    }
    state::create_private_dir(&dir)?;
    Ok(())
}

fn enforce_queue_limit(home: &Path) -> Result<()> {
    let active = load_records(home)?
        .into_iter()
        .filter(|record| {
            !is_expired(record) && matches!(record.status.as_str(), "pending" | "approved")
        })
        .count();
    if active >= MAX_APPROVAL_FILES {
        return Err(anyhow!("too many active Nightward action approvals"));
    }
    Ok(())
}

fn append_approval_audit(
    home: &Path,
    record: &ActionApproval,
    event: &str,
    message: &str,
) -> Result<PathBuf> {
    state::append_audit(
        home,
        &ApprovalAuditEvent {
            schema_version: APPROVAL_SCHEMA_VERSION,
            generated_at: Utc::now(),
            event: format!("approval.{event}"),
            approval_id: record.approval_id.clone(),
            action_id: record.action_id.clone(),
            status: record.status.clone(),
            message: message.to_string(),
        },
    )
}

fn ensure_pending(record: &ActionApproval) -> Result<()> {
    if is_expired(record) {
        return Err(anyhow!("approval {} expired", record.approval_id));
    }
    if record.status != "pending" {
        return Err(anyhow!(
            "approval {} is {}, not pending",
            record.approval_id,
            record.status
        ));
    }
    Ok(())
}

fn mark_expired(mut record: ActionApproval) -> ActionApproval {
    if is_expired(&record) {
        record.status = "expired".to_string();
    }
    record
}

fn is_expired(record: &ActionApproval) -> bool {
    matches!(record.status.as_str(), "pending" | "approved") && Utc::now() > record.expires_at
}

fn preview_digest(
    action_id: &str,
    action_options: &ApprovalActionOptions,
    preview: &ActionPreview,
) -> Result<String> {
    let material = DigestMaterial {
        schema_version: APPROVAL_SCHEMA_VERSION,
        action_id,
        action_options,
        preview,
    };
    let bytes = serde_json::to_vec(&material)?;
    Ok(hex::encode(Sha256::digest(bytes)))
}

fn approval_id(action_id: &str, digest: &str, now: DateTime<Utc>) -> String {
    let mut hasher = Sha256::new();
    hasher.update(action_id.as_bytes());
    hasher.update([0]);
    hasher.update(digest.as_bytes());
    hasher.update([0]);
    hasher.update(now.timestamp_nanos_opt().unwrap_or_default().to_le_bytes());
    let hex = hex::encode(hasher.finalize());
    format!("appr-{}", &hex[..24])
}

fn validate_options_for_action(action_id: &str, options: &ApprovalActionOptions) -> Result<()> {
    let policy_action = matches!(action_id, "policy.init" | "policy.ignore");
    let schedule_action = action_id == "schedule.install";
    if !schedule_action && !options.executable.trim().is_empty() {
        return Err(anyhow!("executable is only accepted for schedule.install"));
    }
    if !policy_action && !options.policy_path.trim().is_empty() {
        return Err(anyhow!("policy_path is only accepted for policy actions"));
    }
    if action_id != "policy.ignore"
        && (!options.finding_id.trim().is_empty()
            || !options.rule.trim().is_empty()
            || !options.reason.trim().is_empty())
    {
        return Err(anyhow!(
            "finding_id, rule, and reason are only accepted for policy.ignore"
        ));
    }
    Ok(())
}

fn safe_requested_by(value: &str) -> String {
    let value = redact_text(value.trim());
    if value.is_empty() {
        default_requested_by()
    } else {
        value.chars().take(120).collect()
    }
}

fn default_requested_by() -> String {
    "mcp-client".to_string()
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::actions::ApplyOptions;

    #[test]
    fn request_requires_out_of_band_disclosure() {
        let home = tempfile::tempdir().expect("home");
        let error = request(
            home.path(),
            ApprovalRequestOptions {
                action_id: "backup.snapshot".to_string(),
                ..default_request()
            },
        )
        .expect_err("disclosure required");
        assert!(error.to_string().contains("accept the Nightward beta"));
    }

    #[test]
    fn request_rejects_disclosure_self_accept() {
        let home = accepted_home();
        let error = request(
            home.path(),
            ApprovalRequestOptions {
                action_id: "disclosure.accept".to_string(),
                ..default_request()
            },
        )
        .expect_err("self accept blocked");
        assert!(error.to_string().contains("cannot accept"));
    }

    #[test]
    fn approval_must_be_local_before_apply_and_is_one_time() {
        let home = accepted_home();
        let approval = request(
            home.path(),
            ApprovalRequestOptions {
                action_id: "backup.snapshot".to_string(),
                ..default_request()
            },
        )
        .expect("request");
        let error =
            apply_approved(home.path(), &approval.approval_id).expect_err("pending blocked");
        assert!(error.to_string().contains("not approved"));
        approve(home.path(), &approval.approval_id, "reviewed locally").expect("approve");
        let result = apply_approved(home.path(), &approval.approval_id).expect("apply");
        assert_eq!(result.approval.status, "applied");
        let replay =
            apply_approved(home.path(), &approval.approval_id).expect_err("replay blocked");
        assert!(replay.to_string().contains("not approved"));
    }

    #[test]
    fn denied_and_expired_approvals_do_not_apply() {
        let home = accepted_home();
        let denied = request(
            home.path(),
            ApprovalRequestOptions {
                action_id: "backup.snapshot".to_string(),
                ..default_request()
            },
        )
        .expect("request");
        deny(home.path(), &denied.approval_id, "no").expect("deny");
        let error = apply_approved(home.path(), &denied.approval_id).expect_err("denied");
        assert!(error.to_string().contains("not approved"));

        let expired = request_with_ttl(
            home.path(),
            ApprovalRequestOptions {
                action_id: "backup.snapshot".to_string(),
                ..default_request()
            },
            1,
        )
        .expect("request");
        let mut record = load_record(home.path(), &expired.approval_id).expect("load");
        record.expires_at = Utc::now() - Duration::seconds(1);
        save_record(home.path(), &record).expect("save");
        approve(home.path(), &expired.approval_id, "late").expect_err("expired");
    }

    #[test]
    fn digest_mismatch_invalidates_approval() {
        let home = accepted_home();
        let approval = request(
            home.path(),
            ApprovalRequestOptions {
                action_id: "backup.snapshot".to_string(),
                ..default_request()
            },
        )
        .expect("request");
        approve(home.path(), &approval.approval_id, "reviewed").expect("approve");
        let mut record = load_record(home.path(), &approval.approval_id).expect("load");
        record.preview_digest = "bad-digest".to_string();
        save_record(home.path(), &record).expect("save");
        let error = apply_approved(home.path(), &approval.approval_id).expect_err("mismatch");
        assert!(error.to_string().contains("no longer matches"));
        assert_eq!(
            load_record(home.path(), &approval.approval_id)
                .expect("load")
                .status,
            "invalidated"
        );
    }

    #[test]
    fn cleanup_persists_expiry_and_prunes_stale_terminal_records() {
        let home = accepted_home();
        let expired = request_with_ttl(
            home.path(),
            ApprovalRequestOptions {
                action_id: "backup.snapshot".to_string(),
                ..default_request()
            },
            1,
        )
        .expect("request");
        let mut record = load_record(home.path(), &expired.approval_id).expect("load");
        record.expires_at = Utc::now() - Duration::seconds(1);
        save_record(home.path(), &record).expect("save");

        cleanup(home.path()).expect("cleanup");
        assert_eq!(
            load_record(home.path(), &expired.approval_id)
                .expect("load expired")
                .status,
            "expired"
        );

        let stale = request(
            home.path(),
            ApprovalRequestOptions {
                action_id: "backup.snapshot".to_string(),
                ..default_request()
            },
        )
        .expect("request");
        deny(home.path(), &stale.approval_id, "no").expect("deny");
        let mut record = load_record(home.path(), &stale.approval_id).expect("load");
        record.expires_at = Utc::now() - Duration::hours(25);
        save_record(home.path(), &record).expect("save");

        cleanup(home.path()).expect("cleanup");
        load_record(home.path(), &stale.approval_id).expect_err("stale terminal record pruned");
    }

    #[cfg(unix)]
    #[test]
    fn approval_storage_rejects_symlinked_files() {
        use std::os::unix::fs::symlink;

        let home = accepted_home();
        let approval = request(
            home.path(),
            ApprovalRequestOptions {
                action_id: "backup.snapshot".to_string(),
                ..default_request()
            },
        )
        .expect("request");
        let outside = tempfile::NamedTempFile::new().expect("outside");
        let path = approval_path(home.path(), &approval.approval_id).expect("path");
        fs::remove_file(&path).expect("remove");
        symlink(outside.path(), &path).expect("symlink");
        let error = status(home.path(), &approval.approval_id).expect_err("symlink rejected");
        assert!(error.to_string().contains("symlinked approval"));
    }

    #[cfg(unix)]
    #[test]
    fn approval_storage_rejects_symlinked_directory() {
        use std::os::unix::fs::symlink;

        let home = accepted_home();
        let outside = tempfile::tempdir().expect("outside");
        let dir = approval_dir(home.path());
        fs::create_dir_all(dir.parent().unwrap()).expect("parent");
        symlink(outside.path(), &dir).expect("approval dir symlink");
        let error = list(home.path()).expect_err("symlink dir rejected");
        assert!(error.to_string().contains("symlinked approval directory"));
    }

    fn accepted_home() -> tempfile::TempDir {
        let home = tempfile::tempdir().expect("home");
        crate::actions::apply(
            home.path(),
            "disclosure.accept",
            ApplyOptions {
                confirm: true,
                ..Default::default()
            },
        )
        .expect("accept disclosure");
        home
    }

    fn default_request() -> ApprovalRequestOptions {
        ApprovalRequestOptions {
            action_id: String::new(),
            action_options: ApprovalActionOptions::default(),
            requested_by: "test-client".to_string(),
        }
    }
}
