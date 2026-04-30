package backupplan

import (
	"testing"
	"time"

	"github.com/shadowbook/nightward/internal/inventory"
)

func TestBuildClassifiesActions(t *testing.T) {
	report := inventory.Report{
		GeneratedAt: time.Date(2026, 4, 30, 2, 17, 0, 0, time.UTC),
		Items: []inventory.Item{
			{Tool: "Codex", Path: "/home/me/.codex/config.toml", Classification: inventory.Portable, Risk: inventory.RiskLow},
			{Tool: "Claude", Path: "/home/me/.claude.json", Classification: inventory.MachineLocal, Risk: inventory.RiskMedium},
			{Tool: "Codex", Path: "/home/me/.codex/auth.json", Classification: inventory.SecretAuth, Risk: inventory.RiskCritical},
			{Tool: "VS Code", Path: "/home/me/.vscode/extensions", Classification: inventory.AppOwned, Risk: inventory.RiskHigh},
			{Tool: "Codex", Path: "/home/me/.codex/cache", Classification: inventory.RuntimeCache, Risk: inventory.RiskInfo},
		},
	}

	plan := Build(report, "/repo")
	if plan.Summary.Included != 1 || plan.Summary.Review != 1 || plan.Summary.Excluded != 3 {
		t.Fatalf("unexpected summary: %#v", plan.Summary)
	}
	for _, entry := range plan.Entries {
		if entry.Classification == inventory.SecretAuth && entry.Action != ActionExclude {
			t.Fatalf("secret entry was not excluded: %#v", entry)
		}
		if entry.Classification == inventory.Portable && entry.Action != ActionInclude {
			t.Fatalf("portable entry was not included: %#v", entry)
		}
	}
}
