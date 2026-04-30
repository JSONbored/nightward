package analysis

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jsonbored/nightward/internal/inventory"
)

func TestAnalysisBuildsOfflineSignalsAndRedacts(t *testing.T) {
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
				Recommendation: "Keep secret values outside dotfiles.",
				Why:            "Secrets should stay local.",
				Confidence:     "high",
			},
		},
		Items: []inventory.Item{
			{
				ID:             "secret-item",
				Tool:           "Secrets",
				Path:           "/tmp/workspace/.env",
				Classification: inventory.SecretAuth,
				Risk:           inventory.RiskCritical,
			},
		},
	}

	out := Run(report, Options{Mode: "workspace", Workspace: "/tmp/workspace"})
	if out.Summary.TotalSignals != 2 {
		t.Fatalf("expected two analysis signals, got %#v", out.Summary)
	}
	if out.Summary.SignalsByCategory[CategorySecrets] != 2 {
		t.Fatalf("expected secret category signals, got %#v", out.Summary.SignalsByCategory)
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "super-secret-value") {
		t.Fatalf("analysis leaked a secret: %s", data)
	}
}

func TestProviderDoctorHonorsOnlineGate(t *testing.T) {
	statuses := ProviderStatuses([]string{"socket"}, false)
	var found bool
	for _, status := range statuses {
		if status.Name == "socket" {
			found = true
			if status.Status != "blocked" {
				t.Fatalf("expected socket to be blocked without --online, got %#v", status)
			}
		}
	}
	if !found {
		t.Fatal("missing socket provider")
	}
}

func TestProviderDoctorFindsFakeLocalProvider(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gitleaks")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	statuses := ProviderStatuses([]string{"gitleaks"}, false)
	for _, status := range statuses {
		if status.Name == "gitleaks" {
			if !status.Available || status.Status != "ready" {
				t.Fatalf("expected fake gitleaks to be ready, got %#v", status)
			}
			return
		}
	}
	t.Fatal("missing gitleaks provider")
}

func TestExplainRelativePathAndHumanProviderSummary(t *testing.T) {
	report := Run(inventory.Report{
		Findings: []inventory.Finding{
			{
				ID:             "mcp_shell_command-111111111111",
				Tool:           "Codex",
				Path:           "/tmp/work/config.toml",
				Rule:           "mcp_shell_command",
				Severity:       inventory.RiskMedium,
				Message:        "Shell wrapper",
				Recommendation: "Use a direct command.",
			},
		},
	}, Options{})

	if len(report.Signals) != 1 {
		t.Fatalf("expected one signal: %#v", report.Signals)
	}
	fullID := report.Signals[0].ID
	if signal, ok := Explain(report, fullID); !ok || signal.ID != fullID {
		t.Fatalf("full signal explain failed: %#v ok=%t", signal, ok)
	}
	if signal, ok := Explain(report, fullID[:8]); !ok || signal.ID != fullID {
		t.Fatalf("prefix signal explain failed: %#v ok=%t", signal, ok)
	}
	if _, ok := Explain(report, "missing"); ok {
		t.Fatal("unexpected match for missing signal")
	}
	if got := RelativeWorkspacePath("/tmp/work", "/tmp/work/config.toml"); got != "config.toml" {
		t.Fatalf("unexpected relative path: %s", got)
	}
	if got := RelativeWorkspacePath("/tmp/work", "/tmp/else/config.toml"); got != "/tmp/else/config.toml" {
		t.Fatalf("unexpected path outside workspace: %s", got)
	}

	enabled := HumanProviderSummary(ProviderStatus{Provider: Provider{Name: "gitleaks"}, Enabled: true, Available: true, Status: "ready"})
	disabled := HumanProviderSummary(ProviderStatus{Provider: Provider{Name: "socket"}, Enabled: false, Available: false, Status: "missing"})
	if !strings.Contains(enabled, "enabled ready (available)") || !strings.Contains(disabled, "disabled missing") {
		t.Fatalf("unexpected provider summaries: %q / %q", enabled, disabled)
	}
}

func TestAnalysisItemCategoriesAndRiskRanks(t *testing.T) {
	report := inventory.Report{Items: []inventory.Item{
		{ID: "machine", Classification: inventory.MachineLocal, Path: "/tmp/socket", Risk: inventory.RiskMedium},
		{ID: "app", Classification: inventory.AppOwned, Path: "/tmp/app.db", Risk: inventory.RiskLow},
		{ID: "portable", Classification: inventory.Portable, Path: "/tmp/config", Risk: inventory.RiskInfo},
	}}
	out := Run(report, Options{})
	if out.Summary.TotalSignals != 2 {
		t.Fatalf("expected machine-local and app-owned signals only: %#v", out.Summary)
	}
	if out.Summary.SignalsByCategory[CategoryLocality] != 1 || out.Summary.SignalsByCategory[CategoryAppState] != 1 {
		t.Fatalf("unexpected item categories: %#v", out.Summary.SignalsByCategory)
	}

	for rule, want := range map[string]SignalCategory{
		"mcp_unpinned_package":  CategorySupplyChain,
		"mcp_secret_header":     CategorySecrets,
		"mcp_broad_filesystem":  CategoryFilesystem,
		"mcp_local_endpoint":    CategoryNetwork,
		"mcp_unknown_command":   CategoryExecution,
		"mcp_server_review":     CategoryExecution,
		"nightward/custom_rule": CategoryUnknown,
	} {
		if got := categoryForRule(rule); got != want {
			t.Fatalf("categoryForRule(%q)=%q, want %q", rule, got, want)
		}
	}
	if riskRank(inventory.RiskCritical) <= riskRank(inventory.RiskHigh) || riskRank(inventory.RiskInfo) != 1 {
		t.Fatal("unexpected risk ranking")
	}
}
