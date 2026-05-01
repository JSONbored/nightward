package policy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jsonbored/nightward/internal/analysis"
	"github.com/jsonbored/nightward/internal/inventory"
)

func TestCheckUsesStrictThreshold(t *testing.T) {
	report := inventory.Report{
		GeneratedAt: time.Date(2026, 4, 30, 7, 0, 0, 0, time.UTC),
		Findings: []inventory.Finding{
			{ID: "low", Severity: inventory.RiskLow, Rule: "mcp_server_review"},
			{ID: "medium", Severity: inventory.RiskMedium, Rule: "mcp_broad_filesystem"},
			{ID: "high", Severity: inventory.RiskHigh, Rule: "mcp_unpinned_package"},
		},
	}

	standard := Check(report, false)
	if standard.Passed || len(standard.Violations) != 1 || standard.Threshold != inventory.RiskHigh {
		t.Fatalf("unexpected standard policy report: %#v", standard)
	}

	strict := Check(report, true)
	if strict.Passed || len(strict.Violations) != 2 || strict.Threshold != inventory.RiskMedium {
		t.Fatalf("unexpected strict policy report: %#v", strict)
	}
}

func TestLoadConfigRejectsUnknownKeysAndReasonlessIgnores(t *testing.T) {
	dir := t.TempDir()
	unknown := filepath.Join(dir, ".nightward.yml")
	if err := os.WriteFile(unknown, []byte("surprise: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(unknown); err == nil {
		t.Fatal("expected unknown policy keys to fail")
	}

	reasonless := filepath.Join(dir, "reasonless.yml")
	if err := os.WriteFile(reasonless, []byte("ignore_rules:\n  - rule: mcp_server_review\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(reasonless); err == nil || !strings.Contains(err.Error(), "requires reason") {
		t.Fatalf("expected missing ignore reason to fail, got %v", err)
	}
}

func TestDefaultConfigExplainValidateAndWriteSARIF(t *testing.T) {
	config := DefaultConfig()
	if config.SeverityThreshold != inventory.RiskHigh || config.SARIF.ToolName != "Nightward" {
		t.Fatalf("unexpected default config: %#v", config)
	}
	yamlText := DefaultConfigYAML()
	for _, want := range []string{"severity_threshold: high", "analysis_threshold: high", "tool_name: Nightward"} {
		if !strings.Contains(yamlText, want) {
			t.Fatalf("default config YAML missing %q:\n%s", want, yamlText)
		}
	}
	if text := Explain(); !strings.Contains(text, "KnownFields") && !strings.Contains(text, "Ignore entries must include a reason") {
		t.Fatalf("unexpected policy explanation:\n%s", text)
	}
	if err := ValidateConfig(Config{SeverityThreshold: "urgent"}); err == nil {
		t.Fatal("expected unsupported severity threshold error")
	}
	if err := ValidateConfig(Config{AnalysisThreshold: "urgent"}); err == nil {
		t.Fatal("expected unsupported analysis threshold error")
	}
	if err := ValidateConfig(Config{IgnoreFindings: []IgnoreFinding{{Reason: "missing id"}}}); err == nil {
		t.Fatal("expected missing ignore finding id error")
	}
	if err := ValidateConfig(Config{IgnoreRules: []IgnoreRule{{Reason: "missing rule"}}}); err == nil {
		t.Fatal("expected missing ignore rule error")
	}

	out := filepath.Join(t.TempDir(), "nested", "nightward.sarif")
	if err := WriteSARIFObject(BuildSARIF(inventory.Report{}), out); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"2.1.0"`) {
		t.Fatalf("unexpected SARIF file:\n%s", data)
	}

	out = filepath.Join(t.TempDir(), "nightward.sarif")
	if err := WriteSARIF(inventory.Report{}, out); err != nil {
		t.Fatal(err)
	}
	if err := WriteSARIFWithConfig(inventory.Report{}, filepath.Join(t.TempDir(), "nightward.sarif"), Config{SARIF: SARIFConfig{ToolName: "Custom Nightward"}}); err != nil {
		t.Fatal(err)
	}
}

func TestCheckWithConfigIgnoresWithReasonAndOverridesThreshold(t *testing.T) {
	report := inventory.Report{
		GeneratedAt: time.Date(2026, 4, 30, 7, 0, 0, 0, time.UTC),
		Findings: []inventory.Finding{
			{ID: "review", Severity: inventory.RiskInfo, Rule: "mcp_server_review"},
			{ID: "medium", Severity: inventory.RiskMedium, Rule: "mcp_broad_filesystem"},
			{ID: "high", Severity: inventory.RiskHigh, Rule: "mcp_unpinned_package", Evidence: "args=@example/server"},
		},
	}

	config := Config{
		SeverityThreshold: inventory.RiskMedium,
		IgnoreRules:       []IgnoreRule{{Rule: "mcp_broad_filesystem", Reason: "fixture path is intentionally broad"}},
		TrustedPackages:   []string{"@example/server"},
	}
	checked := CheckWithOptions(report, Options{Config: config})
	if !checked.Passed {
		t.Fatalf("expected trusted/ignored config to pass: %#v", checked)
	}
	if checked.Summary.Ignored != 2 || len(checked.Ignored) != 2 {
		t.Fatalf("expected two ignored findings: %#v", checked)
	}
}

func TestSARIFRedactsAndIncludesFixMetadata(t *testing.T) {
	report := inventory.Report{Findings: []inventory.Finding{
		{
			ID:             "mcp_secret_env-111111111111",
			Tool:           "Codex",
			Path:           "/tmp/config.toml",
			Severity:       inventory.RiskCritical,
			Rule:           "mcp_secret_env",
			Message:        "MCP server stores a sensitive environment key.",
			Evidence:       "env_key=API_TOKEN",
			Recommendation: "Keep secret values outside dotfiles.",
			FixAvailable:   true,
			FixKind:        inventory.FixExternalizeSecret,
			Confidence:     "high",
			Risk:           inventory.RiskHigh,
			RequiresReview: true,
			FixSummary:     "Move API_TOKEN out of this config.",
			FixSteps:       []string{"Remove the inline value for API_TOKEN."},
		},
	}}

	sarif := BuildSARIF(report)
	data, err := json.Marshal(sarif)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"2.1.0", "mcp_secret_env", "externalize-secret", "API_TOKEN"} {
		if !strings.Contains(text, want) {
			t.Fatalf("SARIF missing %q: %s", want, text)
		}
	}
	if strings.Contains(text, "super-secret-value") {
		t.Fatal("SARIF leaked a secret value")
	}
}

func TestSARIFUsesConfigMetadataAndIgnores(t *testing.T) {
	report := inventory.Report{Findings: []inventory.Finding{
		{ID: "ignored", Severity: inventory.RiskHigh, Rule: "mcp_server_review", Message: "ignored"},
		{ID: "kept", Severity: inventory.RiskHigh, Rule: "mcp_unpinned_package", Message: "kept"},
	}}
	config := Config{
		IgnoreFindings: []IgnoreFinding{{ID: "ignored", Reason: "accepted advisory"}},
		SARIF: SARIFConfig{
			ToolName:       "Nightward CI",
			Category:       "nightward-fixture",
			InformationURI: "https://example.invalid/nightward",
		},
	}

	sarif := BuildSARIFWithConfig(report, config)
	data, err := json.Marshal(sarif)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"Nightward CI", "nightward-fixture", "kept"} {
		if !strings.Contains(text, want) {
			t.Fatalf("SARIF missing %q: %s", want, text)
		}
	}
	if strings.Contains(text, "ignored") {
		t.Fatalf("SARIF included ignored finding: %s", text)
	}
}

func TestPolicyCanIncludeAnalysisSignals(t *testing.T) {
	report := inventory.Report{Findings: []inventory.Finding{
		{ID: "review", Severity: inventory.RiskInfo, Rule: "mcp_server_review", Message: "review"},
	}}
	analysisReport := analysis.Report{Signals: []analysis.Signal{
		{
			ID:             "signal-1",
			Provider:       "nightward",
			Rule:           "nightward/secret_auth_path",
			Category:       analysis.CategorySecrets,
			SubjectID:      "item-1",
			SubjectType:    analysis.SubjectItem,
			Path:           "/tmp/workspace/.env",
			Severity:       inventory.RiskCritical,
			Confidence:     "high",
			Message:        "Secret path present.",
			Evidence:       "classification=secret-auth path=/tmp/workspace/.env",
			Recommendation: "Exclude it.",
		},
	}}

	checked := CheckWithOptions(report, Options{IncludeAnalysis: true, Analysis: analysisReport})
	if checked.Passed || checked.Summary.SignalViolations != 1 || len(checked.SignalViolations) != 1 {
		t.Fatalf("expected analysis signal policy violation: %#v", checked)
	}

	sarif := BuildSARIFWithAnalysis(report, analysisReport, Config{})
	data, err := json.Marshal(sarif)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"nightward/analyze/secret_auth_path", "signal-1", ".env"} {
		if !strings.Contains(text, want) {
			t.Fatalf("SARIF missing analysis signal %q: %s", want, text)
		}
	}
}

func TestGoldenSARIFForURLSecurityFindings(t *testing.T) {
	report := inventory.Report{Findings: []inventory.Finding{
		{
			ID:             "mcp_secret_header-111111111111",
			Tool:           "Codex",
			Path:           "/tmp/nightward-golden-home/.codex/config.toml",
			Server:         "headers",
			Severity:       inventory.RiskCritical,
			Rule:           "mcp_secret_header",
			Message:        "MCP server \"headers\" stores a sensitive header.",
			Evidence:       "header_key=Authorization",
			Recommendation: "Keep sensitive header values outside dotfiles.",
			Impact:         "Credential-bearing headers in agent config can leak.",
			Why:            "Remote MCP servers often use headers for authentication.",
			FixAvailable:   true,
			FixKind:        inventory.FixExternalizeSecret,
			Confidence:     "high",
			Risk:           inventory.RiskHigh,
			RequiresReview: true,
			FixSummary:     "Move the Authorization header value out of this config.",
			FixSteps:       []string{"Remove the inline value for the Authorization header."},
		},
		{
			ID:             "mcp_local_endpoint-222222222222",
			Tool:           "Cursor",
			Path:           "/tmp/nightward-golden-home/.cursor/mcp.json",
			Server:         "local",
			Severity:       inventory.RiskMedium,
			Rule:           "mcp_local_endpoint",
			Message:        "MCP server \"local\" points at a local or private endpoint.",
			Evidence:       "transport=remote-url type=unknown url=http://127.0.0.1:8787",
			Recommendation: "Keep local endpoint assumptions machine-local unless intentionally templated.",
			Impact:         "Local or private MCP endpoints may not exist on another machine.",
			Why:            "Portable dotfiles should distinguish remote service configuration from machine-local development endpoints.",
			FixAvailable:   true,
			FixKind:        inventory.FixManualReview,
			Confidence:     "medium",
			Risk:           inventory.RiskLow,
			RequiresReview: true,
			FixSummary:     "Move local endpoint assumptions into a machine-local overlay.",
			FixSteps:       []string{"Confirm whether this MCP endpoint is intentionally machine-local."},
		},
	}}

	sarif := BuildSARIF(report)
	assertGoldenPolicyJSON(t, "testdata/golden/url-security.sarif.golden.json", sarif)
	data, err := json.Marshal(sarif)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, leaked := range []string{"super-header-secret", "Bearer "} {
		if strings.Contains(text, leaked) {
			t.Fatalf("SARIF leaked secret value %q: %s", leaked, text)
		}
	}
}

func TestGoldenSARIFForAnalysisSignals(t *testing.T) {
	report := inventory.Report{Findings: []inventory.Finding{
		{
			ID:             "mcp_broad_filesystem-111111111111",
			Tool:           "Claude Code",
			Path:           "/tmp/nightward-golden-home/.claude/mcp.json",
			Server:         "filesystem",
			Severity:       inventory.RiskMedium,
			Rule:           "mcp_broad_filesystem",
			Message:        "MCP server \"filesystem\" can access a broad filesystem path.",
			Evidence:       "arg=/Users/test",
			Recommendation: "Narrow filesystem server access to specific project paths.",
			FixAvailable:   true,
			FixKind:        inventory.FixNarrowFilesystem,
			Confidence:     "medium",
			Risk:           inventory.RiskLow,
			RequiresReview: true,
			FixSummary:     "Replace broad filesystem arguments with the smallest project paths needed.",
			FixSteps:       []string{"Confirm which paths this MCP server needs before syncing the config."},
		},
	}}
	analysisReport := analysis.Report{Signals: []analysis.Signal{
		{
			ID:             "signal-provider-1",
			Provider:       "gitleaks",
			Rule:           "nightward/provider/gitleaks",
			Category:       analysis.CategorySecrets,
			SubjectID:      "workspace",
			SubjectType:    analysis.SubjectItem,
			Path:           "/tmp/nightward-golden-home/workspace/.env",
			Severity:       inventory.RiskHigh,
			Confidence:     "medium",
			Message:        "gitleaks reported possible secret material.",
			Evidence:       "provider=gitleaks finding=1 file=.env",
			Recommendation: "Review gitleaks findings locally and rotate any exposed credentials.",
			Why:            "Provider results can reveal workspace secrets outside known agent configs.",
		},
	}}

	sarif := BuildSARIFWithAnalysis(report, analysisReport, Config{})
	assertGoldenPolicyJSON(t, "testdata/golden/analysis-signals.sarif.golden.json", sarif)
	data, err := json.Marshal(sarif)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "super-secret-value") {
		t.Fatalf("SARIF leaked secret value: %s", data)
	}
}

func assertGoldenPolicyJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	actual := string(data) + "\n"
	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read golden %s: %v\nactual:\n%s", path, err, actual)
	}
	if string(expected) != actual {
		t.Fatalf("golden mismatch for %s\nexpected:\n%s\nactual:\n%s", path, expected, actual)
	}
}
