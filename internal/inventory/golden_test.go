package inventory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGoldenScanSummaryAndURLFindings(t *testing.T) {
	home := filepath.Clean("/tmp/nightward-golden-home")
	if err := os.RemoveAll(home); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(home) })

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

	scanner := NewScanner(home)
	scanner.Hostname = "fixture-host"
	scanner.Now = func() time.Time { return time.Date(2026, 4, 30, 7, 0, 0, 0, time.UTC) }
	report := scanner.Scan()

	assertGoldenJSON(t, "testdata/golden/scan-summary.golden.json", report.Summary)
	assertGoldenJSON(t, "testdata/golden/url-findings.golden.json", report.Findings)

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "super-header-secret") {
		t.Fatal("golden fixture scan leaked header secret")
	}
}

func assertGoldenJSON(t *testing.T, path string, value any) {
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
