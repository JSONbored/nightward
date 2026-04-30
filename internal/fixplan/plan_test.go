package fixplan

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/shadowbook/nightward/internal/inventory"
)

func TestBuildGroupsFixStatusesAndRedacts(t *testing.T) {
	report := inventory.Report{
		GeneratedAt: time.Date(2026, 4, 30, 7, 0, 0, 0, time.UTC),
		Findings: []inventory.Finding{
			{
				ID:             "mcp_unpinned_package-111111111111",
				Tool:           "Codex",
				Path:           "/tmp/config.toml",
				Severity:       inventory.RiskHigh,
				Rule:           "mcp_unpinned_package",
				Evidence:       "command=npx args=@modelcontextprotocol/server-filesystem",
				FixAvailable:   true,
				FixKind:        inventory.FixPinPackage,
				Confidence:     "high",
				Risk:           inventory.RiskMedium,
				RequiresReview: true,
				FixSummary:     "Pin the package.",
				FixSteps:       []string{"Change the package arg to an explicit version."},
			},
			{
				ID:             "mcp_server_review-222222222222",
				Tool:           "Claude",
				Path:           "/tmp/claude.json",
				Severity:       inventory.RiskInfo,
				Rule:           "mcp_server_review",
				Evidence:       "command=node",
				FixAvailable:   true,
				FixKind:        inventory.FixIgnoreWithReason,
				Confidence:     "medium",
				Risk:           inventory.RiskLow,
				RequiresReview: false,
				FixSummary:     "Document why this server is expected.",
			},
			{
				ID:       "unknown-333333333333",
				Tool:     "MCP",
				Path:     "/tmp/mcp.json",
				Severity: inventory.RiskMedium,
				Rule:     "unknown",
			},
		},
	}

	plan := Build(report, Selector{All: true})
	if plan.Summary.Total != 3 || plan.Summary.Safe != 1 || plan.Summary.Review != 1 || plan.Summary.Blocked != 1 {
		t.Fatalf("unexpected summary: %#v", plan.Summary)
	}

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "super-secret-value") {
		t.Fatal("fix plan leaked a secret value")
	}

	markdown := Markdown(plan)
	if !strings.Contains(markdown, "Nightward Fix Plan") || !strings.Contains(markdown, "Pin the package.") {
		t.Fatalf("unexpected markdown export:\n%s", markdown)
	}
}

func TestSelectorFiltersFindings(t *testing.T) {
	report := inventory.Report{Findings: []inventory.Finding{
		{ID: "mcp_shell_command-aaaaaaaaaaaa", Rule: "mcp_shell_command", FixAvailable: true},
		{ID: "mcp_secret_env-bbbbbbbbbbbb", Rule: "mcp_secret_env", FixAvailable: true},
	}}

	byRule := Build(report, Selector{Rule: "mcp_secret_env"})
	if len(byRule.Fixes) != 1 || byRule.Fixes[0].FindingID != "mcp_secret_env-bbbbbbbbbbbb" {
		t.Fatalf("unexpected rule selection: %#v", byRule.Fixes)
	}

	finding, ok := Find(report, "mcp_shell_command")
	if !ok || finding.ID != "mcp_shell_command-aaaaaaaaaaaa" {
		t.Fatalf("prefix find failed: %#v ok=%t", finding, ok)
	}
}
