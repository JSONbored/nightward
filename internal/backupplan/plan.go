package backupplan

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jsonbored/nightward/internal/inventory"
)

type Action string

const (
	ActionInclude Action = "include"
	ActionReview  Action = "review"
	ActionExclude Action = "exclude"
)

type Entry struct {
	Source         string                   `json:"source"`
	Target         string                   `json:"target"`
	Tool           string                   `json:"tool"`
	Classification inventory.Classification `json:"classification"`
	Risk           inventory.RiskLevel      `json:"risk"`
	Action         Action                   `json:"action"`
	Reason         string                   `json:"reason"`
	Recommendation string                   `json:"recommended_action"`
}

type Plan struct {
	GeneratedAt time.Time `json:"generated_at"`
	TargetRoot  string    `json:"target_root"`
	Entries     []Entry   `json:"entries"`
	Summary     Summary   `json:"summary"`
}

type Summary struct {
	Included int `json:"included"`
	Review   int `json:"review"`
	Excluded int `json:"excluded"`
}

func Build(report inventory.Report, targetRoot string) Plan {
	plan := Plan{
		GeneratedAt: report.GeneratedAt,
		TargetRoot:  targetRoot,
	}
	for _, item := range report.Items {
		action, reason := actionFor(item)
		entry := Entry{
			Source:         item.Path,
			Target:         filepath.Join(targetRoot, "config", normalize(item.Tool), targetName(item)),
			Tool:           item.Tool,
			Classification: item.Classification,
			Risk:           item.Risk,
			Action:         action,
			Reason:         reason,
			Recommendation: item.Recommendation,
		}
		plan.Entries = append(plan.Entries, entry)
		switch action {
		case ActionInclude:
			plan.Summary.Included++
		case ActionReview:
			plan.Summary.Review++
		default:
			plan.Summary.Excluded++
		}
	}

	sort.Slice(plan.Entries, func(i, j int) bool {
		if plan.Entries[i].Action == plan.Entries[j].Action {
			return plan.Entries[i].Source < plan.Entries[j].Source
		}
		return actionRank(plan.Entries[i].Action) < actionRank(plan.Entries[j].Action)
	})

	return plan
}

func actionFor(item inventory.Item) (Action, string) {
	switch item.Classification {
	case inventory.Portable:
		return ActionInclude, "Portable config can be copied after review."
	case inventory.MachineLocal, inventory.Unknown:
		return ActionReview, "Machine-local or unknown state needs a human portability decision."
	case inventory.SecretAuth:
		return ActionExclude, "Secret/auth material is excluded by default."
	case inventory.RuntimeCache:
		return ActionExclude, "Runtime cache is generated state and should not be backed up."
	case inventory.AppOwned:
		return ActionExclude, "App-owned databases and binaries should use app-supported export paths."
	default:
		return ActionReview, "Unknown classification needs review."
	}
}

func normalize(value string) string {
	value = strings.ToLower(value)
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "/", "-")
	return value
}

func targetName(item inventory.Item) string {
	cleaned := strings.TrimPrefix(item.Path, filepath.VolumeName(item.Path))
	cleaned = strings.TrimPrefix(cleaned, string(filepath.Separator))
	cleaned = strings.ReplaceAll(cleaned, string(filepath.Separator), "__")
	if cleaned == "" {
		return normalize(item.Tool)
	}
	return cleaned
}

func actionRank(action Action) int {
	switch action {
	case ActionInclude:
		return 0
	case ActionReview:
		return 1
	default:
		return 2
	}
}
