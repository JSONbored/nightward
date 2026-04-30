package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/shadowbook/nightward/internal/inventory"
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
