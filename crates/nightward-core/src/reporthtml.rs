use crate::analysis::Report as AnalysisReport;
use crate::fixplan::Plan;
use crate::policy::PolicyReport;
use crate::reportdiff::DiffReport;
use crate::Report as ScanReport;
use anyhow::Result;
use html_escape::encode_text;
use std::fs;
use std::path::Path;

pub fn render(
    scan: &ScanReport,
    analysis: Option<&AnalysisReport>,
    policy: Option<&PolicyReport>,
    plan: Option<&Plan>,
    diff: Option<&DiffReport>,
) -> String {
    let mut out = String::new();
    out.push_str("<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>Nightward Report</title><style>");
    out.push_str("body{margin:0;background:#081012;color:#ecfeff;font:15px/1.5 system-ui,-apple-system,BlinkMacSystemFont,Segoe UI,sans-serif}main{max-width:1180px;margin:0 auto;padding:32px}a{color:#67e8f9}.hero{display:grid;gap:16px;margin-bottom:28px}.kpis{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:12px}.kpi,.panel{border:1px solid #1f3a3d;background:#0c171a;border-radius:10px;padding:16px}.label{color:#93a3a6;font-size:12px;text-transform:uppercase;letter-spacing:.08em}.value{font-size:28px;font-weight:800}.finding{border-left:4px solid #67e8f9}.critical{border-left-color:#ff4d4f}.high{border-left-color:#ff8c42}.medium{border-left-color:#f5c542}code,pre{background:#132226;border-radius:6px;padding:2px 5px}pre{white-space:pre-wrap;padding:12px;overflow:auto}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(320px,1fr));gap:14px}.muted{color:#93a3a6}");
    out.push_str("</style></head><body><main>");
    out.push_str("<section class=\"hero\"><p class=\"label\">Nightward local-first audit</p><h1>AI agent and dotfiles safety report</h1>");
    out.push_str(&format!(
        "<p class=\"muted\">Generated {} on {}</p>",
        scan.generated_at,
        encode_text(&scan.hostname)
    ));
    out.push_str("<div class=\"kpis\">");
    kpi(&mut out, "Findings", scan.summary.total_findings);
    kpi(&mut out, "Items", scan.summary.total_items);
    if let Some(analysis) = analysis {
        kpi(&mut out, "Signals", analysis.summary.total_signals);
    }
    if let Some(policy) = policy {
        kpi(&mut out, "Policy Blocks", policy.blocking_count);
    }
    if let Some(diff) = diff {
        kpi(&mut out, "Added", diff.summary.added);
    }
    out.push_str("</div></section>");
    if let Some(diff) = diff {
        render_diff(&mut out, diff);
    }
    if let Some(analysis) = analysis {
        render_analysis(&mut out, analysis);
    }
    out.push_str("<section class=\"grid\">");
    for finding in &scan.findings {
        out.push_str(&format!("<article class=\"panel finding {}\"><p class=\"label\">{:?} / {}</p><h2>{}</h2><p>{}</p>", format!("{:?}", finding.severity).to_ascii_lowercase(), finding.severity, encode_text(&finding.rule), encode_text(&finding.message), encode_text(&finding.recommended_action)));
        if !finding.evidence.is_empty() {
            out.push_str(&format!(
                "<details><summary>Evidence</summary><pre>{}</pre></details>",
                encode_text(&finding.evidence)
            ));
        }
        out.push_str("</article>");
    }
    out.push_str("</section>");
    if let Some(plan) = plan {
        out.push_str("<section class=\"panel\"><h2>Fix Plan</h2>");
        for group in &plan.groups {
            out.push_str(&format!(
                "<p><strong>{}</strong> <span class=\"muted\">{} findings</span></p>",
                encode_text(&group.title),
                group.finding_count
            ));
        }
        out.push_str("</section>");
    }
    out.push_str("</main></body></html>");
    out
}

pub fn write(path: impl AsRef<Path>, html: &str) -> Result<()> {
    if let Some(parent) = path.as_ref().parent() {
        fs::create_dir_all(parent)?;
    }
    fs::write(path.as_ref(), html)?;
    Ok(())
}

fn kpi(out: &mut String, label: &str, value: usize) {
    out.push_str(&format!(
        "<div class=\"kpi\"><div class=\"label\">{}</div><div class=\"value\">{}</div></div>",
        label, value
    ));
}

fn render_diff(out: &mut String, diff: &DiffReport) {
    out.push_str("<section class=\"panel\"><p class=\"label\">Report history comparison</p>");
    out.push_str(&format!(
        "<h2>{} to {}</h2><p class=\"muted\">Added {}, removed {}, changed {}. Max added severity: {:?}.</p>",
        encode_text(&diff.base),
        encode_text(&diff.head),
        diff.summary.added,
        diff.summary.removed,
        diff.summary.changed,
        diff.summary.max_added_severity
    ));
    out.push_str("<div class=\"grid\">");
    render_finding_list(out, "Added findings", &diff.added);
    render_finding_list(out, "Removed findings", &diff.removed);
    out.push_str("<article><h3>Changed findings</h3>");
    if diff.changed.is_empty() {
        out.push_str("<p class=\"muted\">No severity or message changes.</p>");
    } else {
        for change in diff.changed.iter().take(8) {
            out.push_str(&format!(
                "<p><strong>{}</strong><br><span class=\"muted\">{:?} to {:?}</span><br>{}</p>",
                encode_text(&change.after.rule),
                change.before.severity,
                change.after.severity,
                encode_text(&change.after.message)
            ));
        }
    }
    out.push_str("</article></div></section>");
}

fn render_finding_list(out: &mut String, title: &str, findings: &[crate::Finding]) {
    out.push_str(&format!("<article><h3>{}</h3>", encode_text(title)));
    if findings.is_empty() {
        out.push_str("<p class=\"muted\">None.</p>");
    } else {
        for finding in findings.iter().take(8) {
            out.push_str(&format!(
                "<p><strong>{:?} / {}</strong><br>{}<br><code>{}</code></p>",
                finding.severity,
                encode_text(&finding.rule),
                encode_text(&finding.message),
                encode_text(&finding.path)
            ));
        }
    }
    out.push_str("</article>");
}

fn render_analysis(out: &mut String, analysis: &AnalysisReport) {
    out.push_str("<section class=\"panel\"><p class=\"label\">Analysis</p><h2>Provider and signal summary</h2>");
    out.push_str(&format!(
        "<p class=\"muted\">{} signals across {} subjects. Provider warnings: {}. Highest severity: {:?}.</p>",
        analysis.summary.total_signals,
        analysis.summary.total_subjects,
        analysis.summary.provider_warnings,
        analysis.summary.highest_severity
    ));
    if !analysis.signals.is_empty() {
        out.push_str("<div class=\"grid\">");
        for signal in analysis.signals.iter().take(6) {
            out.push_str(&format!(
                "<article><p class=\"label\">{} / {:?}</p><h3>{}</h3><p>{}</p>",
                encode_text(&signal.provider),
                signal.severity,
                encode_text(&signal.rule),
                encode_text(&signal.message)
            ));
            if !signal.evidence.is_empty() {
                out.push_str(&format!("<pre>{}</pre>", encode_text(&signal.evidence)));
            }
            out.push_str("</article>");
        }
        out.push_str("</div>");
    }
    out.push_str("</section>");
}
