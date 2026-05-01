package policy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jsonbored/nightward/internal/analysis"
	"github.com/jsonbored/nightward/internal/inventory"
	"gopkg.in/yaml.v3"
)

type Report struct {
	SchemaVersion    int                 `json:"schema_version"`
	GeneratedAt      time.Time           `json:"generated_at"`
	Strict           bool                `json:"strict"`
	Passed           bool                `json:"passed"`
	Threshold        inventory.RiskLevel `json:"threshold"`
	ConfigPath       string              `json:"config_path,omitempty"`
	Summary          Summary             `json:"summary"`
	Violations       []inventory.Finding `json:"violations"`
	SignalViolations []analysis.Signal   `json:"signal_violations,omitempty"`
	Ignored          []IgnoredFinding    `json:"ignored,omitempty"`
}

type Badge struct {
	SchemaVersion    int                 `json:"schemaVersion"`
	Label            string              `json:"label"`
	Message          string              `json:"message"`
	Color            string              `json:"color"`
	GeneratedAt      time.Time           `json:"generated_at"`
	Passed           bool                `json:"passed"`
	Threshold        inventory.RiskLevel `json:"threshold"`
	TotalFindings    int                 `json:"total_findings"`
	Violations       int                 `json:"violations"`
	TotalSignals     int                 `json:"total_signals,omitempty"`
	SignalViolations int                 `json:"signal_violations,omitempty"`
	Ignored          int                 `json:"ignored"`
	Critical         int                 `json:"critical"`
	High             int                 `json:"high"`
	Medium           int                 `json:"medium"`
	Low              int                 `json:"low"`
	Info             int                 `json:"info"`
	SARIFURL         string              `json:"sarif_url,omitempty"`
}

type Summary struct {
	TotalFindings    int `json:"total_findings"`
	TotalSignals     int `json:"total_signals,omitempty"`
	Violations       int `json:"violations"`
	SignalViolations int `json:"signal_violations,omitempty"`
	Ignored          int `json:"ignored"`
	Critical         int `json:"critical"`
	High             int `json:"high"`
	Medium           int `json:"medium"`
	Low              int `json:"low"`
	Info             int `json:"info"`
}

type Config struct {
	SeverityThreshold     inventory.RiskLevel `json:"severity_threshold,omitempty" yaml:"severity_threshold"`
	IgnoreFindings        []IgnoreFinding     `json:"ignore_findings,omitempty" yaml:"ignore_findings"`
	IgnoreRules           []IgnoreRule        `json:"ignore_rules,omitempty" yaml:"ignore_rules"`
	TrustedCommands       []string            `json:"trusted_commands,omitempty" yaml:"trusted_commands"`
	TrustedPackages       []string            `json:"trusted_packages,omitempty" yaml:"trusted_packages"`
	PortableAllowPaths    []string            `json:"portable_allow_paths,omitempty" yaml:"portable_allow_paths"`
	MachineLocalDenyPaths []string            `json:"machine_local_deny_paths,omitempty" yaml:"machine_local_deny_paths"`
	IncludeAnalysis       bool                `json:"include_analysis,omitempty" yaml:"include_analysis"`
	AnalysisThreshold     inventory.RiskLevel `json:"analysis_threshold,omitempty" yaml:"analysis_threshold"`
	AnalysisProviders     []string            `json:"analysis_providers,omitempty" yaml:"analysis_providers"`
	AllowOnlineProviders  bool                `json:"allow_online_providers,omitempty" yaml:"allow_online_providers"`
	SARIF                 SARIFConfig         `json:"sarif,omitempty" yaml:"sarif"`
	path                  string
}

type IgnoreFinding struct {
	ID     string `json:"id" yaml:"id"`
	Reason string `json:"reason" yaml:"reason"`
}

type IgnoreRule struct {
	Rule   string `json:"rule" yaml:"rule"`
	Reason string `json:"reason" yaml:"reason"`
}

type IgnoredFinding struct {
	FindingID string `json:"finding_id"`
	Rule      string `json:"rule"`
	Reason    string `json:"reason"`
}

type SARIFConfig struct {
	ToolName        string `json:"tool_name,omitempty" yaml:"tool_name"`
	Category        string `json:"category,omitempty" yaml:"category"`
	InformationURI  string `json:"information_uri,omitempty" yaml:"information_uri"`
	SemanticVersion string `json:"semantic_version,omitempty" yaml:"semantic_version"`
}

const sarifAnalysisRulePrefix = "nightward/" + "analyze/"
const defaultSARIFSemanticVersion = "0.1.4"

type Options struct {
	Strict          bool
	Config          Config
	IncludeAnalysis bool
	Analysis        analysis.Report
}

func Check(report inventory.Report, strict bool) Report {
	return CheckWithOptions(report, Options{Strict: strict})
}

func BuildBadge(report Report, sarifURL string) Badge {
	message := "passing"
	color := "brightgreen"
	if !report.Passed {
		message = fmt.Sprintf("%d violations", report.Summary.Violations+report.Summary.SignalViolations)
		color = "red"
	}
	return Badge{
		SchemaVersion:    1,
		Label:            "nightward",
		Message:          message,
		Color:            color,
		GeneratedAt:      report.GeneratedAt,
		Passed:           report.Passed,
		Threshold:        report.Threshold,
		TotalFindings:    report.Summary.TotalFindings,
		Violations:       report.Summary.Violations,
		TotalSignals:     report.Summary.TotalSignals,
		SignalViolations: report.Summary.SignalViolations,
		Ignored:          report.Summary.Ignored,
		Critical:         report.Summary.Critical,
		High:             report.Summary.High,
		Medium:           report.Summary.Medium,
		Low:              report.Summary.Low,
		Info:             report.Summary.Info,
		SARIFURL:         strings.TrimSpace(sarifURL),
	}
}

func CheckWithOptions(report inventory.Report, options Options) Report {
	threshold := inventory.RiskHigh
	if options.Strict {
		threshold = inventory.RiskMedium
	}
	if options.Config.SeverityThreshold != "" {
		threshold = options.Config.SeverityThreshold
	}
	out := Report{
		SchemaVersion: 1,
		GeneratedAt:   report.GeneratedAt,
		Strict:        options.Strict,
		Threshold:     threshold,
		ConfigPath:    options.Config.path,
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
		if reason, ignored := ignoreReason(finding, options.Config); ignored {
			out.Ignored = append(out.Ignored, IgnoredFinding{FindingID: finding.ID, Rule: finding.Rule, Reason: reason})
			continue
		}
		if riskRank(finding.Severity) >= riskRank(threshold) {
			out.Violations = append(out.Violations, finding)
		}
	}
	if options.IncludeAnalysis || options.Config.IncludeAnalysis {
		analysisThreshold := threshold
		if options.Config.AnalysisThreshold != "" {
			analysisThreshold = options.Config.AnalysisThreshold
		}
		out.Summary.TotalSignals = len(options.Analysis.Signals)
		for _, signal := range options.Analysis.Signals {
			if riskRank(signal.Severity) >= riskRank(analysisThreshold) {
				out.SignalViolations = append(out.SignalViolations, signal)
			}
		}
		out.Summary.SignalViolations = len(out.SignalViolations)
	}
	out.Summary.Violations = len(out.Violations)
	out.Summary.Ignored = len(out.Ignored)
	out.Passed = len(out.Violations) == 0 && len(out.SignalViolations) == 0
	return out
}

func WriteSARIF(report inventory.Report, path string) error {
	return WriteSARIFWithConfig(report, path, Config{})
}

func WriteSARIFWithConfig(report inventory.Report, path string, config Config) error {
	sarif := BuildSARIFWithConfig(report, config)
	return WriteSARIFObject(sarif, path)
}

func WriteSARIFObject(sarif map[string]any, path string) error {
	data, err := json.MarshalIndent(sarif, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path = filepath.Clean(path)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil && dir != "." {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func WriteBadge(badge Badge, path string) error {
	data, err := json.MarshalIndent(badge, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path = filepath.Clean(path)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil && dir != "." {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func BuildSARIF(report inventory.Report) map[string]any {
	return BuildSARIFWithConfig(report, Config{})
}

func BuildSARIFWithConfig(report inventory.Report, config Config) map[string]any {
	return BuildSARIFWithAnalysis(report, analysis.Report{}, config)
}

func BuildSARIFWithAnalysis(report inventory.Report, analysisReport analysis.Report, config Config) map[string]any {
	included := make([]inventory.Finding, 0, len(report.Findings))
	for _, finding := range report.Findings {
		if _, ignored := ignoreReason(finding, config); ignored {
			continue
		}
		included = append(included, finding)
	}
	rules := sarifRules(included)
	if len(analysisReport.Signals) > 0 {
		rules = append(rules, sarifSignalRules(analysisReport.Signals)...)
		sort.Slice(rules, func(i, j int) bool {
			return fmt.Sprint(rules[i]["id"]) < fmt.Sprint(rules[j]["id"])
		})
	}
	results := make([]map[string]any, 0, len(included))
	for _, finding := range included {
		results = append(results, sarifResult(finding))
	}
	for _, signal := range analysisReport.Signals {
		results = append(results, sarifSignalResult(signal))
	}
	name := config.SARIF.ToolName
	if name == "" {
		name = "Nightward"
	}
	infoURI := config.SARIF.InformationURI
	if infoURI == "" {
		infoURI = "https://github.com/JSONbored/nightward"
	}
	semanticVersion := config.SARIF.SemanticVersion
	if semanticVersion == "" {
		semanticVersion = defaultSARIFSemanticVersion
	}
	run := map[string]any{
		"tool": map[string]any{
			"driver": map[string]any{
				"name":            name,
				"informationUri":  infoURI,
				"semanticVersion": semanticVersion,
				"rules":           rules,
			},
		},
		"results": results,
	}
	if config.SARIF.Category != "" {
		run["automationDetails"] = map[string]string{"id": config.SARIF.Category}
	}
	return map[string]any{
		"$schema": "https://json.schemastore.org/sarif-2.1.0.json",
		"version": "2.1.0",
		"runs":    []map[string]any{run},
	}
}

func DefaultConfig() Config {
	return Config{
		SeverityThreshold: inventory.RiskHigh,
		AnalysisThreshold: inventory.RiskHigh,
		SARIF: SARIFConfig{
			ToolName:        "Nightward",
			Category:        "nightward",
			InformationURI:  "https://github.com/JSONbored/nightward",
			SemanticVersion: defaultSARIFSemanticVersion,
		},
	}
}

func DefaultConfigYAML() string {
	data, err := yaml.Marshal(DefaultConfig())
	if err != nil {
		return "severity_threshold: high\n"
	}
	return string(data)
}

func Explain() string {
	return strings.TrimSpace(`Nightward policy config is optional and read-only.

Supported file: .nightward.yml

Fields:
  severity_threshold: info|low|medium|high|critical
  ignore_findings: [{id, reason}]
  ignore_rules: [{rule, reason}]
  trusted_commands: command names to suppress command-trust policy noise when evidence matches
  trusted_packages: package names to suppress unpinned-package policy noise when evidence matches
  portable_allow_paths: reviewed portable path prefixes for future adapter policy
  machine_local_deny_paths: path prefixes that should remain local-only
  include_analysis: include offline analysis signals in policy decisions
  analysis_threshold: optional signal threshold when include_analysis is true
  analysis_providers: optional provider names for future explicit provider analysis
  allow_online_providers: allow selected network-capable providers when analysis_providers requests them
  sarif.tool_name: SARIF tool display name
  sarif.category: SARIF automation category
  sarif.information_uri: SARIF tool information URI
  sarif.semantic_version: SARIF semantic version

Ignore entries must include a reason. Nightward never expands or prints secret values from policy config.`) + "\n"
}

func LoadConfig(path string) (Config, error) {
	if path == "" {
		return Config{}, nil
	}
	data, err := os.ReadFile(filepath.Clean(path)) // #nosec G304 -- policy config path is an explicit local CLI input.
	if err != nil {
		return Config{}, err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return Config{}, err
	}
	config.path = path
	if err := ValidateConfig(config); err != nil {
		return Config{}, err
	}
	return config, nil
}

func ValidateConfig(config Config) error {
	if config.SeverityThreshold != "" && !validRisk(config.SeverityThreshold) {
		return fmt.Errorf("unsupported severity_threshold %q", config.SeverityThreshold)
	}
	if config.AnalysisThreshold != "" && !validRisk(config.AnalysisThreshold) {
		return fmt.Errorf("unsupported analysis_threshold %q", config.AnalysisThreshold)
	}
	for _, entry := range config.IgnoreFindings {
		if strings.TrimSpace(entry.ID) == "" {
			return errors.New("ignore_findings entries require id")
		}
		if strings.TrimSpace(entry.Reason) == "" {
			return fmt.Errorf("ignore_findings entry %q requires reason", entry.ID)
		}
	}
	for _, entry := range config.IgnoreRules {
		if strings.TrimSpace(entry.Rule) == "" {
			return errors.New("ignore_rules entries require rule")
		}
		if strings.TrimSpace(entry.Reason) == "" {
			return fmt.Errorf("ignore_rules entry %q requires reason", entry.Rule)
		}
	}
	return nil
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

func sarifSignalRules(signals []analysis.Signal) []map[string]any {
	byRule := map[string]analysis.Signal{}
	for _, signal := range signals {
		ruleID := sarifAnalysisRulePrefix + strings.TrimPrefix(signal.Rule, "nightward/")
		if _, ok := byRule[ruleID]; !ok {
			byRule[ruleID] = signal
		}
	}
	keys := make([]string, 0, len(byRule))
	for key := range byRule {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	rules := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		signal := byRule[key]
		rules = append(rules, map[string]any{
			"id":   key,
			"name": strings.ReplaceAll(strings.TrimPrefix(key, sarifAnalysisRulePrefix), "_", " "),
			"shortDescription": map[string]string{
				"text": signal.Message,
			},
			"fullDescription": map[string]string{
				"text": signal.Why,
			},
			"help": map[string]any{
				"text":     signal.Recommendation,
				"markdown": signal.Recommendation,
			},
			"properties": map[string]any{
				"severity":   signal.Severity,
				"provider":   signal.Provider,
				"category":   signal.Category,
				"confidence": signal.Confidence,
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

func sarifSignalResult(signal analysis.Signal) map[string]any {
	path := "."
	if signal.Path != "" {
		path = signal.Path
	}
	ruleID := sarifAnalysisRulePrefix + strings.TrimPrefix(signal.Rule, "nightward/")
	return map[string]any{
		"ruleId":  ruleID,
		"level":   sarifLevel(signal.Severity),
		"message": map[string]string{"text": signal.Message},
		"locations": []map[string]any{
			{
				"physicalLocation": map[string]any{
					"artifactLocation": map[string]string{
						"uri": artifactURI(path),
					},
				},
			},
		},
		"properties": map[string]any{
			"signal_id":          signal.ID,
			"provider":           signal.Provider,
			"category":           signal.Category,
			"subject_id":         signal.SubjectID,
			"subject_type":       signal.SubjectType,
			"severity":           signal.Severity,
			"confidence":         signal.Confidence,
			"evidence":           signal.Evidence,
			"recommended_action": signal.Recommendation,
		},
	}
}

func ignoreReason(finding inventory.Finding, config Config) (string, bool) {
	for _, entry := range config.IgnoreFindings {
		if entry.ID == finding.ID {
			return entry.Reason, true
		}
	}
	for _, entry := range config.IgnoreRules {
		if entry.Rule == finding.Rule {
			return entry.Reason, true
		}
	}
	if finding.Rule == "mcp_unpinned_package" {
		for _, pkg := range config.TrustedPackages {
			if pkg != "" && strings.Contains(finding.Evidence, pkg) {
				return "trusted package policy exception", true
			}
		}
	}
	for _, command := range config.TrustedCommands {
		if command != "" && strings.Contains(finding.Evidence, "command="+command) {
			return "trusted command policy exception", true
		}
	}
	return "", false
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

func validRisk(risk inventory.RiskLevel) bool {
	switch risk {
	case inventory.RiskInfo, inventory.RiskLow, inventory.RiskMedium, inventory.RiskHigh, inventory.RiskCritical:
		return true
	default:
		return false
	}
}
