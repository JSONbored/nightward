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
