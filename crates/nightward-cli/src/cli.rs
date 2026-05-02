use crate::tui;
use anyhow::{anyhow, Result};
use nightward_core::analysis::{self, Options as AnalysisOptions};
use nightward_core::fixplan::{self, Selector};
use nightward_core::inventory::{
    home_dir_from_env, load_report, scan_home, scan_workspace, write_report,
};
use nightward_core::policy::{self, PolicyConfig};
use nightward_core::{
    backupplan, mcpserver, providers, reportdiff, reporthtml, rules, schedule, snapshot,
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
        "schedule" => cmd_schedule(&args),
        "help" | "--help" | "-h" => {
            print_help();
            Ok(())
        }
        unknown => Err(anyhow!("unknown command {unknown}; run nightward help")),
    }
}

fn cmd_scan(args: &[String]) -> Result<()> {
    let workspace = value_after(args, "--workspace").or_else(|| value_after(args, "-w"));
    let output = value_after(args, "--output");
    let report = if let Some(workspace) = workspace {
        scan_workspace(workspace)?
    } else {
        scan_home(home_dir_from_env())?
    };
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
        "providers": providers::statuses(&[], false),
        "schedule": schedule::status(home),
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
    let workspace = value_after(args, "--workspace").unwrap_or("");
    let scan = if workspace.is_empty() {
        scan_home(home_dir_from_env())?
    } else {
        scan_workspace(workspace)?
    };
    let options = AnalysisOptions {
        mode: scan.scan_mode.clone(),
        workspace: scan.workspace.clone(),
        with: value_after(args, "--with")
            .map(|value| {
                value
                    .split(',')
                    .map(|part| part.trim().to_string())
                    .collect()
            })
            .unwrap_or_default(),
        online: has(args, "--online"),
        package: if args.first().map(String::as_str) == Some("package") {
            args.get(1).cloned().unwrap_or_default()
        } else {
            String::new()
        },
        finding_id: if args.first().map(String::as_str) == Some("finding") {
            args.get(1).cloned().unwrap_or_default()
        } else {
            String::new()
        },
    };
    let report = analysis::run(&scan, options);
    print_json(&report)
}

fn cmd_providers(args: &[String]) -> Result<()> {
    let selected: Vec<String> = value_after(args, "--with")
        .map(|value| {
            value
                .split(',')
                .map(|part| part.trim().to_string())
                .collect()
        })
        .unwrap_or_default();
    let online = has(args, "--online");
    match args.first().map(String::as_str) {
        Some("list") | None => print_json(&providers::providers()),
        Some("doctor") => print_json(&providers::statuses(&selected, online)),
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
            let scan = if input.is_empty() {
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
            let html = reporthtml::render(&scan, Some(&analysis), None, Some(&plan));
            reporthtml::write(output, &html)?;
            println!("{output}");
            Ok(())
        }
        Some("diff") | Some("changes") => {
            let base = value_after(args, "--base").ok_or_else(|| anyhow!("--base is required"))?;
            let head = value_after(args, "--head").ok_or_else(|| anyhow!("--head is required"))?;
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
            let scan = scan_home(home_dir_from_env())?;
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

fn cmd_schedule(args: &[String]) -> Result<()> {
    match args.first().map(String::as_str) {
        Some("status") | None => print_json(&schedule::status(home_dir_from_env())),
        Some("plan") => print_json(&schedule::plan(true)),
        Some("install") => print_json(&schedule::plan(true)),
        Some("remove") => print_json(&schedule::plan(false)),
        _ => Err(anyhow!("unknown schedule command")),
    }
}

fn selector(args: &[String]) -> Selector {
    Selector {
        all: has(args, "--all") || (!has(args, "--finding") && !has(args, "--rule")),
        finding: value_after(args, "--finding").unwrap_or("").to_string(),
        rule: value_after(args, "--rule").unwrap_or("").to_string(),
    }
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
        "Nightward audits AI agent state, MCP config, and dotfiles sync risk.\n\nUSAGE:\n  nightward                 Open the TUI\n  nightward scan --json     Scan HOME\n  nightward scan --workspace . --json\n  nightward analyze --all --with gitleaks --json\n  nightward providers doctor --with trivy --online --json\n  nightward fix plan --all --json\n  nightward report html --input scan.json --output report.html\n  nightward policy check --json\n  nightward mcp serve\n\nNightward is local-first, read-only by default, and never enables online providers without --online."
    );
}
