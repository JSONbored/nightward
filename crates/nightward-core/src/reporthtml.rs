use crate::analysis::Report as AnalysisReport;
use crate::fixplan::Plan;
use crate::policy::PolicyReport;
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
    out.push_str("</div></section>");
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
