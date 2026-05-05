use crate::inventory::redact_text;
use crate::{Finding, FixKind, Report as ScanReport, RiskLevel};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Plan {
    pub schema_version: u32,
    pub generated_at: DateTime<Utc>,
    pub mode: String,
    pub summary: Summary,
    pub groups: Vec<Group>,
    pub actions: Vec<Action>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct Summary {
    pub total: usize,
    pub safe: usize,
    pub review: usize,
    pub blocked: usize,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Group {
    pub key: String,
    pub title: String,
    pub severity: RiskLevel,
    pub finding_count: usize,
    pub actions: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Action {
    pub id: String,
    pub finding_id: String,
    pub rule: String,
    pub severity: RiskLevel,
    pub safe_to_apply: bool,
    pub requires_review: bool,
    pub title: String,
    pub steps: Vec<String>,
    pub preview: String,
}

pub fn plan(scan: &ScanReport, selector: Selector) -> Plan {
    let findings: Vec<_> = scan
        .findings
        .iter()
        .filter(|finding| selector.matches(finding))
        .cloned()
        .collect();
    let mut actions = Vec::new();
    for finding in findings {
        let action = action_from_finding(&finding);
        actions.push(action);
    }
    let groups = group_actions(&actions);
    let mut summary = Summary {
        total: actions.len(),
        ..Summary::default()
    };
    for action in &actions {
        if action.safe_to_apply {
            summary.safe += 1;
        } else if action.requires_review {
            summary.review += 1;
        } else {
            summary.blocked += 1;
        }
    }
    Plan {
        schema_version: 1,
        generated_at: Utc::now(),
        mode: "plan-only".to_string(),
        summary,
        groups,
        actions,
    }
}

pub fn markdown(plan: &Plan) -> String {
    let mut out = vec![
        "# Nightward Fix Plan".to_string(),
        String::new(),
        "Nightward fix plans are review material only. They do not mutate config.".to_string(),
        String::new(),
        format!(
            "- Total: `{}`\n- Safe: `{}`\n- Review: `{}`\n- Blocked: `{}`",
            plan.summary.total, plan.summary.safe, plan.summary.review, plan.summary.blocked
        ),
    ];
    for group in &plan.groups {
        out.push(String::new());
        out.push(format!("## {}", group.title));
        out.push(format!("- Severity: `{:?}`", group.severity).to_ascii_lowercase());
        out.push(format!("- Findings: `{}`", group.finding_count));
    }
    for action in &plan.actions {
        out.push(String::new());
        out.push(format!("## {}", action.title));
        out.push(format!("- Finding: `{}`", action.finding_id));
        out.push(format!("- Rule: `{}`", action.rule));
        for (index, step) in action.steps.iter().enumerate() {
            out.push(format!("{}. {}", index + 1, step));
        }
        if !action.preview.is_empty() {
            out.push(String::new());
            out.push("```diff".to_string());
            out.push(action.preview.clone());
            out.push("```".to_string());
        }
    }
    out.join("\n")
}

#[derive(Debug, Clone, Default)]
pub struct Selector {
    pub all: bool,
    pub finding: String,
    pub rule: String,
}

impl Selector {
    fn matches(&self, finding: &Finding) -> bool {
        self.all
            || (!self.finding.is_empty()
                && (finding.id == self.finding || finding.id.starts_with(&self.finding)))
            || (!self.rule.is_empty() && finding.rule == self.rule)
            || (self.finding.is_empty() && self.rule.is_empty())
    }
}

fn action_from_finding(finding: &Finding) -> Action {
    let kind = finding.fix_kind.unwrap_or(FixKind::ManualReview);
    let safe_to_apply = false;
    let title = match kind {
        FixKind::PinPackage => "Pin package executor",
        FixKind::ExternalizeSecret => "Externalize inline secret",
        FixKind::ReplaceShellWrapper => "Replace shell wrapper",
        FixKind::NarrowFilesystem => "Narrow filesystem scope",
        FixKind::IgnoreWithReason => "Ignore with documented reason",
        FixKind::ManualReview => "Review finding",
    };
    let preview = preview_for(finding);
    Action {
        id: format!("fix-{}", finding.id),
        finding_id: finding.id.clone(),
        rule: finding.rule.clone(),
        severity: finding.severity,
        safe_to_apply,
        requires_review: true,
        title: title.to_string(),
        steps: if finding.fix_steps.is_empty() {
            vec![
                "Inspect the redacted finding evidence.".to_string(),
                redact_text(&finding.recommended_action),
                "Re-run Nightward and compare the next report.".to_string(),
            ]
        } else {
            finding
                .fix_steps
                .iter()
                .map(|step| redact_text(step))
                .collect()
        },
        preview,
    }
}

fn preview_for(finding: &Finding) -> String {
    let Some(hint) = &finding.patch_hint else {
        return redact_text(&format!(
            "# plan-only review\n# {}\n# {}",
            finding.path, finding.recommended_action
        ));
    };
    match hint.kind {
        Some(FixKind::ExternalizeSecret) => {
            let key = if hint.env_key.is_empty() {
                "SECRET_VALUE"
            } else {
                &hint.env_key
            };
            redact_text(&format!(
                "- inline secret value in {}\n+ external reference to ${}\n# review required before editing",
                finding.path, key
            ))
        }
        Some(FixKind::PinPackage) if !hint.package.is_empty() => redact_text(&format!(
            "- {}\n+ {}@<reviewed-version>\n# choose and review an explicit version",
            hint.package, hint.package
        )),
        Some(FixKind::PinPackage) => {
            "# choose and review an explicit package version manually".to_string()
        }
        Some(FixKind::NarrowFilesystem) => {
            "- broad filesystem path\n+ <specific-reviewed-path>".to_string()
        }
        _ => redact_text(&format!("# review required for {}", finding.path)),
    }
}

fn group_actions(actions: &[Action]) -> Vec<Group> {
    let mut map: BTreeMap<String, Group> = BTreeMap::new();
    for action in actions {
        let key = group_key(action);
        let entry = map.entry(key.clone()).or_insert_with(|| Group {
            key: key.clone(),
            title: group_title(action),
            severity: action.severity,
            finding_count: 0,
            actions: Vec::new(),
        });
        entry.finding_count += 1;
        if action.severity.rank() > entry.severity.rank() {
            entry.severity = action.severity;
        }
        entry.actions.push(action.id.clone());
    }
    map.into_values().collect()
}

fn group_key(action: &Action) -> String {
    format!("{}:{:?}", action.rule, action.severity)
}

fn group_title(action: &Action) -> String {
    match action.rule.as_str() {
        "mcp_unpinned_package" => "Pin package executors".to_string(),
        "mcp_secret_env" | "mcp_secret_header" => "Externalize inline secrets".to_string(),
        "mcp_local_endpoint" => "Review machine-local endpoints".to_string(),
        "mcp_broad_filesystem" => "Narrow filesystem access".to_string(),
        _ => action.title.clone(),
    }
}
