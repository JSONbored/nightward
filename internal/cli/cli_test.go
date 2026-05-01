package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jsonbored/nightward/internal/analysis"
	"github.com/jsonbored/nightward/internal/backupplan"
	"github.com/jsonbored/nightward/internal/inventory"
	"github.com/jsonbored/nightward/internal/policy"
	"github.com/jsonbored/nightward/internal/schedule"
	"github.com/jsonbored/nightward/internal/snapshot"
)

func TestReadOnlyCommandsDoNotMutateHome(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, filepath.Join(home, ".mcp.json"), `{"mcpServers":{"demo":{"command":"node","args":["server.js"]}}}`)
	outputDir := t.TempDir()
	targetDir := filepath.Join(t.TempDir(), "backup-target")
	t.Setenv("NIGHTWARD_HOME", home)

	before := listTestFiles(t, home)
	targetBefore := listOptionalTestFiles(t, targetDir)
	commands := [][]string{
		{"scan", "--json"},
		{"doctor", "--json"},
		{"plan", "backup", "--target", targetDir, "--json"},
		{"adapters", "explain", "Codex", "--json"},
		{"adapters", "template", "Codex", "--json"},
		{"findings", "list", "--json"},
		{"fix", "plan", "--all", "--json"},
		{"fix", "preview", "--all", "--format", "json"},
		{"fix", "export", "--format", "markdown"},
		{"analyze", "--all", "--json"},
		{"providers", "list", "--json"},
		{"providers", "doctor", "--json"},
		{"rules", "list", "--json"},
		{"rules", "explain", "--json", "mcp_secret_header"},
		{"policy", "check", "--json"},
		{"policy", "check", "--include-analysis", "--json"},
		{"policy", "badge", "--include-analysis", "--output", "-"},
		{"policy", "init", "--dry-run"},
		{"policy", "explain"},
		{"snapshot", "plan", "--target", filepath.Join(home, "snapshot"), "--json"},
		{"policy", "sarif", "--output", filepath.Join(outputDir, "nightward.sarif")},
		{"policy", "sarif", "--include-analysis", "--output", "-"},
		{"report", "history", "--json"},
		{"schedule", "plan", "--preset", "nightly", "--json"},
		{"schedule", "install", "--preset", "nightly", "--dry-run", "--json"},
		{"schedule", "remove", "--dry-run", "--json"},
	}
	for _, args := range commands {
		var stdout, stderr bytes.Buffer
		if code := RunWithName("nw", args, &stdout, &stderr); code != 0 {
			t.Fatalf("%s failed with %d: %s", strings.Join(args, " "), code, stderr.String())
		}
	}
	var stdout, stderr bytes.Buffer
	if code := RunWithName("nw", []string{"findings", "explain", "--json", "mcp_server_review"}, &stdout, &stderr); code != 0 {
		t.Fatalf("findings explain failed with %d: %s", code, stderr.String())
	}
	after := listTestFiles(t, home)
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatalf("read-only commands mutated home\nbefore=%v\nafter=%v", before, after)
	}
	targetAfter := listOptionalTestFiles(t, targetDir)
	if strings.Join(targetBefore, "\n") != strings.Join(targetAfter, "\n") {
		t.Fatalf("read-only backup plan mutated target\nbefore=%v\nafter=%v", targetBefore, targetAfter)
	}
}

func TestPublicCommandMatrixAndRedaction(t *testing.T) {
	home := t.TempDir()
	secretValue := "super-" + "secret-value"
	writeTestFile(t, filepath.Join(home, ".codex", "config.toml"), `[mcp_servers.demo]
command = "npx"
args = ["-y", "@modelcontextprotocol/server-filesystem", "/"]

[mcp_servers.demo.env]
API_TOKEN = "`+secretValue+`"
`)
	t.Setenv("NIGHTWARD_HOME", home)

	findingID := firstFindingID(t, []string{"findings", "list", "--json"})
	jsonCommands := [][]string{
		{"scan", "--json"},
		{"doctor", "--json"},
		{"adapters", "list", "--json"},
		{"adapters", "explain", "--json", "Codex"},
		{"adapters", "template", "--json", "Codex"},
		{"findings", "list", "--json"},
		{"findings", "explain", "--json", findingID},
		{"fix", "plan", "--all", "--json"},
		{"fix", "preview", "--all", "--format", "json"},
		{"analyze", "--all", "--json"},
		{"analyze", "finding", "--json", findingID},
		{"trust", "explain", "--json", findingID},
		{"providers", "list", "--json"},
		{"providers", "doctor", "--json"},
		{"rules", "list", "--json"},
		{"rules", "explain", "--json", "mcp_secret_header"},
		{"policy", "check", "--json"},
		{"policy", "badge", "--output", "-", "--sarif-url", "https://example.invalid/nightward.sarif"},
		{"policy", "sarif", "--output", "-"},
		{"report", "history", "--json"},
		{"snapshot", "plan", "--target", filepath.Join(home, "snapshots"), "--json"},
		{"schedule", "plan", "--json"},
		{"schedule", "install", "--dry-run", "--json"},
		{"schedule", "remove", "--dry-run", "--json"},
	}
	for _, args := range jsonCommands {
		stdout, stderr, code := runCLI(args)
		if args[0] == "policy" && args[1] == "check" {
			if code != 1 {
				t.Fatalf("%s expected policy violation exit 1, got %d stderr=%s", strings.Join(args, " "), code, stderr)
			}
		} else if code != 0 {
			t.Fatalf("%s failed with %d: %s", strings.Join(args, " "), code, stderr)
		}
		if !json.Valid([]byte(stdout)) {
			t.Fatalf("%s did not emit valid JSON:\n%s\nstderr=%s", strings.Join(args, " "), stdout, stderr)
		}
		assertNoSecret(t, stdout, secretValue)
	}

	textCommands := [][]string{
		{"scan"},
		{"doctor"},
		{"plan", "backup", "--target", filepath.Join(home, "backup")},
		{"findings", "list"},
		{"findings", "explain", findingID},
		{"fix", "plan", "--all"},
		{"fix", "preview", "--all", "--format", "markdown"},
		{"fix", "export", "--all", "--format", "markdown"},
		{"analyze", "--all"},
		{"trust", "explain", findingID},
		{"rules", "list"},
		{"rules", "explain", "mcp_secret_header"},
		{"adapters", "explain", "Codex"},
		{"adapters", "template", "Codex"},
		{"policy", "init", "--dry-run"},
		{"policy", "explain"},
		{"schedule", "install", "--dry-run"},
	}
	for _, args := range textCommands {
		stdout, stderr, code := runCLI(args)
		if code != 0 {
			t.Fatalf("%s failed with %d: %s", strings.Join(args, " "), code, stderr)
		}
		if stdout == "" {
			t.Fatalf("%s produced no stdout", strings.Join(args, " "))
		}
		assertNoSecret(t, stdout, secretValue)
	}

	failures := [][]string{
		{"plan", "backup", "--json"},
		{"fix", "preview", "--all", "--format", "xml"},
		{"policy", "init"},
		{"rules", "explain", "mcp_secret"},
		{"report", "html", "--input", "missing.json", "--output", filepath.Join(home, "report.html")},
		{"snapshot", "diff", "--from", "missing.json"},
		{"schedule", "install", "--preset", "bogus", "--dry-run"},
	}
	for _, args := range failures {
		_, _, code := runCLI(args)
		if code == 0 {
			t.Fatalf("%s unexpectedly succeeded", strings.Join(args, " "))
		}
	}
}

func TestReportHTMLCommandWritesPrivateReport(t *testing.T) {
	home := t.TempDir()
	t.Setenv("NIGHTWARD_HOME", home)
	scanPath := filepath.Join(home, "scan.json")
	reportPath := filepath.Join(home, "reports", "nightward.html")
	writeTestFile(t, scanPath, `{
  "generated_at": "2026-05-01T00:00:00Z",
  "hostname": "host<script>",
  "home": "/tmp/home",
  "summary": {
    "total_items": 1,
    "total_findings": 1,
    "findings_by_severity": {"high": 1}
  },
  "items": [{"id":"item-1","tool":"codex","path":"/tmp/<secret>","classification":"portable","risk":"low"}],
  "findings": [{"id":"finding-1","tool":"codex","path":"/tmp/config","severity":"high","rule":"mcp_secret_header","message":"<bad>","recommended_action":"externalize"}]
}`)

	stdout, stderr, code := runCLI([]string{"report", "html", "--input", scanPath, "--output", reportPath})
	if code != 0 {
		t.Fatalf("report html failed: stdout=%s stderr=%s", stdout, stderr)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	html := string(data)
	if strings.Contains(html, "<bad>") || strings.Contains(html, "host<script>") || strings.Contains(html, "/tmp/<secret>") {
		t.Fatalf("expected escaped HTML report:\n%s", html)
	}
	info, err := os.Stat(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("expected 0600 report, got %o", got)
	}
}

func TestReportDiffHistoryAndIndexCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("NIGHTWARD_HOME", home)
	reportsDir := filepath.Join(home, ".local", "state", "nightward", "reports")
	beforePath := filepath.Join(reportsDir, "before.json")
	afterPath := filepath.Join(reportsDir, "after.json")
	indexPath := filepath.Join(home, "reports", "index.html")
	htmlPath := filepath.Join(home, "reports", "current.html")
	writeTestFile(t, beforePath, `{
  "generated_at": "2026-05-01T00:00:00Z",
  "hostname": "host",
  "home": "/tmp/home",
  "summary": {"total_findings": 1, "findings_by_severity": {"high": 1}},
  "items": [],
  "findings": [{"id":"finding-1","tool":"codex","path":"/tmp/config","severity":"high","rule":"mcp_secret_header","message":"old","recommended_action":"externalize"}],
  "adapters": []
}`)
	writeTestFile(t, afterPath, `{
  "generated_at": "2026-05-01T00:05:00Z",
  "hostname": "host",
  "home": "/tmp/home",
  "summary": {"total_findings": 2, "findings_by_severity": {"critical": 1, "high": 1}},
  "items": [],
  "findings": [
    {"id":"finding-1","tool":"codex","path":"/tmp/config","severity":"high","rule":"mcp_secret_header","message":"new","recommended_action":"externalize"},
    {"id":"finding-2","tool":"cursor","path":"/tmp/mcp.json","severity":"critical","rule":"mcp_secret_env","message":"added","recommended_action":"externalize"}
  ],
  "adapters": []
}`)

	stdout, stderr, code := runCLI([]string{"report", "diff", "--from", beforePath, "--to", afterPath, "--json"})
	if code != 0 {
		t.Fatalf("report diff failed: %s", stderr)
	}
	if !json.Valid([]byte(stdout)) || !strings.Contains(stdout, `"added": 1`) || !strings.Contains(stdout, `"changed": 1`) {
		t.Fatalf("unexpected report diff JSON:\n%s", stdout)
	}

	stdout, stderr, code = runCLI([]string{"report", "diff", "--from", beforePath, "--to", afterPath})
	if code != 0 || !strings.Contains(stdout, "Report diff: added=1 removed=0 changed=1") {
		t.Fatalf("unexpected report diff text stdout=%s stderr=%s", stdout, stderr)
	}

	stdout, stderr, code = runCLI([]string{"report", "history", "--json"})
	if code != 0 {
		t.Fatalf("report history failed: %s", stderr)
	}
	if !json.Valid([]byte(stdout)) || !strings.Contains(stdout, "after.json") || !strings.Contains(stdout, `"highest_severity": "critical"`) {
		t.Fatalf("unexpected history JSON:\n%s", stdout)
	}

	stdout, stderr, code = runCLI([]string{"report", "latest"})
	if code != 0 || !strings.Contains(stdout, "Latest Nightward report:") || !strings.Contains(stdout, "highest:  critical") {
		t.Fatalf("unexpected latest report stdout=%s stderr=%s", stdout, stderr)
	}

	stdout, stderr, code = runCLI([]string{"report", "latest", "--json"})
	if code != 0 || !json.Valid([]byte(stdout)) || !strings.Contains(stdout, `"report_name": "after.json"`) {
		t.Fatalf("unexpected latest report JSON stdout=%s stderr=%s", stdout, stderr)
	}

	stdout, stderr, code = runCLI([]string{"report", "html", "--input", afterPath, "--previous", beforePath, "--output", htmlPath})
	if code != 0 {
		t.Fatalf("report html with previous failed: stdout=%s stderr=%s", stdout, stderr)
	}
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatal(err)
	}
	if html := string(data); !strings.Contains(html, "Changes Since Previous Scan") || !strings.Contains(html, "finding-2") {
		t.Fatalf("expected report diff section:\n%s", html)
	}

	stdout, stderr, code = runCLI([]string{"report", "index", "--output", indexPath})
	if code != 0 {
		t.Fatalf("report index failed: stdout=%s stderr=%s", stdout, stderr)
	}
	data, err = os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if html := string(data); !strings.Contains(html, "Nightward Report History") || !strings.Contains(html, "after.json") {
		t.Fatalf("unexpected report index:\n%s", html)
	}
	if !strings.Contains(string(data), "critical") || !strings.Contains(string(data), "latest") {
		t.Fatalf("expected severity and latest markers in report index:\n%s", string(data))
	}
}

func TestHelpVersionAndCommandErrors(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("help failed: %d %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Usage:") || !strings.Contains(stdout.String(), "Canonical command") {
		t.Fatalf("unexpected help output:\n%s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := RunWithName("nw", []string{"version"}, &stdout, &stderr); code != 0 || strings.TrimSpace(stdout.String()) == "" {
		t.Fatalf("version failed: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	oldVersion := version
	version = "9.8.7-test"
	t.Cleanup(func() { version = oldVersion })
	if code := RunWithName("nightward", []string{"--version"}, &stdout, &stderr); code != 0 || strings.TrimSpace(stdout.String()) != "9.8.7-test" {
		t.Fatalf("version override failed: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := RunWithName("nw", []string{"providers", "bogus"}, &stdout, &stderr); code == 0 || !strings.Contains(stderr.String(), "usage: nightward providers") {
		t.Fatalf("expected providers error, code=%d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := RunWithName("nw", []string{"bogus"}, &stdout, &stderr); code == 0 || !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("expected unknown command error, code=%d stderr=%q", code, stderr.String())
	}
}

func TestScanOutputModesHaveEquivalentSummaries(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, filepath.Join(home, ".mcp.json"), `{"mcpServers":{"demo":{"command":"node","args":["server.js"]}}}`)
	t.Setenv("NIGHTWARD_HOME", home)
	outputFile := filepath.Join(t.TempDir(), "scan.json")
	outputDir := t.TempDir()

	stdoutJSON, stderr, code := runCLI([]string{"scan", "--json"})
	if code != 0 {
		t.Fatalf("scan --json failed: %s", stderr)
	}
	stdoutDash, stderr, code := runCLI([]string{"scan", "--output", "-"})
	if code != 0 {
		t.Fatalf("scan --output - failed: %s", stderr)
	}
	_, stderr, code = runCLI([]string{"scan", "--output", outputFile})
	if code != 0 {
		t.Fatalf("scan --output file failed: %s", stderr)
	}
	_, stderr, code = runCLI([]string{"scan", "--output-dir", outputDir})
	if code != 0 {
		t.Fatalf("scan --output-dir failed: %s", stderr)
	}

	want := scanSummary(t, stdoutJSON)
	if got := scanSummary(t, stdoutDash); got != want {
		t.Fatalf("stdout summary mismatch: got=%s want=%s", got, want)
	}
	fileData, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := scanSummary(t, string(fileData)); got != want {
		t.Fatalf("file summary mismatch: got=%s want=%s", got, want)
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "nightward-scan-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one output-dir report, got %v", matches)
	}
	dirData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if got := scanSummary(t, string(dirData)); got != want {
		t.Fatalf("output-dir summary mismatch: got=%s want=%s", got, want)
	}
}

func TestWorkspaceCommandsAreReadOnlyAndSupportStdoutSARIF(t *testing.T) {
	workspace := t.TempDir()
	writeTestFile(t, filepath.Join(workspace, ".cursor", "mcp.json"), `{"mcpServers":{"demo":{"url":"http://127.0.0.1:8787/mcp","headers":{"Authorization":"Bearer super-secret-value"}}}}`)
	before := listTestFiles(t, workspace)

	commands := [][]string{
		{"scan", "--workspace", workspace, "--json"},
		{"scan", "--workspace", workspace, "--output", "-"},
		{"analyze", "--all", "--workspace", workspace, "--json"},
		{"policy", "check", "--workspace", workspace, "--include-analysis", "--json"},
		{"policy", "badge", "--workspace", workspace, "--include-analysis", "--output", "-"},
		{"policy", "sarif", "--workspace", workspace, "--include-analysis", "--output", "-"},
	}
	for _, args := range commands {
		var stdout, stderr bytes.Buffer
		code := RunWithName("nw", args, &stdout, &stderr)
		if args[0] == "policy" && args[1] == "check" {
			if code != 1 {
				t.Fatalf("expected policy check to report violations, got %d: %s", code, stderr.String())
			}
		} else if code != 0 {
			t.Fatalf("%s failed with %d: %s", strings.Join(args, " "), code, stderr.String())
		}
		if stdout.Len() == 0 {
			t.Fatalf("%s produced no stdout", strings.Join(args, " "))
		}
	}
	after := listTestFiles(t, workspace)
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatalf("workspace commands mutated files\nbefore=%v\nafter=%v", before, after)
	}
}

func TestPolicyBadgeWritesArtifactWithoutFailingPolicy(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, filepath.Join(home, ".mcp.json"), `{"mcpServers":{"demo":{"command":"npx","args":["@example/server"]}}}`)
	t.Setenv("NIGHTWARD_HOME", home)
	out := filepath.Join(home, "reports", "badge.json")

	stdout, stderr, code := runCLI([]string{"policy", "badge", "--output", out, "--sarif-url", "https://example.invalid/nightward.sarif"})
	if code != 0 {
		t.Fatalf("policy badge failed: stdout=%s stderr=%s", stdout, stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var badge struct {
		SchemaVersion int    `json:"schemaVersion"`
		Passed        bool   `json:"passed"`
		Message       string `json:"message"`
		SARIFURL      string `json:"sarif_url"`
	}
	if err := json.Unmarshal(data, &badge); err != nil {
		t.Fatal(err)
	}
	if badge.SchemaVersion != 1 || badge.Passed || !strings.Contains(badge.Message, "violations") || badge.SARIFURL == "" {
		t.Fatalf("unexpected badge artifact: %#v\n%s", badge, data)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("expected badge mode 0600, got %o", got)
	}
}

func TestPolicyBadgeDoesNotRunConfiguredProvidersUnlessAnalysisIncluded(t *testing.T) {
	workspace := t.TempDir()
	writeTestFile(t, filepath.Join(workspace, "config.txt"), "API_TOKEN=super-secret-value\n")
	configPath := filepath.Join(workspace, ".nightward.yml")
	writeTestFile(t, configPath, "analysis_providers:\n  - gitleaks\n")
	binDir := t.TempDir()
	writeTestFile(t, filepath.Join(binDir, "gitleaks"), `#!/bin/sh
printf '[{"RuleID":"generic-api-key","Description":"API_TOKEN=super-secret-value","File":"config.txt","StartLine":1}]'
`)
	if err := os.Chmod(filepath.Join(binDir, "gitleaks"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	stdout, stderr, code := runCLI([]string{"policy", "badge", "--workspace", workspace, "--config", configPath, "--output", "-"})
	if code != 0 {
		t.Fatalf("policy badge failed: %s", stderr)
	}
	if strings.Contains(stdout, "generic-api-key") || strings.Contains(stdout, "super-secret-value") {
		t.Fatalf("badge ran providers without include-analysis:\n%s", stdout)
	}

	stdout, stderr, code = runCLI([]string{"policy", "badge", "--workspace", workspace, "--config", configPath, "--include-analysis", "--output", "-"})
	if code != 0 {
		t.Fatalf("policy badge with analysis failed: %s", stderr)
	}
	assertNoSecret(t, stdout, "super-secret-value")
	if !strings.Contains(stdout, `"total_signals": 1`) || !strings.Contains(stdout, `"signal_violations": 1`) {
		t.Fatalf("badge did not include provider signals when requested:\n%s", stdout)
	}
}

func TestAnalyzeWithExplicitProviderIsReadOnly(t *testing.T) {
	workspace := t.TempDir()
	writeTestFile(t, filepath.Join(workspace, "config.txt"), "API_TOKEN=super-secret-value\n")
	writeTestFile(t, filepath.Join(workspace, "semgrep.yml"), "rules: []\n")
	binDir := t.TempDir()
	writeTestFile(t, filepath.Join(binDir, "gitleaks"), `#!/bin/sh
printf '[{"RuleID":"generic-api-key","Description":"API_TOKEN=super-secret-value","File":"config.txt","StartLine":1}]'
`)
	if err := os.Chmod(filepath.Join(binDir, "gitleaks"), 0700); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(binDir, "trufflehog"), `#!/bin/sh
printf '{"DetectorName":"GitHub","Verified":true,"SourceMetadata":{"Data":{"Filesystem":{"file":"config.txt"}}}}\n'
`)
	if err := os.Chmod(filepath.Join(binDir, "trufflehog"), 0700); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(binDir, "semgrep"), `#!/bin/sh
printf '{"results":[{"check_id":"nightward.fixture","path":"config.txt","extra":{"message":"API_TOKEN=super-secret-value","severity":"WARNING"}}]}'
`)
	if err := os.Chmod(filepath.Join(binDir, "semgrep"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	before := listTestFiles(t, workspace)

	stdout, stderr, code := runCLI([]string{"analyze", "--all", "--workspace", workspace, "--with", "gitleaks,trufflehog,semgrep", "--json"})
	if code != 0 {
		t.Fatalf("analyze with provider failed: %s", stderr)
	}
	if !json.Valid([]byte(stdout)) {
		t.Fatalf("analyze with provider did not emit JSON: %s", stdout)
	}
	assertNoSecret(t, stdout, "super-secret-value")
	for _, want := range []string{"generic-api-key", "GitHub", "nightward.fixture"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("provider finding %q missing from analysis output: %s", want, stdout)
		}
	}

	after := listTestFiles(t, workspace)
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatalf("provider analyze mutated workspace\nbefore=%v\nafter=%v", before, after)
	}
}

func TestAnalyzeWithOnlineProvidersRequiresExplicitOnline(t *testing.T) {
	workspace := t.TempDir()
	writeTestFile(t, filepath.Join(workspace, "package.json"), `{"dependencies":{"demo":"1.0.0"}}`)
	binDir := t.TempDir()
	writeTestFile(t, filepath.Join(binDir, "trivy"), `#!/bin/sh
if [ "$PWD" != "$7" ] || [ "$1" != "filesystem" ] || [ "$2" != "--format" ] || [ "$3" != "json" ] || [ "$4" != "--scanners" ] || [ "$5" != "vuln,secret,misconfig" ] || [ "$6" != "--skip-version-check" ]; then
  echo "unexpected trivy args: $*" >&2
  exit 2
fi
printf '{"Results":[{"Target":"package.json","Vulnerabilities":[{"VulnerabilityID":"CVE-2026-0001","PkgName":"demo","Severity":"HIGH","Title":"demo vulnerable"}]}]}'
`)
	if err := os.Chmod(filepath.Join(binDir, "trivy"), 0700); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(binDir, "osv-scanner"), `#!/bin/sh
if [ "$PWD" != "$6" ] || [ "$1" != "scan" ] || [ "$2" != "source" ] || [ "$3" != "-r" ] || [ "$4" != "--format" ] || [ "$5" != "json" ]; then
  echo "unexpected osv-scanner args: $*" >&2
  exit 2
fi
printf '{"results":[{"source":"package.json","packages":[{"package":{"name":"demo"},"vulnerabilities":[{"id":"GHSA-123","summary":"bad package"}]}]}]}'
`)
	if err := os.Chmod(filepath.Join(binDir, "osv-scanner"), 0700); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(binDir, "socket"), `#!/bin/sh
if [ "$PWD" != "$3" ] || [ "$1" != "scan" ] || [ "$2" != "create" ] || [ "$4" != "--json" ]; then
  echo "unexpected socket args: $*" >&2
  exit 2
fi
printf '{"issues":[{"type":"malware","severity":"high","message":"API_TOKEN=super-secret-value","package":"demo","file":"package.json"}]}'
`)
	if err := os.Chmod(filepath.Join(binDir, "socket"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	before := listTestFiles(t, workspace)

	blocked, stderr, code := runCLI([]string{"analyze", "--all", "--workspace", workspace, "--with", "trivy,osv-scanner,socket", "--json"})
	if code != 0 {
		t.Fatalf("blocked online-provider posture failed: %s", stderr)
	}
	if strings.Contains(blocked, "CVE-2026-0001") || strings.Contains(blocked, "GHSA-123") || strings.Contains(blocked, "malware") {
		t.Fatalf("online providers ran without --online:\n%s", blocked)
	}

	stdout, stderr, code := runCLI([]string{"analyze", "--all", "--workspace", workspace, "--with", "trivy,osv-scanner,socket", "--online", "--json"})
	if code != 0 {
		t.Fatalf("online provider analyze failed: %s", stderr)
	}
	assertNoSecret(t, stdout, "super-secret-value")
	for _, want := range []string{"CVE-2026-0001", "GHSA-123", "malware"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("online provider finding %q missing from analysis output: %s", want, stdout)
		}
	}
	after := listTestFiles(t, workspace)
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatalf("online provider analyze mutated workspace\nbefore=%v\nafter=%v", before, after)
	}
}

func TestWorkspaceScanDoesNotReadHomeFixtures(t *testing.T) {
	home := t.TempDir()
	workspace := t.TempDir()
	writeTestFile(t, filepath.Join(home, ".codex", "config.toml"), `[mcp_servers.home_only]
command = "npx"
args = ["-y", "@example/home-only"]
`)
	writeTestFile(t, filepath.Join(workspace, ".cursor", "mcp.json"), `{"mcpServers":{"workspaceOnly":{"url":"https://mcp.example.test/server"}}}`)
	t.Setenv("NIGHTWARD_HOME", home)

	stdout, stderr, code := runCLI([]string{"scan", "--workspace", workspace, "--json"})
	if code != 0 {
		t.Fatalf("workspace scan failed: %s", stderr)
	}
	if strings.Contains(stdout, "home_only") || strings.Contains(stdout, filepath.Join(home, ".codex")) {
		t.Fatalf("workspace scan included HOME-only config:\n%s", stdout)
	}
	if !strings.Contains(stdout, "workspaceOnly") {
		t.Fatalf("workspace scan missed workspace config:\n%s", stdout)
	}
}

func TestFixPlanFindingSelectorRequiresUniquePrefix(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, filepath.Join(home, ".mcp.json"), `{"mcpServers":{"one":{"command":"node","args":["one.js"]},"two":{"command":"node","args":["two.js"]}}}`)
	t.Setenv("NIGHTWARD_HOME", home)

	var stdout, stderr bytes.Buffer
	code := RunWithName("nw", []string{"fix", "plan", "--finding", "mcp_server_review", "--json"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected ambiguous finding prefix to fail, stdout=%s stderr=%s", stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = RunWithName("nw", []string{"findings", "list", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("findings list failed: %s", stderr.String())
	}
	var findings []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &findings); err != nil {
		t.Fatal(err)
	}
	if len(findings) == 0 {
		t.Fatal("expected findings")
	}

	stdout.Reset()
	stderr.Reset()
	code = RunWithName("nw", []string{"fix", "plan", "--finding", findings[0].ID, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exact finding selector failed: %s", stderr.String())
	}
}

func TestPolicyCheckUsesConfig(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, filepath.Join(home, ".mcp.json"), `{"mcpServers":{"demo":{"command":"npx","args":["@example/server"]}}}`)
	config := filepath.Join(home, ".nightward.yml")
	writeTestFile(t, config, "severity_threshold: medium\ntrusted_packages:\n  - '@example/server'\n")
	t.Setenv("NIGHTWARD_HOME", home)

	var stdout, stderr bytes.Buffer
	code := RunWithName("nw", []string{"policy", "check", "--config", config, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("policy check with config failed: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"ignored": 1`) {
		t.Fatalf("expected ignored finding in output: %s", stdout.String())
	}
}

func TestHumanPrintersCoverPolicySnapshotAndSchedule(t *testing.T) {
	var out bytes.Buffer
	printPolicy(&out, policy.Report{
		Strict:    true,
		Passed:    false,
		Threshold: inventory.RiskMedium,
		Summary:   policy.Summary{TotalFindings: 2, Violations: 1, Ignored: 1},
		Violations: []inventory.Finding{
			{ID: "mcp_secret_env-1", Severity: inventory.RiskCritical, Rule: "mcp_secret_env", Message: "Secret env"},
		},
	})
	if text := out.String(); !strings.Contains(text, "policy failed") || !strings.Contains(text, "mcp_secret_env") {
		t.Fatalf("unexpected policy printer output:\n%s", text)
	}

	out.Reset()
	printSnapshotPlan(&out, snapshot.Plan{
		TargetRoot: "/tmp/snapshots",
		Summary:    snapshot.Summary{Total: 1, Include: 1},
		Entries: []snapshot.Entry{
			{Source: "/tmp/config.toml", Target: "/tmp/snapshots/config.toml", Tool: "Codex", Action: backupplan.ActionInclude},
		},
	})
	if text := out.String(); !strings.Contains(text, "Snapshot dry-run plan") || !strings.Contains(text, "config.toml") {
		t.Fatalf("unexpected snapshot plan output:\n%s", text)
	}

	out.Reset()
	printSnapshotDiff(&out, snapshot.Diff{
		Summary: snapshot.DiffSummary{Added: 1, Removed: 1, Changed: 1},
		Added:   []snapshot.Entry{{Source: "/tmp/new", Tool: "Codex"}},
		Removed: []snapshot.Entry{{Source: "/tmp/old", Tool: "Claude"}},
		Changed: []snapshot.Change{{Source: "/tmp/config", Before: snapshot.Entry{Action: backupplan.ActionReview}, After: snapshot.Entry{Tool: "Cursor", Action: backupplan.ActionInclude}}},
	})
	if text := out.String(); !strings.Contains(text, "added") || !strings.Contains(text, "removed") || !strings.Contains(text, "changed") {
		t.Fatalf("unexpected snapshot diff output:\n%s", text)
	}

	now := time.Date(2026, 4, 30, 2, 17, 0, 0, time.UTC)
	next := now.Add(-24 * time.Hour)
	out.Reset()
	printSchedulePlan(&out, schedule.Plan{
		Preset:     "nightly",
		Platform:   "darwin",
		Command:    []string{"nw", "scan"},
		ReportDir:  "/tmp/reports",
		LastReport: "/tmp/reports/latest.json",
		LastRun:    &now,
		Files:      []schedule.GeneratedFile{{Path: "/tmp/agent.plist", Content: "<plist/>", Mode: 0644}},
		Notes:      []string{"dry run"},
	})
	if text := out.String(); !strings.Contains(text, "Schedule preset") || !strings.Contains(text, "agent.plist") || !strings.Contains(text, "dry run") {
		t.Fatalf("unexpected schedule output:\n%s", text)
	}

	out.Reset()
	printReportHistory(&out, []schedule.ReportRecord{
		{Path: "/tmp/latest.json", ModTime: now, Findings: 4, ReportName: "latest.json"},
		{Path: "/tmp/previous.json", ModTime: next, Findings: 2, ReportName: "previous.json"},
	})
	if text := out.String(); !strings.Contains(text, "latest") || !strings.Contains(text, "-2 vs newer") {
		t.Fatalf("unexpected report history output:\n%s", text)
	}
	out.Reset()
	printReportHistory(&out, nil)
	if !strings.Contains(out.String(), "No Nightward reports found") {
		t.Fatalf("expected empty report history output: %s", out.String())
	}

	adapter := inventory.AdapterStatus{
		Name:        "Codex",
		Description: "Codex configs",
		Checked:     []string{filepath.Join("/tmp/home", ".codex", "config.toml"), "/outside/config.toml"},
	}
	template := buildAdapterFixtureTemplate("/tmp/home", "", adapter)
	if template.SchemaVersion != 1 || !strings.Contains(strings.Join(template.Paths, "\n"), filepath.Join("testdata", "homes", "codex", ".codex", "config.toml")) {
		t.Fatalf("unexpected adapter fixture template: %#v", template)
	}
	out.Reset()
	printAdapterFixtureTemplate(&out, template)
	if !strings.Contains(out.String(), "Nightward adapter fixture") {
		t.Fatalf("unexpected adapter template text:\n%s", out.String())
	}
	if _, ok := findAdapter([]inventory.AdapterStatus{{Name: "Codex"}, {Name: "Cursor"}}, "c"); ok {
		t.Fatal("expected ambiguous adapter prefix")
	}
}

func TestHumanAnalysisSignalPrinter(t *testing.T) {
	var out bytes.Buffer
	printSignal(&out, analysis.Signal{
		ID:             "signal-1",
		Rule:           "nightward/mcp_secret_env",
		Severity:       inventory.RiskCritical,
		Confidence:     "high",
		Provider:       "nightward",
		Path:           "/tmp/config.toml",
		Evidence:       "env_key=API_TOKEN",
		Recommendation: "Externalize the secret.",
		Why:            "Secrets should not live in portable config.",
	})
	if text := out.String(); !strings.Contains(text, "signal-1") || !strings.Contains(text, "Externalize") || !strings.Contains(text, "Why this matters") {
		t.Fatalf("unexpected signal output:\n%s", text)
	}
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
}

func listTestFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func listOptionalTestFiles(t *testing.T, root string) []string {
	t.Helper()
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil
	}
	return listTestFiles(t, root)
}

func runCLI(args []string) (string, string, int) {
	var stdout, stderr bytes.Buffer
	code := RunWithName("nw", args, &stdout, &stderr)
	return stdout.String(), stderr.String(), code
}

func firstFindingID(t *testing.T, args []string) string {
	t.Helper()
	stdout, stderr, code := runCLI(args)
	if code != 0 {
		t.Fatalf("%s failed: %s", strings.Join(args, " "), stderr)
	}
	var findings []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(stdout), &findings); err != nil {
		t.Fatal(err)
	}
	if len(findings) == 0 {
		t.Fatal("expected at least one finding")
	}
	return findings[0].ID
}

func assertNoSecret(t *testing.T, output, secret string) {
	t.Helper()
	if strings.Contains(output, secret) {
		t.Fatalf("output leaked secret %q:\n%s", secret, output)
	}
}

func scanSummary(t *testing.T, data string) string {
	t.Helper()
	var decoded struct {
		Summary json.RawMessage `json:"summary"`
	}
	if err := json.Unmarshal([]byte(data), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Summary) == 0 {
		t.Fatalf("missing summary in %s", data)
	}
	return string(decoded.Summary)
}
