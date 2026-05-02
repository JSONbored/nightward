use crate::rules;
use crate::{AdapterStatus, Classification, Finding, FixKind, Item, PatchHint, Report, RiskLevel};
use anyhow::{Context, Result};
use chrono::{DateTime, Utc};
use regex::Regex;
use serde_json::Value;
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::fs;
use std::path::{Path, PathBuf};

const MAX_CONFIG_BYTES: u64 = 2 * 1024 * 1024;

#[derive(Debug, Clone)]
struct Adapter {
    name: &'static str,
    description: &'static str,
    paths: &'static [&'static str],
}

const HOME_ADAPTERS: &[Adapter] = &[
    Adapter {
        name: "Codex",
        description: "OpenAI Codex CLI and agent configuration",
        paths: &[".codex/config.toml", ".codex/auth.json"],
    },
    Adapter {
        name: "Claude",
        description: "Claude Code and Claude Desktop MCP configuration",
        paths: &[
            ".claude.json",
            "Library/Application Support/Claude/claude_desktop_config.json",
        ],
    },
    Adapter {
        name: "Cursor",
        description: "Cursor MCP configuration",
        paths: &[".cursor/mcp.json"],
    },
    Adapter {
        name: "Windsurf",
        description: "Windsurf MCP configuration",
        paths: &[".codeium/windsurf/mcp_config.json", ".windsurf/mcp_config.json"],
    },
    Adapter {
        name: "VS Code",
        description: "VS Code and compatible MCP settings",
        paths: &[
            "Library/Application Support/Code/User/mcp.json",
            "Library/Application Support/Code/User/settings.json",
            ".config/Code/User/mcp.json",
            ".config/Code/User/settings.json",
        ],
    },
    Adapter {
        name: "Cline/Roo",
        description: "Cline and Roo Code MCP settings",
        paths: &[
            "Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json",
            "Library/Application Support/Code/User/globalStorage/rooveterinaryinc.roo-cline/settings/mcp_settings.json",
            ".cline/mcp_settings.json",
            ".roo/mcp_settings.json",
        ],
    },
    Adapter {
        name: "OpenCode",
        description: "OpenCode local MCP configuration",
        paths: &[".opencode/config.json", ".config/opencode/opencode.json"],
    },
    Adapter {
        name: "Goose",
        description: "Goose agent and MCP configuration",
        paths: &[".config/goose/config.yaml"],
    },
    Adapter {
        name: "Ollama/Open WebUI",
        description: "Local model identity and app-owned runtime state",
        paths: &[".ollama/id_ed25519", ".open-webui"],
    },
];

const WORKSPACE_CANDIDATES: &[&str] = &[
    ".mcp.json",
    "mcp.json",
    ".cursor/mcp.json",
    ".vscode/mcp.json",
    ".codex/config.toml",
    ".claude/settings.json",
    "claude_desktop_config.json",
];

pub fn home_dir_from_env() -> PathBuf {
    std::env::var_os("NIGHTWARD_HOME")
        .map(PathBuf::from)
        .or_else(dirs::home_dir)
        .unwrap_or_else(|| PathBuf::from("."))
}

pub fn scan_home(home: impl AsRef<Path>) -> Result<Report> {
    let home = home.as_ref().to_path_buf();
    let mut report = Report::empty(
        home.display().to_string(),
        String::new(),
        "home".to_string(),
    );

    for adapter in HOME_ADAPTERS {
        let mut status = AdapterStatus {
            name: adapter.name.to_string(),
            description: adapter.description.to_string(),
            available: false,
            checked: adapter.paths.iter().map(|path| path.to_string()).collect(),
            found: Vec::new(),
        };
        for rel in adapter.paths {
            let path = home.join(rel);
            if !path.exists() {
                continue;
            }
            status.available = true;
            status.found.push(path.display().to_string());
            add_item_for_path(&mut report, adapter.name, &path);
            inspect_config(&mut report, adapter.name, &path);
        }
        report.adapters.push(status);
    }

    finalize_report(&mut report);
    Ok(report)
}

pub fn scan_workspace(workspace: impl AsRef<Path>) -> Result<Report> {
    let workspace = workspace.as_ref().to_path_buf();
    let home = home_dir_from_env();
    let mut report = Report::empty(
        home.display().to_string(),
        workspace.display().to_string(),
        "workspace".to_string(),
    );

    let mut found = Vec::new();
    for rel in WORKSPACE_CANDIDATES {
        let path = workspace.join(rel);
        if path.exists() {
            found.push(path.display().to_string());
            add_item_for_path(&mut report, "Workspace", &path);
            inspect_config(&mut report, "Workspace", &path);
        }
    }
    report.adapters.push(AdapterStatus {
        name: "Workspace".to_string(),
        description: "Workspace-local AI and MCP configuration".to_string(),
        available: !found.is_empty(),
        checked: WORKSPACE_CANDIDATES
            .iter()
            .map(|path| path.to_string())
            .collect(),
        found,
    });

    finalize_report(&mut report);
    Ok(report)
}

pub fn load_report(path: impl AsRef<Path>) -> Result<Report> {
    let text = fs::read_to_string(path.as_ref())
        .with_context(|| format!("read {}", path.as_ref().display()))?;
    serde_json::from_str(&text).context("parse Nightward report JSON")
}

pub fn write_report(path: impl AsRef<Path>, report: &Report) -> Result<()> {
    if let Some(parent) = path.as_ref().parent() {
        fs::create_dir_all(parent)?;
    }
    let text = serde_json::to_string_pretty(report)?;
    fs::write(path.as_ref(), format!("{text}\n"))?;
    Ok(())
}

fn add_item_for_path(report: &mut Report, tool: &str, path: &Path) {
    let metadata = fs::symlink_metadata(path);
    let (exists, size_bytes, mod_time, classification, risk, kind, reason, action) = match metadata
    {
        Ok(meta) => {
            let mod_time = meta.modified().ok().map(DateTime::<Utc>::from);
            let path_text = path.display().to_string();
            let (classification, risk, reason, action) = classify_path(&path_text);
            (
                true,
                Some(meta.len()),
                mod_time,
                classification,
                risk,
                if meta.file_type().is_symlink() {
                    "symlink".to_string()
                } else if meta.is_dir() {
                    "directory".to_string()
                } else {
                    "file".to_string()
                },
                reason,
                action,
            )
        }
        Err(_) => (
            false,
            None,
            None,
            Classification::Unknown,
            RiskLevel::Info,
            "missing".to_string(),
            "Path was not found during scan.".to_string(),
            "No action needed unless this path should exist.".to_string(),
        ),
    };

    report.items.push(Item {
        id: stable_id(&["item", tool, &path.display().to_string()]),
        tool: tool.to_string(),
        path: path.display().to_string(),
        kind,
        classification,
        risk,
        reason,
        recommended_action: action,
        exists,
        size_bytes,
        mod_time,
        metadata: BTreeMap::new(),
    });
}

fn classify_path(path: &str) -> (Classification, RiskLevel, String, String) {
    let lower = path.to_ascii_lowercase();
    if lower.contains("auth")
        || lower.contains("credential")
        || lower.contains("id_ed25519")
        || lower.contains("token")
        || lower.contains("keychain")
    {
        return (
            Classification::SecretAuth,
            RiskLevel::Critical,
            "Path appears to contain local authentication or credential material.".to_string(),
            "Keep this path out of portable dotfiles and encrypted backups unless explicitly reviewed.".to_string(),
        );
    }
    if lower.contains("cache") || lower.contains("logs") {
        return (
            Classification::RuntimeCache,
            RiskLevel::Info,
            "Path looks like runtime cache or logs.".to_string(),
            "Do not sync unless a tool explicitly requires it.".to_string(),
        );
    }
    if lower.contains("library/application support") || lower.contains(".config") {
        return (
            Classification::MachineLocal,
            RiskLevel::Medium,
            "Path is likely machine-local application configuration.".to_string(),
            "Review before syncing because it may contain local endpoints, paths, or tokens."
                .to_string(),
        );
    }
    (
        Classification::Portable,
        RiskLevel::Info,
        "Path is a portable-looking configuration file.".to_string(),
        "Review generated findings before syncing.".to_string(),
    )
}

fn inspect_config(report: &mut Report, tool: &str, path: &Path) {
    if let Ok(meta) = fs::symlink_metadata(path) {
        if meta.file_type().is_symlink() {
            push_finding(
                report,
                tool,
                path,
                "",
                "config_symlink",
                RiskLevel::Info,
                "Config file is a symlink; review the target before syncing.",
                "symlink config",
                "Review the link target and keep machine-local paths out of portable dotfiles.",
                FixKind::ManualReview,
                None,
            );
        }
        if meta.len() > MAX_CONFIG_BYTES {
            push_finding(
                report,
                tool,
                path,
                "",
                "config_too_large",
                RiskLevel::Medium,
                "Config file is too large for safe inline review.",
                &format!("size_bytes={}", meta.len()),
                "Review this file manually and split generated/cache material away from portable config.",
                FixKind::ManualReview,
                None,
            );
            return;
        }
    }

    let bytes = match fs::read(path) {
        Ok(bytes) => bytes,
        Err(error) => {
            push_finding(
                report,
                tool,
                path,
                "",
                "config_read_failed",
                RiskLevel::Medium,
                "Nightward could not read a config file.",
                &format!("error={}", redact_text(&error.to_string())),
                "Check file permissions and review manually.",
                FixKind::ManualReview,
                None,
            );
            return;
        }
    };
    let text = String::from_utf8_lossy(&bytes);
    let value = match parse_config(path, &text) {
        Ok(value) => value,
        Err(error) => {
            push_finding(
                report,
                tool,
                path,
                "",
                "config_parse_failed",
                RiskLevel::Medium,
                "Nightward could not parse a config file.",
                &format!("error={}", redact_text(&error.to_string())),
                "Fix the syntax error or review this file manually before syncing.",
                FixKind::ManualReview,
                None,
            );
            return;
        }
    };
    inspect_mcp_servers(report, tool, path, &value);
}

fn parse_config(path: &Path, text: &str) -> Result<Value> {
    let ext = path
        .extension()
        .and_then(|ext| ext.to_str())
        .unwrap_or_default()
        .to_ascii_lowercase();
    if ext == "toml" {
        let value: toml::Value = toml::from_str(text)?;
        return Ok(serde_json::to_value(value)?);
    }
    if ext == "yaml" || ext == "yml" {
        let value: serde_yaml::Value = serde_yaml::from_str(text)?;
        return Ok(serde_json::to_value(value)?);
    }
    Ok(serde_json::from_str(text)?)
}

fn inspect_mcp_servers(report: &mut Report, tool: &str, path: &Path, value: &Value) {
    for (server, config) in mcp_server_entries(value) {
        inspect_server(report, tool, path, &server, config);
    }
}

fn mcp_server_entries(value: &Value) -> Vec<(String, &Value)> {
    let mut out = Vec::new();
    for key in ["mcpServers", "mcp_servers", "servers"] {
        if let Some(object) = value.get(key).and_then(Value::as_object) {
            for (name, config) in object {
                out.push((name.clone(), config));
            }
        }
    }
    if out.is_empty() {
        if let Some(object) = value.as_object() {
            for (name, config) in object {
                if config.get("command").is_some()
                    || config.get("cmd").is_some()
                    || config.get("url").is_some()
                {
                    out.push((name.clone(), config));
                }
            }
        }
    }
    out
}

fn inspect_server(report: &mut Report, tool: &str, path: &Path, server: &str, config: &Value) {
    let command = str_field(config, &["command", "cmd"]);
    let args = array_field(config, &["args", "arguments"]);
    let url = str_field(config, &["url", "endpoint"]);
    let evidence = redact_text(&format!(
        "command={} args={} url={}",
        command,
        args.join(" "),
        url
    ));

    if shell_command(&command) {
        push_finding(
            report,
            tool,
            path,
            server,
            "mcp_shell_wrapper",
            RiskLevel::High,
            "MCP server runs through a shell wrapper.",
            &evidence,
            "Replace shell-wrapper launchers with direct command and argument arrays where possible.",
            FixKind::ReplaceShellWrapper,
            Some(PatchHint {
                kind: Some(FixKind::ReplaceShellWrapper),
                package: String::new(),
                env_key: String::new(),
                header_key: String::new(),
                inline_secret: false,
                direct_command: command.clone(),
                direct_args: args.clone(),
                replacement: String::new(),
            }),
        );
    }

    if let Some(package) = unpinned_package(&command, &args) {
        push_finding(
            report,
            tool,
            path,
            server,
            "mcp_unpinned_package",
            RiskLevel::High,
            &format!(
                "MCP server \"{}\" runs a package executor without an obvious pinned package version.",
                server
            ),
            &evidence,
            "Replace unversioned or @latest package references with a reviewed explicit version.",
            FixKind::PinPackage,
            Some(PatchHint {
                kind: Some(FixKind::PinPackage),
                package,
                env_key: String::new(),
                header_key: String::new(),
                inline_secret: false,
                direct_command: String::new(),
                direct_args: Vec::new(),
                replacement: String::new(),
            }),
        );
    }

    for (key, value) in object_entries(config, &["env"]) {
        if secret_key(&key) || secret_value(&value) {
            let value_is_inline_secret = secret_value(&value) || !env_reference(&value);
            let severity = if value_is_inline_secret {
                RiskLevel::Critical
            } else {
                RiskLevel::Medium
            };
            let message = if value_is_inline_secret {
                format!(
                    "MCP server \"{}\" stores sensitive env key {} inline.",
                    server, key
                )
            } else {
                format!(
                    "MCP server \"{}\" references a sensitive environment key.",
                    server
                )
            };
            let action = if value_is_inline_secret {
                "Move the value to a local secret source and keep only the variable name in portable config."
            } else {
                "Keep secret values outside dotfiles and document required env names only."
            };
            push_finding(
                report,
                tool,
                path,
                server,
                "mcp_secret_env",
                severity,
                &message,
                &format!("env.{}={}", key, redact_secret_field_value(&value)),
                action,
                FixKind::ExternalizeSecret,
                Some(PatchHint {
                    kind: Some(FixKind::ExternalizeSecret),
                    package: String::new(),
                    env_key: key,
                    header_key: String::new(),
                    inline_secret: value_is_inline_secret,
                    direct_command: String::new(),
                    direct_args: Vec::new(),
                    replacement: String::new(),
                }),
            );
        }
    }

    for (key, value) in object_entries(config, &["headers", "request_headers"]) {
        if secret_key(&key) || secret_value(&value) {
            push_finding(
                report,
                tool,
                path,
                server,
                "mcp_secret_header",
                RiskLevel::Critical,
                &format!(
                    "MCP server \"{}\" stores sensitive header key {} inline.",
                    server, key
                ),
                &format!("headers.{}={}", key, redact_secret_field_value(&value)),
                "Move the header value into a local environment variable or secret manager.",
                FixKind::ExternalizeSecret,
                Some(PatchHint {
                    kind: Some(FixKind::ExternalizeSecret),
                    package: String::new(),
                    env_key: env_name_for_header(&key),
                    header_key: key,
                    inline_secret: true,
                    direct_command: String::new(),
                    direct_args: Vec::new(),
                    replacement: String::new(),
                }),
            );
        }
    }

    let combined = format!("{} {} {}", command, args.join(" "), url);
    if local_endpoint(&combined) {
        push_finding(
            report,
            tool,
            path,
            server,
            "mcp_local_endpoint",
            RiskLevel::Medium,
            "MCP server references a localhost or private-network endpoint.",
            &redact_text(&combined),
            "Treat this as machine-local setup state and avoid syncing it unless documented.",
            FixKind::ManualReview,
            None,
        );
    }
    if broad_filesystem(&args) {
        push_finding(
            report,
            tool,
            path,
            server,
            "mcp_broad_filesystem",
            RiskLevel::Medium,
            &format!(
                "MCP server \"{}\" appears to reference broad filesystem access.",
                server
            ),
            &redact_text(&combined),
            "Narrow filesystem arguments to explicit project or vault directories.",
            FixKind::NarrowFilesystem,
            None,
        );
    }
    if local_token_path(&combined) {
        push_finding(
            report,
            tool,
            path,
            server,
            "mcp_local_token_path",
            RiskLevel::High,
            "MCP server references a local credential path.",
            &redact_text(&combined),
            "Keep credential paths out of portable config and document local setup separately.",
            FixKind::ManualReview,
            None,
        );
    }
    push_finding(
        report,
        tool,
        path,
        server,
        "mcp_server_review",
        RiskLevel::Info,
        &format!(
            "Review MCP server \"{}\" before syncing this config.",
            server
        ),
        &redact_text(&combined),
        "Confirm this server is intentional and safe for the target machine before syncing.",
        FixKind::ManualReview,
        None,
    );
}

#[allow(clippy::too_many_arguments)]
fn push_finding(
    report: &mut Report,
    tool: &str,
    path: &Path,
    server: &str,
    rule: &str,
    severity: RiskLevel,
    message: &str,
    evidence: &str,
    action: &str,
    fix_kind: FixKind,
    patch_hint: Option<PatchHint>,
) {
    let docs_url = rules::explain_rule(rule)
        .map(|rule| rule.docs_url.to_string())
        .unwrap_or_else(|| "https://jsonbored.github.io/nightward/reference/rules".to_string());
    report.findings.push(Finding {
        id: finding_id(rule, tool, &path.display().to_string(), server, evidence),
        tool: tool.to_string(),
        path: path.display().to_string(),
        server: server.to_string(),
        severity,
        rule: rule.to_string(),
        message: message.to_string(),
        evidence: evidence.to_string(),
        recommended_action: action.to_string(),
        impact: "Unsafe portable config can expose secrets, stale local state, or unexpected agent capabilities.".to_string(),
        why: "AI agent and MCP configuration often sits in dotfiles and sync folders, so local-only values can leak or break on another machine.".to_string(),
        docs_url,
        fix_available: true,
        fix_kind: Some(fix_kind),
        confidence: "medium".to_string(),
        risk: Some(severity),
        requires_review: true,
        fix_summary: action.to_string(),
        fix_steps: vec![
            "Inspect the redacted evidence.".to_string(),
            action.to_string(),
            "Re-run Nightward and compare the next report.".to_string(),
        ],
        patch_hint,
    });
}

fn finalize_report(report: &mut Report) {
    report.items.sort_by(|a, b| a.path.cmp(&b.path));
    report.findings.sort_by(|a, b| {
        b.severity
            .rank()
            .cmp(&a.severity.rank())
            .then_with(|| a.tool.cmp(&b.tool))
            .then_with(|| rule_sort_rank(&a.rule).cmp(&rule_sort_rank(&b.rule)))
            .then_with(|| a.id.cmp(&b.id))
    });
    report.recompute_summary();
}

fn rule_sort_rank(rule: &str) -> usize {
    match rule {
        "mcp_unpinned_package" => 0,
        "mcp_secret_env" | "mcp_secret_header" => 1,
        "mcp_broad_filesystem" => 2,
        "mcp_local_endpoint" => 3,
        "mcp_server_review" => 4,
        _ => 10,
    }
}

fn str_field(config: &Value, keys: &[&str]) -> String {
    for key in keys {
        if let Some(value) = config.get(*key).and_then(Value::as_str) {
            return value.to_string();
        }
    }
    String::new()
}

fn array_field(config: &Value, keys: &[&str]) -> Vec<String> {
    for key in keys {
        if let Some(values) = config.get(*key).and_then(Value::as_array) {
            return values
                .iter()
                .filter_map(|value| value.as_str().map(ToString::to_string))
                .collect();
        }
    }
    Vec::new()
}

fn object_entries(config: &Value, keys: &[&str]) -> Vec<(String, String)> {
    let mut out = Vec::new();
    for key in keys {
        if let Some(object) = config.get(*key).and_then(Value::as_object) {
            for (entry_key, entry_value) in object {
                let value = entry_value
                    .as_str()
                    .map(ToString::to_string)
                    .unwrap_or_else(|| entry_value.to_string());
                out.push((entry_key.clone(), value));
            }
        }
    }
    out
}

fn unpinned_package(command: &str, args: &[String]) -> Option<String> {
    let command_base = command.rsplit('/').next().unwrap_or(command);
    if !matches!(
        command_base,
        "npx" | "uvx" | "pipx" | "pnpm" | "yarn" | "bunx"
    ) {
        return None;
    }
    for arg in args {
        if arg.starts_with('-') || arg == "exec" || arg == "dlx" {
            continue;
        }
        if package_has_version_pin(arg) {
            return None;
        }
        return Some(arg.clone());
    }
    Some(command.to_string())
}

fn package_has_version_pin(package: &str) -> bool {
    let after_scope = if package.starts_with('@') {
        package
            .split_once('/')
            .map(|(_, rest)| rest)
            .unwrap_or(package)
    } else {
        package
    };
    after_scope.contains('@') && !after_scope.ends_with("@latest")
}

fn shell_command(command: &str) -> bool {
    matches!(
        command.rsplit('/').next().unwrap_or(command),
        "sh" | "bash" | "zsh" | "fish" | "pwsh" | "powershell" | "cmd"
    )
}

fn secret_key(key: &str) -> bool {
    Regex::new("(?i)(token|secret|password|passwd|api[_-]?key|auth|credential|private[_-]?key)")
        .expect("valid regex")
        .is_match(key)
}

fn secret_value(value: &str) -> bool {
    Regex::new(r"\b(sk-[A-Za-z0-9_-]{12,}|gh[pousr]_[A-Za-z0-9_]{20,}|glpat-[A-Za-z0-9_-]{20,}|npm_[A-Za-z0-9]{20,}|xox[abprs]-[A-Za-z0-9-]{20,})\b")
        .expect("valid regex")
        .is_match(value)
}

fn env_reference(value: &str) -> bool {
    Regex::new(r"^\$\{?[A-Za-z_][A-Za-z0-9_]*\}?$")
        .expect("valid regex")
        .is_match(value.trim())
}

fn redact_secret_field_value(value: &str) -> String {
    if env_reference(value) {
        value.trim().to_string()
    } else {
        "[REDACTED]".to_string()
    }
}

pub fn redact_text(value: &str) -> String {
    let assignment = Regex::new(r#"(?i)((?:token|secret|password|passwd|api[_-]?key|auth|credential|private[_-]?key)[\w.-]*\s*[:=]\s*)(["']?)[^"',\s}]+"#)
        .expect("valid regex");
    let provider = Regex::new(r"\b(?:sk-[A-Za-z0-9_-]{12,}|gh[pousr]_[A-Za-z0-9_]{20,}|glpat-[A-Za-z0-9_-]{20,}|npm_[A-Za-z0-9]{20,}|xox[abprs]-[A-Za-z0-9-]{20,}|eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,})\b")
        .expect("valid regex");
    let redacted = assignment.replace_all(value, "$1$2[redacted]");
    provider.replace_all(&redacted, "[redacted]").to_string()
}

fn local_endpoint(value: &str) -> bool {
    Regex::new(r"(?i)\b(localhost|127\.0\.0\.1|0\.0\.0\.0|10\.\d+\.\d+\.\d+|192\.168\.\d+\.\d+|172\.(1[6-9]|2\d|3[01])\.\d+\.\d+)\b")
        .expect("valid regex")
        .is_match(value)
}

fn broad_filesystem(args: &[String]) -> bool {
    args.iter().any(|arg| {
        matches!(arg.as_str(), "$HOME" | "~" | "/" | ".")
            || arg.starts_with("~/")
            || arg.starts_with("/Users/")
            || arg.starts_with("/home/")
    })
}

fn local_token_path(value: &str) -> bool {
    Regex::new("(?i)(auth\\.json|credentials?\\.json|id_ed25519|id_rsa|keychain|token\\.json)")
        .expect("valid regex")
        .is_match(value)
}

fn env_name_for_header(key: &str) -> String {
    key.chars()
        .map(|ch| {
            if ch.is_ascii_alphanumeric() {
                ch.to_ascii_uppercase()
            } else {
                '_'
            }
        })
        .collect::<String>()
}

fn finding_id(rule: &str, tool: &str, path: &str, server: &str, evidence: &str) -> String {
    format!(
        "{rule}-{}",
        stable_id(&[rule, tool, path, server, evidence])
    )
}

pub fn stable_id(parts: &[&str]) -> String {
    let mut hasher = Sha256::new();
    for part in parts {
        hasher.update(part.as_bytes());
        hasher.update([0]);
    }
    hex::encode(hasher.finalize())[..12].to_string()
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::BTreeSet;
    use std::fs;

    #[test]
    fn detects_json_mcp_secret_and_unpinned_package() {
        let dir = tempfile::tempdir().unwrap();
        let config = dir.path().join(".cursor");
        fs::create_dir_all(&config).unwrap();
        let key = ["API_", "KEY"].concat();
        let token = ["sk-", "1234567890abcdef"].concat();
        fs::write(
            config.join("mcp.json"),
            format!(
                r#"{{"mcpServers":{{"demo":{{"command":"npx","args":["@modelcontextprotocol/server-filesystem","$HOME"],"env":{{"{key}":"{token}"}}}}}}}}"#
            ),
        )
        .unwrap();

        let report = scan_home(dir.path()).unwrap();
        let rules: BTreeSet<_> = report.findings.iter().map(|f| f.rule.as_str()).collect();
        assert!(rules.contains("mcp_secret_env"));
        assert!(rules.contains("mcp_unpinned_package"));
        assert!(rules.contains("mcp_broad_filesystem"));
        assert!(report
            .findings
            .iter()
            .all(|finding| !finding.evidence.contains(&token)));
    }

    #[test]
    fn redacts_secret_key_values_even_when_token_shape_is_unknown() {
        let dir = tempfile::tempdir().unwrap();
        let config = dir.path().join(".claude.json");
        let header_key = ["CONTEXT7_", "API_", "KEY"].concat();
        let header_value = ["ctx", "7sk-", "7f4e75c9-e4f3-4e18-b22f-832367f85b48"].concat();
        fs::write(
            &config,
            format!(
                r#"{{"mcpServers":{{"remote":{{"url":"https://example.test/mcp","headers":{{"{header_key}":"{header_value}"}}}}}}}}"#
            ),
        )
        .unwrap();

        let report = scan_home(dir.path()).unwrap();
        let finding = report
            .findings
            .iter()
            .find(|finding| finding.rule == "mcp_secret_header")
            .expect("secret header finding");
        assert!(finding.evidence.contains("[REDACTED]"));
        assert!(!finding.evidence.contains(&header_value));
    }

    #[test]
    fn parses_toml_mcp_servers() {
        let dir = tempfile::tempdir().unwrap();
        let config = dir.path().join(".codex");
        fs::create_dir_all(&config).unwrap();
        fs::write(
            config.join("config.toml"),
            r#"[mcp_servers.demo]
command = "npx"
args = ["package", "http://127.0.0.1:3000"]
"#,
        )
        .unwrap();

        let report = scan_home(dir.path()).unwrap();
        assert!(report
            .findings
            .iter()
            .any(|finding| finding.rule == "mcp_local_endpoint"));
    }

    #[test]
    fn adds_advisory_review_for_clean_mcp_server() {
        let dir = tempfile::tempdir().unwrap();
        let config = dir.path().join(".codex");
        fs::create_dir_all(&config).unwrap();
        fs::write(
            config.join("config.toml"),
            r#"[mcp_servers.notes]
command = "node"
args = ["./tools/notes-mcp.js"]
"#,
        )
        .unwrap();

        let report = scan_home(dir.path()).unwrap();
        assert!(report
            .findings
            .iter()
            .any(|finding| finding.rule == "mcp_server_review"
                && finding.severity == RiskLevel::Info
                && finding.server == "notes"));
    }
}
