use crate::tui;
use anyhow::{anyhow, Result};
use nightward_core::analysis::{self, Options as AnalysisOptions};
use nightward_core::fixplan::{self, Selector};
use nightward_core::inventory::{
    home_dir_from_env, load_report, scan_home, scan_workspace, write_report,
};
use nightward_core::policy::{self, PolicyConfig};
use nightward_core::{
    actions, backupplan, mcpserver, providers, reportdiff, reporthtml, rules, schedule, snapshot,
    state,
};
use serde::Serialize;
use std::env;

pub fn run() -> Result<()> {
    let mut args: Vec<String> = env::args().skip(1).collect();
    if args.is_empty() {
        let report = scan_home(home_dir_from_env())?;
        return tui::run(&report);
    }
    if args[0] == "--version" || args[0] == "version" {
        println!("{}", version());
        return Ok(());
    }
    let command = args.remove(0);
    match command.as_str() {
        "scan" => cmd_scan(&args),
        "tui" => cmd_tui(&args),
        "doctor" => cmd_doctor(&args),
        "plan" => cmd_plan(&args),
        "adapters" => cmd_adapters(&args),
        "findings" => cmd_findings(&args),
        "fix" => cmd_fix(&args),
        "analyze" => cmd_analyze(&args),
        "providers" => cmd_providers(&args),
        "rules" => cmd_rules(&args),
        "report" => cmd_report(&args),
        "policy" => cmd_policy(&args),
        "mcp" => cmd_mcp(&args),
        "snapshot" => cmd_snapshot(&args),
        "backup" => cmd_backup(&args),
        "schedule" => cmd_schedule(&args),
        "actions" => cmd_actions(&args),
        "disclosure" => cmd_disclosure(&args),
        "help" | "--help" | "-h" => {
            print_help();
            Ok(())
        }
        unknown => Err(anyhow!("unknown command {unknown}; run nightward help")),
    }
}

fn cmd_tui(args: &[String]) -> Result<()> {
    if has_report_diff_paths(args) {
        let (base, head) = report_diff_paths(args)?;
        let base_report = load_report(base)?;
        let head_report = load_report(head)?;
        let diff = reportdiff::diff(
            base.to_string(),
            &base_report,
            head.to_string(),
            &head_report,
        );
        return tui::run_compare(&diff);
    }
    let input = value_after(args, "--input");
    let workspace = value_after(args, "--workspace").or_else(|| value_after(args, "-w"));
    let report = if let Some(input) = input {
        load_report(input)?
    } else if let Some(workspace) = workspace {
        scan_workspace(workspace)?
    } else {
        scan_home(home_dir_from_env())?
    };
    tui::run(&report)
}

fn cmd_scan(args: &[String]) -> Result<()> {
    let output = value_after(args, "--output");
    let report = scan_for_args(args)?;
    if let Some(output) = output.filter(|value| *value != "-") {
        write_report(output, &report)?;
    }
    if has(args, "--json") || output.is_none() {
        print_json(&report)?;
    } else {
        println!(
            "{} findings across {} scanned items",
            report.summary.total_findings, report.summary.total_items
        );
    }
    Ok(())
}

fn cmd_doctor(args: &[String]) -> Result<()> {
    let home = home_dir_from_env();
    let payload = serde_json::json!({
        "schema_version": 1,
        "providers": providers::statuses(&selected_providers_for_args(args), online_for_args(args)),
        "schedule": schedule::status(&home),
        "disclosure": state::disclosure_status(home),
    });
    if has(args, "--json") {
        print_json(&payload)?;
    } else {
        println!("Nightward doctor: provider and schedule posture ready");
    }
    Ok(())
}

fn cmd_plan(args: &[String]) -> Result<()> {
    if args.first().map(String::as_str) == Some("backup") {
        print_json(&backupplan::plan(home_dir_from_env()))?;
        return Ok(());
    }
    Err(anyhow!("unknown plan command"))
}

fn cmd_adapters(args: &[String]) -> Result<()> {
    let report = scan_home(home_dir_from_env())?;
    match args.first().map(String::as_str) {
        Some("list") | None => print_json(&report.adapters),
        Some("explain") => {
            let name = args
                .get(1)
                .ok_or_else(|| anyhow!("adapter name required"))?;
            let adapter = report
                .adapters
                .into_iter()
                .find(|adapter| adapter.name.eq_ignore_ascii_case(name))
                .ok_or_else(|| anyhow!("adapter not found"))?;
            print_json(&adapter)
        }
        Some("template") => {
            println!(
                "# Nightward adapter fixture\n# Add representative MCP config files under testdata/homes/<name>."
            );
            Ok(())
        }
        _ => Err(anyhow!("unknown adapters command")),
    }
}

fn cmd_findings(args: &[String]) -> Result<()> {
    let report = scan_home(home_dir_from_env())?;
    match args.first().map(String::as_str) {
        Some("list") | None => {
            if has(args, "--json") {
                print_json(&report.findings)
            } else {
                for finding in report.findings {
                    println!(
                        "{} {} {}",
                        risk_label(finding.severity),
                        finding.rule,
                        finding.id
                    );
                }
                Ok(())
            }
        }
        Some("explain") => {
            let id = args
                .iter()
                .find(|arg| !arg.starts_with('-') && arg.as_str() != "explain")
                .ok_or_else(|| anyhow!("finding id required"))?;
            let finding = report
                .findings
                .iter()
                .find(|finding| finding.id == *id || finding.id.starts_with(id.as_str()))
                .ok_or_else(|| anyhow!("finding not found"))?;
            print_json(finding)
        }
        _ => Err(anyhow!("unknown findings command")),
    }
}

fn cmd_fix(args: &[String]) -> Result<()> {
    let report = scan_home(home_dir_from_env())?;
    let selector = selector(args);
    match args.first().map(String::as_str) {
        Some("plan") | Some("preview") | None => print_json(&fixplan::plan(&report, selector)),
        Some("export") => {
            let plan = fixplan::plan(&report, selector);
            let format = value_after(args, "--format").unwrap_or("markdown");
            if format == "json" {
                print_json(&plan)
            } else {
                println!("{}", fixplan::markdown(&plan));
                Ok(())
            }
        }
        _ => Err(anyhow!("unknown fix command")),
    }
}

fn cmd_analyze(args: &[String]) -> Result<()> {
    let scan = scan_for_args(args)?;
    let options = AnalysisOptions {
        mode: scan.scan_mode.clone(),
        workspace: scan.workspace.clone(),
        with: selected_providers_for_args(args),
        online: online_for_args(args),
        package: if args.iter().any(|arg| arg == "package") {
            positional_after_command(args, "package")
                .unwrap_or_default()
                .to_string()
        } else {
            String::new()
        },
        finding_id: if args.iter().any(|arg| arg == "finding") {
            positional_after_command(args, "finding")
                .unwrap_or_default()
                .to_string()
        } else {
            String::new()
        },
    };
    let report = analysis::run(&scan, options);
    print_json(&report)
}

fn cmd_providers(args: &[String]) -> Result<()> {
    match args.first().map(String::as_str) {
        Some("list") | None => print_json(&providers::providers()),
        Some("doctor") => print_json(&providers::statuses(
            &selected_providers_for_args(args),
            online_for_args(args),
        )),
        Some("enable") => {
            let name = args
                .get(1)
                .ok_or_else(|| anyhow!("provider name required"))?;
            let id = format!("provider.enable.{name}");
            if !has(args, "--confirm") {
                return print_json(&actions::preview(home_dir_from_env(), &id)?);
            }
            print_json(&actions::apply(
                home_dir_from_env(),
                &id,
                actions::ApplyOptions {
                    confirm: true,
                    executable: current_executable(),
                    ..Default::default()
                },
            )?)
        }
        Some("disable") => {
            let name = args
                .get(1)
                .ok_or_else(|| anyhow!("provider name required"))?;
            let id = format!("provider.disable.{name}");
            if !has(args, "--confirm") {
                return print_json(&actions::preview(home_dir_from_env(), &id)?);
            }
            print_json(&actions::apply(
                home_dir_from_env(),
                &id,
                actions::ApplyOptions {
                    confirm: true,
                    executable: current_executable(),
                    ..Default::default()
                },
            )?)
        }
        Some("install") => {
            let name = args
                .get(1)
                .ok_or_else(|| anyhow!("provider name required"))?;
            let id = format!("provider.install.{name}");
            if !has(args, "--confirm") {
                return print_json(&actions::preview(home_dir_from_env(), &id)?);
            }
            print_json(&actions::apply(
                home_dir_from_env(),
                &id,
                actions::ApplyOptions {
                    confirm: true,
                    executable: current_executable(),
                    ..Default::default()
                },
            )?)
        }
        _ => Err(anyhow!("unknown providers command")),
    }
}

fn cmd_rules(args: &[String]) -> Result<()> {
    match args.first().map(String::as_str) {
        Some("list") | None => print_json(&rules::all_rules()),
        Some("explain") => {
            let id = args.get(1).ok_or_else(|| anyhow!("rule id required"))?;
            let rule = rules::explain_rule(id).ok_or_else(|| anyhow!("rule not found"))?;
            print_json(&rule)
        }
        _ => Err(anyhow!("unknown rules command")),
    }
}

fn cmd_report(args: &[String]) -> Result<()> {
    match args.first().map(String::as_str) {
        Some("html") => {
            let input = value_after(args, "--input").unwrap_or("");
            let output = value_after(args, "--output").unwrap_or("nightward-report.html");
            let diff = if has_report_diff_paths(args) {
                let (base, head) = report_diff_paths(args)?;
                let base_report = load_report(base)?;
                let head_report = load_report(head)?;
                Some(reportdiff::diff(
                    base.to_string(),
                    &base_report,
                    head.to_string(),
                    &head_report,
                ))
            } else {
                None
            };
            let scan = if let Some(diff) = diff.as_ref() {
                load_report(&diff.head)?
            } else if input.is_empty() {
                scan_home(home_dir_from_env())?
            } else {
                load_report(input)?
            };
            let analysis = analysis::run(
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
            let plan = fixplan::plan(
                &scan,
                Selector {
                    all: true,
                    ..Selector::default()
                },
            );
            let html = reporthtml::render(&scan, Some(&analysis), None, Some(&plan), diff.as_ref());
            reporthtml::write(output, &html)?;
            println!("{output}");
            Ok(())
        }
        Some("diff") | Some("changes") => {
            let (base, head) = report_diff_paths(args)?;
            let base_report = load_report(base)?;
            let head_report = load_report(head)?;
            print_json(&reportdiff::diff(
                base.to_string(),
                &base_report,
                head.to_string(),
                &head_report,
            ))
        }
        Some("history") => print_json(&schedule::status(home_dir_from_env()).history),
        Some("latest") => {
            let status = schedule::status(home_dir_from_env());
            if let Some(path) = status.last_report {
                println!("{path}");
            }
            Ok(())
        }
        Some("index") => print_json(&schedule::status(home_dir_from_env())),
        _ => Err(anyhow!("unknown report command")),
    }
}

fn cmd_policy(args: &[String]) -> Result<()> {
    match args.first().map(String::as_str) {
        Some("init") => {
            let path = value_after(args, "--output").unwrap_or("nightward-policy.yml");
            policy::init_file(path)?;
            println!("{path}");
            Ok(())
        }
        Some("explain") => print_json(&PolicyConfig::default()),
        Some("check") | Some("sarif") | Some("badge") => {
            let config = value_after(args, "--config")
                .map(policy::load)
                .transpose()?
                .unwrap_or_default();
            let scan = scan_for_args(args)?;
            let include_analysis = config.include_analysis || has(args, "--include-analysis");
            let analysis_providers: Vec<String> = value_after(args, "--with")
                .map(|value| {
                    value
                        .split(',')
                        .map(|part| part.trim().to_string())
                        .collect()
                })
                .unwrap_or_else(|| config.analysis_providers.clone());
            let analysis = if include_analysis {
                Some(analysis::run(
                    &scan,
                    AnalysisOptions {
                        mode: scan.scan_mode.clone(),
                        workspace: scan.workspace.clone(),
                        with: analysis_providers,
                        online: config.allow_online_providers || has(args, "--online"),
                        package: String::new(),
                        finding_id: String::new(),
                    },
                ))
            } else {
                None
            };
            let report = policy::check(&scan, &config, analysis.as_ref());
            let output = value_after(args, "--output");
            let value = match args.first().map(String::as_str) {
                Some("sarif") => policy::sarif(&scan, Some(&report)),
                Some("badge") => serde_json::to_value(policy::badge(
                    &report,
                    value_after(args, "--sarif-url")
                        .or_else(|| value_after(args, "--sarif"))
                        .unwrap_or("")
                        .to_string(),
                ))?,
                _ => serde_json::to_value(&report)?,
            };
            if let Some(output) = output.filter(|value| *value != "-") {
                std::fs::write(
                    output,
                    format!("{}\n", serde_json::to_string_pretty(&value)?),
                )?;
            }
            println!("{}", serde_json::to_string_pretty(&value)?);
            if args.first().map(String::as_str) == Some("check")
                && has(args, "--strict")
                && !report.passed
            {
                return Err(anyhow!("policy failed"));
            }
            Ok(())
        }
        _ => Err(anyhow!("unknown policy command")),
    }
}

fn cmd_mcp(args: &[String]) -> Result<()> {
    if args.first().map(String::as_str) == Some("serve") {
        return mcpserver::serve();
    }
    Err(anyhow!("unknown mcp command"))
}

fn cmd_snapshot(args: &[String]) -> Result<()> {
    if args.first().map(String::as_str) == Some("plan") || args.is_empty() {
        let destination = value_after(args, "--output").unwrap_or("nightward-snapshot");
        print_json(&snapshot::plan(home_dir_from_env(), destination))
    } else {
        Err(anyhow!("unknown snapshot command"))
    }
}

fn cmd_backup(args: &[String]) -> Result<()> {
    match args.first().map(String::as_str) {
        Some("plan") | None => print_json(&backupplan::plan(home_dir_from_env())),
        Some("create") | Some("snapshot") => {
            if !has(args, "--confirm") {
                return print_json(&actions::preview(home_dir_from_env(), "backup.snapshot")?);
            }
            print_json(&actions::apply(
                home_dir_from_env(),
                "backup.snapshot",
                actions::ApplyOptions {
                    confirm: true,
                    executable: current_executable(),
                    ..Default::default()
                },
            )?)
        }
        _ => Err(anyhow!("unknown backup command")),
    }
}

fn cmd_schedule(args: &[String]) -> Result<()> {
    match args.first().map(String::as_str) {
        Some("status") | None => print_json(&schedule::status(home_dir_from_env())),
        Some("plan") => print_json(&schedule::plan(
            home_dir_from_env(),
            true,
            &current_executable(),
        )),
        Some("install") => {
            if !has(args, "--confirm") {
                return print_json(&actions::preview(home_dir_from_env(), "schedule.install")?);
            }
            print_json(&actions::apply(
                home_dir_from_env(),
                "schedule.install",
                actions::ApplyOptions {
                    confirm: true,
                    executable: current_executable(),
                    ..Default::default()
                },
            )?)
        }
        Some("remove") => {
            if !has(args, "--confirm") {
                return print_json(&actions::preview(home_dir_from_env(), "schedule.remove")?);
            }
            print_json(&actions::apply(
                home_dir_from_env(),
                "schedule.remove",
                actions::ApplyOptions {
                    confirm: true,
                    executable: current_executable(),
                    ..Default::default()
                },
            )?)
        }
        _ => Err(anyhow!("unknown schedule command")),
    }
}

fn cmd_actions(args: &[String]) -> Result<()> {
    match args.first().map(String::as_str) {
        Some("list") | None => print_json(&actions::list(home_dir_from_env())),
        Some("preview") => {
            let id = args.get(1).ok_or_else(|| anyhow!("action id required"))?;
            print_json(&actions::preview(home_dir_from_env(), id)?)
        }
        Some("apply") => {
            let id = args.get(1).ok_or_else(|| anyhow!("action id required"))?;
            print_json(&actions::apply(
                home_dir_from_env(),
                id,
                actions::ApplyOptions {
                    confirm: has(args, "--confirm"),
                    executable: current_executable(),
                    policy_path: value_after(args, "--policy")
                        .or_else(|| value_after(args, "--config"))
                        .unwrap_or("")
                        .to_string(),
                    finding_id: value_after(args, "--finding").unwrap_or("").to_string(),
                    rule: value_after(args, "--rule").unwrap_or("").to_string(),
                    reason: value_after(args, "--reason").unwrap_or("").to_string(),
                },
            )?)
        }
        _ => Err(anyhow!("unknown actions command")),
    }
}

fn cmd_disclosure(args: &[String]) -> Result<()> {
    match args.first().map(String::as_str) {
        Some("status") | None => print_json(&state::disclosure_status(home_dir_from_env())),
        Some("accept") => print_json(&actions::apply(
            home_dir_from_env(),
            "disclosure.accept",
            actions::ApplyOptions {
                confirm: true,
                executable: current_executable(),
                ..Default::default()
            },
        )?),
        _ => Err(anyhow!("unknown disclosure command")),
    }
}

fn selector(args: &[String]) -> Selector {
    Selector {
        all: has(args, "--all") || (!has(args, "--finding") && !has(args, "--rule")),
        finding: value_after(args, "--finding").unwrap_or("").to_string(),
        rule: value_after(args, "--rule").unwrap_or("").to_string(),
    }
}

fn scan_for_args(args: &[String]) -> Result<nightward_core::Report> {
    if let Some(workspace) = value_after(args, "--workspace")
        .or_else(|| value_after(args, "-w"))
        .filter(|workspace| !workspace.is_empty())
    {
        scan_workspace(workspace)
    } else {
        scan_home(home_dir_from_env())
    }
}

fn report_diff_paths(args: &[String]) -> Result<(&str, &str)> {
    let base = value_after(args, "--base")
        .or_else(|| value_after(args, "--from"))
        .ok_or_else(|| anyhow!("--base or --from is required"))?;
    let head = value_after(args, "--head")
        .or_else(|| value_after(args, "--to"))
        .ok_or_else(|| anyhow!("--head or --to is required"))?;
    Ok((base, head))
}

fn has_report_diff_paths(args: &[String]) -> bool {
    (value_after(args, "--base").is_some() || value_after(args, "--from").is_some())
        && (value_after(args, "--head").is_some() || value_after(args, "--to").is_some())
}

fn print_json(value: &impl Serialize) -> Result<()> {
    println!("{}", serde_json::to_string_pretty(value)?);
    Ok(())
}

fn value_after<'a>(args: &'a [String], key: &str) -> Option<&'a str> {
    args.iter()
        .position(|arg| arg == key)
        .and_then(|index| args.get(index + 1))
        .map(String::as_str)
}

fn has(args: &[String], key: &str) -> bool {
    args.iter().any(|arg| arg == key)
}

fn selected_providers_for_args(args: &[String]) -> Vec<String> {
    value_after(args, "--with")
        .map(|value| {
            value
                .split(',')
                .map(|part| part.trim().to_string())
                .filter(|part| !part.is_empty())
                .collect()
        })
        .unwrap_or_else(|| {
            state::load_settings(home_dir_from_env())
                .map(|settings| settings.selected_providers)
                .unwrap_or_default()
        })
}

fn online_for_args(args: &[String]) -> bool {
    has(args, "--online")
        || state::load_settings(home_dir_from_env())
            .map(|settings| settings.allow_online_providers)
            .unwrap_or(false)
}

fn current_executable() -> String {
    env::current_exe()
        .ok()
        .map(|path| path.display().to_string())
        .unwrap_or_else(|| "nightward".to_string())
}

fn positional_after_command<'a>(args: &'a [String], command: &str) -> Option<&'a str> {
    let mut iter = args
        .iter()
        .skip_while(|arg| arg.as_str() != command)
        .skip(1);
    while let Some(arg) = iter.next() {
        if arg.starts_with('-') {
            if option_takes_value(arg) {
                iter.next();
            }
            continue;
        }
        return Some(arg.as_str());
    }
    None
}

fn option_takes_value(option: &str) -> bool {
    matches!(
        option,
        "--workspace"
            | "-w"
            | "--with"
            | "--output"
            | "--config"
            | "--sarif-url"
            | "--sarif"
            | "--base"
            | "--head"
            | "--from"
            | "--to"
            | "--finding"
            | "--rule"
            | "--format"
            | "--input"
    )
}

fn risk_label(risk: nightward_core::RiskLevel) -> &'static str {
    match risk {
        nightward_core::RiskLevel::Critical => "critical",
        nightward_core::RiskLevel::High => "high",
        nightward_core::RiskLevel::Medium => "medium",
        nightward_core::RiskLevel::Low => "low",
        nightward_core::RiskLevel::Info => "info",
    }
}

fn version() -> &'static str {
    option_env!("NIGHTWARD_VERSION").unwrap_or(env!("CARGO_PKG_VERSION"))
}

fn print_help() {
    println!(
        "Nightward audits AI agent state, MCP config, and dotfiles sync risk.\n\nUSAGE:\n  nightward                         Open the TUI\n  nightward tui --input scan.json   Review a saved report in the TUI\n  nightward tui --from old.json --to new.json\n  nightward scan --json             Scan HOME\n  nightward scan --workspace . --json\n  nightward analyze --all --with gitleaks --json\n  nightward providers doctor --with trivy --online --json\n  nightward providers enable gitleaks --confirm\n  nightward providers install gitleaks --confirm\n  nightward disclosure accept\n  nightward fix plan --all --json\n  nightward backup create --confirm\n  nightward schedule install --confirm\n  nightward actions list --json\n  nightward actions apply backup.snapshot --confirm\n  nightward actions apply reports.cleanup --confirm\n  nightward actions apply cache.cleanup --confirm\n  nightward actions apply policy.ignore --finding <id> --reason \"reviewed\" --confirm\n  nightward report html --input scan.json --output report.html\n  nightward report html --from old.json --to new.json --output report.html\n  nightward policy check --json\n  nightward mcp serve\n\nNightward is local-first and read-only by default. Write-capable actions require disclosure acceptance and explicit confirmation."
    );
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    #[test]
    fn scan_for_args_prefers_workspace_target() {
        let dir = tempfile::tempdir().expect("temp dir");
        fs::write(
            dir.path().join(".mcp.json"),
            r#"{"mcpServers":{"demo":{"command":"npx","args":["@modelcontextprotocol/server-filesystem"]}}}"#,
        )
        .expect("write fixture config");
        let args = vec![
            "check".to_string(),
            "--workspace".to_string(),
            dir.path().display().to_string(),
        ];

        let report = scan_for_args(&args).expect("workspace scan");

        assert_eq!(report.scan_mode, "workspace");
        assert_eq!(report.summary.total_items, 1);
        assert!(report.summary.total_findings > 0);
    }

    #[test]
    fn analyze_subject_argument_skips_flags() {
        let args = vec![
            "finding".to_string(),
            "--json".to_string(),
            "mcp_unpinned_package-123".to_string(),
        ];

        assert_eq!(
            positional_after_command(&args, "finding"),
            Some("mcp_unpinned_package-123")
        );
    }

    #[test]
    fn report_diff_accepts_from_to_aliases() {
        let args = vec![
            "diff".to_string(),
            "--from".to_string(),
            "before.json".to_string(),
            "--to".to_string(),
            "after.json".to_string(),
        ];

        assert_eq!(
            report_diff_paths(&args).expect("diff paths"),
            ("before.json", "after.json")
        );
    }
}
