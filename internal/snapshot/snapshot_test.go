package snapshot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jsonbored/nightward/internal/backupplan"
	"github.com/jsonbored/nightward/internal/inventory"
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

func TestLoadSnapshotPlan(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.json")
	original := Plan{
		GeneratedAt: time.Date(2026, 4, 30, 7, 0, 0, 0, time.UTC),
		TargetRoot:  "/backup",
		Entries: []Entry{
			{Source: "/home/me/.codex/config.toml", Target: "/backup/config/codex/config.toml", Tool: "Codex", Classification: inventory.Portable, Action: backupplan.ActionInclude},
		},
		Summary: Summary{Total: 1, Include: 1},
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TargetRoot != original.TargetRoot || len(loaded.Entries) != 1 || loaded.Entries[0].Source != original.Entries[0].Source {
		t.Fatalf("unexpected loaded snapshot: %#v", loaded)
	}
	if _, err := Load(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("expected missing snapshot load error")
	}
}
