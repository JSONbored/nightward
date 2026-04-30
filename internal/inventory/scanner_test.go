package inventory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestScannerFindsMCPRisksAndRedactsValues(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".codex", "config.toml"), `
[mcp_servers.filesystem]
command = "npx"
args = ["-y", "@modelcontextprotocol/server-filesystem", "/Users/example"]

[mcp_servers.filesystem.env]
API_TOKEN = "super-secret-value"

[mcp_servers.reference.env]
GITHUB_TOKEN = "${GITHUB_TOKEN}"
`)

	scanner := NewScanner(home)
	scanner.Now = func() time.Time { return time.Date(2026, 4, 30, 2, 17, 0, 0, time.UTC) }
	report := scanner.Scan()

	if report.Summary.TotalItems != 1 {
		t.Fatalf("expected 1 item, got %d", report.Summary.TotalItems)
	}
	if report.Items[0].Classification != Portable {
		t.Fatalf("expected portable Codex config, got %s", report.Items[0].Classification)
	}

	rules := map[string]bool{}
	for _, finding := range report.Findings {
		rules[finding.Rule] = true
	}
	for _, rule := range []string{"mcp_server_review", "mcp_unpinned_package", "mcp_secret_env", "mcp_broad_filesystem"} {
		if !rules[rule] {
			t.Fatalf("expected finding rule %s in %#v", rule, rules)
		}
	}
	var inlineSecret, envReference bool
	for _, finding := range report.Findings {
		if finding.Rule != "mcp_secret_env" {
			continue
		}
		if !finding.FixAvailable || finding.FixKind != FixExternalizeSecret {
			t.Fatalf("secret finding missing externalize-secret plan: %#v", finding)
		}
		if strings.Contains(finding.Evidence, "API_TOKEN") && finding.Severity == RiskCritical {
			inlineSecret = true
		}
		if strings.Contains(finding.Evidence, "GITHUB_TOKEN") && finding.Severity == RiskMedium {
			envReference = true
		}
	}
	if !inlineSecret {
		t.Fatal("expected inline secret env to be critical")
	}
	if !envReference {
		t.Fatal("expected env reference to be medium guidance")
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "super-secret-value") {
		t.Fatal("scan report leaked an env secret value")
	}
}

func TestScannerRedactsSecretArgumentValues(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".mcp.json"), `{
  "mcpServers": {
    "leaky": {
      "command": "bash",
      "args": ["-c", "node server.js --api-key super-secret-value --token=another-secret"]
    }
  }
}`)

	report := NewScanner(home).Scan()
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, leaked := range []string{"super-secret-value", "another-secret"} {
		if strings.Contains(text, leaked) {
			t.Fatalf("scan report leaked secret argument value %q: %s", leaked, text)
		}
	}
	if !strings.Contains(text, "[redacted]") {
		t.Fatalf("expected redacted evidence in scan report: %s", text)
	}
}

func TestScannerDoesNotWriteToHome(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".mcp.json")
	writeFile(t, path, `{"mcpServers":{"demo":{"command":"node","args":["server.js"]}}}`)

	before := listFiles(t, home)
	_ = NewScanner(home).Scan()
	after := listFiles(t, home)

	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatalf("scan mutated home\nbefore=%v\nafter=%v", before, after)
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
}

func listFiles(t *testing.T, root string) []string {
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
