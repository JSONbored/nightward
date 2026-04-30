package policy

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/shadowbook/nightward/internal/inventory"
)

func TestCheckUsesStrictThreshold(t *testing.T) {
	report := inventory.Report{
		GeneratedAt: time.Date(2026, 4, 30, 7, 0, 0, 0, time.UTC),
		Findings: []inventory.Finding{
			{ID: "low", Severity: inventory.RiskLow, Rule: "mcp_server_review"},
			{ID: "medium", Severity: inventory.RiskMedium, Rule: "mcp_broad_filesystem"},
			{ID: "high", Severity: inventory.RiskHigh, Rule: "mcp_unpinned_package"},
		},
	}

	standard := Check(report, false)
	if standard.Passed || len(standard.Violations) != 1 || standard.Threshold != inventory.RiskHigh {
		t.Fatalf("unexpected standard policy report: %#v", standard)
	}

	strict := Check(report, true)
	if strict.Passed || len(strict.Violations) != 2 || strict.Threshold != inventory.RiskMedium {
		t.Fatalf("unexpected strict policy report: %#v", strict)
	}
}

func TestSARIFRedactsAndIncludesFixMetadata(t *testing.T) {
	report := inventory.Report{Findings: []inventory.Finding{
		{
			ID:             "mcp_secret_env-111111111111",
			Tool:           "Codex",
			Path:           "/tmp/config.toml",
			Severity:       inventory.RiskCritical,
			Rule:           "mcp_secret_env",
			Message:        "MCP server stores a sensitive environment key.",
			Evidence:       "env_key=API_TOKEN",
			Recommendation: "Keep secret values outside dotfiles.",
			FixAvailable:   true,
			FixKind:        inventory.FixExternalizeSecret,
			Confidence:     "high",
			Risk:           inventory.RiskHigh,
			RequiresReview: true,
			FixSummary:     "Move API_TOKEN out of this config.",
			FixSteps:       []string{"Remove the inline value for API_TOKEN."},
		},
	}}

	sarif := BuildSARIF(report)
	data, err := json.Marshal(sarif)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"2.1.0", "mcp_secret_env", "externalize-secret", "API_TOKEN"} {
		if !strings.Contains(text, want) {
			t.Fatalf("SARIF missing %q: %s", want, text)
		}
	}
	if strings.Contains(text, "super-secret-value") {
		t.Fatal("SARIF leaked a secret value")
	}
}
