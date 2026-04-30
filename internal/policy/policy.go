package policy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/shadowbook/nightward/internal/inventory"
)

type Report struct {
	GeneratedAt time.Time           `json:"generated_at"`
	Strict      bool                `json:"strict"`
	Passed      bool                `json:"passed"`
	Threshold   inventory.RiskLevel `json:"threshold"`
	Summary     Summary             `json:"summary"`
	Violations  []inventory.Finding `json:"violations"`
}

type Summary struct {
	TotalFindings int `json:"total_findings"`
	Violations    int `json:"violations"`
	Critical      int `json:"critical"`
	High          int `json:"high"`
	Medium        int `json:"medium"`
	Low           int `json:"low"`
	Info          int `json:"info"`
}

func Check(report inventory.Report, strict bool) Report {
	threshold := inventory.RiskHigh
	if strict {
		threshold = inventory.RiskMedium
	}
	out := Report{
		GeneratedAt: report.GeneratedAt,
		Strict:      strict,
		Threshold:   threshold,
		Summary: Summary{
			TotalFindings: len(report.Findings),
		},
	}
	for _, finding := range report.Findings {
		switch finding.Severity {
		case inventory.RiskCritical:
			out.Summary.Critical++
		case inventory.RiskHigh:
			out.Summary.High++
		case inventory.RiskMedium:
			out.Summary.Medium++
		case inventory.RiskLow:
			out.Summary.Low++
		default:
			out.Summary.Info++
		}
		if riskRank(finding.Severity) >= riskRank(threshold) {
			out.Violations = append(out.Violations, finding)
		}
	}
	out.Summary.Violations = len(out.Violations)
	out.Passed = len(out.Violations) == 0
	return out
}

func WriteSARIF(report inventory.Report, path string) error {
	sarif := BuildSARIF(report)
	data, err := json.MarshalIndent(sarif, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func BuildSARIF(report inventory.Report) map[string]any {
	rules := sarifRules(report.Findings)
	results := make([]map[string]any, 0, len(report.Findings))
	for _, finding := range report.Findings {
		results = append(results, sarifResult(finding))
	}
	return map[string]any{
		"$schema": "https://json.schemastore.org/sarif-2.1.0.json",
		"version": "2.1.0",
		"runs": []map[string]any{
			{
				"tool": map[string]any{
					"driver": map[string]any{
						"name":            "Nightward",
						"informationUri":  "https://github.com/JSONbored/nightward",
						"semanticVersion": "0.1.0",
						"rules":           rules,
					},
				},
				"results": results,
			},
		},
	}
}

func sarifRules(findings []inventory.Finding) []map[string]any {
	byRule := map[string]inventory.Finding{}
	for _, finding := range findings {
		if _, ok := byRule[finding.Rule]; !ok {
			byRule[finding.Rule] = finding
		}
	}
	keys := make([]string, 0, len(byRule))
	for key := range byRule {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	rules := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		finding := byRule[key]
		rules = append(rules, map[string]any{
			"id":   key,
			"name": strings.ReplaceAll(key, "_", " "),
			"shortDescription": map[string]string{
				"text": finding.Message,
			},
			"fullDescription": map[string]string{
				"text": finding.Why,
			},
			"help": map[string]any{
				"text":     finding.Recommendation,
				"markdown": finding.FixSummary,
			},
			"properties": map[string]any{
				"severity":        finding.Severity,
				"fix_kind":        finding.FixKind,
				"fix_available":   finding.FixAvailable,
				"requires_review": finding.RequiresReview,
			},
		})
	}
	return rules
}

func sarifResult(finding inventory.Finding) map[string]any {
	return map[string]any{
		"ruleId":  finding.Rule,
		"level":   sarifLevel(finding.Severity),
		"message": map[string]string{"text": finding.Message},
		"locations": []map[string]any{
			{
				"physicalLocation": map[string]any{
					"artifactLocation": map[string]string{
						"uri": artifactURI(finding.Path),
					},
				},
			},
		},
		"properties": map[string]any{
			"finding_id":      finding.ID,
			"tool":            finding.Tool,
			"severity":        finding.Severity,
			"evidence":        finding.Evidence,
			"recommended_fix": finding.FixSummary,
			"fix_steps":       finding.FixSteps,
			"fix_kind":        finding.FixKind,
			"fix_risk":        finding.Risk,
			"confidence":      finding.Confidence,
			"requires_review": finding.RequiresReview,
		},
	}
}

func artifactURI(path string) string {
	cwd, err := os.Getwd()
	if err == nil {
		if rel, relErr := filepath.Rel(cwd, path); relErr == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(path)
}

func sarifLevel(risk inventory.RiskLevel) string {
	switch risk {
	case inventory.RiskCritical, inventory.RiskHigh:
		return "error"
	case inventory.RiskMedium:
		return "warning"
	default:
		return "note"
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
