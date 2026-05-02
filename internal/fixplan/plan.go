package fixplan

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jsonbored/nightward/internal/inventory"
)

type Status string

const (
	StatusSafe    Status = "safe"
	StatusReview  Status = "review"
	StatusBlocked Status = "blocked"
)

type Selector struct {
	FindingID string
	Rule      string
	All       bool
}

type Plan struct {
	SchemaVersion int       `json:"schema_version"`
	GeneratedAt   time.Time `json:"generated_at"`
	Summary       Summary   `json:"summary"`
	Groups        []Group   `json:"groups,omitempty"`
	Fixes         []Fix     `json:"fixes"`
}

type Summary struct {
	Total   int `json:"total"`
	Safe    int `json:"safe"`
	Review  int `json:"review"`
	Blocked int `json:"blocked"`
}

type Fix struct {
	FindingID      string              `json:"finding_id"`
	Tool           string              `json:"tool"`
	Path           string              `json:"path"`
	Server         string              `json:"server,omitempty"`
	Severity       inventory.RiskLevel `json:"severity"`
	Rule           string              `json:"rule"`
	Package        string              `json:"package,omitempty"`
	FixAvailable   bool                `json:"fix_available"`
	FixKind        inventory.FixKind   `json:"fix_kind,omitempty"`
	Confidence     string              `json:"confidence,omitempty"`
	Risk           inventory.RiskLevel `json:"risk,omitempty"`
	RequiresReview bool                `json:"requires_review"`
	Status         Status              `json:"status"`
	Summary        string              `json:"summary"`
	Steps          []string            `json:"steps,omitempty"`
	Evidence       string              `json:"evidence,omitempty"`
	Impact         string              `json:"impact,omitempty"`
	Why            string              `json:"why_this_matters,omitempty"`
}

type Group struct {
	Key        string              `json:"key"`
	Label      string              `json:"label"`
	Rule       string              `json:"rule"`
	FixKind    inventory.FixKind   `json:"fix_kind,omitempty"`
	Package    string              `json:"package,omitempty"`
	Severity   inventory.RiskLevel `json:"severity"`
	Status     Status              `json:"status"`
	Count      int                 `json:"count"`
	FindingIDs []string            `json:"finding_ids"`
	Paths      []string            `json:"paths,omitempty"`
	Servers    []string            `json:"servers,omitempty"`
	Summary    string              `json:"summary"`
	Steps      []string            `json:"steps,omitempty"`
}

func Build(report inventory.Report, selector Selector) Plan {
	plan := Plan{SchemaVersion: 1, GeneratedAt: report.GeneratedAt}
	for _, finding := range report.Findings {
		if !matches(finding, selector) {
			continue
		}
		fix := fromFinding(finding)
		plan.Fixes = append(plan.Fixes, fix)
		plan.Summary.Total++
		switch fix.Status {
		case StatusSafe:
			plan.Summary.Safe++
		case StatusReview:
			plan.Summary.Review++
		default:
			plan.Summary.Blocked++
		}
	}
	plan.Groups = groupFixes(plan.Fixes)
	sort.Slice(plan.Fixes, func(i, j int) bool {
		if plan.Fixes[i].Status == plan.Fixes[j].Status {
			if plan.Fixes[i].Severity == plan.Fixes[j].Severity {
				return plan.Fixes[i].FindingID < plan.Fixes[j].FindingID
			}
			return riskRank(plan.Fixes[i].Severity) > riskRank(plan.Fixes[j].Severity)
		}
		return statusRank(plan.Fixes[i].Status) < statusRank(plan.Fixes[j].Status)
	})
	return plan
}

func Find(report inventory.Report, idOrPrefix string) (inventory.Finding, bool) {
	for _, finding := range report.Findings {
		if finding.ID == idOrPrefix {
			return finding, true
		}
	}
	var matched []inventory.Finding
	for _, finding := range report.Findings {
		if strings.HasPrefix(finding.ID, idOrPrefix) {
			matched = append(matched, finding)
		}
	}
	if len(matched) == 1 {
		return matched[0], true
	}
	return inventory.Finding{}, false
}

func Markdown(plan Plan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Nightward Fix Plan\n\n")
	fmt.Fprintf(&b, "Generated: %s\n\n", plan.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "Total: %d  Safe: %d  Review: %d  Blocked: %d\n\n", plan.Summary.Total, plan.Summary.Safe, plan.Summary.Review, plan.Summary.Blocked)
	if len(plan.Fixes) == 0 {
		b.WriteString("No fixes matched the selected findings.\n")
		return b.String()
	}
	if len(plan.Groups) > 0 {
		b.WriteString("## Grouped Review\n\n")
		for _, group := range plan.Groups {
			fmt.Fprintf(&b, "- `%s` (%d finding", group.Label, group.Count)
			if group.Count != 1 {
				b.WriteString("s")
			}
			fmt.Fprintf(&b, "): %s\n", group.Summary)
		}
		b.WriteString("\n")
	}
	for _, fix := range plan.Fixes {
		fmt.Fprintf(&b, "## %s\n\n", fix.FindingID)
		fmt.Fprintf(&b, "- Tool: `%s`\n", fix.Tool)
		fmt.Fprintf(&b, "- Path: `%s`\n", fix.Path)
		if fix.Server != "" {
			fmt.Fprintf(&b, "- Server: `%s`\n", fix.Server)
		}
		fmt.Fprintf(&b, "- Rule: `%s`\n", fix.Rule)
		if fix.Package != "" {
			fmt.Fprintf(&b, "- Package: `%s`\n", fix.Package)
		}
		fmt.Fprintf(&b, "- Severity: `%s`\n", fix.Severity)
		fmt.Fprintf(&b, "- Status: `%s`\n", fix.Status)
		if fix.FixKind != "" {
			fmt.Fprintf(&b, "- Fix kind: `%s`\n", fix.FixKind)
		}
		if fix.Risk != "" {
			fmt.Fprintf(&b, "- Fix risk: `%s`\n", fix.Risk)
		}
		if fix.Confidence != "" {
			fmt.Fprintf(&b, "- Confidence: `%s`\n", fix.Confidence)
		}
		fmt.Fprintf(&b, "- Requires review: `%t`\n", fix.RequiresReview)
		if fix.Evidence != "" {
			fmt.Fprintf(&b, "- Evidence: `%s`\n", fix.Evidence)
		}
		if fix.Summary != "" {
			fmt.Fprintf(&b, "\n%s\n", fix.Summary)
		}
		if len(fix.Steps) > 0 {
			b.WriteString("\nSteps:\n")
			for i, step := range fix.Steps {
				fmt.Fprintf(&b, "%d. %s\n", i+1, step)
			}
		}
		if fix.Why != "" {
			fmt.Fprintf(&b, "\nWhy this matters: %s\n", fix.Why)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func matches(finding inventory.Finding, selector Selector) bool {
	if selector.FindingID != "" {
		return finding.ID == selector.FindingID || strings.HasPrefix(finding.ID, selector.FindingID)
	}
	if selector.Rule != "" {
		return finding.Rule == selector.Rule
	}
	return selector.All || selector.FindingID == "" && selector.Rule == ""
}

func fromFinding(finding inventory.Finding) Fix {
	status := StatusBlocked
	if finding.FixAvailable {
		status = StatusSafe
		if finding.RequiresReview || riskRank(finding.Risk) >= riskRank(inventory.RiskMedium) {
			status = StatusReview
		}
	}
	return Fix{
		FindingID:      finding.ID,
		Tool:           finding.Tool,
		Path:           finding.Path,
		Server:         finding.Server,
		Severity:       finding.Severity,
		Rule:           finding.Rule,
		Package:        patchPackage(finding),
		FixAvailable:   finding.FixAvailable,
		FixKind:        finding.FixKind,
		Confidence:     finding.Confidence,
		Risk:           finding.Risk,
		RequiresReview: finding.RequiresReview,
		Status:         status,
		Summary:        finding.FixSummary,
		Steps:          finding.FixSteps,
		Evidence:       finding.Evidence,
		Impact:         finding.Impact,
		Why:            finding.Why,
	}
}

func patchPackage(finding inventory.Finding) string {
	if finding.PatchHint == nil {
		return ""
	}
	return finding.PatchHint.Package
}

func groupFixes(fixes []Fix) []Group {
	byKey := map[string]*Group{}
	for _, fix := range fixes {
		key := fixGroupKey(fix)
		if key == "" {
			continue
		}
		group, ok := byKey[key]
		if !ok {
			group = &Group{
				Key:      key,
				Label:    fixGroupLabel(fix),
				Rule:     fix.Rule,
				FixKind:  fix.FixKind,
				Package:  fix.Package,
				Severity: fix.Severity,
				Status:   fix.Status,
				Summary:  fixGroupSummary(fix),
				Steps:    fixGroupSteps(fix),
			}
			byKey[key] = group
		}
		group.Count++
		group.FindingIDs = appendUnique(group.FindingIDs, fix.FindingID)
		group.Paths = appendUnique(group.Paths, fix.Path)
		group.Servers = appendUnique(group.Servers, fix.Server)
		if riskRank(fix.Severity) > riskRank(group.Severity) {
			group.Severity = fix.Severity
		}
		if statusRank(fix.Status) > statusRank(group.Status) {
			group.Status = fix.Status
		}
	}
	groups := make([]Group, 0, len(byKey))
	for _, group := range byKey {
		sort.Strings(group.FindingIDs)
		sort.Strings(group.Paths)
		sort.Strings(group.Servers)
		groups = append(groups, *group)
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Count == groups[j].Count {
			return groups[i].Key < groups[j].Key
		}
		return groups[i].Count > groups[j].Count
	})
	return groups
}

func fixGroupKey(fix Fix) string {
	switch {
	case fix.Rule == "mcp_unpinned_package" && fix.Package != "":
		return "package:" + fix.Package
	case fix.Rule == "mcp_local_endpoint":
		return "local-endpoint:" + fix.Tool
	case fix.Rule == "mcp_broad_filesystem":
		return "filesystem-scope:" + fix.Tool
	default:
		return ""
	}
}

func fixGroupLabel(fix Fix) string {
	switch {
	case fix.Rule == "mcp_unpinned_package" && fix.Package != "":
		return "Pin " + fix.Package
	case fix.Rule == "mcp_local_endpoint":
		return fix.Tool + " local endpoints"
	case fix.Rule == "mcp_broad_filesystem":
		return fix.Tool + " filesystem scope"
	default:
		return fix.Rule
	}
}

func fixGroupSummary(fix Fix) string {
	switch {
	case fix.Rule == "mcp_unpinned_package" && fix.Package != "":
		return "Choose one reviewed version and apply it consistently anywhere this package executor is used."
	case fix.Rule == "mcp_local_endpoint":
		return "Move machine-local service assumptions into an ignored local overlay or document them as prerequisites."
	case fix.Rule == "mcp_broad_filesystem":
		return "Replace broad filesystem arguments with explicit reviewed paths."
	default:
		return fix.Summary
	}
}

func fixGroupSteps(fix Fix) []string {
	switch {
	case fix.Rule == "mcp_unpinned_package" && fix.Package != "":
		return []string{
			"Pick a reviewed package version once.",
			"Replace each matching package token with the same pinned version.",
			"Rerun `nw findings list --json` and confirm the grouped unpinned-package findings are gone.",
		}
	case len(fix.Steps) > 0:
		return append([]string(nil), fix.Steps...)
	default:
		return nil
	}
}

func appendUnique(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func statusRank(status Status) int {
	switch status {
	case StatusSafe:
		return 0
	case StatusReview:
		return 1
	default:
		return 2
	}
}

func riskRank(risk inventory.RiskLevel) int {
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
