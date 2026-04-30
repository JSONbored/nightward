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
	if report.Summary.ItemsByClassification[Portable] != 1 {
		t.Fatalf("expected portable item summary, got %#v", report.Summary.ItemsByClassification)
	}
	if report.Summary.FindingsByRule["mcp_secret_env"] != 2 {
		t.Fatalf("expected finding rule summary, got %#v", report.Summary.FindingsByRule)
	}
	if report.Summary.FindingsBySeverity[RiskCritical] != 1 || report.Summary.FindingsBySeverity[RiskMedium] < 1 {
		t.Fatalf("expected finding severity summary, got %#v", report.Summary.FindingsBySeverity)
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

func TestScannerHandlesRemoteURLMCPShapesAndHeaders(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".codex", "config.toml"), `
[mcp_servers.remote]
url = "https://mcp.example.test/sse"

[mcp_servers.local]
url = "http://127.0.0.1:8787/mcp"

[mcp_servers.header]
type = "sse"
url = "https://headers.example.test/mcp"

[mcp_servers.header.headers]
Authorization = "Bearer super-header-secret"
X-API-Key = "${MCP_API_KEY}"

[mcp_servers.unknown]
type = "custom"
`)
	writeFile(t, filepath.Join(home, ".claude.json"), `{
  "mcpServers": {
    "context7": {
      "type": "sse",
      "url": "https://context.example.test/mcp",
      "headers": {
        "Authorization": "${CONTEXT_TOKEN}"
      }
    }
  }
}`)
	writeFile(t, filepath.Join(home, ".cursor", "mcp.json"), `{
  "mcpServers": {
    "socket-mcp": {
      "type": "streamable-http",
      "url": "https://socket.example.test/mcp"
    }
  }
}`)

	report := NewScanner(home).Scan()
	ruleCounts := map[string]int{}
	for _, finding := range report.Findings {
		ruleCounts[finding.Rule]++
		if finding.Server != "unknown" && strings.Contains(finding.Evidence, "super-header-secret") {
			t.Fatalf("finding leaked header value: %#v", finding)
		}
		if finding.Rule == "mcp_unknown_command" && finding.Server != "unknown" {
			t.Fatalf("URL-shaped server produced unknown-command finding: %#v", finding)
		}
	}

	if ruleCounts["mcp_secret_header"] != 3 {
		t.Fatalf("expected three header findings, got %#v", ruleCounts)
	}
	if ruleCounts["mcp_local_endpoint"] != 1 {
		t.Fatalf("expected one local endpoint finding, got %#v", ruleCounts)
	}
	if ruleCounts["mcp_unknown_command"] != 1 {
		t.Fatalf("expected only malformed server to be unknown, got %#v", ruleCounts)
	}
	if report.Summary.FindingsByRule["mcp_secret_header"] != 3 || report.Summary.FindingsByTool["Codex"] == 0 {
		t.Fatalf("summary did not include new finding buckets: %#v", report.Summary)
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, "super-header-secret") {
		t.Fatalf("scan report leaked header value: %s", text)
	}
	for _, want := range []string{"mcp_secret_header", "header_key=Authorization", "transport=remote-url", "url=https://mcp.example.test"} {
		if !strings.Contains(text, want) {
			t.Fatalf("scan report missing %q: %s", want, text)
		}
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

func TestScannerFindsExpandedAdaptersWithConservativeClassifications(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".config", "zed", "settings.json"), `{}`)
	writeFile(t, filepath.Join(home, ".continue", "config.json"), `{"mcpServers":{"demo":{"command":"node","args":["server.js"]}}}`)
	writeFile(t, filepath.Join(home, ".aider.conf.yml"), "model: sonnet\n")
	writeFile(t, filepath.Join(home, ".ollama", "id_ed25519"), "PRIVATE KEY")
	writeFile(t, filepath.Join(home, ".config", "nvim", "init.lua"), "-- config\n")

	report := NewScanner(home).Scan()
	found := map[string]Item{}
	for _, item := range report.Items {
		found[item.Tool+":"+item.Path] = item
	}

	assertClass := func(tool, rel string, class Classification) {
		t.Helper()
		path := filepath.Join(home, rel)
		item, ok := found[tool+":"+path]
		if !ok {
			t.Fatalf("missing %s item for %s", tool, path)
		}
		if item.Classification != class {
			t.Fatalf("%s classified as %s, want %s", path, item.Classification, class)
		}
	}

	assertClass("Zed", filepath.Join(".config", "zed", "settings.json"), Portable)
	assertClass("Continue", filepath.Join(".continue", "config.json"), Portable)
	assertClass("Aider", ".aider.conf.yml", Portable)
	assertClass("Ollama/Open WebUI", filepath.Join(".ollama", "id_ed25519"), SecretAuth)
	assertClass("Neovim", filepath.Join(".config", "nvim"), Portable)
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
