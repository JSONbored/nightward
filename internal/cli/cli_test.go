package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		{"findings", "list", "--json"},
		{"fix", "plan", "--all", "--json"},
		{"fix", "preview", "--all", "--format", "json"},
		{"fix", "export", "--format", "markdown"},
		{"analyze", "--all", "--json"},
		{"providers", "list", "--json"},
		{"providers", "doctor", "--json"},
		{"policy", "check", "--json"},
		{"policy", "check", "--include-analysis", "--json"},
		{"policy", "init", "--dry-run"},
		{"policy", "explain"},
		{"snapshot", "plan", "--target", filepath.Join(home, "snapshot"), "--json"},
		{"policy", "sarif", "--output", filepath.Join(outputDir, "nightward.sarif")},
		{"policy", "sarif", "--include-analysis", "--output", "-"},
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
		{"findings", "list", "--json"},
		{"findings", "explain", "--json", findingID},
		{"fix", "plan", "--all", "--json"},
		{"fix", "preview", "--all", "--format", "json"},
		{"analyze", "--all", "--json"},
		{"analyze", "finding", "--json", findingID},
		{"trust", "explain", "--json", findingID},
		{"providers", "list", "--json"},
		{"providers", "doctor", "--json"},
		{"policy", "check", "--json"},
		{"policy", "sarif", "--output", "-"},
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
