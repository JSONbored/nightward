package reporthtml

import (
	"strings"
	"testing"
	"time"

	"github.com/jsonbored/nightward/internal/inventory"
	"github.com/jsonbored/nightward/internal/schedule"
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
	for _, want := range []string{"finding-search", "severity-filter", "data-finding-card", "Filters run locally"} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected interactive review affordance %q in report:\n%s", want, html)
		}
	}
}

func TestRenderIndexEscapesReportHistory(t *testing.T) {
	html, err := RenderIndex([]schedule.ReportRecord{
		{
			Path:               "/tmp/<report>.json",
			ReportName:         "<report>.json",
			Findings:           2,
			HighestSeverity:    inventory.RiskHigh,
			FindingsBySeverity: map[inventory.RiskLevel]int{inventory.RiskHigh: 2},
			SizeBytes:          123,
			ModTime:            time.Date(2026, 5, 1, 1, 0, 0, 0, time.UTC),
		},
		{
			Path:       "/tmp/older.json",
			ReportName: "older.json",
			Findings:   1,
			SizeBytes:  100,
			ModTime:    time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "/tmp/<report>.json") || !strings.Contains(html, "Nightward Report History") {
		t.Fatalf("expected escaped report history index:\n%s", html)
	}
	for _, want := range []string{"Highest", "high", "latest", "-1 vs newer", "&lt;report&gt;.json"} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected richer report history field %q:\n%s", want, html)
		}
	}
	empty, err := RenderIndex(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(empty, "No JSON reports found.") {
		t.Fatalf("expected empty report history copy:\n%s", empty)
	}
}

func TestRenderIncludesFixFilters(t *testing.T) {
	html, err := Render(inventory.Report{
		Home: "/tmp/home",
		Summary: inventory.Summary{
			TotalFindings:      2,
			FindingsBySeverity: map[inventory.RiskLevel]int{inventory.RiskHigh: 2},
		},
		Findings: []inventory.Finding{
			{
				ID:           "fixable",
				Tool:         "Codex",
				Path:         "/tmp/config.toml",
				Severity:     inventory.RiskHigh,
				Rule:         "mcp_secret_header",
				Message:      "header secret",
				FixAvailable: true,
				FixKind:      inventory.FixExternalizeSecret,
			},
			{
				ID:       "manual",
				Tool:     "Codex",
				Path:     "/tmp/config.toml",
				Severity: inventory.RiskHigh,
				Rule:     "mcp_server_review",
				Message:  "review",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`data-fix="externalize-secret"`, `data-fix="manual review"`, `<option value="externalize-secret">externalize-secret</option>`} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected fix filter marker %q:\n%s", want, html)
		}
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
