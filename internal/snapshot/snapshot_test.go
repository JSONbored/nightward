package snapshot

import (
	"testing"
	"time"

	"github.com/shadowbook/nightward/internal/backupplan"
	"github.com/shadowbook/nightward/internal/inventory"
)

func TestBuildCreatesReadOnlySnapshotPlan(t *testing.T) {
	report := inventory.Report{
		GeneratedAt: time.Date(2026, 4, 30, 7, 0, 0, 0, time.UTC),
		Items: []inventory.Item{
			{Tool: "Codex", Path: "/home/me/.codex/config.toml", Classification: inventory.Portable},
			{Tool: "Codex", Path: "/home/me/.codex/auth.json", Classification: inventory.SecretAuth},
		},
	}

	plan := Build(report, "/backup")
	if plan.Summary.Total != 2 || plan.Summary.Include != 1 || plan.Summary.Excluded != 1 {
		t.Fatalf("unexpected summary: %#v", plan.Summary)
	}
}

func TestCompareReportsAddedRemovedAndChangedEntries(t *testing.T) {
	before := Plan{Entries: []Entry{
		{Source: "/a", Target: "/repo/a", Tool: "Codex", Classification: inventory.Portable, Action: backupplan.ActionInclude},
		{Source: "/b", Target: "/repo/b", Tool: "Claude", Classification: inventory.MachineLocal, Action: backupplan.ActionReview},
	}}
	after := Plan{Entries: []Entry{
		{Source: "/a", Target: "/repo/a", Tool: "Codex", Classification: inventory.MachineLocal, Action: backupplan.ActionReview},
		{Source: "/c", Target: "/repo/c", Tool: "Cursor", Classification: inventory.Portable, Action: backupplan.ActionInclude},
	}}

	diff := Compare("before.json", "after.json", before, after)
	if diff.Summary.Added != 1 || diff.Summary.Removed != 1 || diff.Summary.Changed != 1 {
		t.Fatalf("unexpected diff: %#v", diff.Summary)
	}
}
