package rules

import (
	"sort"
	"strings"

	"github.com/jsonbored/nightward/internal/inventory"
)

type Rule struct {
	ID              string              `json:"id"`
	Category        string              `json:"category"`
	DefaultSeverity inventory.RiskLevel `json:"default_severity"`
	Title           string              `json:"title"`
	Description     string              `json:"description"`
	Recommendation  string              `json:"recommended_action"`
	DocsURL         string              `json:"docs_url,omitempty"`
}

func List() []Rule {
	out := []Rule{
		{ID: "mcp_server_review", Category: "mcp", DefaultSeverity: inventory.RiskInfo, Title: "Review MCP server", Description: "Every MCP server is an executable trust boundary that should be reviewed before syncing.", Recommendation: "Confirm source, permissions, and local assumptions before backing up config.", DocsURL: "https://github.com/JSONbored/nightward/blob/main/docs/remediation.md"},
		{ID: "mcp_unpinned_package", Category: "mcp", DefaultSeverity: inventory.RiskHigh, Title: "Unpinned package executor", Description: "An MCP server invokes a package executor without an obvious pinned package version.", Recommendation: "Pin package versions or replace the executor with a reviewed local binary.", DocsURL: "https://github.com/JSONbored/nightward/blob/main/docs/remediation.md"},
		{ID: "mcp_secret_env", Category: "mcp", DefaultSeverity: inventory.RiskMedium, Title: "Sensitive environment key", Description: "An MCP server references a sensitive environment key.", Recommendation: "Keep secret values outside portable config and document required env keys.", DocsURL: "https://github.com/JSONbored/nightward/blob/main/docs/privacy-model.md"},
		{ID: "mcp_secret_header", Category: "mcp", DefaultSeverity: inventory.RiskHigh, Title: "Sensitive header key", Description: "A URL-shaped MCP server references a sensitive header key.", Recommendation: "Externalize inline header values and keep only reviewed env references in config.", DocsURL: "https://github.com/JSONbored/nightward/blob/main/docs/privacy-model.md"},
		{ID: "mcp_shell_command", Category: "mcp", DefaultSeverity: inventory.RiskHigh, Title: "Shell wrapper", Description: "An MCP server executes through a shell wrapper.", Recommendation: "Prefer direct executable invocation when the wrapper is a simple passthrough.", DocsURL: "https://github.com/JSONbored/nightward/blob/main/docs/remediation.md"},
		{ID: "mcp_broad_filesystem", Category: "mcp", DefaultSeverity: inventory.RiskMedium, Title: "Broad filesystem access", Description: "An MCP server appears to request broad filesystem access.", Recommendation: "Replace broad mounts with explicit reviewed paths when possible.", DocsURL: "https://github.com/JSONbored/nightward/blob/main/docs/remediation.md"},
		{ID: "mcp_local_endpoint", Category: "mcp", DefaultSeverity: inventory.RiskMedium, Title: "Local endpoint", Description: "A URL-shaped MCP server points at loopback, localhost, or private network state.", Recommendation: "Document the local service dependency and avoid assuming it exists on every machine.", DocsURL: "https://github.com/JSONbored/nightward/blob/main/docs/privacy-model.md"},
		{ID: "mcp_local_token_path", Category: "mcp", DefaultSeverity: inventory.RiskMedium, Title: "Local token path", Description: "An MCP server argument appears to reference local token or credential material.", Recommendation: "Keep token paths out of portable config and recreate them per machine.", DocsURL: "https://github.com/JSONbored/nightward/blob/main/docs/privacy-model.md"},
		{ID: "mcp_symlink_config", Category: "mcp", DefaultSeverity: inventory.RiskMedium, Title: "Symlinked MCP config", Description: "An MCP config path resolves through a symlink, which can hide the real source being synced or reviewed.", Recommendation: "Review the resolved target and keep machine-local symlinks out of portable dotfiles.", DocsURL: "https://github.com/JSONbored/nightward/blob/main/docs/privacy-model.md"},
		{ID: "mcp_unknown_command", Category: "mcp", DefaultSeverity: inventory.RiskMedium, Title: "Unknown MCP server shape", Description: "An MCP server lacks both an executable command and URL endpoint.", Recommendation: "Review the config manually and update Nightward fixtures if this is a supported shape.", DocsURL: "https://github.com/JSONbored/nightward/blob/main/docs/adapters.md"},
		{ID: "mcp_parse_failed", Category: "mcp", DefaultSeverity: inventory.RiskMedium, Title: "MCP parse failure", Description: "Nightward could not parse an MCP config file.", Recommendation: "Fix the config syntax, then rerun scan and policy checks.", DocsURL: "https://github.com/JSONbored/nightward/blob/main/docs/adapters.md"},
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func Find(idOrPrefix string) (Rule, bool) {
	if idOrPrefix == "" {
		return Rule{}, false
	}
	for _, rule := range List() {
		if rule.ID == idOrPrefix {
			return rule, true
		}
	}
	var matched []Rule
	for _, rule := range List() {
		if strings.HasPrefix(rule.ID, idOrPrefix) {
			matched = append(matched, rule)
		}
	}
	if len(matched) == 1 {
		return matched[0], true
	}
	return Rule{}, false
}
