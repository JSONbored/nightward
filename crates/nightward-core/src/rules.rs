use crate::{FixKind, RiskLevel};
use serde::Serialize;

#[derive(Debug, Clone, Serialize)]
pub struct Rule {
    pub id: &'static str,
    pub severity: RiskLevel,
    pub fix_kind: FixKind,
    pub title: &'static str,
    pub docs_url: &'static str,
}

pub fn all_rules() -> Vec<Rule> {
    vec![
        Rule {
            id: "mcp_secret_env",
            severity: RiskLevel::Critical,
            fix_kind: FixKind::ExternalizeSecret,
            title: "MCP server stores a sensitive environment variable inline",
            docs_url: "https://nightward.aethereal.dev/guide/remediation",
        },
        Rule {
            id: "mcp_secret_header",
            severity: RiskLevel::Critical,
            fix_kind: FixKind::ExternalizeSecret,
            title: "MCP server stores a sensitive header inline",
            docs_url: "https://nightward.aethereal.dev/guide/remediation",
        },
        Rule {
            id: "mcp_unpinned_package",
            severity: RiskLevel::High,
            fix_kind: FixKind::PinPackage,
            title: "MCP server runs a package executor without an obvious pin",
            docs_url: "https://nightward.aethereal.dev/guide/mcp-security",
        },
        Rule {
            id: "mcp_shell_wrapper",
            severity: RiskLevel::High,
            fix_kind: FixKind::ReplaceShellWrapper,
            title: "MCP server runs through a shell wrapper",
            docs_url: "https://nightward.aethereal.dev/guide/mcp-security",
        },
        Rule {
            id: "mcp_local_endpoint",
            severity: RiskLevel::Medium,
            fix_kind: FixKind::ManualReview,
            title: "MCP server references a machine-local endpoint",
            docs_url: "https://nightward.aethereal.dev/guide/mcp-security",
        },
        Rule {
            id: "mcp_broad_filesystem",
            severity: RiskLevel::Medium,
            fix_kind: FixKind::NarrowFilesystem,
            title: "MCP server can access a broad filesystem path",
            docs_url: "https://nightward.aethereal.dev/guide/mcp-security",
        },
        Rule {
            id: "mcp_local_token_path",
            severity: RiskLevel::High,
            fix_kind: FixKind::ManualReview,
            title: "MCP server references a local credential path",
            docs_url: "https://nightward.aethereal.dev/guide/privacy-model",
        },
        Rule {
            id: "mcp_docker_socket",
            severity: RiskLevel::High,
            fix_kind: FixKind::ManualReview,
            title: "MCP server can control Docker or container host state",
            docs_url: "https://nightward.aethereal.dev/guide/mcp-security",
        },
        Rule {
            id: "mcp_typosquat_package",
            severity: RiskLevel::Medium,
            fix_kind: FixKind::ManualReview,
            title: "MCP server package resembles a trusted namespace",
            docs_url: "https://nightward.aethereal.dev/guide/mcp-security",
        },
        Rule {
            id: "mcp_untrusted_package_source",
            severity: RiskLevel::Medium,
            fix_kind: FixKind::ManualReview,
            title: "MCP server launches a remote package or script source",
            docs_url: "https://nightward.aethereal.dev/guide/mcp-security",
        },
        Rule {
            id: "mcp_server_review",
            severity: RiskLevel::Info,
            fix_kind: FixKind::ManualReview,
            title: "MCP server should be reviewed",
            docs_url: "https://nightward.aethereal.dev/reference/rules",
        },
        Rule {
            id: "mcp_unknown_command",
            severity: RiskLevel::Info,
            fix_kind: FixKind::ManualReview,
            title: "MCP server has an unsupported command shape",
            docs_url: "https://nightward.aethereal.dev/reference/rules",
        },
        Rule {
            id: "config_parse_failed",
            severity: RiskLevel::Medium,
            fix_kind: FixKind::ManualReview,
            title: "Nightward could not parse a config file",
            docs_url: "https://nightward.aethereal.dev/use/troubleshooting",
        },
        Rule {
            id: "config_symlink",
            severity: RiskLevel::Info,
            fix_kind: FixKind::ManualReview,
            title: "Config file is a symbolic link",
            docs_url: "https://nightward.aethereal.dev/guide/privacy-model",
        },
        Rule {
            id: "config_stale",
            severity: RiskLevel::Low,
            fix_kind: FixKind::ManualReview,
            title: "Config file has not changed in over 180 days",
            docs_url: "https://nightward.aethereal.dev/guide/privacy-model",
        },
    ]
}

pub fn explain_rule(id: &str) -> Option<Rule> {
    all_rules().into_iter().find(|rule| rule.id == id)
}
