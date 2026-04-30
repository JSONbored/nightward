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
