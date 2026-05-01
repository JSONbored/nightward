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

func TestRenderHandlesEmptyAndLimitsInventorySample(t *testing.T) {
	items := make([]inventory.Item, 205)
	for i := range items {
		items[i] = inventory.Item{Tool: "tool", Path: "/tmp/item", Classification: inventory.Portable, Risk: inventory.RiskLow}
	}
	html, err := Render(inventory.Report{
		Home:      "/tmp/home",
		Workspace: "/tmp/workspace",
		Summary: inventory.Summary{
			TotalItems:         len(items),
			FindingsBySeverity: map[inventory.RiskLevel]int{inventory.RiskMedium: 2},
		},
		Items: items,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "Generated unknown") || !strings.Contains(html, "/tmp/workspace") || !strings.Contains(html, "No findings.") {
		t.Fatalf("expected empty report metadata:\n%s", html)
	}
	if got := strings.Count(html, "<tr><td>tool</td>"); got != 200 {
		t.Fatalf("expected inventory sample to be limited to 200 rows, got %d", got)
	}
	if len(limitItems(items, 100)) != 100 || len(limitItems(items[:1], 100)) != 1 {
		t.Fatal("limitItems returned unexpected lengths")
	}
}
