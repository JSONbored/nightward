use crate::analysis::{ProviderFinding, SignalCategory};
use crate::inventory::redact_text;
use crate::RiskLevel;
use anyhow::{anyhow, Context, Result};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::env;
use std::io::Read;
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::thread;
use std::time::Duration;
use wait_timeout::ChildExt;

const DEFAULT_STDOUT_CAP: usize = 2 * 1024 * 1024;
const DEFAULT_STDERR_CAP: usize = 64 * 1024;
const DEFAULT_PROVIDER_TIMEOUT: Duration = Duration::from_secs(20);

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Provider {
    pub name: String,
    pub kind: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub command: String,
    pub online: bool,
    pub default: bool,
    pub privacy: String,
    pub capabilities: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderStatus {
    #[serde(flatten)]
    pub provider: Provider,
    pub enabled: bool,
    pub available: bool,
    pub status: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub detail: String,
}

pub fn providers() -> Vec<Provider> {
    vec![
        Provider {
            name: "nightward".to_string(),
            kind: "built-in".to_string(),
            command: String::new(),
            online: false,
            default: true,
            privacy: "local-only".to_string(),
            capabilities: "inventory, MCP config posture, dotfiles safety".to_string(),
        },
        local_provider("gitleaks", "secret scanning"),
        local_provider("trufflehog", "secret scanning"),
        local_provider("semgrep", "local rule scanning"),
        online_provider(
            "trivy",
            "filesystem vulnerability, secret, and misconfig scanning",
        ),
        online_provider("osv-scanner", "dependency vulnerability scanning"),
        Provider {
            name: "socket".to_string(),
            kind: "local-command".to_string(),
            command: "socket".to_string(),
            online: true,
            default: false,
            privacy: "online-capable; creates a remote Socket scan artifact".to_string(),
            capabilities: "dependency risk metadata and Socket scan creation".to_string(),
        },
    ]
}

pub fn statuses(selected: &[String], online: bool) -> Vec<ProviderStatus> {
    let selected = selected_set(selected);
    providers()
        .into_iter()
        .map(|provider| {
            let enabled =
                provider.default || selected.contains("all") || selected.contains(&provider.name);
            let available = provider.kind == "built-in" || which::which(&provider.name).is_ok();
            let (status, detail) = if !enabled {
                (
                    "skipped".to_string(),
                    "provider not selected for this analysis run".to_string(),
                )
            } else if provider.online && !online {
                (
                    "blocked".to_string(),
                    "online-capable provider requires --online or allow_online_providers"
                        .to_string(),
                )
            } else if provider.kind == "built-in" {
                (
                    "ready".to_string(),
                    "Nightward built-in analysis".to_string(),
                )
            } else if available {
                ("ready".to_string(), "command found on PATH".to_string())
            } else {
                (
                    "missing".to_string(),
                    "command not found on PATH".to_string(),
                )
            };
            ProviderStatus {
                provider,
                enabled,
                available,
                status,
                detail,
            }
        })
        .collect()
}

pub fn run_selected(
    root: &Path,
    selected: &[String],
    online: bool,
) -> Vec<(String, Result<Vec<ProviderFinding>>)> {
    let selected = selected_set(selected);
    let mut out = Vec::new();
    for status in statuses(
        selected.iter().cloned().collect::<Vec<_>>().as_slice(),
        online,
    ) {
        if !status.enabled || status.provider.kind != "local-command" {
            continue;
        }
        if status.provider.online && !online {
            out.push((
                status.provider.name.clone(),
                Err(anyhow!("online-capable provider requires --online")),
            ));
            continue;
        }
        if !selected.contains("all") && !selected.contains(&status.provider.name) {
            continue;
        }
        if status.status != "ready" {
            out.push((status.provider.name.clone(), Err(anyhow!(status.detail))));
            continue;
        }
        out.push((
            status.provider.name.clone(),
            run_provider(&status.provider.name, root),
        ));
    }
    out
}

pub fn run_provider(name: &str, root: &Path) -> Result<Vec<ProviderFinding>> {
    let args = provider_args(name, root)?;
    let timeout = provider_timeout();
    let stdout_cap = provider_stdout_cap();
    let stderr_cap = provider_stderr_cap();
    let mut child = Command::new(name)
        .args(&args)
        .current_dir(root)
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .with_context(|| format!("spawn provider {name}"))?;

    let stdout_handle = child.stdout.take().map(|mut stream| {
        thread::spawn(move || {
            let mut buf = Vec::new();
            let _ = stream
                .by_ref()
                .take(stdout_cap as u64 + 1)
                .read_to_end(&mut buf);
            buf
        })
    });
    let stderr_handle = child.stderr.take().map(|mut stream| {
        thread::spawn(move || {
            let mut buf = Vec::new();
            let _ = stream
                .by_ref()
                .take(stderr_cap as u64 + 1)
                .read_to_end(&mut buf);
            buf
        })
    });

    let status = match child.wait_timeout(timeout)? {
        Some(status) => status,
        None => {
            let _ = child.kill();
            let _ = child.wait();
            return Err(anyhow!("provider timed out after {:?}", timeout));
        }
    };

    let stdout = stdout_handle
        .map(|handle| handle.join().unwrap_or_default())
        .unwrap_or_default();
    let stderr = stderr_handle
        .map(|handle| handle.join().unwrap_or_default())
        .unwrap_or_default();
    let (stdout, stdout_truncated) = capped_string(stdout, stdout_cap);
    let (stderr, _) = capped_string(stderr, stderr_cap);
    if stdout_truncated {
        return Err(anyhow!("provider stdout exceeded {stdout_cap} byte cap"));
    }
    if !status.success() && stdout.trim().is_empty() {
        return Err(anyhow!("{}: {}", status, first_line(&stderr)));
    }
    parse_provider_output(name, root, &stdout)
}

pub fn parse_provider_output(
    name: &str,
    root: &Path,
    output: &str,
) -> Result<Vec<ProviderFinding>> {
    match name {
        "gitleaks" => parse_gitleaks(root, output),
        "trufflehog" => parse_trufflehog(root, output),
        "semgrep" => parse_semgrep(root, output),
        "trivy" => parse_trivy(root, output),
        "osv-scanner" => parse_osv(root, output),
        "socket" => parse_socket(root, output),
        _ => Ok(Vec::new()),
    }
}

fn local_provider(name: &str, capabilities: &str) -> Provider {
    Provider {
        name: name.to_string(),
        kind: "local-command".to_string(),
        command: name.to_string(),
        online: false,
        default: false,
        privacy: "local command; no network enabled by Nightward".to_string(),
        capabilities: capabilities.to_string(),
    }
}

fn online_provider(name: &str, capabilities: &str) -> Provider {
    Provider {
        name: name.to_string(),
        kind: "local-command".to_string(),
        command: name.to_string(),
        online: true,
        default: false,
        privacy: "online-capable; blocked unless explicitly enabled".to_string(),
        capabilities: capabilities.to_string(),
    }
}

fn selected_set(selected: &[String]) -> std::collections::BTreeSet<String> {
    selected
        .iter()
        .flat_map(|value| value.split(','))
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(|value| value.to_ascii_lowercase())
        .collect()
}

fn provider_args(name: &str, root: &Path) -> Result<Vec<String>> {
    let root = root.display().to_string();
    Ok(match name {
        "gitleaks" => vec![
            "detect",
            "--no-git",
            "--redact",
            "--no-banner",
            "--source",
            &root,
            "--report-format",
            "json",
            "--exit-code",
            "0",
        ]
        .into_iter()
        .map(str::to_string)
        .collect(),
        "trufflehog" => vec!["filesystem", "--json", "--no-update", &root]
            .into_iter()
            .map(str::to_string)
            .collect(),
        "semgrep" => {
            let config = local_semgrep_config(Path::new(&root))
                .ok_or_else(|| anyhow!("semgrep local config not found"))?;
            vec![
                "scan".to_string(),
                "--json".to_string(),
                "--metrics=off".to_string(),
                "--disable-version-check".to_string(),
                "--config".to_string(),
                config.display().to_string(),
                root,
            ]
        }
        "trivy" => vec![
            "filesystem",
            "--format",
            "json",
            "--scanners",
            "vuln,secret,misconfig",
            "--skip-version-check",
            &root,
        ]
        .into_iter()
        .map(str::to_string)
        .collect(),
        "osv-scanner" => vec!["scan", "source", "-r", "--format", "json", &root]
            .into_iter()
            .map(str::to_string)
            .collect(),
        "socket" => vec!["scan", "create", &root, "--json"]
            .into_iter()
            .map(str::to_string)
            .collect(),
        _ => return Err(anyhow!("unknown provider {name}")),
    })
}

fn local_semgrep_config(root: &Path) -> Option<PathBuf> {
    [
        "semgrep.yml",
        "semgrep.yaml",
        ".semgrep.yml",
        ".semgrep.yaml",
        ".semgrep/config.yml",
        ".semgrep/config.yaml",
    ]
    .into_iter()
    .map(|rel| root.join(rel))
    .find(|path| path.is_file())
}

fn parse_gitleaks(root: &Path, output: &str) -> Result<Vec<ProviderFinding>> {
    if output.trim().is_empty() {
        return Ok(Vec::new());
    }
    let records: Vec<Value> = serde_json::from_str(output)?;
    Ok(records
        .into_iter()
        .map(|record| {
            let rule = first_string(&record, &["RuleID", "ruleID", "Rule", "rule"])
                .unwrap_or_else(|| "secret".to_string());
            let file = first_string(&record, &["File", "file"]).unwrap_or_default();
            ProviderFinding {
                rule,
                path: normalize_provider_path(root, &file),
                message: "Gitleaks reported a secret-like value.".to_string(),
                evidence: redact_text(&record.to_string()),
                severity: RiskLevel::Critical,
                category: SignalCategory::SecretsExposure,
            }
        })
        .collect())
}

fn parse_trufflehog(root: &Path, output: &str) -> Result<Vec<ProviderFinding>> {
    let mut out = Vec::new();
    for line in output.lines().filter(|line| !line.trim().is_empty()) {
        let value: Value = serde_json::from_str(line)?;
        let path = first_string(
            &value,
            &["SourceMetadata.Data.Filesystem.file", "path", "file"],
        )
        .unwrap_or_default();
        let detector = first_string(&value, &["DetectorName", "detector_name"])
            .unwrap_or_else(|| "secret".to_string());
        out.push(ProviderFinding {
            rule: detector,
            path: normalize_provider_path(root, &path),
            message: "TruffleHog reported a verified or likely secret.".to_string(),
            evidence: redact_text(&value.to_string()),
            severity: RiskLevel::Critical,
            category: SignalCategory::SecretsExposure,
        });
    }
    Ok(out)
}

fn parse_semgrep(root: &Path, output: &str) -> Result<Vec<ProviderFinding>> {
    if output.trim().is_empty() {
        return Ok(Vec::new());
    }
    let value: Value = serde_json::from_str(output)?;
    let mut out = Vec::new();
    for result in value
        .get("results")
        .and_then(Value::as_array)
        .into_iter()
        .flatten()
    {
        let check_id = first_string(result, &["check_id"]).unwrap_or_else(|| "semgrep".to_string());
        let path = first_string(result, &["path"]).unwrap_or_default();
        let message = nested_string(result, &["extra", "message"])
            .unwrap_or_else(|| "Semgrep reported a rule match.".to_string());
        out.push(ProviderFinding {
            rule: check_id,
            path: normalize_provider_path(root, &path),
            message: redact_text(&message),
            evidence: redact_text(&result.to_string()),
            severity: RiskLevel::Medium,
            category: SignalCategory::ExecutionRisk,
        });
    }
    Ok(out)
}

fn parse_trivy(root: &Path, output: &str) -> Result<Vec<ProviderFinding>> {
    if output.trim().is_empty() {
        return Ok(Vec::new());
    }
    let value: Value = serde_json::from_str(output)?;
    let mut out = Vec::new();
    for result in value
        .get("Results")
        .and_then(Value::as_array)
        .into_iter()
        .flatten()
    {
        let target = first_string(result, &["Target"]).unwrap_or_default();
        for vuln in result
            .get("Vulnerabilities")
            .and_then(Value::as_array)
            .into_iter()
            .flatten()
        {
            let id = first_string(vuln, &["VulnerabilityID"])
                .unwrap_or_else(|| "vulnerability".to_string());
            let package = first_string(vuln, &["PkgName"]).unwrap_or_default();
            out.push(ProviderFinding {
                rule: id.clone(),
                path: normalize_provider_path(root, &target),
                message: redact_text(&format!("Trivy reported {id} in {package}.")),
                evidence: redact_text(&vuln.to_string()),
                severity: trivy_severity(vuln),
                category: SignalCategory::SupplyChain,
            });
        }
        for secret in result
            .get("Secrets")
            .and_then(Value::as_array)
            .into_iter()
            .flatten()
        {
            let rule =
                first_string(secret, &["RuleID", "Title"]).unwrap_or_else(|| "secret".to_string());
            out.push(ProviderFinding {
                rule,
                path: normalize_provider_path(root, &target),
                message: "Trivy reported a secret-like value.".to_string(),
                evidence: redact_text(&secret.to_string()),
                severity: RiskLevel::Critical,
                category: SignalCategory::SecretsExposure,
            });
        }
        for misconfig in result
            .get("Misconfigurations")
            .and_then(Value::as_array)
            .into_iter()
            .flatten()
        {
            let rule = first_string(misconfig, &["ID", "AVDID"])
                .unwrap_or_else(|| "misconfiguration".to_string());
            let title = first_string(misconfig, &["Title"])
                .unwrap_or_else(|| "Trivy reported a misconfiguration.".to_string());
            out.push(ProviderFinding {
                rule,
                path: normalize_provider_path(root, &target),
                message: redact_text(&title),
                evidence: redact_text(&misconfig.to_string()),
                severity: trivy_severity(misconfig),
                category: SignalCategory::ExecutionRisk,
            });
        }
    }
    Ok(out)
}

fn parse_osv(root: &Path, output: &str) -> Result<Vec<ProviderFinding>> {
    if output.trim().is_empty() {
        return Ok(Vec::new());
    }
    let value: Value = serde_json::from_str(output)?;
    let mut out = Vec::new();
    collect_osv_results(root, &value, &mut out);
    Ok(out)
}

fn collect_osv_results(root: &Path, value: &Value, out: &mut Vec<ProviderFinding>) {
    if let Some(vulns) = value.get("vulnerabilities").and_then(Value::as_array) {
        let package = nested_string(value, &["package", "name"])
            .or_else(|| first_string(value, &["package"]))
            .unwrap_or_default();
        let source = first_string(value, &["source", "path", "lockfile"]).unwrap_or_default();
        for vuln in vulns {
            let id =
                first_string(vuln, &["id", "ID"]).unwrap_or_else(|| "vulnerability".to_string());
            out.push(ProviderFinding {
                rule: id.clone(),
                path: normalize_provider_path(root, &source),
                message: redact_text(&format!("OSV reported {id} for {package}.")),
                evidence: redact_text(&vuln.to_string()),
                severity: RiskLevel::High,
                category: SignalCategory::SupplyChain,
            });
        }
    }
    match value {
        Value::Array(values) => {
            for child in values {
                collect_osv_results(root, child, out);
            }
        }
        Value::Object(object) => {
            for child in object.values() {
                collect_osv_results(root, child, out);
            }
        }
        _ => {}
    }
}

fn parse_socket(root: &Path, output: &str) -> Result<Vec<ProviderFinding>> {
    if output.trim().is_empty() {
        return Ok(Vec::new());
    }
    let value: Value = serde_json::from_str(output)?;
    let mut out = Vec::new();
    let issue_arrays = ["issues", "alerts", "vulnerabilities", "findings"];
    for key in issue_arrays {
        for issue in value
            .get(key)
            .and_then(Value::as_array)
            .into_iter()
            .flatten()
        {
            let rule = first_string(issue, &["type", "rule", "code", "id"])
                .unwrap_or_else(|| "socket_issue".to_string());
            let path = first_string(issue, &["file", "path", "manifest"]).unwrap_or_default();
            let message = first_string(issue, &["message", "title", "description"])
                .unwrap_or_else(|| "Socket reported a dependency risk signal.".to_string());
            out.push(ProviderFinding {
                rule,
                path: normalize_provider_path(root, &path),
                message: redact_text(&message),
                evidence: redact_text(&issue.to_string()),
                severity: RiskLevel::Medium,
                category: SignalCategory::SupplyChain,
            });
        }
    }
    if out.is_empty() {
        if let Some(scan_id) = first_string(&value, &["scanId", "scan_id", "id"]) {
            out.push(ProviderFinding {
                rule: "socket_scan_created".to_string(),
                path: root.display().to_string(),
                message: "Socket created a remote scan artifact; Nightward did not fetch a remote report.".to_string(),
                evidence: redact_text(&format!("scan_id={scan_id}")),
                severity: RiskLevel::Info,
                category: SignalCategory::SupplyChain,
            });
        }
    }
    Ok(out)
}

fn trivy_severity(value: &Value) -> RiskLevel {
    match first_string(value, &["Severity"])
        .unwrap_or_default()
        .to_ascii_uppercase()
        .as_str()
    {
        "CRITICAL" => RiskLevel::Critical,
        "HIGH" => RiskLevel::High,
        "MEDIUM" => RiskLevel::Medium,
        "LOW" => RiskLevel::Low,
        _ => RiskLevel::Info,
    }
}

fn first_string(value: &Value, keys: &[&str]) -> Option<String> {
    for key in keys {
        if key.contains('.') {
            if let Some(value) = nested_string(value, &key.split('.').collect::<Vec<_>>()) {
                return Some(value);
            }
        } else if let Some(found) = value.get(*key).and_then(Value::as_str) {
            return Some(found.to_string());
        }
    }
    None
}

fn nested_string(value: &Value, keys: &[&str]) -> Option<String> {
    let mut current = value;
    for key in keys {
        current = current.get(*key)?;
    }
    current.as_str().map(ToString::to_string)
}

fn normalize_provider_path(root: &Path, path: &str) -> String {
    if path.is_empty() {
        return root.display().to_string();
    }
    let path = Path::new(path);
    if path.is_absolute() {
        path.display().to_string()
    } else {
        root.join(path).display().to_string()
    }
}

fn provider_timeout() -> Duration {
    env::var("NIGHTWARD_PROVIDER_TIMEOUT_MS")
        .ok()
        .and_then(|value| value.parse::<u64>().ok())
        .map(Duration::from_millis)
        .unwrap_or(DEFAULT_PROVIDER_TIMEOUT)
}

fn provider_stdout_cap() -> usize {
    env::var("NIGHTWARD_PROVIDER_STDOUT_CAP")
        .ok()
        .and_then(|value| value.parse::<usize>().ok())
        .unwrap_or(DEFAULT_STDOUT_CAP)
}

fn provider_stderr_cap() -> usize {
    env::var("NIGHTWARD_PROVIDER_STDERR_CAP")
        .ok()
        .and_then(|value| value.parse::<usize>().ok())
        .unwrap_or(DEFAULT_STDERR_CAP)
}

fn capped_string(bytes: Vec<u8>, cap: usize) -> (String, bool) {
    let truncated = bytes.len() > cap;
    let mut value = String::from_utf8_lossy(&bytes[..bytes.len().min(cap)]).to_string();
    if truncated {
        value.push_str("\n[provider output truncated]");
    }
    (redact_text(&value), truncated)
}

fn first_line(value: &str) -> String {
    value
        .lines()
        .find(|line| !line.trim().is_empty())
        .unwrap_or("")
        .to_string()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_trivy_vulns_secrets_and_misconfigurations_with_redaction() {
        let root = Path::new("/tmp/project");
        let key = ["API_", "KEY"].concat();
        let token = ["sk-", "1234567890abcdef"].concat();
        let json = format!(
            r#"{{"Results":[{{"Target":"package-lock.json","Vulnerabilities":[{{"VulnerabilityID":"CVE-1","PkgName":"demo","Severity":"HIGH"}}],"Secrets":[{{"RuleID":"secret","Match":"{key}={token}"}}],"Misconfigurations":[{{"ID":"AVD-1","Title":"bad config","Severity":"MEDIUM"}}]}}]}}"#
        );
        let findings = parse_trivy(root, &json).unwrap();
        assert_eq!(findings.len(), 3);
        assert!(findings
            .iter()
            .all(|finding| !finding.evidence.contains(&token)));
        assert!(findings.iter().any(|finding| finding.rule == "CVE-1"));
    }

    #[test]
    fn parses_semgrep_without_leaking_bearer_tokens() {
        let root = Path::new("/tmp/project");
        let token = ["opaque", "-secret", "-12345"].concat();
        let json = format!(
            r#"{{"results":[{{"check_id":"nightward.secret","path":"mcp.json","extra":{{"message":"Authorization: Bearer {token}"}}}}]}}"#
        );
        let findings = parse_semgrep(root, &json).unwrap();

        assert_eq!(findings.len(), 1);
        assert!(!findings[0].message.contains(&token));
        assert!(!findings[0].evidence.contains(&token));
    }

    #[test]
    fn parses_osv_nested_results() {
        let root = Path::new("/tmp/project");
        let json = r#"{"results":[{"source":{"path":"package-lock.json"},"packages":[{"package":{"name":"leftpad"},"vulnerabilities":[{"id":"GHSA-demo"}]}]}]}"#;
        let findings = parse_osv(root, json).unwrap();
        assert!(findings.iter().any(|finding| finding.rule == "GHSA-demo"));
    }

    #[test]
    fn parses_socket_scan_id_only_response() {
        let root = Path::new("/tmp/project");
        let findings = parse_socket(root, r#"{"scanId":"scan_123"}"#).unwrap();
        assert_eq!(findings.len(), 1);
        assert_eq!(findings[0].rule, "socket_scan_created");
    }

    #[test]
    fn statuses_block_online_without_gate() {
        let statuses = statuses(&["trivy".to_string()], false);
        let trivy = statuses
            .iter()
            .find(|status| status.provider.name == "trivy")
            .unwrap();
        assert_eq!(trivy.status, "blocked");
    }
}
