package reportdiff

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/jsonbored/nightward/internal/inventory"
)

const SchemaVersion = 1

type Diff struct {
	SchemaVersion int             `json:"schema_version"`
	GeneratedAt   time.Time       `json:"generated_at"`
	From          string          `json:"from"`
	To            string          `json:"to"`
	Summary       Summary         `json:"summary"`
	Added         []FindingRecord `json:"added_findings,omitempty"`
	Removed       []FindingRecord `json:"removed_findings,omitempty"`
	Changed       []Change        `json:"changed_findings,omitempty"`
}

type Summary struct {
	Added     int `json:"added"`
	Removed   int `json:"removed"`
	Changed   int `json:"changed"`
	Unchanged int `json:"unchanged"`
}

type FindingRecord struct {
	Key     string            `json:"key"`
	Finding inventory.Finding `json:"finding"`
}

type Change struct {
	Key    string            `json:"key"`
	Fields []string          `json:"fields"`
	Before inventory.Finding `json:"before"`
	After  inventory.Finding `json:"after"`
}

func Compare(fromName, toName string, from, to inventory.Report) Diff {
	diff := Diff{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Now().UTC(),
		From:          fromName,
		To:            toName,
	}
	fromIndex := indexFindings(from.Findings)
	toIndex := indexFindings(to.Findings)

	for key, after := range toIndex {
		before, ok := fromIndex[key]
		if !ok {
			diff.Added = append(diff.Added, FindingRecord{Key: key, Finding: after})
			continue
		}
		if fields := changedFields(before, after); len(fields) > 0 {
			diff.Changed = append(diff.Changed, Change{Key: key, Fields: fields, Before: before, After: after})
		} else {
			diff.Summary.Unchanged++
		}
	}
	for key, before := range fromIndex {
		if _, ok := toIndex[key]; !ok {
			diff.Removed = append(diff.Removed, FindingRecord{Key: key, Finding: before})
		}
	}

	sort.Slice(diff.Added, func(i, j int) bool { return lessFinding(diff.Added[i].Finding, diff.Added[j].Finding) })
	sort.Slice(diff.Removed, func(i, j int) bool { return lessFinding(diff.Removed[i].Finding, diff.Removed[j].Finding) })
	sort.Slice(diff.Changed, func(i, j int) bool { return lessFinding(diff.Changed[i].After, diff.Changed[j].After) })
	diff.Summary.Added = len(diff.Added)
	diff.Summary.Removed = len(diff.Removed)
	diff.Summary.Changed = len(diff.Changed)
	return diff
}

func IsEmpty(diff Diff) bool {
	return diff.Summary.Added == 0 && diff.Summary.Removed == 0 && diff.Summary.Changed == 0
}

func indexFindings(findings []inventory.Finding) map[string]inventory.Finding {
	out := map[string]inventory.Finding{}
	for _, finding := range findings {
		key := findingKey(finding)
		if key == "" {
			continue
		}
		out[key] = finding
	}
	return out
}

func findingKey(finding inventory.Finding) string {
	if strings.TrimSpace(finding.ID) != "" {
		return finding.ID
	}
	parts := []string{
		finding.Tool,
		finding.Path,
		finding.Server,
		string(finding.Severity),
		finding.Rule,
		finding.Message,
		finding.Evidence,
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return "generated-" + hex.EncodeToString(sum[:])[:12]
}

func changedFields(before, after inventory.Finding) []string {
	var fields []string
	check := func(name string, left, right any) {
		if !reflect.DeepEqual(left, right) {
			fields = append(fields, name)
		}
	}
	check("tool", before.Tool, after.Tool)
	check("path", before.Path, after.Path)
	check("server", before.Server, after.Server)
	check("severity", before.Severity, after.Severity)
	check("rule", before.Rule, after.Rule)
	check("message", before.Message, after.Message)
	check("evidence", before.Evidence, after.Evidence)
	check("recommended_action", before.Recommendation, after.Recommendation)
	check("impact", before.Impact, after.Impact)
	check("fix_available", before.FixAvailable, after.FixAvailable)
	check("fix_kind", before.FixKind, after.FixKind)
	check("fix_steps", before.FixSteps, after.FixSteps)
	sort.Strings(fields)
	return fields
}

func lessFinding(left, right inventory.Finding) bool {
	if rank(left.Severity) != rank(right.Severity) {
		return rank(left.Severity) > rank(right.Severity)
	}
	for _, pair := range [][2]string{
		{left.Rule, right.Rule},
		{left.Tool, right.Tool},
		{left.Path, right.Path},
		{findingKey(left), findingKey(right)},
	} {
		if pair[0] != pair[1] {
			return pair[0] < pair[1]
		}
	}
	return fmt.Sprint(left) < fmt.Sprint(right)
}

func rank(risk inventory.RiskLevel) int {
	switch risk {
	case inventory.RiskCritical:
		return 5
	case inventory.RiskHigh:
		return 4
	case inventory.RiskMedium:
		return 3
	case inventory.RiskLow:
		return 2
	default:
		return 1
	}
}
