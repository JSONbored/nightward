package fixplan

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func TestBuildPreviewProducesRedactedPatchForInlineSecret(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mcp.json")
	if err := os.WriteFile(path, []byte(`{
  "mcpServers": {
    "demo": {
      "command": "node",
      "args": ["server.js"],
      "env": {
        "API_TOKEN": "super-secret-value"
      }
    }
  }
}`), 0600); err != nil {
		t.Fatal(err)
	}

	report := inventory.Report{
		GeneratedAt: time.Date(2026, 4, 30, 7, 0, 0, 0, time.UTC),
		Findings: []inventory.Finding{
			{
				ID:             "mcp_secret_env-111111111111",
				Tool:           "Generic MCP",
				Path:           path,
				Server:         "demo",
				Severity:       inventory.RiskCritical,
				Rule:           "mcp_secret_env",
				FixAvailable:   true,
				FixKind:        inventory.FixExternalizeSecret,
				RequiresReview: true,
				FixSteps:       []string{"Remove the inline value."},
				PatchHint:      &inventory.PatchHint{Kind: inventory.FixExternalizeSecret, EnvKey: "API_TOKEN", InlineSecret: true},
			},
		},
	}

	preview := BuildPreview(report, Selector{All: true})
	if preview.Summary.Patchable != 1 {
		t.Fatalf("expected patchable preview: %#v", preview)
	}
	diff := PreviewDiff(preview)
	if !strings.Contains(diff, "${API_TOKEN}") || !strings.Contains(diff, "[redacted]") {
		t.Fatalf("unexpected preview diff:\n%s", diff)
	}
	if strings.Contains(diff, "super-secret-value") {
		t.Fatalf("preview leaked secret value:\n%s", diff)
	}
}

func TestBuildPreviewDoesNotGuessPackageVersions(t *testing.T) {
	report := inventory.Report{Findings: []inventory.Finding{
		{
			ID:             "mcp_unpinned_package-111111111111",
			Path:           filepath.Join(t.TempDir(), ".mcp.json"),
			Server:         "demo",
			Rule:           "mcp_unpinned_package",
			FixKind:        inventory.FixPinPackage,
			RequiresReview: true,
			PatchHint:      &inventory.PatchHint{Kind: inventory.FixPinPackage, Package: "@example/server"},
		},
	}}

	preview := BuildPreview(report, Selector{All: true})
	if preview.Summary.Patchable != 0 || !strings.Contains(preview.Patches[0].Reason, "will not guess") {
		t.Fatalf("expected blocked package preview: %#v", preview)
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
