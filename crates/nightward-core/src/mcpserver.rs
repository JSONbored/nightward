use crate::actions;
use crate::analysis::{run as analyze, Options as AnalysisOptions};
use crate::fixplan::{plan as fix_plan, Selector};
use crate::inventory::{home_dir_from_env, load_report, redact_text, scan_home, scan_workspace};
use crate::policy::{check as policy_check, PolicyConfig};
use crate::{approvals, providers, reportdiff, rules, schedule, state};
use anyhow::{anyhow, Context, Result};
use serde::Serialize;
use serde_json::{json, Map, Value};
use std::fs;
use std::io::{self, BufRead, Write};
use std::path::{Component, Path, PathBuf};

const PROTOCOL_LATEST: &str = "2025-11-25";
const PROTOCOL_COMPAT: &str = "2025-06-18";
const SUPPORTED_PROTOCOLS: &[&str] = &[PROTOCOL_LATEST, PROTOCOL_COMPAT];

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
        if response.is_null() {
            continue;
        }
        writeln!(stdout, "{}", serde_json::to_string(&response)?)?;
        stdout.flush()?;
    }
    Ok(())
}

pub fn handle_request(request: Value) -> Value {
    handle_request_with_home(request, &home_dir_from_env())
}

fn handle_request_with_home(request: Value, home: &Path) -> Value {
    let id = request.get("id").cloned();
    let method = request
        .get("method")
        .and_then(Value::as_str)
        .unwrap_or_default();

    if id.is_none() && method.starts_with("notifications/") {
        return Value::Null;
    }

    let result = match method {
        "initialize" => Ok(initialize_result(
            request
                .get("params")
                .and_then(|params| params.get("protocolVersion"))
                .and_then(Value::as_str),
        )),
        "ping" => Ok(json!({})),
        "tools/list" => Ok(json!({ "tools": tools() })),
        "resources/list" => Ok(json!({ "resources": resources() })),
        "resources/read" => read_resource(request.get("params").cloned().unwrap_or_default(), home),
        "prompts/list" => Ok(json!({ "prompts": prompts() })),
        "prompts/get" => read_prompt(request.get("params").cloned().unwrap_or_default()),
        "tools/call" => Ok(call_tool(
            request.get("params").cloned().unwrap_or_default(),
            home,
        )),
        _ => Err(anyhow!("unknown method {method}")),
    };

    let id = id.unwrap_or(Value::Null);
    match result {
        Ok(result) => json!({ "jsonrpc": "2.0", "id": id, "result": result }),
        Err(error) => json!({
            "jsonrpc": "2.0",
            "id": id,
            "error": { "code": -32000, "message": error.to_string() }
        }),
    }
}

fn initialize_result(requested: Option<&str>) -> Value {
    let protocol_version = requested
        .filter(|version| SUPPORTED_PROTOCOLS.contains(version))
        .unwrap_or(PROTOCOL_LATEST);
    json!({
        "protocolVersion": protocol_version,
        "capabilities": {
            "tools": { "listChanged": false },
            "resources": { "subscribe": false, "listChanged": false },
            "prompts": { "listChanged": false }
        },
        "serverInfo": {
            "name": "nightward",
            "title": "Nightward",
            "version": env!("CARGO_PKG_VERSION"),
            "description": "Local-first AI agent, MCP, provider, and dotfiles security posture."
        },
        "instructions": "Nightward returns redacted local security posture. MCP can request bounded action approvals, but local writes require an out-of-band Nightward approval from the CLI, TUI, or Raycast extension before MCP can apply the exact approved ticket."
    })
}

fn tools() -> Vec<Value> {
    vec![
        tool(
            "nightward_scan",
            "Nightward Scan",
            "Run a redacted HOME or workspace scan.",
            schema_scan(),
            read_only_annotations("Nightward scan", false),
        ),
        tool(
            "nightward_doctor",
            "Nightward Doctor",
            "Return provider, schedule, disclosure, and settings posture.",
            schema_provider_context(),
            read_only_annotations("Nightward doctor", false),
        ),
        tool(
            "nightward_findings",
            "Nightward Findings",
            "Return findings with optional severity, rule, and limit filters.",
            schema_findings(),
            read_only_annotations("Nightward findings", false),
        ),
        tool(
            "nightward_explain_finding",
            "Explain Finding",
            "Return one finding by full ID or unique prefix.",
            schema_id_only(),
            read_only_annotations("Explain finding", false),
        ),
        tool(
            "nightward_analysis",
            "Nightward Analysis",
            "Run Nightward analysis with selected local or explicitly allowed online providers.",
            schema_analysis(),
            read_only_annotations("Nightward analysis", true),
        ),
        tool(
            "nightward_explain_signal",
            "Explain Signal",
            "Return one analysis signal by full ID or unique prefix.",
            schema_explain_signal(),
            read_only_annotations("Explain signal", true),
        ),
        tool(
            "nightward_policy_check",
            "Policy Check",
            "Run a read-only Nightward policy check.",
            schema_policy_check(),
            read_only_annotations("Policy check", true),
        ),
        tool(
            "nightward_fix_plan",
            "Fix Plan",
            "Generate plan-only remediation directions for all findings, one finding, or one rule.",
            schema_fix_plan(),
            read_only_annotations("Fix plan", false),
        ),
        tool(
            "nightward_report_history",
            "Report History",
            "List saved scheduled report history.",
            no_args_schema(),
            read_only_annotations("Report history", false),
        ),
        tool(
            "nightward_report_changes",
            "Report Changes",
            "Compare two saved report files, or the latest two saved reports when paths are omitted.",
            schema_report_changes(),
            read_only_annotations("Report changes", false),
        ),
        tool(
            "nightward_actions_list",
            "Actions List",
            "List bounded Nightward actions available through the shared action registry.",
            no_args_schema(),
            read_only_annotations("Actions list", false),
        ),
        tool(
            "nightward_action_preview",
            "Action Preview",
            "Preview one bounded Nightward action before applying it.",
            schema_action_id(),
            read_only_annotations("Action preview", false),
        ),
        tool(
            "nightward_action_request",
            "Action Request",
            "Request local approval for one exact bounded Nightward action. This only writes Nightward approval state.",
            schema_action_request(),
            write_annotations("Action request", false, false),
        ),
        tool(
            "nightward_action_status",
            "Action Status",
            "Read one Nightward action approval request status.",
            schema_approval_id(),
            read_only_annotations("Action approval status", false),
        ),
        tool(
            "nightward_action_apply_approved",
            "Apply Approved Action",
            "Apply an already-approved, unexpired, one-time Nightward action ticket.",
            schema_approval_id(),
            write_annotations("Apply approved action", true, false),
        ),
        tool(
            "nightward_rules",
            "Nightward Rules",
            "List Nightward rules and remediation metadata.",
            no_args_schema(),
            read_only_annotations("Rules", false),
        ),
        tool(
            "nightward_providers",
            "Nightward Providers",
            "List provider capabilities and current status.",
            schema_provider_context(),
            read_only_annotations("Providers", false),
        ),
    ]
}

fn resources() -> Vec<Value> {
    vec![
        resource(
            "nightward://latest-summary",
            "Latest Nightward summary",
            "Live HOME scan summary if no saved report is requested.",
        ),
        resource(
            "nightward://latest-report",
            "Latest Nightward report",
            "Latest saved report when available, otherwise a live HOME scan report.",
        ),
        resource(
            "nightward://rules",
            "Nightward rules",
            "Rule catalog and remediation metadata.",
        ),
        resource(
            "nightward://providers",
            "Nightward providers",
            "Provider catalog, configured selections, and local status.",
        ),
        resource(
            "nightward://schedule",
            "Nightward schedule",
            "User-level scheduled scan status and report paths.",
        ),
        resource(
            "nightward://actions",
            "Nightward actions",
            "Bounded action registry with availability and risk metadata.",
        ),
        resource(
            "nightward://disclosure",
            "Nightward disclosure",
            "Disclosure acceptance status and responsibility text.",
        ),
        resource(
            "nightward://action-approvals",
            "Nightward action approvals",
            "Pending and recent MCP action approval requests.",
        ),
        resource(
            "nightward://report-history",
            "Nightward report history",
            "Saved scheduled report history.",
        ),
    ]
}

fn prompts() -> Vec<Value> {
    prompt(
        "audit_my_ai_setup",
        "Audit My AI Setup",
        "Have an AI client run Nightward scan, analysis, and policy checks, then summarize local AI/MCP risk.",
        &[],
    )
    .into_iter()
    .chain(prompt(
        "explain_top_risks",
        "Explain Top Risks",
        "Explain the highest-severity Nightward findings and signals in plain language.",
        &[],
    ))
    .chain(prompt(
        "fix_this_finding",
        "Fix This Finding Safely",
        "Generate a cautious fix plan for a specific finding without mutating raw agent config.",
        &[("finding_id", "Finding ID or unique prefix.")],
    ))
    .chain(prompt(
        "set_up_providers",
        "Set Up Providers",
        "Review provider status and propose bounded provider install/enable actions.",
        &[],
    ))
    .chain(prompt(
        "compare_reports",
        "Compare Reports",
        "Compare the last two saved Nightward reports and explain what changed.",
        &[],
    ))
    .collect()
}

fn prompt(name: &str, title: &str, description: &str, arguments: &[(&str, &str)]) -> Vec<Value> {
    vec![json!({
        "name": name,
        "title": title,
        "description": description,
        "arguments": arguments
            .iter()
            .map(|(name, description)| json!({
                "name": name,
                "description": description,
                "required": true
            }))
            .collect::<Vec<_>>()
    })]
}

fn tool(
    name: &str,
    title: &str,
    description: &str,
    input_schema: Value,
    annotations: Value,
) -> Value {
    json!({
        "name": name,
        "title": title,
        "description": description,
        "inputSchema": input_schema,
        "outputSchema": {
            "type": "object",
            "additionalProperties": true
        },
        "annotations": annotations,
        "execution": {
            "taskSupport": "forbidden"
        }
    })
}

fn resource(uri: &str, name: &str, description: &str) -> Value {
    json!({
        "uri": uri,
        "name": name,
        "description": description,
        "mimeType": "application/json"
    })
}

fn read_only_annotations(title: &str, open_world: bool) -> Value {
    json!({
        "title": title,
        "readOnlyHint": true,
        "destructiveHint": false,
        "idempotentHint": true,
        "openWorldHint": open_world
    })
}

fn write_annotations(title: &str, destructive: bool, open_world: bool) -> Value {
    json!({
        "title": title,
        "readOnlyHint": false,
        "destructiveHint": destructive,
        "idempotentHint": false,
        "openWorldHint": open_world
    })
}

fn read_resource(params: Value, home: &Path) -> Result<Value> {
    let uri = params
        .get("uri")
        .and_then(Value::as_str)
        .unwrap_or_default();
    match uri {
        "nightward://latest-summary" => {
            let report = scan_home(home)?;
            json_resource(uri, &report.summary)
        }
        "nightward://latest-report" => json_resource(uri, &latest_report(home)?),
        "nightward://rules" => json_resource(uri, &rules::all_rules()),
        "nightward://providers" => json_resource(uri, &provider_context(home, &Value::Null)),
        "nightward://schedule" => json_resource(uri, &schedule::status(home)),
        "nightward://actions" => json_resource(
            uri,
            &json!({
                "schema_version": 1,
                "actions": actions::list(home)
            }),
        ),
        "nightward://disclosure" => json_resource(uri, &state::disclosure_status(home)),
        "nightward://action-approvals" => json_resource(uri, &approvals::list(home)?),
        "nightward://report-history" => json_resource(
            uri,
            &json!({
                "schema_version": 1,
                "history": schedule::status(home).history
            }),
        ),
        _ => Err(anyhow!("unknown resource {uri}")),
    }
}

fn read_prompt(params: Value) -> Result<Value> {
    let name = params
        .get("name")
        .and_then(Value::as_str)
        .unwrap_or_default();
    let args = params.get("arguments").cloned().unwrap_or_default();
    let finding_id = string_arg(&args, "finding_id");
    let text = match name {
        "audit_my_ai_setup" => {
            "Use Nightward MCP tools to run nightward_scan, nightward_analysis, and nightward_policy_check with compact output. Explain the highest-risk AI/MCP configuration issues, provider posture, and the safest next actions. Preview any relevant action, request local approval only when a bounded registry action is clearly useful, then apply only the exact approved ticket."
        }
        "explain_top_risks" => {
            "Use nightward_findings and nightward_analysis to identify the top risks. Explain what can actually break or leak, what is probably just review noise, and what should be fixed first."
        }
        "fix_this_finding" => {
            return Ok(prompt_result(
                "Generate a safe Nightward fix workflow.",
                format!(
                    "Use nightward_explain_finding and nightward_fix_plan for finding `{}`. If a bounded registry action is relevant, use nightward_action_preview first, then nightward_action_request. Apply only after the user approves the exact ticket locally.",
                    if finding_id.is_empty() {
                        "<finding-id>"
                    } else {
                        &finding_id
                    }
                ),
            ));
        }
        "set_up_providers" => {
            "Use nightward_providers and nightward_actions_list to show missing, blocked, selected, and online-capable providers. Recommend provider.install/provider.enable actions only through nightward_action_preview and nightward_action_request, call out online/network behavior, and apply only an exact locally approved ticket."
        }
        "compare_reports" => {
            "Use nightward_report_history and nightward_report_changes to compare the last two reports. Summarize new, removed, and changed findings, then recommend which changes actually matter."
        }
        _ => return Err(anyhow!("unknown prompt {name}")),
    };
    Ok(prompt_result(name, text.to_string()))
}

fn prompt_result(description: impl Into<String>, text: String) -> Value {
    json!({
        "description": description.into(),
        "messages": [{
            "role": "user",
            "content": {
                "type": "text",
                "text": text
            }
        }]
    })
}

fn call_tool(params: Value, home: &Path) -> Value {
    let result = call_tool_inner(params, home);
    match result {
        Ok(value) => value,
        Err(error) => tool_error(error),
    }
}

fn call_tool_inner(params: Value, home: &Path) -> Result<Value> {
    let name = params
        .get("name")
        .and_then(Value::as_str)
        .ok_or_else(|| anyhow!("tools/call requires a tool name"))?;
    if name == "nightward_action_apply" {
        return Err(anyhow!(
            "nightward_action_apply is disabled in MCP because MCP clients cannot provide out-of-band local confirmation; use nightward_action_preview, then apply writes in the Nightward CLI, TUI, or Raycast extension"
        ));
    }
    let args = validate_tool_args(
        name,
        params
            .get("arguments")
            .cloned()
            .unwrap_or_else(|| json!({})),
    )?;
    match name {
        "nightward_scan" => {
            let scan = scan_for_args(home, &args)?;
            let structured = if bool_arg(&args, "compact", false) {
                json!({
                    "schema_version": 1,
                    "summary": scan.summary,
                    "findings": limited_values(scan.findings, limit_arg(&args, 25))?
                })
            } else {
                sanitized_value(&scan)?
            };
            tool_result(structured)
        }
        "nightward_doctor" => tool_result(json!({
            "schema_version": 1,
            "providers": provider_context(home, &args),
            "schedule": schedule::status(home),
            "disclosure": state::disclosure_status(home),
            "actions": {
                "available": actions::list(home).into_iter().filter(|action| action.available).count()
            }
        })),
        "nightward_findings" => {
            let scan = scan_for_args(home, &args)?;
            let severity = string_arg(&args, "severity");
            let rule = string_arg(&args, "rule");
            let limit = limit_arg(&args, 50);
            let findings: Vec<_> = scan
                .findings
                .into_iter()
                .filter(|finding| {
                    (severity.is_empty()
                        || format!("{:?}", finding.severity).eq_ignore_ascii_case(&severity))
                        && (rule.is_empty() || finding.rule == rule)
                })
                .take(limit)
                .collect();
            tool_result(json!({
                "schema_version": 1,
                "count": findings.len(),
                "findings": sanitized_value(&findings)?
            }))
        }
        "nightward_explain_finding" => {
            let id = string_arg(&args, "id");
            if id.is_empty() {
                return Err(anyhow!("id is required"));
            }
            let scan = scan_for_args(home, &args)?;
            let finding = scan
                .findings
                .iter()
                .find(|finding| finding.id == id || finding.id.starts_with(&id))
                .ok_or_else(|| anyhow!("finding not found"))?;
            tool_result(json!({
                "schema_version": 1,
                "finding": sanitized_value(finding)?
            }))
        }
        "nightward_analysis" => {
            let scan = scan_for_args(home, &args)?;
            let report = analysis_for_args(home, &scan, &args);
            let structured = if bool_arg(&args, "compact", false) {
                json!({
                    "schema_version": 1,
                    "summary": report.summary,
                    "providers": report.providers,
                    "signals": limited_values(report.signals, limit_arg(&args, 25))?
                })
            } else {
                sanitized_value(&report)?
            };
            tool_result(structured)
        }
        "nightward_explain_signal" => {
            let id = string_arg(&args, "id");
            if id.is_empty() {
                return Err(anyhow!("id is required"));
            }
            let scan = scan_for_args(home, &args)?;
            let report = analysis_for_args(home, &scan, &args);
            let signal = crate::analysis::explain(&report, &id)
                .ok_or_else(|| anyhow!("analysis signal not found"))?;
            tool_result(json!({
                "schema_version": 1,
                "signal": sanitized_value(&signal)?
            }))
        }
        "nightward_policy_check" => {
            let scan = scan_for_args(home, &args)?;
            let include_analysis = bool_arg(&args, "include_analysis", false);
            let analysis = include_analysis.then(|| analysis_for_args(home, &scan, &args));
            let mut config = PolicyConfig {
                include_analysis,
                ..PolicyConfig::default()
            };
            let provider_selection = selected_providers(home, &args);
            if !provider_selection.is_empty() {
                config.analysis_providers = provider_selection;
            }
            config.allow_online_providers = online_allowed(home, &args);
            let policy = policy_check(&scan, &config, analysis.as_ref());
            let structured = if bool_arg(&args, "compact", false) {
                json!({
                    "schema_version": 1,
                    "passed": policy.passed,
                    "threshold": policy.threshold,
                    "finding_count": policy.finding_count,
                    "blocking_count": policy.blocking_count,
                    "ignored_count": policy.ignored_count,
                    "analysis_violation_count": policy.analysis_violation_count,
                    "max_severity": policy.max_severity,
                    "findings": limited_values(policy.findings, limit_arg(&args, 25))?,
                    "analysis_violations": limited_values(policy.analysis_violations, limit_arg(&args, 25))?
                })
            } else {
                sanitized_value(&policy)?
            };
            tool_result(structured)
        }
        "nightward_fix_plan" => {
            let scan = scan_for_args(home, &args)?;
            let id = string_arg(&args, "id");
            let rule = string_arg(&args, "rule");
            let selector = Selector {
                all: bool_arg(&args, "all", id.is_empty() && rule.is_empty()),
                finding: id,
                rule,
            };
            tool_result(sanitized_value(&fix_plan(&scan, selector))?)
        }
        "nightward_report_history" => tool_result(json!({
            "schema_version": 1,
            "history": schedule::status(home).history
        })),
        "nightward_report_changes" => tool_result(sanitized_value(&report_changes(home, &args)?)?),
        "nightward_actions_list" => tool_result(json!({
            "schema_version": 1,
            "actions": actions::list(home)
        })),
        "nightward_action_preview" => {
            let id = string_arg(&args, "action_id");
            if id.is_empty() {
                return Err(anyhow!("action_id is required"));
            }
            tool_result(sanitized_value(&actions::preview(home, &id)?)?)
        }
        "nightward_action_request" => {
            let id = string_arg(&args, "action_id");
            if id.is_empty() {
                return Err(anyhow!("action_id is required"));
            }
            let requested = approvals::request(
                home,
                approvals::ApprovalRequestOptions {
                    action_id: id.clone(),
                    action_options: approval_options_from_mcp(&id, &args),
                    requested_by: string_arg(&args, "client"),
                },
            )?;
            tool_result(sanitized_value(&requested)?)
        }
        "nightward_action_status" => {
            let id = string_arg(&args, "approval_id");
            if id.is_empty() {
                return Err(anyhow!("approval_id is required"));
            }
            tool_result(sanitized_value(&approvals::status(home, &id)?)?)
        }
        "nightward_action_apply_approved" => {
            let id = string_arg(&args, "approval_id");
            if id.is_empty() {
                return Err(anyhow!("approval_id is required"));
            }
            tool_result(sanitized_value(&approvals::apply_approved(home, &id)?)?)
        }
        "nightward_rules" => tool_result(json!({
            "schema_version": 1,
            "rules": rules::all_rules()
        })),
        "nightward_providers" => tool_result(provider_context(home, &args)),
        _ => Err(anyhow!("unknown tool {name}")),
    }
}

#[derive(Clone, Copy)]
enum ToolArgKind {
    String,
    Bool,
    Limit,
    Severity,
    StringList,
}

#[derive(Clone, Copy)]
struct ToolArgSpec {
    name: &'static str,
    kind: ToolArgKind,
    required: bool,
}

impl ToolArgSpec {
    const fn optional(name: &'static str, kind: ToolArgKind) -> Self {
        Self {
            name,
            kind,
            required: false,
        }
    }

    const fn required(name: &'static str, kind: ToolArgKind) -> Self {
        Self {
            name,
            kind,
            required: true,
        }
    }
}

fn validate_tool_args(name: &str, args: Value) -> Result<Value> {
    let specs = tool_arg_specs(name)?;
    let object = args
        .as_object()
        .ok_or_else(|| anyhow!("{name} arguments must be an object"))?;
    for key in object.keys() {
        if !specs.iter().any(|spec| spec.name == key) {
            return Err(anyhow!("{name} does not accept argument `{key}`"));
        }
    }
    for spec in &specs {
        match object.get(spec.name) {
            Some(value) => validate_arg_value(name, *spec, value)?,
            None if spec.required => {
                return Err(anyhow!("{name} requires argument `{}`", spec.name));
            }
            None => {}
        }
    }
    Ok(Value::Object(object.clone()))
}

fn validate_arg_value(tool: &str, spec: ToolArgSpec, value: &Value) -> Result<()> {
    match spec.kind {
        ToolArgKind::String => {
            if value.is_string() {
                Ok(())
            } else {
                Err(anyhow!("{tool} argument `{}` must be a string", spec.name))
            }
        }
        ToolArgKind::Bool => {
            if value.is_boolean() {
                Ok(())
            } else {
                Err(anyhow!("{tool} argument `{}` must be a boolean", spec.name))
            }
        }
        ToolArgKind::Limit => {
            let Some(value) = value.as_u64() else {
                return Err(anyhow!(
                    "{tool} argument `{}` must be an integer",
                    spec.name
                ));
            };
            if (1..=250).contains(&value) {
                Ok(())
            } else {
                Err(anyhow!(
                    "{tool} argument `{}` must be between 1 and 250",
                    spec.name
                ))
            }
        }
        ToolArgKind::Severity => {
            let Some(value) = value.as_str() else {
                return Err(anyhow!("{tool} argument `{}` must be a string", spec.name));
            };
            if matches!(
                value,
                "info"
                    | "low"
                    | "medium"
                    | "high"
                    | "critical"
                    | "Info"
                    | "Low"
                    | "Medium"
                    | "High"
                    | "Critical"
            ) {
                Ok(())
            } else {
                Err(anyhow!(
                    "{tool} argument `{}` must be a known severity",
                    spec.name
                ))
            }
        }
        ToolArgKind::StringList => match value {
            Value::String(_) => Ok(()),
            Value::Array(values) if values.iter().all(Value::is_string) => Ok(()),
            _ => Err(anyhow!(
                "{tool} argument `{}` must be a string or array of strings",
                spec.name
            )),
        },
    }
}

fn tool_arg_specs(name: &str) -> Result<Vec<ToolArgSpec>> {
    use ToolArgKind::*;
    let specs = match name {
        "nightward_scan" => vec![
            ToolArgSpec::optional("workspace", String),
            ToolArgSpec::optional("compact", Bool),
            ToolArgSpec::optional("limit", Limit),
        ],
        "nightward_doctor" | "nightward_providers" => vec![
            ToolArgSpec::optional("with", StringList),
            ToolArgSpec::optional("online", Bool),
        ],
        "nightward_findings" => vec![
            ToolArgSpec::optional("workspace", String),
            ToolArgSpec::optional("severity", Severity),
            ToolArgSpec::optional("rule", String),
            ToolArgSpec::optional("limit", Limit),
        ],
        "nightward_explain_finding" => vec![
            ToolArgSpec::optional("workspace", String),
            ToolArgSpec::required("id", String),
        ],
        "nightward_analysis" => vec![
            ToolArgSpec::optional("workspace", String),
            ToolArgSpec::optional("with", StringList),
            ToolArgSpec::optional("online", Bool),
            ToolArgSpec::optional("package", String),
            ToolArgSpec::optional("finding_id", String),
            ToolArgSpec::optional("compact", Bool),
            ToolArgSpec::optional("limit", Limit),
        ],
        "nightward_explain_signal" => vec![
            ToolArgSpec::optional("workspace", String),
            ToolArgSpec::optional("with", StringList),
            ToolArgSpec::optional("online", Bool),
            ToolArgSpec::optional("package", String),
            ToolArgSpec::optional("finding_id", String),
            ToolArgSpec::required("id", String),
        ],
        "nightward_policy_check" => vec![
            ToolArgSpec::optional("workspace", String),
            ToolArgSpec::optional("include_analysis", Bool),
            ToolArgSpec::optional("with", StringList),
            ToolArgSpec::optional("online", Bool),
            ToolArgSpec::optional("compact", Bool),
            ToolArgSpec::optional("limit", Limit),
        ],
        "nightward_fix_plan" => vec![
            ToolArgSpec::optional("workspace", String),
            ToolArgSpec::optional("id", String),
            ToolArgSpec::optional("rule", String),
            ToolArgSpec::optional("all", Bool),
        ],
        "nightward_report_history" | "nightward_actions_list" | "nightward_rules" => Vec::new(),
        "nightward_report_changes" => vec![
            ToolArgSpec::optional("base", String),
            ToolArgSpec::optional("head", String),
        ],
        "nightward_action_preview" => vec![ToolArgSpec::required("action_id", String)],
        "nightward_action_request" => vec![
            ToolArgSpec::required("action_id", String),
            ToolArgSpec::optional("client", String),
            ToolArgSpec::optional("policy_path", String),
            ToolArgSpec::optional("finding_id", String),
            ToolArgSpec::optional("rule", String),
            ToolArgSpec::optional("reason", String),
        ],
        "nightward_action_status" | "nightward_action_apply_approved" => {
            vec![ToolArgSpec::required("approval_id", String)]
        }
        _ => return Err(anyhow!("unknown tool {name}")),
    };
    Ok(specs)
}

fn latest_report(home: &Path) -> Result<Value> {
    let status = schedule::status(home);
    let Some(path) = status.last_report else {
        let scan = scan_home(home)?;
        return Ok(json!({
            "schema_version": 1,
            "source": "live-scan",
            "report": sanitized_value(&scan)?
        }));
    };
    let report = load_report(&path)?;
    Ok(json!({
        "schema_version": 1,
        "source": "saved-report",
        "path": path,
        "report": sanitized_value(&report)?
    }))
}

fn provider_context(home: &Path, args: &Value) -> Value {
    let selected = selected_providers(home, args);
    let online = online_allowed(home, args);
    json!({
        "schema_version": 1,
        "providers": providers::providers(),
        "statuses": providers::statuses(&selected, online),
        "selected": selected,
        "online_allowed": online
    })
}

fn approval_options_from_mcp(action_id: &str, args: &Value) -> approvals::ApprovalActionOptions {
    approvals::ApprovalActionOptions {
        executable: if action_id == "schedule.install" {
            std::env::current_exe()
                .ok()
                .map(|path| path.display().to_string())
                .unwrap_or_else(|| "nightward".to_string())
        } else {
            String::new()
        },
        policy_path: string_arg(args, "policy_path"),
        finding_id: string_arg(args, "finding_id"),
        rule: string_arg(args, "rule"),
        reason: string_arg(args, "reason"),
    }
}

fn report_changes(home: &Path, args: &Value) -> Result<reportdiff::DiffReport> {
    let base_path = string_arg(args, "base");
    let head_path = string_arg(args, "head");
    if base_path.is_empty() != head_path.is_empty() {
        return Err(anyhow!(
            "base and head must both be provided or both omitted"
        ));
    }
    if !base_path.is_empty() {
        let base_path = scoped_mcp_existing_path(home, &base_path, ScopedPathKind::File)?;
        let head_path = scoped_mcp_existing_path(home, &head_path, ScopedPathKind::File)?;
        let base = load_report(&base_path)?;
        let head = load_report(&head_path)?;
        return Ok(reportdiff::diff(
            base_path.display().to_string(),
            &base,
            head_path.display().to_string(),
            &head,
        ));
    }

    let history = schedule::status(home).history;
    if history.len() < 2 {
        return Err(anyhow!(
            "at least two saved reports are required for nightward_report_changes"
        ));
    }
    let head_entry = &history[0];
    let base_entry = &history[1];
    let base = load_report(&base_entry.path)
        .with_context(|| format!("load base report {}", base_entry.path))?;
    let head = load_report(&head_entry.path)
        .with_context(|| format!("load head report {}", head_entry.path))?;
    Ok(reportdiff::diff(
        base_entry.report_name.clone(),
        &base,
        head_entry.report_name.clone(),
        &head,
    ))
}

fn scan_for_args(home: &Path, args: &Value) -> Result<crate::Report> {
    let workspace = string_arg(args, "workspace");
    if workspace.is_empty() {
        scan_home(home)
    } else {
        scan_workspace(scoped_mcp_existing_path(
            home,
            &workspace,
            ScopedPathKind::Directory,
        )?)
    }
}

#[derive(Clone, Copy)]
enum ScopedPathKind {
    Directory,
    File,
}

fn scoped_mcp_existing_path(home: &Path, requested: &str, kind: ScopedPathKind) -> Result<PathBuf> {
    let requested = requested.trim();
    if requested.is_empty() {
        return Err(anyhow!("path cannot be empty"));
    }
    let requested_path = Path::new(requested);
    let relative = if requested_path.is_absolute() {
        if requested_path
            .components()
            .any(|component| matches!(component, Component::ParentDir | Component::CurDir))
        {
            return Err(anyhow!("path cannot contain relative components"));
        }
        requested_path
            .strip_prefix(home)
            .map_err(|_| anyhow!("path must stay under NIGHTWARD_HOME"))?
            .to_path_buf()
    } else {
        requested_path.to_path_buf()
    };
    let parts = normal_mcp_relative_components(&relative)?;
    if parts.is_empty() {
        return Err(anyhow!("path cannot resolve to NIGHTWARD_HOME itself"));
    }

    ensure_scoped_path_component(home, true)?;
    let mut current = home.to_path_buf();
    for part in parts {
        current.push(part);
        let is_final = current == home.join(&relative);
        ensure_scoped_path_component(
            &current,
            matches!(kind, ScopedPathKind::Directory) && is_final,
        )?;
    }

    let metadata =
        fs::symlink_metadata(&current).with_context(|| format!("inspect {}", current.display()))?;
    if metadata.file_type().is_symlink() {
        return Err(anyhow!("path cannot be a symlink"));
    }
    match kind {
        ScopedPathKind::Directory if !metadata.is_dir() => {
            Err(anyhow!("workspace path must be an existing directory"))
        }
        ScopedPathKind::File if !metadata.is_file() => {
            Err(anyhow!("report path must be an existing regular file"))
        }
        _ => Ok(current),
    }
}

fn normal_mcp_relative_components(path: &Path) -> Result<Vec<String>> {
    let mut parts = Vec::new();
    for component in path.components() {
        match component {
            Component::Normal(part) => parts.push(part.to_string_lossy().to_string()),
            Component::ParentDir => return Err(anyhow!("path cannot contain parent directories")),
            Component::CurDir => return Err(anyhow!("path cannot contain current directories")),
            Component::RootDir | Component::Prefix(_) => {
                return Err(anyhow!("path must be relative"))
            }
        }
    }
    Ok(parts)
}

fn ensure_scoped_path_component(path: &Path, final_directory: bool) -> Result<()> {
    let metadata =
        fs::symlink_metadata(path).with_context(|| format!("inspect {}", path.display()))?;
    if metadata.file_type().is_symlink() {
        return Err(anyhow!("path cannot contain symlinks"));
    }
    if final_directory {
        if !metadata.is_dir() {
            return Err(anyhow!("workspace path must be an existing directory"));
        }
    } else if !metadata.is_dir() && !metadata.is_file() {
        return Err(anyhow!("path must be a regular file or directory"));
    }
    Ok(())
}

fn analysis_for_args(home: &Path, scan: &crate::Report, args: &Value) -> crate::analysis::Report {
    analyze(
        scan,
        AnalysisOptions {
            mode: scan.scan_mode.clone(),
            workspace: if scan.workspace.is_empty() {
                string_arg(args, "workspace")
            } else {
                scan.workspace.clone()
            },
            with: selected_providers(home, args),
            online: online_allowed(home, args),
            package: string_arg(args, "package"),
            finding_id: string_arg(args, "finding_id"),
        },
    )
}

fn selected_providers(home: &Path, args: &Value) -> Vec<String> {
    let requested = string_array_arg(args, "with");
    if !requested.is_empty() {
        return requested;
    }
    state::load_settings(home)
        .map(|settings| settings.selected_providers)
        .unwrap_or_default()
}

fn online_allowed(home: &Path, args: &Value) -> bool {
    args.get("online")
        .and_then(Value::as_bool)
        .unwrap_or_else(|| {
            state::load_settings(home)
                .map(|settings| settings.allow_online_providers)
                .unwrap_or(false)
        })
}

fn tool_result(structured: Value) -> Result<Value> {
    let structured = sanitize_structured(structured);
    let text = serde_json::to_string_pretty(&structured)?;
    Ok(json!({
        "content": [{ "type": "text", "text": text }],
        "structuredContent": structured,
        "isError": false
    }))
}

fn tool_error(error: anyhow::Error) -> Value {
    let message = redact_text(&error.to_string());
    json!({
        "content": [{ "type": "text", "text": message }],
        "structuredContent": {
            "schema_version": 1,
            "error": message
        },
        "isError": true
    })
}

fn json_resource(uri: &str, value: &impl Serialize) -> Result<Value> {
    let structured = sanitized_value(value)?;
    let text = serde_json::to_string_pretty(&structured)?;
    Ok(json!({
        "contents": [{
            "uri": uri,
            "mimeType": "application/json",
            "text": text
        }]
    }))
}

fn sanitized_value(value: &impl Serialize) -> Result<Value> {
    let text = serde_json::to_string(value)?;
    let redacted = redact_text(&text);
    Ok(serde_json::from_str(&redacted).unwrap_or_else(|_| {
        json!({
            "schema_version": 1,
            "redacted_text": redacted
        })
    }))
}

fn sanitize_structured(value: Value) -> Value {
    sanitized_value(&value).unwrap_or_else(|_| {
        json!({
            "schema_version": 1,
            "redacted_text": redact_text(&value.to_string())
        })
    })
}

fn limited_values<T: Serialize>(values: Vec<T>, limit: usize) -> Result<Value> {
    let values = values.into_iter().take(limit).collect::<Vec<_>>();
    sanitized_value(&values)
}

fn string_arg(args: &Value, name: &str) -> String {
    args.get(name)
        .and_then(Value::as_str)
        .map(str::trim)
        .unwrap_or_default()
        .to_string()
}

fn bool_arg(args: &Value, name: &str, default: bool) -> bool {
    args.get(name).and_then(Value::as_bool).unwrap_or(default)
}

fn limit_arg(args: &Value, default: usize) -> usize {
    args.get("limit")
        .and_then(Value::as_u64)
        .map(|value| value.clamp(1, 250) as usize)
        .unwrap_or(default)
}

fn string_array_arg(args: &Value, name: &str) -> Vec<String> {
    match args.get(name) {
        Some(Value::String(value)) => value
            .split(',')
            .map(str::trim)
            .filter(|value| !value.is_empty())
            .map(ToString::to_string)
            .collect(),
        Some(Value::Array(values)) => values
            .iter()
            .filter_map(Value::as_str)
            .map(str::trim)
            .filter(|value| !value.is_empty())
            .map(ToString::to_string)
            .collect(),
        _ => Vec::new(),
    }
}

fn no_args_schema() -> Value {
    json!({
        "type": "object",
        "additionalProperties": false,
        "properties": {}
    })
}

fn schema_object(properties: Value, required: &[&str]) -> Value {
    let mut schema = Map::new();
    schema.insert("type".to_string(), json!("object"));
    schema.insert("additionalProperties".to_string(), json!(false));
    schema.insert("properties".to_string(), properties);
    if !required.is_empty() {
        schema.insert("required".to_string(), json!(required));
    }
    Value::Object(schema)
}

fn schema_scan() -> Value {
    schema_object(
        json!({
            "workspace": { "type": "string", "description": "Workspace path to scan instead of HOME." },
            "compact": { "type": "boolean", "description": "Return summary plus bounded findings." },
            "limit": { "type": "integer", "minimum": 1, "maximum": 250 }
        }),
        &[],
    )
}

fn schema_provider_context() -> Value {
    schema_object(
        json!({
            "with": {
                "oneOf": [
                    { "type": "array", "items": { "type": "string" } },
                    { "type": "string" }
                ],
                "description": "Provider names or comma-separated provider list."
            },
            "online": { "type": "boolean", "description": "Allow online-capable providers for this call." }
        }),
        &[],
    )
}

fn schema_findings() -> Value {
    schema_object(
        json!({
            "workspace": { "type": "string" },
            "severity": { "type": "string", "enum": ["info", "low", "medium", "high", "critical", "Info", "Low", "Medium", "High", "Critical"] },
            "rule": { "type": "string" },
            "limit": { "type": "integer", "minimum": 1, "maximum": 250 }
        }),
        &[],
    )
}

fn schema_id_only() -> Value {
    schema_object(
        json!({
            "workspace": { "type": "string" },
            "id": { "type": "string", "description": "Finding ID or unique prefix." }
        }),
        &["id"],
    )
}

fn schema_analysis() -> Value {
    schema_object(
        json!({
            "workspace": { "type": "string" },
            "with": {
                "oneOf": [
                    { "type": "array", "items": { "type": "string" } },
                    { "type": "string" }
                ]
            },
            "online": { "type": "boolean" },
            "package": { "type": "string" },
            "finding_id": { "type": "string" },
            "compact": { "type": "boolean" },
            "limit": { "type": "integer", "minimum": 1, "maximum": 250 }
        }),
        &[],
    )
}

fn schema_explain_signal() -> Value {
    schema_object(
        json!({
            "workspace": { "type": "string" },
            "with": {
                "oneOf": [
                    { "type": "array", "items": { "type": "string" } },
                    { "type": "string" }
                ]
            },
            "online": { "type": "boolean" },
            "package": { "type": "string" },
            "finding_id": { "type": "string" },
            "id": { "type": "string", "description": "Analysis signal ID or unique prefix." }
        }),
        &["id"],
    )
}

fn schema_policy_check() -> Value {
    schema_object(
        json!({
            "workspace": { "type": "string" },
            "include_analysis": { "type": "boolean" },
            "with": {
                "oneOf": [
                    { "type": "array", "items": { "type": "string" } },
                    { "type": "string" }
                ]
            },
            "online": { "type": "boolean" },
            "compact": { "type": "boolean" },
            "limit": { "type": "integer", "minimum": 1, "maximum": 250 }
        }),
        &[],
    )
}

fn schema_fix_plan() -> Value {
    schema_object(
        json!({
            "workspace": { "type": "string" },
            "id": { "type": "string", "description": "Finding ID or unique prefix." },
            "rule": { "type": "string", "description": "Rule ID." },
            "all": { "type": "boolean" }
        }),
        &[],
    )
}

fn schema_report_changes() -> Value {
    schema_object(
        json!({
            "base": { "type": "string", "description": "Base report JSON path." },
            "head": { "type": "string", "description": "Head report JSON path." }
        }),
        &[],
    )
}

fn schema_action_id() -> Value {
    schema_object(
        json!({
            "action_id": { "type": "string" }
        }),
        &["action_id"],
    )
}

fn schema_action_request() -> Value {
    schema_object(
        json!({
            "action_id": { "type": "string" },
            "client": { "type": "string", "description": "Optional local client/session label for the approval queue." },
            "policy_path": { "type": "string", "description": "Optional policy path under NIGHTWARD_HOME for policy actions." },
            "finding_id": { "type": "string", "description": "Finding ID for policy.ignore." },
            "rule": { "type": "string", "description": "Rule ID for policy.ignore." },
            "reason": { "type": "string", "description": "Reviewed reason for policy.ignore." }
        }),
        &["action_id"],
    )
}

fn schema_approval_id() -> Value {
    schema_object(
        json!({
            "approval_id": { "type": "string" }
        }),
        &["approval_id"],
    )
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::actions::ApplyOptions;
    use std::collections::BTreeSet;
    use std::fs;

    #[test]
    fn initialize_negotiates_latest_and_compat_protocols() {
        let home = tempfile::tempdir().expect("temp home");
        let latest = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}),
            home.path(),
        );
        assert_eq!(latest["result"]["protocolVersion"], "2025-11-25");

        let compat = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}),
            home.path(),
        );
        assert_eq!(compat["result"]["protocolVersion"], "2025-06-18");

        let fallback = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}),
            home.path(),
        );
        assert_eq!(fallback["result"]["protocolVersion"], "2025-11-25");
    }

    #[test]
    fn lists_schema_backed_tools_resources_and_prompts() {
        let home = tempfile::tempdir().expect("temp home");
        let tools_response = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":1,"method":"tools/list"}),
            home.path(),
        );
        let tools = tools_response["result"]["tools"].as_array().unwrap();
        let names: BTreeSet<_> = tools
            .iter()
            .map(|tool| tool["name"].as_str().unwrap())
            .collect();
        for name in [
            "nightward_scan",
            "nightward_doctor",
            "nightward_findings",
            "nightward_explain_finding",
            "nightward_analysis",
            "nightward_explain_signal",
            "nightward_policy_check",
            "nightward_fix_plan",
            "nightward_report_history",
            "nightward_report_changes",
            "nightward_actions_list",
            "nightward_action_preview",
            "nightward_action_request",
            "nightward_action_status",
            "nightward_action_apply_approved",
            "nightward_rules",
            "nightward_providers",
        ] {
            assert!(names.contains(name), "missing {name}");
        }
        assert!(!names.contains("nightward_action_apply"));
        assert!(tools
            .iter()
            .all(|tool| tool["inputSchema"]["additionalProperties"] == false));
        assert!(tools.iter().all(|tool| tool.get("outputSchema").is_some()));
        assert!(tools.iter().all(|tool| tool.get("annotations").is_some()));
        let request_tool = tools
            .iter()
            .find(|tool| tool["name"] == "nightward_action_request")
            .unwrap();
        assert_eq!(request_tool["annotations"]["readOnlyHint"], false);
        assert_eq!(request_tool["annotations"]["destructiveHint"], false);
        let apply_tool = tools
            .iter()
            .find(|tool| tool["name"] == "nightward_action_apply_approved")
            .unwrap();
        assert_eq!(apply_tool["annotations"]["readOnlyHint"], false);
        assert_eq!(apply_tool["annotations"]["destructiveHint"], true);
        assert!(tools
            .iter()
            .filter(|tool| !matches!(
                tool["name"].as_str(),
                Some("nightward_action_request" | "nightward_action_apply_approved")
            ))
            .all(|tool| tool["annotations"]["readOnlyHint"] == true));

        let resources_response = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":2,"method":"resources/list"}),
            home.path(),
        );
        let resource_uris: BTreeSet<_> = resources_response["result"]["resources"]
            .as_array()
            .unwrap()
            .iter()
            .map(|resource| resource["uri"].as_str().unwrap())
            .collect();
        for uri in [
            "nightward://providers",
            "nightward://schedule",
            "nightward://latest-report",
            "nightward://action-approvals",
            "nightward://report-history",
        ] {
            assert!(resource_uris.contains(uri), "missing {uri}");
        }

        let prompts_response = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":3,"method":"prompts/list"}),
            home.path(),
        );
        assert!(prompts_response["result"]["prompts"]
            .as_array()
            .unwrap()
            .iter()
            .any(|prompt| prompt["name"] == "audit_my_ai_setup"));
    }

    #[test]
    fn prompt_get_returns_messages() {
        let home = tempfile::tempdir().expect("temp home");
        let response = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":1,"method":"prompts/get","params":{"name":"fix_this_finding","arguments":{"finding_id":"abc"}}}),
            home.path(),
        );
        assert!(response["result"]["messages"][0]["content"]["text"]
            .as_str()
            .unwrap()
            .contains("abc"));
    }

    #[test]
    fn tool_errors_use_mcp_tool_result_errors() {
        let home = tempfile::tempdir().expect("temp home");
        let response = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nightward_missing","arguments":{}}}),
            home.path(),
        );
        assert!(response.get("error").is_none());
        assert_eq!(response["result"]["isError"], true);
        assert!(response["result"]["structuredContent"]["error"]
            .as_str()
            .unwrap()
            .contains("unknown tool"));
    }

    #[test]
    fn action_apply_is_disabled_and_cannot_accept_disclosure() {
        let home = tempfile::tempdir().expect("temp home");
        let response = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nightward_action_apply","arguments":{"action_id":"disclosure.accept","confirm":true}}}),
            home.path(),
        );
        assert_eq!(response["result"]["isError"], true);
        assert!(response["result"]["content"][0]["text"]
            .as_str()
            .unwrap()
            .contains("disabled in MCP"));
        assert!(!state::disclosure_status(home.path()).accepted);
        assert!(!state::settings_path(home.path()).exists());
    }

    #[test]
    fn mcp_action_request_requires_local_disclosure_and_cannot_self_confirm() {
        let home = tempfile::tempdir().expect("temp home");
        let blocked = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nightward_action_request","arguments":{"action_id":"backup.snapshot","confirm":true}}}),
            home.path(),
        );
        assert_eq!(blocked["result"]["isError"], true);
        assert!(blocked["result"]["content"][0]["text"]
            .as_str()
            .unwrap()
            .contains("does not accept argument `confirm`"));

        let no_disclosure = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nightward_action_request","arguments":{"action_id":"backup.snapshot"}}}),
            home.path(),
        );
        assert_eq!(no_disclosure["result"]["isError"], true);
        assert!(no_disclosure["result"]["content"][0]["text"]
            .as_str()
            .unwrap()
            .contains("accept the Nightward beta"));

        let self_accept = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nightward_action_request","arguments":{"action_id":"disclosure.accept"}}}),
            home.path(),
        );
        assert_eq!(self_accept["result"]["isError"], true);
        assert!(!state::disclosure_status(home.path()).accepted);
    }

    #[test]
    fn mcp_can_request_and_apply_only_after_local_approval_once() {
        let home = tempfile::tempdir().expect("temp home");
        fs::create_dir_all(home.path().join(".codex")).expect("codex dir");
        fs::write(home.path().join(".codex/config.toml"), "model = \"test\"\n").expect("config");
        actions::apply(
            home.path(),
            "disclosure.accept",
            ApplyOptions {
                confirm: true,
                executable: "nightward".to_string(),
                ..Default::default()
            },
        )
        .expect("accept disclosure");

        let requested = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nightward_action_request","arguments":{"action_id":"backup.snapshot","client":"test-mcp"}}}),
            home.path(),
        );
        assert_eq!(requested["result"]["isError"], false);
        let approval_id = requested["result"]["structuredContent"]["approval_id"]
            .as_str()
            .unwrap()
            .to_string();

        let pending_apply = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"nightward_action_apply_approved","arguments":{"approval_id":approval_id}}}),
            home.path(),
        );
        assert_eq!(pending_apply["result"]["isError"], true);
        assert!(pending_apply["result"]["content"][0]["text"]
            .as_str()
            .unwrap()
            .contains("not approved"));

        approvals::approve(home.path(), &approval_id, "reviewed in test").expect("approve");
        let applied = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"nightward_action_apply_approved","arguments":{"approval_id":approval_id}}}),
            home.path(),
        );
        assert_eq!(applied["result"]["isError"], false);
        assert_eq!(
            applied["result"]["structuredContent"]["approval"]["status"],
            "applied"
        );
        assert!(state::state_dir(home.path()).join("snapshots").exists());

        let replay = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"nightward_action_apply_approved","arguments":{"approval_id":approval_id}}}),
            home.path(),
        );
        assert_eq!(replay["result"]["isError"], true);
        assert!(replay["result"]["content"][0]["text"]
            .as_str()
            .unwrap()
            .contains("not approved"));
    }

    #[test]
    fn tool_calls_reject_invalid_arguments_against_strict_schemas() {
        let home = tempfile::tempdir().expect("temp home");
        for (arguments, expected) in [
            (
                json!({"name":"nightward_actions_list","arguments":{"extra":true}}),
                "does not accept argument",
            ),
            (
                json!({"name":"nightward_scan","arguments":{"workspace":123}}),
                "workspace` must be a string",
            ),
            (
                json!({"name":"nightward_findings","arguments":{"limit":999}}),
                "between 1 and 250",
            ),
            (
                json!({"name":"nightward_findings","arguments":{"severity":"urgent"}}),
                "known severity",
            ),
        ] {
            let response = handle_request_with_home(
                json!({"jsonrpc":"2.0","id":1,"method":"tools/call","params":arguments}),
                home.path(),
            );
            assert_eq!(response["result"]["isError"], true);
            assert!(
                response["result"]["content"][0]["text"]
                    .as_str()
                    .unwrap()
                    .contains(expected),
                "expected {expected}, got {}",
                response["result"]["content"][0]["text"]
            );
        }
    }

    #[test]
    fn mcp_workspace_paths_must_stay_under_home_and_exist_without_symlinks() {
        let home = tempfile::tempdir().expect("temp home");
        let workspace = home.path().join("workspace");
        fs::create_dir_all(&workspace).expect("workspace dir");
        let response = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nightward_scan","arguments":{"workspace":workspace.display().to_string(),"compact":true}}}),
            home.path(),
        );
        assert_eq!(response["result"]["isError"], false);

        let outside = tempfile::tempdir().expect("outside");
        let response = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nightward_scan","arguments":{"workspace":outside.path().display().to_string()}}}),
            home.path(),
        );
        assert_eq!(response["result"]["isError"], true);
        let text = response["result"]["content"][0]["text"].as_str().unwrap();
        assert!(text.contains("NIGHTWARD_HOME"));
        assert!(!text.contains(outside.path().to_string_lossy().as_ref()));
    }

    #[cfg(unix)]
    #[test]
    fn mcp_workspace_paths_reject_symlink_components() {
        use std::os::unix::fs::symlink;

        let home = tempfile::tempdir().expect("temp home");
        let outside = tempfile::tempdir().expect("outside");
        symlink(outside.path(), home.path().join("workspace")).expect("workspace symlink");
        let response = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nightward_scan","arguments":{"workspace":"workspace"}}}),
            home.path(),
        );

        assert_eq!(response["result"]["isError"], true);
        assert!(response["result"]["content"][0]["text"]
            .as_str()
            .unwrap()
            .contains("symlinks"));
    }

    #[test]
    fn mcp_report_changes_paths_are_scoped_to_home() {
        let home = tempfile::tempdir().expect("temp home");
        let outside = tempfile::tempdir().expect("outside");
        let base = outside.path().join("base.json");
        let head = outside.path().join("head.json");
        fs::write(&base, "{}").expect("base");
        fs::write(&head, "{}").expect("head");

        let response = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nightward_report_changes","arguments":{"base":base.display().to_string(),"head":head.display().to_string()}}}),
            home.path(),
        );

        assert_eq!(response["result"]["isError"], true);
        let text = response["result"]["content"][0]["text"].as_str().unwrap();
        assert!(text.contains("NIGHTWARD_HOME"));
        assert!(!text.contains(outside.path().to_string_lossy().as_ref()));
    }

    #[test]
    fn action_apply_remains_disabled_after_out_of_band_disclosure() {
        let home = tempfile::tempdir().expect("temp home");
        fs::create_dir_all(home.path().join(".codex")).expect("codex dir");
        fs::write(home.path().join(".codex/config.toml"), "model = \"test\"\n").expect("config");
        actions::apply(
            home.path(),
            "disclosure.accept",
            ApplyOptions {
                confirm: true,
                executable: "nightward".to_string(),
                ..Default::default()
            },
        )
        .expect("accept disclosure");

        let response = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nightward_action_apply","arguments":{"action_id":"backup.snapshot","confirm":true}}}),
            home.path(),
        );

        assert_eq!(response["result"]["isError"], true);
        assert!(response["result"]["content"][0]["text"]
            .as_str()
            .unwrap()
            .contains("disabled in MCP"));
        assert!(!state::state_dir(home.path()).join("snapshots").exists());
    }

    #[test]
    fn report_changes_errors_when_not_enough_history() {
        let home = tempfile::tempdir().expect("temp home");
        let response = handle_request_with_home(
            json!({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nightward_report_changes","arguments":{}}}),
            home.path(),
        );
        assert_eq!(response["result"]["isError"], true);
        assert!(response["result"]["content"][0]["text"]
            .as_str()
            .unwrap()
            .contains("at least two saved reports"));
    }
}
