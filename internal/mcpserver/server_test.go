package mcpserver

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jsonbored/nightward/internal/inventory"
)

func TestServeInitializeToolsAndResources(t *testing.T) {
	home := fixtureHome(t)
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"resources/list"}`,
	}, "\n") + "\n"
	var out bytes.Buffer
	server := Server{Home: home, Version: "0.1.99", Now: fixedNow}
	if err := server.Serve(strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, out.String())
	if len(responses) != 3 {
		t.Fatalf("expected 3 responses because initialized is a notification, got %d: %s", len(responses), out.String())
	}
	init := responses[0]["result"].(map[string]any)
	if got := init["protocolVersion"]; got != protocolVersion {
		t.Fatalf("unexpected protocol version: %v", got)
	}
	if !strings.Contains(init["instructions"].(string), "read-only") {
		t.Fatalf("missing read-only instructions: %v", init["instructions"])
	}
	if tools := responses[1]["result"].(map[string]any)["tools"].([]any); len(tools) < 7 {
		t.Fatalf("expected Nightward tools, got %#v", tools)
	}
	if resources := responses[2]["result"].(map[string]any)["resources"].([]any); len(resources) < 4 {
		t.Fatalf("expected Nightward resources, got %#v", resources)
	}
}

func TestToolCallsAreRedactedAndReadOnly(t *testing.T) {
	home := fixtureHome(t)
	before := listFiles(t, home)
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"scan","method":"tools/call","params":{"name":"nightward_scan","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":"findings","method":"tools/call","params":{"name":"nightward_findings","arguments":{"severity":"high"}}}`,
		`{"jsonrpc":"2.0","id":"fix","method":"tools/call","params":{"name":"nightward_fix_plan","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":"policy","method":"tools/call","params":{"name":"nightward_policy_check","arguments":{"strict":true,"include_analysis":true}}}`,
		`{"jsonrpc":"2.0","id":"bad","method":"tools/call","params":{"name":"nightward_missing","arguments":{}}}`,
	}, "\n") + "\n"
	var out bytes.Buffer
	server := Server{Home: home, Version: "test", Now: fixedNow}
	if err := server.Serve(strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	assertScrubbed(t, text)
	responses := decodeResponses(t, text)
	if len(responses) != 5 {
		t.Fatalf("expected 5 responses, got %d", len(responses))
	}
	for i, response := range responses[:4] {
		result := response["result"].(map[string]any)
		if result["isError"] == true {
			t.Fatalf("response %d unexpectedly errored: %#v", i, result)
		}
		if len(result["content"].([]any)) == 0 {
			t.Fatalf("response %d missing content", i)
		}
	}
	if result := responses[4]["result"].(map[string]any); result["isError"] != true {
		t.Fatalf("expected unknown tool to return tool error result, got %#v", result)
	}
	after := listFiles(t, home)
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatalf("MCP tool calls mutated fixture home\nbefore=%v\nafter=%v", before, after)
	}
}

func TestResourceReadLatestReportAndRules(t *testing.T) {
	home := t.TempDir()
	reportsDir := filepath.Join(home, ".local", "state", "nightward", "reports")
	writeFile(t, filepath.Join(reportsDir, "latest.json"), `{
  "schema_version": 1,
  "generated_at": "2026-05-01T00:00:00Z",
  "hostname": "fixture",
  "home": "`+home+`",
  "summary": {"total_findings": 0},
  "items": [],
  "findings": [],
  "adapters": []
}`)
	var out bytes.Buffer
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"nightward://latest-report"}}`,
		`{"jsonrpc":"2.0","id":2,"method":"resources/read","params":{"uri":"nightward://rules"}}`,
	}, "\n") + "\n"
	if err := (Server{Home: home, Version: "test", Now: fixedNow}).Serve(strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, out.String())
	for _, response := range responses {
		result := response["result"].(map[string]any)
		contents := result["contents"].([]any)
		if len(contents) != 1 {
			t.Fatalf("unexpected contents: %#v", contents)
		}
		if text := contents[0].(map[string]any)["text"].(string); !json.Valid([]byte(text)) {
			t.Fatalf("resource text is not JSON: %s", text)
		}
	}
}

func TestDoctorResourcesAndMissingLatestReport(t *testing.T) {
	home := fixtureHome(t)
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nightward_doctor","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"resources/read","params":{"uri":"nightward://providers"}}`,
		`{"jsonrpc":"2.0","id":3,"method":"resources/read","params":{"uri":"nightward://schedule"}}`,
		`{"jsonrpc":"2.0","id":4,"method":"resources/read","params":{"uri":"nightward://latest-report"}}`,
	}, "\n") + "\n"
	var out bytes.Buffer
	if err := Serve(home, "", strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, out.String())
	if len(responses) != 4 {
		t.Fatalf("expected 4 responses, got %d", len(responses))
	}
	doctor := toolJSON(t, responses[0])
	if doctor["version"] != "devel" {
		t.Fatalf("expected default version devel, got %#v", doctor["version"])
	}
	if doctor["home"] != home {
		t.Fatalf("expected doctor home %q, got %#v", home, doctor["home"])
	}
	for _, response := range responses[1:] {
		if text := resourceText(t, response); !json.Valid([]byte(text)) {
			t.Fatalf("resource text is not JSON: %s", text)
		}
	}
	latest := resourceJSON(t, responses[3])
	if latest["available"] != false {
		t.Fatalf("expected latest-report unavailable, got %#v", latest)
	}
}

func TestFindingExplainReportChangesAndPolicyTool(t *testing.T) {
	home := fixtureHome(t)
	server := Server{Home: home, Version: "test", Now: fixedNow}
	report := server.scan("")
	if len(report.Findings) == 0 {
		t.Fatal("fixture did not produce findings")
	}
	reportsDir := defaultReportDir(home)
	writeReportFile(t, filepath.Join(reportsDir, "older.json"), inventory.Report{
		SchemaVersion: inventory.ReportSchemaVersion,
		GeneratedAt:   fixedNow().Add(-time.Hour),
		Home:          home,
		Summary:       inventory.Summary{TotalFindings: 1},
		Findings: []inventory.Finding{{
			ID:             "same-finding",
			Tool:           "Codex",
			Path:           filepath.Join(home, ".codex", "config.toml"),
			Severity:       inventory.RiskLow,
			Rule:           "mcp_server_review",
			Message:        "old message",
			Recommendation: "review",
		}},
	})
	writeReportFile(t, filepath.Join(reportsDir, "newer.json"), inventory.Report{
		SchemaVersion: inventory.ReportSchemaVersion,
		GeneratedAt:   fixedNow(),
		Home:          home,
		Summary:       inventory.Summary{TotalFindings: 2},
		Findings: []inventory.Finding{
			{
				ID:             "same-finding",
				Tool:           "Codex",
				Path:           filepath.Join(home, ".codex", "config.toml"),
				Severity:       inventory.RiskHigh,
				Rule:           "mcp_server_review",
				Message:        "new message",
				Recommendation: "review",
			},
			{
				ID:             "added-finding",
				Tool:           "Codex",
				Path:           filepath.Join(home, ".codex", "config.toml"),
				Severity:       inventory.RiskMedium,
				Rule:           "mcp_secret_env",
				Message:        "added message",
				Recommendation: "externalize",
			},
		},
	})
	olderTime := fixedNow().Add(-time.Hour)
	newerTime := fixedNow()
	if err := os.Chtimes(filepath.Join(reportsDir, "older.json"), olderTime, olderTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(reportsDir, "newer.json"), newerTime, newerTime); err != nil {
		t.Fatal(err)
	}

	prefix := report.Findings[0].ID
	if len(prefix) > 12 {
		prefix = prefix[:12]
	}
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"filter","method":"tools/call","params":{"name":"nightward_findings","arguments":{"severity":"high","search":"package"}}}`,
		`{"jsonrpc":"2.0","id":"explain","method":"tools/call","params":{"name":"nightward_explain_finding","arguments":{"finding_id":"` + prefix + `"}}}`,
		`{"jsonrpc":"2.0","id":"missing","method":"tools/call","params":{"name":"nightward_explain_finding","arguments":{"finding_id":"not-a-finding"}}}`,
		`{"jsonrpc":"2.0","id":"required","method":"tools/call","params":{"name":"nightward_explain_finding","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":"diff","method":"tools/call","params":{"name":"nightward_report_changes","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":"policy","method":"tools/call","params":{"name":"nightward_policy_check","arguments":{"strict":true,"include_analysis":true,"providers":"missing-provider"}}}`,
	}, "\n") + "\n"
	var out bytes.Buffer
	if err := server.Serve(strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	assertScrubbed(t, out.String())
	responses := decodeResponses(t, out.String())
	filtered := toolJSONArray(t, responses[0])
	if len(filtered) == 0 {
		t.Fatalf("expected filtered findings, got %#v", filtered)
	}
	explained := toolJSON(t, responses[1])
	if explained["found"] != true {
		t.Fatalf("expected finding explanation, got %#v", explained)
	}
	missing := toolJSON(t, responses[2])
	if missing["found"] != false {
		t.Fatalf("expected missing finding response, got %#v", missing)
	}
	if result := responses[3]["result"].(map[string]any); result["isError"] != true {
		t.Fatalf("expected missing finding_id to return tool error, got %#v", result)
	}
	diff := toolJSON(t, responses[4])
	if diff["available"] != true {
		t.Fatalf("expected report diff, got %#v", diff)
	}
	policyReport := toolJSON(t, responses[5])
	if policyReport["schema_version"] == nil {
		t.Fatalf("expected policy report JSON, got %#v", policyReport)
	}
}

func TestResourceAndToolErrorPaths(t *testing.T) {
	server := Server{Home: t.TempDir(), Version: "test", Now: fixedNow}
	cases := []request{
		{ID: json.RawMessage(`1`), Method: "resources/read", Params: json.RawMessage(`{}`)},
		{ID: json.RawMessage(`2`), Method: "resources/read", Params: json.RawMessage(`{"uri":"nightward://missing"}`)},
		{ID: json.RawMessage(`3`), Method: "tools/call", Params: json.RawMessage(`{not-json}`)},
	}
	for _, req := range cases {
		resp, ok := server.Handle(req)
		if !ok {
			t.Fatalf("expected response for %s", req.Method)
		}
		if resp.Error == nil && resp.Result == nil {
			t.Fatalf("expected error or tool error result for %#v", req)
		}
	}
}

func TestBoundsAndArgumentHelpers(t *testing.T) {
	if got := strings.Join(stringListArg(map[string]any{"providers": "gitleaks, semgrep, "}, "providers"), ","); got != "gitleaks,semgrep" {
		t.Fatalf("unexpected CSV providers: %q", got)
	}
	if got := strings.Join(stringListArg(map[string]any{"providers": []any{"trivy", 12, "socket", ""}}, "providers"), ","); got != "trivy,socket" {
		t.Fatalf("unexpected array providers: %q", got)
	}
	home := t.TempDir()
	if got := expandHome(home, "~"); got != home {
		t.Fatalf("expand ~ = %q, want %q", got, home)
	}
	if got := expandHome(home, "~/repo"); got != filepath.Join(home, "repo") {
		t.Fatalf("expand ~/repo = %q", got)
	}
	if got := scanMode(inventory.Report{Workspace: "/tmp/workspace"}); got != "workspace" {
		t.Fatalf("scan mode = %q", got)
	}
	if got := (Server{Version: "  ", Now: fixedNow}).version(); got != "devel" {
		t.Fatalf("default version = %q", got)
	}
	if got := (Server{Now: fixedNow}).now(); !got.Equal(fixedNow()) {
		t.Fatalf("now = %s", got)
	}

	large := strings.Repeat("x", maxTextBytes+1)
	text, err := jsonText(map[string]string{"value": large})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, `"truncated": true`) {
		t.Fatalf("expected truncated jsonText, got %.120q", text)
	}
	var items []inventory.Item
	for i := 0; i < 6000; i++ {
		items = append(items, inventory.Item{ID: strings.Repeat("item", 20), Tool: "Codex", Path: strings.Repeat("/tmp/path", 10)})
	}
	boundedReport := bounded("scan", inventory.Report{Items: items})
	if summary, ok := boundedReport.(map[string]any); !ok || summary["truncated"] != true {
		t.Fatalf("expected bounded report summary, got %#v", boundedReport)
	}
	boundedBlob := bounded("blob", map[string]string{"value": large})
	if summary, ok := boundedBlob.(map[string]any); !ok || summary["truncated"] != true {
		t.Fatalf("expected bounded generic summary, got %#v", boundedBlob)
	}
}

func TestInvalidJSONAndUnknownMethod(t *testing.T) {
	var out bytes.Buffer
	input := "{not-json}\n" + `{"jsonrpc":"2.0","id":2,"method":"missing"}` + "\n"
	if err := (Server{Home: t.TempDir(), Version: "test"}).Serve(strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, out.String())
	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}
	if code := responses[0]["error"].(map[string]any)["code"]; code != float64(-32700) {
		t.Fatalf("expected parse error, got %#v", responses[0])
	}
	if code := responses[1]["error"].(map[string]any)["code"]; code != float64(-32601) {
		t.Fatalf("expected method not found, got %#v", responses[1])
	}
}

func fixtureHome(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".codex", "config.toml"), `[mcp_servers.demo]
command = "npx"
args = ["-y", "@modelcontextprotocol/server-filesystem", "/"]

[mcp_servers.demo.env]
API_TOKEN = "super-secret-value"
`)
	return root
}

func decodeResponses(t *testing.T, output string) []map[string]any {
	t.Helper()
	var responses []map[string]any
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		var response map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &response); err != nil {
			t.Fatalf("invalid response JSON %q: %v", scanner.Text(), err)
		}
		responses = append(responses, response)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return responses
}

func assertScrubbed(t *testing.T, text string) {
	t.Helper()
	for _, forbidden := range []string{"super-secret-value"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("MCP output leaked %q:\n%s", forbidden, text)
		}
	}
}

func listFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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
		files = append(files, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func writeReportFile(t *testing.T, path string, report inventory.Report) {
	t.Helper()
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, string(data))
}

func toolJSON(t *testing.T, response map[string]any) map[string]any {
	t.Helper()
	text := toolText(t, response)
	var value map[string]any
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		t.Fatalf("tool result is not JSON object: %s: %v", text, err)
	}
	return value
}

func toolJSONArray(t *testing.T, response map[string]any) []any {
	t.Helper()
	text := toolText(t, response)
	var value []any
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		t.Fatalf("tool result is not JSON array: %s: %v", text, err)
	}
	return value
}

func toolText(t *testing.T, response map[string]any) string {
	t.Helper()
	result := response["result"].(map[string]any)
	content := result["content"].([]any)
	if len(content) == 0 {
		t.Fatalf("tool result missing content: %#v", result)
	}
	return content[0].(map[string]any)["text"].(string)
}

func resourceJSON(t *testing.T, response map[string]any) map[string]any {
	t.Helper()
	text := resourceText(t, response)
	var value map[string]any
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		t.Fatalf("resource result is not JSON object: %s: %v", text, err)
	}
	return value
}

func resourceText(t *testing.T, response map[string]any) string {
	t.Helper()
	result := response["result"].(map[string]any)
	contents := result["contents"].([]any)
	if len(contents) != 1 {
		t.Fatalf("unexpected resource contents: %#v", contents)
	}
	return contents[0].(map[string]any)["text"].(string)
}

func fixedNow() time.Time {
	return time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
}
