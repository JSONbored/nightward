package reportdiff

import (
	"testing"

	"github.com/jsonbored/nightward/internal/inventory"
)

func TestCompareFindings(t *testing.T) {
	before := inventory.Report{Findings: []inventory.Finding{
		{ID: "same", Tool: "codex", Rule: "mcp_secret_env", Path: "/tmp/a", Severity: inventory.RiskHigh, Message: "old"},
		{ID: "removed", Tool: "cursor", Rule: "mcp_local_endpoint", Path: "/tmp/b", Severity: inventory.RiskMedium, Message: "gone"},
		{ID: "changed", Tool: "claude", Rule: "mcp_unpinned_package", Path: "/tmp/c", Severity: inventory.RiskMedium, Message: "old"},
	}}
	after := inventory.Report{Findings: []inventory.Finding{
		{ID: "same", Tool: "codex", Rule: "mcp_secret_env", Path: "/tmp/a", Severity: inventory.RiskHigh, Message: "old"},
		{ID: "changed", Tool: "claude", Rule: "mcp_unpinned_package", Path: "/tmp/c", Severity: inventory.RiskHigh, Message: "new"},
		{ID: "added", Tool: "goose", Rule: "mcp_broad_filesystem", Path: "/tmp/d", Severity: inventory.RiskCritical, Message: "new"},
	}}

	diff := Compare("before.json", "after.json", before, after)
	if diff.SchemaVersion != 1 {
		t.Fatalf("expected schema version 1, got %d", diff.SchemaVersion)
	}
	if diff.Summary.Added != 1 || diff.Summary.Removed != 1 || diff.Summary.Changed != 1 || diff.Summary.Unchanged != 1 {
		t.Fatalf("unexpected summary: %#v", diff.Summary)
	}
	if diff.Added[0].Finding.ID != "added" || diff.Removed[0].Finding.ID != "removed" || diff.Changed[0].Key != "changed" {
		t.Fatalf("unexpected diff: %#v", diff)
	}
	if len(diff.Changed[0].Fields) == 0 {
		t.Fatalf("expected changed fields")
	}
}

func TestCompareUsesGeneratedKeyForLegacyFindings(t *testing.T) {
	finding := inventory.Finding{Tool: "codex", Rule: "mcp_secret_env", Path: "/tmp/a", Severity: inventory.RiskHigh, Message: "same"}
	diff := Compare("before.json", "after.json", inventory.Report{Findings: []inventory.Finding{finding}}, inventory.Report{Findings: []inventory.Finding{finding}})
	if !IsEmpty(diff) || diff.Summary.Unchanged != 1 {
		t.Fatalf("expected legacy finding to compare as unchanged: %#v", diff)
	}
}

func TestCompareSortsAddedFindingsByRisk(t *testing.T) {
	after := inventory.Report{Findings: []inventory.Finding{
		{ID: "low", Tool: "codex", Rule: "mcp_server_review", Path: "/tmp/low", Severity: inventory.RiskLow, Message: "low"},
		{ID: "critical", Tool: "codex", Rule: "mcp_secret_env", Path: "/tmp/critical", Severity: inventory.RiskCritical, Message: "critical"},
		{ID: "high", Tool: "codex", Rule: "mcp_secret_header", Path: "/tmp/high", Severity: inventory.RiskHigh, Message: "high"},
	}}
	diff := Compare("before.json", "after.json", inventory.Report{}, after)
	got := []string{diff.Added[0].Finding.ID, diff.Added[1].Finding.ID, diff.Added[2].Finding.ID}
	want := []string{"critical", "high", "low"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected sort order got=%v want=%v", got, want)
		}
	}
}
