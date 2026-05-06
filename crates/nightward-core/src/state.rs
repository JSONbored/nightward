use anyhow::{Context, Result};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::fs::{self, OpenOptions};
use std::io::{ErrorKind, Write};
use std::path::{Path, PathBuf};

pub const DISCLOSURE_VERSION: u32 = 1;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Settings {
    pub schema_version: u32,
    #[serde(default)]
    pub accepted_disclosure_version: u32,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub accepted_disclosure_at: Option<DateTime<Utc>>,
    #[serde(default)]
    pub selected_providers: Vec<String>,
    #[serde(default)]
    pub allow_online_providers: bool,
}

impl Default for Settings {
    fn default() -> Self {
        Self {
            schema_version: 1,
            accepted_disclosure_version: 0,
            accepted_disclosure_at: None,
            selected_providers: Vec::new(),
            allow_online_providers: false,
        }
    }
}

#[derive(Debug, Clone, Serialize)]
pub struct DisclosureStatus {
    pub schema_version: u32,
    pub required_version: u32,
    pub accepted: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub accepted_at: Option<DateTime<Utc>>,
    pub text: String,
}

pub fn config_dir(home: impl AsRef<Path>) -> PathBuf {
    home.as_ref().join(".config/nightward")
}

pub fn state_dir(home: impl AsRef<Path>) -> PathBuf {
    home.as_ref().join(".local/state/nightward")
}

pub fn cache_dir(home: impl AsRef<Path>) -> PathBuf {
    home.as_ref().join(".cache/nightward")
}

pub fn settings_path(home: impl AsRef<Path>) -> PathBuf {
    config_dir(home).join("settings.json")
}

pub fn audit_path(home: impl AsRef<Path>) -> PathBuf {
    state_dir(home).join("audit.jsonl")
}

pub fn disclosure_text() -> String {
    [
        "Nightward is a beta local security/devtool assistant.",
        "It can inspect and, when explicitly confirmed, change local AI-agent, MCP, schedule, provider, and backup state.",
        "You are responsible for reviewing previews, backups, provider behavior, and any resulting system changes.",
        "Nightward provides no warranty and the maintainers are not liable for broken configs, lost data, exposed secrets, package-manager side effects, or third-party tool behavior.",
    ]
    .join(" ")
}

pub fn disclosure_status(home: impl AsRef<Path>) -> DisclosureStatus {
    let settings = load_settings(home).unwrap_or_default();
    DisclosureStatus {
        schema_version: 1,
        required_version: DISCLOSURE_VERSION,
        accepted: settings.accepted_disclosure_version >= DISCLOSURE_VERSION,
        accepted_at: settings.accepted_disclosure_at,
        text: disclosure_text(),
    }
}

pub fn accept_disclosure(home: impl AsRef<Path>) -> Result<DisclosureStatus> {
    let home = home.as_ref();
    let mut settings = load_settings(home).unwrap_or_default();
    settings.accepted_disclosure_version = DISCLOSURE_VERSION;
    settings.accepted_disclosure_at = Some(Utc::now());
    save_settings(home, &settings)?;
    Ok(disclosure_status(home))
}

pub fn load_settings(home: impl AsRef<Path>) -> Result<Settings> {
    let path = settings_path(home);
    if !path.exists() {
        return Ok(Settings::default());
    }
    let text = fs::read_to_string(&path).with_context(|| format!("read {}", path.display()))?;
    serde_json::from_str(&text).with_context(|| format!("parse {}", path.display()))
}

pub fn save_settings(home: impl AsRef<Path>, settings: &Settings) -> Result<()> {
    let path = settings_path(home);
    write_private_file(
        &path,
        format!("{}\n", serde_json::to_string_pretty(settings)?),
    )
}

pub fn set_provider_selected(
    home: impl AsRef<Path>,
    provider: &str,
    selected: bool,
) -> Result<Settings> {
    let home = home.as_ref();
    let mut settings = load_settings(home).unwrap_or_default();
    let normalized = provider.trim().to_ascii_lowercase();
    settings
        .selected_providers
        .retain(|existing| existing != &normalized);
    if selected && !normalized.is_empty() {
        settings.selected_providers.push(normalized);
        settings.selected_providers.sort();
    }
    save_settings(home, &settings)?;
    Ok(settings)
}

pub fn set_online_providers_allowed(home: impl AsRef<Path>, allowed: bool) -> Result<Settings> {
    let home = home.as_ref();
    let mut settings = load_settings(home).unwrap_or_default();
    settings.allow_online_providers = allowed;
    save_settings(home, &settings)?;
    Ok(settings)
}

pub fn append_audit(home: impl AsRef<Path>, value: &impl Serialize) -> Result<PathBuf> {
    let path = audit_path(home);
    if let Some(parent) = path.parent() {
        create_private_dir(parent)?;
    }
    ensure_regular_file_or_missing(&path)?;
    let mut file = OpenOptions::new()
        .create(true)
        .append(true)
        .open(&path)
        .with_context(|| format!("open {}", path.display()))?;
    set_private_file_permissions(&path)?;
    writeln!(file, "{}", serde_json::to_string(value)?)
        .with_context(|| format!("append {}", path.display()))?;
    Ok(path)
}

pub fn create_private_dir(path: &Path) -> Result<()> {
    if path.as_os_str().is_empty() {
        return Ok(());
    }

    let mut missing = Vec::new();
    let mut cursor = path;
    loop {
        if cursor.as_os_str().is_empty() {
            break;
        }
        match fs::symlink_metadata(cursor) {
            Ok(metadata) => {
                ensure_private_dir_metadata(cursor, &metadata)?;
                break;
            }
            Err(error) if error.kind() == ErrorKind::NotFound => {
                missing.push(cursor.to_path_buf());
                let Some(parent) = cursor.parent() else {
                    break;
                };
                cursor = parent;
            }
            Err(error) => {
                return Err(error).with_context(|| format!("inspect {}", cursor.display()));
            }
        }
    }

    for dir in missing.iter().rev() {
        match fs::create_dir(dir) {
            Ok(()) => {}
            Err(error) if error.kind() == ErrorKind::AlreadyExists => {}
            Err(error) => return Err(error).with_context(|| format!("create {}", dir.display())),
        }
        ensure_private_dir(dir)?;
        set_private_dir_permissions(dir)?;
    }

    ensure_private_dir(path)?;
    set_private_dir_permissions(path)?;
    Ok(())
}

pub fn ensure_regular_file_or_missing(path: &Path) -> Result<()> {
    match fs::symlink_metadata(path) {
        Ok(metadata) => {
            if metadata.file_type().is_symlink() {
                return Err(anyhow::anyhow!(
                    "refusing to write through symlinked Nightward path {}",
                    path.display()
                ));
            }
            if !metadata.is_file() {
                return Err(anyhow::anyhow!(
                    "refusing to write non-regular Nightward path {}",
                    path.display()
                ));
            }
            Ok(())
        }
        Err(error) if error.kind() == ErrorKind::NotFound => Ok(()),
        Err(error) => Err(error).with_context(|| format!("inspect {}", path.display())),
    }
}

pub fn write_private_file(path: &Path, contents: impl AsRef<[u8]>) -> Result<()> {
    if let Some(parent) = path.parent() {
        create_private_dir(parent)?;
    }
    ensure_regular_file_or_missing(path)?;
    let temp_path = private_temp_path(path)?;
    let write_result = (|| -> Result<()> {
        {
            let mut file = OpenOptions::new()
                .create_new(true)
                .write(true)
                .open(&temp_path)
                .with_context(|| format!("create {}", temp_path.display()))?;
            file.write_all(contents.as_ref())
                .with_context(|| format!("write {}", temp_path.display()))?;
            file.sync_all()
                .with_context(|| format!("sync {}", temp_path.display()))?;
        }
        set_private_file_permissions(&temp_path)?;
        #[cfg(windows)]
        if path.exists() {
            ensure_regular_file_or_missing(path)?;
            fs::remove_file(path).with_context(|| format!("remove {}", path.display()))?;
        }
        fs::rename(&temp_path, path)
            .with_context(|| format!("rename {} to {}", temp_path.display(), path.display()))?;
        set_private_file_permissions(path)?;
        Ok(())
    })();
    if write_result.is_err() {
        let _ = fs::remove_file(&temp_path);
    }
    write_result
}

fn private_temp_path(path: &Path) -> Result<PathBuf> {
    let file_name = path
        .file_name()
        .and_then(|name| name.to_str())
        .ok_or_else(|| anyhow::anyhow!("Nightward write path must include a file name"))?;
    Ok(path.with_file_name(format!(
        ".{file_name}.tmp-{}-{}",
        std::process::id(),
        Utc::now()
            .timestamp_nanos_opt()
            .unwrap_or_else(|| Utc::now().timestamp_micros())
    )))
}

fn ensure_private_dir(path: &Path) -> Result<()> {
    let metadata =
        fs::symlink_metadata(path).with_context(|| format!("inspect {}", path.display()))?;
    ensure_private_dir_metadata(path, &metadata)
}

fn ensure_private_dir_metadata(path: &Path, metadata: &fs::Metadata) -> Result<()> {
    if metadata.file_type().is_symlink() {
        return Err(anyhow::anyhow!(
            "refusing to use symlinked Nightward directory {}",
            path.display()
        ));
    }
    if !metadata.is_dir() {
        return Err(anyhow::anyhow!(
            "refusing to use non-directory Nightward path {}",
            path.display()
        ));
    }
    Ok(())
}

#[cfg(unix)]
pub fn set_private_dir_permissions(path: &Path) -> Result<()> {
    use std::os::unix::fs::PermissionsExt;
    fs::set_permissions(path, fs::Permissions::from_mode(0o700))
        .with_context(|| format!("chmod 700 {}", path.display()))
}

#[cfg(not(unix))]
pub fn set_private_dir_permissions(_path: &Path) -> Result<()> {
    Ok(())
}

#[cfg(unix)]
pub fn set_private_file_permissions(path: &Path) -> Result<()> {
    use std::os::unix::fs::PermissionsExt;
    fs::set_permissions(path, fs::Permissions::from_mode(0o600))
        .with_context(|| format!("chmod 600 {}", path.display()))
}

#[cfg(not(unix))]
pub fn set_private_file_permissions(_path: &Path) -> Result<()> {
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[cfg(unix)]
    #[test]
    fn private_file_writes_reject_symlinked_file_and_directory_paths() {
        use std::os::unix::fs::symlink;

        let home = tempfile::tempdir().expect("home");
        let outside = tempfile::tempdir().expect("outside");
        let target = outside.path().join("settings.json");
        fs::write(&target, "{}\n").expect("target");
        let config = config_dir(home.path());
        fs::create_dir_all(&config).expect("config dir");
        symlink(&target, settings_path(home.path())).expect("settings symlink");

        let error = save_settings(home.path(), &Settings::default()).expect_err("symlink file");
        assert!(error.to_string().contains("symlinked Nightward path"));

        fs::remove_file(settings_path(home.path())).expect("remove symlink");
        fs::remove_dir_all(&config).expect("remove config");
        symlink(outside.path(), &config).expect("config dir symlink");

        let error = save_settings(home.path(), &Settings::default()).expect_err("symlink dir");
        assert!(error.to_string().contains("symlinked Nightward directory"));
    }

    #[cfg(unix)]
    #[test]
    fn audit_append_rejects_symlinked_audit_file() {
        use std::os::unix::fs::symlink;

        let home = tempfile::tempdir().expect("home");
        let outside = tempfile::tempdir().expect("outside");
        let target = outside.path().join("audit.jsonl");
        fs::write(&target, "{}\n").expect("target");
        let audit = audit_path(home.path());
        fs::create_dir_all(audit.parent().unwrap()).expect("audit dir");
        symlink(&target, &audit).expect("audit symlink");

        let error =
            append_audit(home.path(), &serde_json::json!({"ok":true})).expect_err("symlink audit");
        assert!(error.to_string().contains("symlinked Nightward path"));
    }
}
