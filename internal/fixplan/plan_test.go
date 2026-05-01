package fixplan

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jsonbored/nightward/internal/inventory"
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

func TestBuildPreviewProducesRedactedPatchForInlineHeaderSecret(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mcp.yaml")
	if err := os.WriteFile(path, []byte(`mcpServers:
  remote:
    url: https://mcp.example.test
    headers:
      Authorization: Bearer super-secret-value
`), 0600); err != nil {
		t.Fatal(err)
	}

	report := inventory.Report{Findings: []inventory.Finding{
		{
			ID:             "mcp_secret_header-111111111111",
			Tool:           "Generic MCP",
			Path:           path,
			Server:         "remote",
			Rule:           "mcp_secret_header",
			Severity:       inventory.RiskCritical,
			FixKind:        inventory.FixExternalizeSecret,
			RequiresReview: true,
			PatchHint:      &inventory.PatchHint{Kind: inventory.FixExternalizeSecret, EnvKey: "AUTHORIZATION", HeaderKey: "Authorization", InlineSecret: true},
		},
	}}

	preview := BuildPreview(report, Selector{All: true})
	if preview.Summary.Patchable != 1 {
		t.Fatalf("expected patchable header preview: %#v", preview)
	}
	diff := PreviewDiff(preview)
	if !strings.Contains(diff, "headers.Authorization") || !strings.Contains(diff, "${AUTHORIZATION}") || strings.Contains(diff, "super-secret-value") {
		t.Fatalf("unexpected header preview diff:\n%s", diff)
	}
}

func TestBuildPreviewProducesReviewDiffsForEndpointAndFilesystemScope(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`[mcp_servers.local]
url = "http://127.0.0.1:8787/mcp"

[mcp_servers.files]
command = "npx"
args = ["@modelcontextprotocol/server-filesystem", "/Users/example"]
`), 0600); err != nil {
		t.Fatal(err)
	}
	report := inventory.Report{Findings: []inventory.Finding{
		{
			ID:             "mcp_local_endpoint-111111111111",
			Path:           path,
			Server:         "local",
			Rule:           "mcp_local_endpoint",
			FixKind:        inventory.FixManualReview,
			RequiresReview: true,
			PatchHint:      &inventory.PatchHint{Kind: inventory.FixManualReview, Replacement: "<reviewed-portable-or-local-overlay-url>"},
		},
		{
			ID:             "mcp_broad_filesystem-222222222222",
			Path:           path,
			Server:         "files",
			Rule:           "mcp_broad_filesystem",
			FixKind:        inventory.FixNarrowFilesystem,
			RequiresReview: true,
			PatchHint:      &inventory.PatchHint{Kind: inventory.FixNarrowFilesystem, DirectArgs: []string{"@modelcontextprotocol/server-filesystem", "<explicit-project-or-config-path>"}},
		},
	}}

	preview := BuildPreview(report, Selector{All: true})
	if preview.Summary.Patchable != 0 || preview.Summary.Review != 2 {
		t.Fatalf("expected review-only previews: %#v", preview.Summary)
	}
	diff := PreviewDiff(preview)
	for _, want := range []string{"<reviewed-portable-or-local-overlay-url>", "<explicit-project-or-config-path>"} {
		if !strings.Contains(diff, want) {
			t.Fatalf("review diff missing %q:\n%s", want, diff)
		}
	}
}

func TestBuildPreviewProducesReviewDiffForCredentialPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	if err := os.WriteFile(path, []byte(`{"mcpServers":{"token":{"command":"node","args":["server.js","--token-file","/Users/example/.token"]}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	report := inventory.Report{Findings: []inventory.Finding{
		{
			ID:             "mcp_local_token_path-111111111111",
			Path:           path,
			Server:         "token",
			Rule:           "mcp_local_token_path",
			FixKind:        inventory.FixManualReview,
			RequiresReview: true,
			PatchHint:      &inventory.PatchHint{Kind: inventory.FixManualReview, DirectArgs: []string{"server.js", "--token-file", "/Users/example/.token"}, Replacement: "<local-secret-path-kept-out-of-dotfiles>"},
		},
	}}

	preview := BuildPreview(report, Selector{All: true})
	if preview.Summary.Review != 1 {
		t.Fatalf("expected review-only token path preview: %#v", preview.Summary)
	}
	diff := PreviewDiff(preview)
	if strings.Contains(diff, "/Users/example/.token") || !strings.Contains(diff, "[redacted]") {
		t.Fatalf("credential path preview was not redacted:\n%s", diff)
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

func TestBuildPreviewReplacesSimpleShellWrapperAndRedactsArgs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`[mcp_servers.demo]
command = "sh"
args = ["-c", "node server.js --api-key super-secret-value"]
`), 0600); err != nil {
		t.Fatal(err)
	}

	report := inventory.Report{Findings: []inventory.Finding{
		{
			ID:             "mcp_shell_command-111111111111",
			Tool:           "Codex",
			Path:           path,
			Server:         "demo",
			Rule:           "mcp_shell_command",
			Severity:       inventory.RiskMedium,
			FixKind:        inventory.FixReplaceShellWrapper,
			RequiresReview: true,
			PatchHint: &inventory.PatchHint{
				Kind:          inventory.FixReplaceShellWrapper,
				DirectCommand: "node",
				DirectArgs:    []string{"server.js", "--api-key", "super-secret-value"},
			},
		},
	}}

	preview := BuildPreview(report, Selector{All: true})
	if preview.Summary.Patchable != 1 {
		t.Fatalf("expected shell wrapper patch: %#v", preview)
	}
	markdown := PreviewMarkdown(preview)
	if !strings.Contains(markdown, `command = "node"`) || !strings.Contains(markdown, "[redacted]") {
		t.Fatalf("unexpected shell preview markdown:\n%s", markdown)
	}
	if strings.Contains(markdown, "super-secret-value") {
		t.Fatalf("shell wrapper preview leaked secret:\n%s", markdown)
	}
}

func TestBuildPreviewExplainsUnpatchableSecretShapes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")
	if err := os.WriteFile(path, []byte(`{"mcpServers":{"demo":{"env":{"API_TOKEN":"[redacted]"}}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	report := inventory.Report{Findings: []inventory.Finding{
		{
			ID:        "missing-server",
			Rule:      "mcp_secret_env",
			PatchHint: &inventory.PatchHint{Kind: inventory.FixExternalizeSecret, EnvKey: "API_TOKEN", InlineSecret: true},
		},
		{
			ID:        "missing-env-key",
			Server:    "demo",
			Rule:      "mcp_secret_env",
			Path:      path,
			PatchHint: &inventory.PatchHint{Kind: inventory.FixExternalizeSecret, InlineSecret: true},
		},
	}}

	preview := BuildPreview(report, Selector{All: true})
	if preview.Summary.Blocked != 2 || preview.Summary.Patchable != 0 {
		t.Fatalf("expected blocked previews: %#v", preview.Summary)
	}
	combined := PreviewMarkdown(preview)
	for _, want := range []string{"server name is unknown", "env key is unknown"} {
		if !strings.Contains(combined, want) {
			t.Fatalf("preview missing reason %q:\n%s", want, combined)
		}
	}
}

func TestPreviewMarkdownAndDiffEmptySelection(t *testing.T) {
	preview := BuildPreview(inventory.Report{}, Selector{All: true})
	if got := PreviewDiff(preview); !strings.Contains(got, "No redacted patch previews") {
		t.Fatalf("unexpected empty diff: %s", got)
	}
	if got := PreviewMarkdown(preview); !strings.Contains(got, "No findings matched") {
		t.Fatalf("unexpected empty markdown: %s", got)
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
