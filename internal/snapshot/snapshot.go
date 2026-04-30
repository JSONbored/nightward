package snapshot

import (
	"encoding/json"
	"os"
	"sort"
	"time"

	"github.com/jsonbored/nightward/internal/backupplan"
	"github.com/jsonbored/nightward/internal/inventory"
)

type Plan struct {
	GeneratedAt time.Time `json:"generated_at"`
	TargetRoot  string    `json:"target_root"`
	Entries     []Entry   `json:"entries"`
	Summary     Summary   `json:"summary"`
}

type Entry struct {
	Source         string                   `json:"source"`
	Target         string                   `json:"target"`
	Tool           string                   `json:"tool"`
	Classification inventory.Classification `json:"classification"`
	Action         backupplan.Action        `json:"action"`
	Reason         string                   `json:"reason"`
}

type Summary struct {
	Total    int `json:"total"`
	Include  int `json:"include"`
	Review   int `json:"review"`
	Excluded int `json:"excluded"`
}

type Diff struct {
	GeneratedAt time.Time   `json:"generated_at"`
	From        string      `json:"from"`
	To          string      `json:"to"`
	Summary     DiffSummary `json:"summary"`
	Added       []Entry     `json:"added,omitempty"`
	Removed     []Entry     `json:"removed,omitempty"`
	Changed     []Change    `json:"changed,omitempty"`
}

type DiffSummary struct {
	Added   int `json:"added"`
	Removed int `json:"removed"`
	Changed int `json:"changed"`
}

type Change struct {
	Source string `json:"source"`
	Before Entry  `json:"before"`
	After  Entry  `json:"after"`
}

func Build(report inventory.Report, targetRoot string) Plan {
	backup := backupplan.Build(report, targetRoot)
	plan := Plan{GeneratedAt: report.GeneratedAt, TargetRoot: targetRoot}
	for _, backupEntry := range backup.Entries {
		entry := Entry{
			Source:         backupEntry.Source,
			Target:         backupEntry.Target,
			Tool:           backupEntry.Tool,
			Classification: backupEntry.Classification,
			Action:         backupEntry.Action,
			Reason:         backupEntry.Reason,
		}
		plan.Entries = append(plan.Entries, entry)
		plan.Summary.Total++
		switch entry.Action {
		case backupplan.ActionInclude:
			plan.Summary.Include++
		case backupplan.ActionReview:
			plan.Summary.Review++
		default:
			plan.Summary.Excluded++
		}
	}
	return plan
}

func Load(path string) (Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Plan{}, err
	}
	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func Compare(fromPath, toPath string, from, to Plan) Diff {
	diff := Diff{
		GeneratedAt: time.Now().UTC(),
		From:        fromPath,
		To:          toPath,
	}
	fromEntries := indexEntries(from)
	toEntries := indexEntries(to)
	for source, after := range toEntries {
		before, ok := fromEntries[source]
		if !ok {
			diff.Added = append(diff.Added, after)
			continue
		}
		if before.Target != after.Target || before.Action != after.Action || before.Classification != after.Classification || before.Tool != after.Tool {
			diff.Changed = append(diff.Changed, Change{Source: source, Before: before, After: after})
		}
	}
	for source, before := range fromEntries {
		if _, ok := toEntries[source]; !ok {
			diff.Removed = append(diff.Removed, before)
		}
	}
	sort.Slice(diff.Added, func(i, j int) bool { return diff.Added[i].Source < diff.Added[j].Source })
	sort.Slice(diff.Removed, func(i, j int) bool { return diff.Removed[i].Source < diff.Removed[j].Source })
	sort.Slice(diff.Changed, func(i, j int) bool { return diff.Changed[i].Source < diff.Changed[j].Source })
	diff.Summary.Added = len(diff.Added)
	diff.Summary.Removed = len(diff.Removed)
	diff.Summary.Changed = len(diff.Changed)
	return diff
}

func indexEntries(plan Plan) map[string]Entry {
	out := map[string]Entry{}
	for _, entry := range plan.Entries {
		out[entry.Source] = entry
	}
	return out
}
