use crate::analysis::{run as analyze, Options as AnalysisOptions};
use crate::fixplan::{plan as fix_plan, Selector};
use crate::inventory::{home_dir_from_env, scan_home, scan_workspace};
use crate::policy::{check as policy_check, PolicyConfig};
use crate::rules;
use anyhow::{anyhow, Result};
use serde_json::{json, Value};
use std::io::{self, BufRead, Write};
use std::path::PathBuf;

const PROTOCOL_VERSION: &str = "2025-06-18";

pub fn serve() -> Result<()> {
    let stdin = io::stdin();
    let mut stdout = io::stdout();
    for line in stdin.lock().lines() {
        let line = line?;
        if line.trim().is_empty() {
            continue;
        }
        let request: Value = serde_json::from_str(&line)?;
        let response = handle_request(request);
        writeln!(stdout, "{}", serde_json::to_string(&response)?)?;
        stdout.flush()?;
    }
    Ok(())
}

pub fn handle_request(request: Value) -> Value {
    let id = request.get("id").cloned().unwrap_or(Value::Null);
    let method = request
        .get("method")
        .and_then(Value::as_str)
        .unwrap_or_default();
    let result = match method {
        "initialize" => Ok(json!({
            "protocolVersion": PROTOCOL_VERSION,
            "capabilities": {
                "tools": {},
                "resources": {}
            },
            "serverInfo": {
                "name": "nightward",
                "version": env!("CARGO_PKG_VERSION")
            }
        })),
        "ping" => Ok(json!({})),
        "tools/list" => Ok(json!({ "tools": tools() })),
        "resources/list" => Ok(json!({ "resources": resources() })),
        "resources/read" => read_resource(request.get("params").cloned().unwrap_or_default()),
        "tools/call" => call_tool(request.get("params").cloned().unwrap_or_default()),
        _ => Err(anyhow!("unknown method {method}")),
    };
    match result {
        Ok(result) => json!({ "jsonrpc": "2.0", "id": id, "result": result }),
        Err(error) => json!({
            "jsonrpc": "2.0",
            "id": id,
            "error": { "code": -32000, "message": error.to_string() }
        }),
    }
}

fn tools() -> Vec<Value> {
    vec![
        tool(
            "nightward_scan",
            "Scan home or workspace with Nightward read-only defaults.",
        ),
        tool("nightward_doctor", "Return provider and schedule posture."),
        tool(
            "nightward_findings",
            "Return current findings with optional severity filtering.",
        ),
        tool(
            "nightward_explain_finding",
            "Explain one finding by id or prefix.",
        ),
        tool(
            "nightward_fix_plan",
            "Generate a plan-only remediation preview.",
        ),
        tool("nightward_policy_check", "Run a read-only policy check."),
        tool("nightward_rules", "List Nightward rules."),
    ]
}

fn resources() -> Vec<Value> {
    vec![
        json!({
            "uri": "nightward://latest-summary",
            "name": "Latest Nightward summary",
            "mimeType": "application/json"
        }),
        json!({
            "uri": "nightward://rules",
            "name": "Nightward rules",
            "mimeType": "application/json"
        }),
    ]
}

fn tool(name: &str, description: &str) -> Value {
    json!({
        "name": name,
        "description": description,
        "inputSchema": {
            "type": "object",
            "additionalProperties": true,
            "properties": {
                "workspace": { "type": "string" },
                "severity": { "type": "string" },
                "id": { "type": "string" },
                "compact": { "type": "boolean" }
            }
        }
    })
}

fn read_resource(params: Value) -> Result<Value> {
    let uri = params
        .get("uri")
        .and_then(Value::as_str)
        .unwrap_or_default();
    match uri {
        "nightward://latest-summary" => {
            let report = scan_home(home_dir_from_env())?;
            text_resource(uri, serde_json::to_string_pretty(&report.summary)?)
        }
        "nightward://rules" => {
            text_resource(uri, serde_json::to_string_pretty(&rules::all_rules())?)
        }
        _ => Err(anyhow!("unknown resource {uri}")),
    }
}

fn call_tool(params: Value) -> Result<Value> {
    let name = params
        .get("name")
        .and_then(Value::as_str)
        .unwrap_or_default();
    let args = params.get("arguments").cloned().unwrap_or_default();
    let workspace = args
        .get("workspace")
        .and_then(Value::as_str)
        .unwrap_or_default();
    let scan = if workspace.is_empty() {
        scan_home(home_dir_from_env())?
    } else {
        scan_workspace(PathBuf::from(workspace))?
    };
    match name {
        "nightward_scan" => text_result(serde_json::to_string_pretty(&scan)?),
        "nightward_doctor" => {
            let doctor = json!({
                "schema_version": 1,
                "providers": crate::providers::statuses(&[], false),
                "schedule": crate::schedule::status(home_dir_from_env())
            });
            text_result(serde_json::to_string_pretty(&doctor)?)
        }
        "nightward_findings" => {
            let severity = args
                .get("severity")
                .and_then(Value::as_str)
                .unwrap_or_default();
            let findings: Vec<_> = scan
                .findings
                .iter()
                .filter(|finding| {
                    severity.is_empty()
                        || format!("{:?}", finding.severity).eq_ignore_ascii_case(severity)
                })
                .cloned()
                .collect();
            text_result(serde_json::to_string_pretty(&findings)?)
        }
        "nightward_explain_finding" => {
            let id = args.get("id").and_then(Value::as_str).unwrap_or_default();
            let Some(finding) = scan
                .findings
                .iter()
                .find(|finding| finding.id == id || finding.id.starts_with(id))
            else {
                return Err(anyhow!("finding not found"));
            };
            text_result(serde_json::to_string_pretty(finding)?)
        }
        "nightward_fix_plan" => {
            let id = args.get("id").and_then(Value::as_str).unwrap_or_default();
            let selector = Selector {
                all: id.is_empty(),
                finding: id.to_string(),
                rule: String::new(),
            };
            text_result(serde_json::to_string_pretty(&fix_plan(&scan, selector))?)
        }
        "nightward_policy_check" => {
            let analysis = analyze(
                &scan,
                AnalysisOptions {
                    mode: scan.scan_mode.clone(),
                    workspace: scan.workspace.clone(),
                    with: Vec::new(),
                    online: false,
                    package: String::new(),
                    finding_id: String::new(),
                },
            );
            let policy = policy_check(&scan, &PolicyConfig::default(), Some(&analysis));
            let compact = args
                .get("compact")
                .and_then(Value::as_bool)
                .unwrap_or(false);
            if compact {
                text_result(serde_json::to_string_pretty(&json!({
                    "passed": policy.passed,
                    "blocking_count": policy.blocking_count,
                    "finding_count": policy.finding_count,
                    "max_severity": policy.max_severity,
                }))?)
            } else {
                text_result(serde_json::to_string_pretty(&policy)?)
            }
        }
        "nightward_rules" => text_result(serde_json::to_string_pretty(&rules::all_rules())?),
        _ => Err(anyhow!("unknown tool {name}")),
    }
}

fn text_result(text: String) -> Result<Value> {
    Ok(json!({
        "content": [{ "type": "text", "text": text }]
    }))
}

fn text_resource(uri: &str, text: String) -> Result<Value> {
    Ok(json!({
        "contents": [{
            "uri": uri,
            "mimeType": "application/json",
            "text": text
        }]
    }))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn lists_tools() {
        let response = handle_request(json!({"jsonrpc":"2.0","id":1,"method":"tools/list"}));
        assert!(response["result"]["tools"].as_array().unwrap().len() >= 5);
    }
}
