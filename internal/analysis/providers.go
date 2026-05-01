package analysis

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jsonbored/nightward/internal/inventory"
)

const (
	providerTimeout        = 20 * time.Second
	providerOutputMaxBytes = 2 * 1024 * 1024
)

type providerFinding struct {
	Rule     string
	Path     string
	Message  string
	Evidence string
	Severity inventory.RiskLevel
	Category SignalCategory
}

type limitedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	accepted := len(p)
	if b.buf.Len()+len(p) > b.limit {
		remaining := b.limit - b.buf.Len()
		if remaining > 0 {
			_, _ = b.buf.Write(p[:remaining])
		}
		b.truncated = true
		return accepted, nil
	}
	_, _ = b.buf.Write(p)
	return accepted, nil
}

func (b *limitedBuffer) String() string {
	value := b.buf.String()
	if b.truncated {
		value += "\n[provider output truncated]"
	}
	return value
}

var providerSecretPattern = regexp.MustCompile(`(?i)((?:token|secret|password|passwd|api[_-]?key|auth|credential|private[_-]?key)[\w.-]*\s*[:=]\s*)(["']?)[^"',\s}]+`)

func appendProviderSignals(out *Report, report inventory.Report, options Options) {
	selected := selectedProviders(options.With)
	if len(selected) == 0 {
		return
	}
	root := report.Workspace
	if root == "" {
		root = options.Workspace
	}
	if root == "" {
		root = report.Home
	}
	if root == "" {
		return
	}
	for _, status := range out.Providers {
		if !status.Enabled || status.Kind != "local-command" || status.Status != "ready" {
			continue
		}
		if !selected["all"] && !selected[status.Name] {
			continue
		}
		findings, err := runProvider(status.Name, root)
		if err != nil {
			appendProviderSignal(out, status.Name, providerFinding{
				Rule:     "provider_execution_failed",
				Path:     root,
				Message:  fmt.Sprintf("%s provider execution failed.", status.Name),
				Evidence: redactProviderText(err.Error()),
				Severity: inventory.RiskLow,
				Category: CategoryUnknown,
			})
			continue
		}
		for _, finding := range findings {
			appendProviderSignal(out, status.Name, finding)
		}
	}
}

func appendProviderSignal(out *Report, provider string, finding providerFinding) {
	rule := strings.TrimSpace(finding.Rule)
	if rule == "" {
		rule = "provider_signal"
	}
	severity := finding.Severity
	if severity == "" {
		severity = inventory.RiskMedium
	}
	category := finding.Category
	if category == "" {
		category = CategoryUnknown
	}
	subjectID := stableID("provider-subject", provider, rule, finding.Path, finding.Evidence)
	out.Subjects = append(out.Subjects, Subject{
		ID:       subjectID,
		Type:     SubjectItem,
		Name:     provider + "/" + rule,
		Tool:     provider,
		Path:     finding.Path,
		Rule:     provider + "/" + rule,
		Evidence: finding.Evidence,
	})
	out.Signals = append(out.Signals, Signal{
		ID:             stableID("signal", provider, rule, finding.Path, finding.Evidence),
		Provider:       provider,
		Rule:           provider + "/" + rule,
		Category:       category,
		SubjectID:      subjectID,
		SubjectType:    SubjectItem,
		Path:           finding.Path,
		Severity:       severity,
		Confidence:     "medium",
		Message:        finding.Message,
		Evidence:       finding.Evidence,
		Recommendation: providerRecommendation(provider, category),
		Why:            "Provider execution was explicitly requested, so Nightward preserves only redacted finding metadata for review.",
	})
}

func runProvider(name, root string) ([]providerFinding, error) {
	args, ok, err := providerArgs(name, root)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), providerTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...) // #nosec G204 -- provider name is selected from Nightward's built-in provider allowlist.
	var stdout, stderr limitedBuffer
	stdout.limit = providerOutputMaxBytes
	stderr.limit = 64 * 1024
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("provider timed out after %s", providerTimeout)
	}
	if err != nil && strings.TrimSpace(stdout.String()) == "" {
		return nil, fmt.Errorf("%v: %s", err, firstProviderLine(stderr.String()))
	}
	return parseProviderOutput(name, root, stdout.String())
}

func providerArgs(name, root string) ([]string, bool, error) {
	switch name {
	case "gitleaks":
		return []string{"detect", "--no-git", "--redact", "--no-banner", "--source", root, "--report-format", "json", "--exit-code", "0"}, true, nil
	case "trufflehog":
		return []string{"filesystem", "--json", "--no-update", root}, true, nil
	case "semgrep":
		config, ok := localSemgrepConfig(root)
		if !ok {
			return nil, true, fmt.Errorf("semgrep local config not found; add semgrep.yml, semgrep.yaml, .semgrep.yml, .semgrep.yaml, or .semgrep/config.yml")
		}
		return []string{"scan", "--json", "--metrics=off", "--disable-version-check", "--config", config, root}, true, nil
	default:
		return nil, false, nil
	}
}

func localSemgrepConfig(root string) (string, bool) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	if resolvedRoot, err := filepath.EvalSymlinks(rootAbs); err == nil {
		rootAbs = resolvedRoot
	}
	for _, rel := range []string{
		"semgrep.yml",
		"semgrep.yaml",
		".semgrep.yml",
		".semgrep.yaml",
		filepath.Join(".semgrep", "config.yml"),
		filepath.Join(".semgrep", "config.yaml"),
	} {
		path := filepath.Join(root, rel)
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			continue
		}
		resolvedAbs, err := filepath.Abs(resolved)
		if err != nil {
			continue
		}
		relToRoot, err := filepath.Rel(rootAbs, resolvedAbs)
		if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
			continue
		}
		return resolvedAbs, true
	}
	return "", false
}

func parseProviderOutput(name, root, output string) ([]providerFinding, error) {
	switch name {
	case "gitleaks":
		return parseGitleaks(root, output)
	case "trufflehog":
		return parseTrufflehog(root, output)
	case "semgrep":
		return parseSemgrep(root, output)
	default:
		return nil, nil
	}
}

func parseGitleaks(root, output string) ([]providerFinding, error) {
	var records []map[string]any
	if strings.TrimSpace(output) == "" {
		return nil, nil
	}
	if err := json.Unmarshal([]byte(output), &records); err != nil {
		return nil, err
	}
	findings := make([]providerFinding, 0, len(records))
	for _, record := range records {
		rule := firstString(record, "RuleID", "ruleID", "Rule", "rule")
		if rule == "" {
			rule = "secret"
		}
		path := relativeProviderPath(root, firstString(record, "File", "file", "Path", "path"))
		line := firstNumber(record, "StartLine", "line")
		evidence := fmt.Sprintf("rule=%s file=%s", redactProviderText(rule), redactProviderText(path))
		if line != "" {
			evidence += " line=" + line
		}
		message := firstString(record, "Description", "description")
		if message == "" {
			message = "Gitleaks reported a secret-like value."
		}
		findings = append(findings, providerFinding{Rule: rule, Path: path, Message: redactProviderText(message), Evidence: evidence, Severity: inventory.RiskHigh, Category: CategorySecrets})
	}
	return findings, nil
}

func parseTrufflehog(root, output string) ([]providerFinding, error) {
	var findings []providerFinding
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, err
		}
		rule := firstString(record, "DetectorName", "detector_name", "SourceName")
		if rule == "" {
			rule = "secret"
		}
		path := relativeProviderPath(root, nestedString(record, "SourceMetadata", "Data", "Filesystem", "file"))
		if path == "" {
			path = relativeProviderPath(root, firstString(record, "File", "file", "Path", "path"))
		}
		severity := inventory.RiskHigh
		if verified, ok := record["Verified"].(bool); ok && verified {
			severity = inventory.RiskCritical
		}
		evidence := fmt.Sprintf("detector=%s file=%s", redactProviderText(rule), redactProviderText(path))
		findings = append(findings, providerFinding{Rule: rule, Path: path, Message: "TruffleHog reported a secret-like value.", Evidence: evidence, Severity: severity, Category: CategorySecrets})
	}
	return findings, scanner.Err()
}

func parseSemgrep(root, output string) ([]providerFinding, error) {
	if strings.TrimSpace(output) == "" {
		return nil, nil
	}
	var doc struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal([]byte(output), &doc); err != nil {
		return nil, err
	}
	findings := make([]providerFinding, 0, len(doc.Results))
	for _, record := range doc.Results {
		rule := firstString(record, "check_id", "rule_id")
		if rule == "" {
			rule = "semgrep"
		}
		path := relativeProviderPath(root, firstString(record, "path"))
		extra, _ := record["extra"].(map[string]any)
		message := firstString(extra, "message")
		if message == "" {
			message = "Semgrep reported a static analysis finding."
		}
		severity := semgrepSeverity(firstString(extra, "severity"))
		evidence := fmt.Sprintf("rule=%s file=%s", redactProviderText(rule), redactProviderText(path))
		findings = append(findings, providerFinding{Rule: rule, Path: path, Message: redactProviderText(message), Evidence: evidence, Severity: severity, Category: CategoryExecution})
	}
	return findings, nil
}

func semgrepSeverity(value string) inventory.RiskLevel {
	switch strings.ToUpper(value) {
	case "ERROR", "HIGH", "CRITICAL":
		return inventory.RiskHigh
	case "WARNING", "WARN", "MEDIUM":
		return inventory.RiskMedium
	case "INFO", "LOW":
		return inventory.RiskLow
	default:
		return inventory.RiskMedium
	}
}

func providerRecommendation(provider string, category SignalCategory) string {
	if category == CategorySecrets {
		return "Rotate exposed credentials if confirmed, remove secret material from portable config, and keep only local secret references."
	}
	return fmt.Sprintf("Review the %s finding locally before trusting or syncing this configuration.", provider)
}

func relativeProviderPath(root, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return root
	}
	if filepath.IsAbs(path) {
		return RelativeWorkspacePath(root, path)
	}
	return path
}

func firstString(record map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := record[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstNumber(record map[string]any, keys ...string) string {
	for _, key := range keys {
		switch value := record[key].(type) {
		case float64:
			return fmt.Sprintf("%.0f", value)
		case int:
			return fmt.Sprintf("%d", value)
		case string:
			return value
		}
	}
	return ""
}

func nestedString(record map[string]any, keys ...string) string {
	var current any = record
	for _, key := range keys {
		values, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = values[key]
	}
	if value, ok := current.(string); ok {
		return value
	}
	return ""
}

func firstProviderLine(value string) string {
	line := strings.Split(strings.TrimSpace(value), "\n")[0]
	if len(line) > 300 {
		return line[:300]
	}
	return redactProviderText(line)
}

func redactProviderText(value string) string {
	value = providerSecretPattern.ReplaceAllString(value, "$1$2[redacted]")
	if len(value) > 500 {
		return value[:500] + "..."
	}
	return value
}
