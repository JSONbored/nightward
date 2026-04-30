package tui

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jsonbored/nightward/internal/inventory"
)

func TestFindingsAndFixPlanViewsRenderRedactedDetails(t *testing.T) {
	report := inventory.Report{
		GeneratedAt: time.Date(2026, 4, 30, 7, 0, 0, 0, time.UTC),
		Findings: []inventory.Finding{
			{
				ID:             "mcp_secret_env-111111111111",
				Tool:           "Codex",
				Path:           "/tmp/config.toml",
				Severity:       inventory.RiskCritical,
				Rule:           "mcp_secret_env",
				Message:        "MCP server stores a sensitive environment key.",
				Evidence:       "env_key=API_TOKEN",
				FixAvailable:   true,
				FixKind:        inventory.FixExternalizeSecret,
				Confidence:     "high",
				Risk:           inventory.RiskHigh,
				RequiresReview: true,
				FixSummary:     "Move API_TOKEN out of this config.",
				FixSteps:       []string{"Remove the inline value for API_TOKEN."},
				Impact:         "Credential material can leak.",
				Why:            "Secrets should stay in secret stores.",
			},
		},
	}
	m := model{report: report, width: 120, height: 40}

	findings := m.findings(116, 32)
	if !strings.Contains(findings, "Suggested Fix") || !strings.Contains(findings, "API_TOKEN") {
		t.Fatalf("findings view missing detail:\n%s", findings)
	}
	if strings.Contains(findings, "super-secret-value") {
		t.Fatal("findings view leaked a secret value")
	}

	fixes := m.fixPlan(116, 32)
	if !strings.Contains(fixes, "Fix Plan") || !strings.Contains(fixes, "externalize-secret") {
		t.Fatalf("fix plan view missing detail:\n%s", fixes)
	}
}

func TestFindingSearchAndHelpRender(t *testing.T) {
	report := inventory.Report{Findings: []inventory.Finding{
		{ID: "one", Tool: "Codex", Rule: "mcp_secret_env", Message: "Sensitive key", Evidence: "env_key=API_TOKEN"},
		{ID: "two", Tool: "Cursor", Rule: "mcp_server_review", Message: "Review server"},
	}}
	m := model{report: report, search: "api_token", width: 100, height: 30}
	filtered := m.filteredFindings()
	if len(filtered) != 1 || filtered[0].ID != "one" {
		t.Fatalf("unexpected filtered findings: %#v", filtered)
	}
	help := m.help(90)
	if !strings.Contains(help, "search findings") || !strings.Contains(help, "do not mutate") {
		t.Fatalf("help text missing expected content:\n%s", help)
	}
	if strings.Contains(help, "show first suggested") {
		t.Fatalf("help text contains stale action wording:\n%s", help)
	}
}

func TestTUIActionSelectionAndDocsURL(t *testing.T) {
	report := inventory.Report{
		Home: "/tmp/nightward-home",
		Items: []inventory.Item{
			{Path: "/tmp/nightward-home/.codex/config.toml"},
		},
		Findings: []inventory.Finding{
			{
				ID:             "mcp_secret_env-111111111111",
				Tool:           "Codex",
				Path:           "/tmp/nightward-home/.codex/config.toml",
				Severity:       inventory.RiskCritical,
				Rule:           "mcp_secret_env",
				Recommendation: "Move API_TOKEN into an environment variable.",
				FixAvailable:   true,
				FixKind:        inventory.FixExternalizeSecret,
				RequiresReview: true,
				FixSteps:       []string{"Remove API_TOKEN=super-" + "secret-value from the MCP config."},
			},
		},
	}

	inventoryModel := model{report: report, tab: 1}
	if got, label, ok := inventoryModel.copySelection(); !ok || label != "path" || got != "/tmp/nightward-home/.codex/config.toml" {
		t.Fatalf("unexpected inventory copy selection: got=%q label=%q ok=%t", got, label, ok)
	}

	findingModel := model{report: report, tab: 2}
	got, label, ok := findingModel.copySelection()
	if !ok || label != "finding action" || strings.Contains(got, "secret-value") || !strings.Contains(got, "[redacted]") {
		t.Fatalf("unexpected finding copy selection: got=%q label=%q ok=%t", got, label, ok)
	}
	if docsURL, ok := findingModel.currentDocsURL(); !ok || docsURL != ruleDocsURL("mcp_secret_env") {
		t.Fatalf("unexpected docs URL: %q ok=%t", docsURL, ok)
	}
}

func TestExportFixPlanWritesRedactedMarkdown(t *testing.T) {
	home := t.TempDir()
	secretValue := "super-" + "secret-value"
	report := inventory.Report{
		GeneratedAt: time.Date(2026, 4, 30, 7, 0, 0, 0, time.UTC),
		Home:        home,
		Findings: []inventory.Finding{
			{
				ID:             "mcp_secret_env-111111111111",
				Tool:           "Codex",
				Path:           filepath.Join(home, ".codex", "config.toml"),
				Severity:       inventory.RiskCritical,
				Rule:           "mcp_secret_env",
				Evidence:       "API_TOKEN=" + secretValue,
				FixAvailable:   true,
				FixKind:        inventory.FixExternalizeSecret,
				Confidence:     "high",
				Risk:           inventory.RiskHigh,
				RequiresReview: true,
				FixSummary:     "Move API_TOKEN=" + secretValue + " out of this config.",
				FixSteps:       []string{"Remove API_TOKEN=" + secretValue + " from the MCP config."},
			},
		},
	}

	path, err := exportFixPlan(report, time.Date(2026, 4, 30, 12, 13, 14, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".local", "state", "nightward", "exports", "fix-plan-20260430T121314Z.md"); path != want {
		t.Fatalf("unexpected export path: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, secretValue) {
		t.Fatalf("export leaked secret value:\n%s", text)
	}
	if !strings.Contains(text, filepath.Join(home, ".codex", "config.toml")) {
		t.Fatalf("export redacted non-secret path:\n%s", text)
	}
	if !strings.Contains(text, "[redacted]") || !strings.Contains(text, "API_TOKEN") {
		t.Fatalf("export missing redacted guidance:\n%s", text)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("expected private export mode 0600, got %s", got)
	}
}

func TestClipboardAndOpenCommandBuilders(t *testing.T) {
	lookup := func(name string) (string, error) {
		if name == "xclip" {
			return "/usr/bin/xclip", nil
		}
		return "", os.ErrNotExist
	}
	cmd, err := clipboardCommandFor("linux", "copy me", lookup)
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Path != "/usr/bin/xclip" || strings.Join(cmd.Args[1:], " ") != "-selection clipboard" {
		t.Fatalf("unexpected clipboard command: path=%s args=%v", cmd.Path, cmd.Args)
	}
	data, err := io.ReadAll(cmd.Stdin)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "copy me" {
		t.Fatalf("unexpected clipboard stdin: %q", data)
	}
	if _, err := openURLCommandFor("darwin", "file:///tmp/secret"); err == nil {
		t.Fatal("expected non-http URL to be rejected")
	}
	openCmd, err := openURLCommandFor("darwin", remediationDocsURL)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(openCmd.Path) != "open" || openCmd.Args[len(openCmd.Args)-1] != remediationDocsURL {
		t.Fatalf("unexpected open command: path=%s args=%v", openCmd.Path, openCmd.Args)
	}
}
