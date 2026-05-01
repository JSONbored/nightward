package reporthtml

import (
	"strings"
	"testing"
	"time"

	"github.com/jsonbored/nightward/internal/inventory"
)

func TestRenderEscapesHTML(t *testing.T) {
	html, err := Render(inventory.Report{
		GeneratedAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		Hostname:    "host<script>",
		Home:        "/tmp/home",
		Summary: inventory.Summary{
			TotalItems:         1,
			TotalFindings:      1,
			FindingsBySeverity: map[inventory.RiskLevel]int{inventory.RiskHigh: 1},
		},
		Items: []inventory.Item{{Tool: "codex", Path: "/tmp/<secret>", Classification: inventory.Portable, Risk: inventory.RiskLow}},
		Findings: []inventory.Finding{{
			Tool:           "codex",
			Path:           "/tmp/config",
			Rule:           "mcp_secret_header",
			Severity:       inventory.RiskHigh,
			Message:        "<bad>",
			Evidence:       "header=authorization",
			Recommendation: "externalize",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "<bad>") || strings.Contains(html, "host<script>") || strings.Contains(html, "/tmp/<secret>") {
		t.Fatalf("expected escaped HTML, got: %s", html)
	}
	if !strings.Contains(html, "mcp_secret_header") {
		t.Fatalf("expected finding rule in report: %s", html)
	}
}
