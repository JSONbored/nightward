package analysis

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jsonbored/nightward/internal/inventory"
)

func TestAnalysisBuildsOfflineSignalsAndRedacts(t *testing.T) {
	report := inventory.Report{
		GeneratedAt: time.Date(2026, 4, 30, 7, 0, 0, 0, time.UTC),
		Findings: []inventory.Finding{
			{
				ID:             "mcp_secret_env-111111111111",
				Tool:           "Codex",
				Path:           "/tmp/config.toml",
				Severity:       inventory.RiskCritical,
				Rule:           "mcp_secret_env",
				Message:        "MCP server stores a sensitive environment key.",
				Evidence:       "env_key=API_TOKEN",
				Recommendation: "Keep secret values outside dotfiles.",
				Why:            "Secrets should stay local.",
				Confidence:     "high",
			},
		},
		Items: []inventory.Item{
			{
				ID:             "secret-item",
				Tool:           "Secrets",
				Path:           "/tmp/workspace/.env",
				Classification: inventory.SecretAuth,
				Risk:           inventory.RiskCritical,
			},
		},
	}

	out := Run(report, Options{Mode: "workspace", Workspace: "/tmp/workspace"})
	if out.Summary.TotalSignals != 2 {
		t.Fatalf("expected two analysis signals, got %#v", out.Summary)
	}
	if out.Summary.SignalsByCategory[CategorySecrets] != 2 {
		t.Fatalf("expected secret category signals, got %#v", out.Summary.SignalsByCategory)
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "super-secret-value") {
		t.Fatalf("analysis leaked a secret: %s", data)
	}
}

func TestProviderDoctorHonorsOnlineGate(t *testing.T) {
	statuses := ProviderStatuses([]string{"socket"}, false)
	var found bool
	for _, status := range statuses {
		if status.Name == "socket" {
			found = true
			if status.Status != "blocked" {
				t.Fatalf("expected socket to be blocked without --online, got %#v", status)
			}
		}
	}
	if !found {
		t.Fatal("missing socket provider")
	}
}

func TestProviderDoctorEnablesOnlineProvidersOnlyWithOnlineOptIn(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "socket"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", dir)

	blocked := ProviderStatuses([]string{"socket"}, false)
	ready := ProviderStatuses([]string{"socket"}, true)
	var blockedStatus, readyStatus ProviderStatus
	for _, status := range blocked {
		if status.Name == "socket" {
			blockedStatus = status
		}
	}
	for _, status := range ready {
		if status.Name == "socket" {
			readyStatus = status
		}
	}
	if blockedStatus.Status != "blocked" {
		t.Fatalf("expected socket blocked without online opt-in, got %#v", blockedStatus)
	}
	if readyStatus.Status != "ready" || !readyStatus.Enabled || !readyStatus.Available {
		t.Fatalf("expected socket ready with online opt-in, got %#v", readyStatus)
	}
}

func TestProviderDoctorFindsFakeLocalProvider(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gitleaks")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	statuses := ProviderStatuses([]string{"gitleaks"}, false)
	for _, status := range statuses {
		if status.Name == "gitleaks" {
			if !status.Available || status.Status != "ready" {
				t.Fatalf("expected fake gitleaks to be ready, got %#v", status)
			}
			return
		}
	}
	t.Fatal("missing gitleaks provider")
}

func TestRunExecutesExplicitLocalProviderAndRedacts(t *testing.T) {
	dir := t.TempDir()
	workspace := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "gitleaks"), `#!/bin/sh
printf '[{"RuleID":"generic-api-key","Description":"API_TOKEN=super-secret-value","File":"config.toml","StartLine":7}]'
`)
	t.Setenv("PATH", dir)

	report := inventory.NewWorkspaceScanner(workspace).Scan()
	out := Run(report, Options{Mode: "workspace", Workspace: workspace, With: []string{"gitleaks"}})
	if out.Summary.SignalsByProvider["gitleaks"] != 1 {
		t.Fatalf("expected one gitleaks signal, got %#v", out.Summary)
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, "super-secret-value") {
		t.Fatalf("provider signal leaked secret: %s", text)
	}
	if !strings.Contains(text, "generic-api-key") || !strings.Contains(text, "config.toml") {
		t.Fatalf("provider signal missed finding metadata: %s", text)
	}
}

func TestRunReportsProviderExecutionFailureAsSignal(t *testing.T) {
	dir := t.TempDir()
	workspace := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "gitleaks"), `#!/bin/sh
echo 'API_TOKEN=super-secret-value failed' >&2
exit 2
`)
	t.Setenv("PATH", dir)

	report := inventory.NewWorkspaceScanner(workspace).Scan()
	out := Run(report, Options{Mode: "workspace", Workspace: workspace, With: []string{"gitleaks"}})
	if out.Summary.SignalsByProvider["gitleaks"] != 1 {
		t.Fatalf("expected provider failure signal, got %#v", out.Summary)
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "super-secret-value") {
		t.Fatalf("provider failure leaked stderr: %s", data)
	}
}

func TestProviderParsersCoverSupportedFormats(t *testing.T) {
	root := "/tmp/workspace"
	gitleaks, err := parseProviderOutput("gitleaks", root, `[{"RuleID":"generic-api-key","Description":"secret match","File":"/tmp/workspace/config.toml","StartLine":9}]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(gitleaks) != 1 || gitleaks[0].Path != "config.toml" || gitleaks[0].Severity != inventory.RiskHigh || !strings.Contains(gitleaks[0].Evidence, "line=9") {
		t.Fatalf("unexpected gitleaks parse: %#v", gitleaks)
	}

	trufflehog, err := parseProviderOutput("trufflehog", root, `{"DetectorName":"GitHub","Verified":true,"SourceMetadata":{"Data":{"Filesystem":{"file":"/tmp/workspace/.env"}}}}`+"\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(trufflehog) != 1 || trufflehog[0].Path != ".env" || trufflehog[0].Severity != inventory.RiskCritical {
		t.Fatalf("unexpected trufflehog parse: %#v", trufflehog)
	}

	semgrep, err := parseProviderOutput("semgrep", root, `{"results":[{"check_id":"go.lang.security","path":"/tmp/workspace/main.go","extra":{"message":"review this","severity":"ERROR"}}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(semgrep) != 1 || semgrep[0].Path != "main.go" || semgrep[0].Severity != inventory.RiskHigh || semgrep[0].Category != CategoryExecution {
		t.Fatalf("unexpected semgrep parse: %#v", semgrep)
	}

	trivy, err := parseProviderOutput("trivy", root, `{"Results":[{"Target":"/tmp/workspace/package-lock.json","Vulnerabilities":[{"VulnerabilityID":"CVE-2026-0001","PkgName":"demo","Severity":"CRITICAL","Title":"demo vulnerable"}],"Misconfigurations":[{"ID":"AVD-1","Severity":"MEDIUM","Title":"bad config"}],"Secrets":[{"RuleID":"aws-access-key","Severity":"HIGH","Target":"/tmp/workspace/.env","Title":"AWS key"}]}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(trivy) != 3 || trivy[0].Severity != inventory.RiskCritical || trivy[1].Category != CategorySecrets || trivy[2].Category != CategoryExecution {
		t.Fatalf("unexpected trivy parse: %#v", trivy)
	}

	osv, err := parseProviderOutput("osv-scanner", root, `{"results":[{"source":{"path":"/tmp/workspace/package-lock.json","type":"lockfile"},"packages":[{"package":{"name":"demo"},"vulnerabilities":[{"id":"GHSA-123","summary":"bad package"}]}]}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(osv) != 1 || osv[0].Rule != "GHSA-123" || osv[0].Path != "package-lock.json" || osv[0].Category != CategorySupplyChain {
		t.Fatalf("unexpected osv parse: %#v", osv)
	}
	osvDirect, err := parseProviderOutput("osv-scanner", root, `{"results":[{"path":"/tmp/workspace/go.mod","vulnerabilities":[{"id":"GO-2026-0001","details":"stdlib issue"}]}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(osvDirect) != 1 || osvDirect[0].Rule != "GO-2026-0001" || osvDirect[0].Path != "go.mod" {
		t.Fatalf("unexpected direct osv parse: %#v", osvDirect)
	}

	socket, err := parseProviderOutput("socket", root, `{"issues":[{"type":"malware","severity":"high","message":"API_TOKEN=super-secret-value","package":"demo","file":"package.json"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(socket) != 1 || socket[0].Rule != "malware" || strings.Contains(socket[0].Message, "super-secret-value") {
		t.Fatalf("unexpected socket parse: %#v", socket)
	}
	socketScan, err := parseProviderOutput("socket", root, `{"scanId":"scan_123"}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(socketScan) != 1 || socketScan[0].Rule != "scan-created" || socketScan[0].Severity != inventory.RiskInfo {
		t.Fatalf("unexpected socket scan parse: %#v", socketScan)
	}
}

func TestOnlineProviderParsersRedactSecretLikeFields(t *testing.T) {
	root := "/tmp/workspace"
	cases := map[string]string{
		"trivy":       `{"Results":[{"Target":"/tmp/workspace/package-lock.json","Vulnerabilities":[{"VulnerabilityID":"CVE-2026-0001","PkgName":"demo","Severity":"CRITICAL","Title":"API_TOKEN=super-secret-value vulnerable dependency"}],"Misconfigurations":[{"ID":"AVD-1","Severity":"MEDIUM","Title":"password=super-secret-value in config"}],"Secrets":[{"RuleID":"aws-access-key","Severity":"HIGH","Target":"/tmp/workspace/.env","Title":"private_key=super-secret-value"}]}]}`,
		"osv-scanner": `{"results":[{"source":{"path":"/tmp/workspace/package-lock.json","type":"lockfile"},"packages":[{"package":{"name":"demo"},"vulnerabilities":[{"id":"GHSA-123","summary":"api_key=super-secret-value vulnerable package"}]}]},{"path":"/tmp/workspace/go.mod","vulnerabilities":[{"id":"GO-2026-0001","details":"password=super-secret-value detail"}]}]}`,
		"socket":      `{"issues":[{"type":"malware","severity":"high","message":"API_TOKEN=super-secret-value","package":{"name":"demo"},"file":"package.json"}],"scanId":"scan_123"}`,
	}
	for provider, output := range cases {
		findings, err := parseProviderOutput(provider, root, output)
		if err != nil {
			t.Fatalf("%s parse failed: %v", provider, err)
		}
		if len(findings) == 0 {
			t.Fatalf("%s parse returned no findings", provider)
		}
		data, err := json.Marshal(findings)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "super-secret-value") {
			t.Fatalf("%s parser leaked provider secret material: %s", provider, data)
		}
		if !strings.Contains(string(data), "[redacted]") {
			t.Fatalf("%s parser did not preserve redaction marker: %s", provider, data)
		}
	}
}

func TestOnlineProviderParsersRejectInvalidJSON(t *testing.T) {
	for _, provider := range []string{"trivy", "osv-scanner", "socket"} {
		if _, err := parseProviderOutput(provider, "/tmp/workspace", `{"not-json"`); err == nil {
			t.Fatalf("%s parser accepted invalid JSON", provider)
		}
	}
}

func TestProviderHelpersAndOutputLimits(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "semgrep.yml"), []byte("rules: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"gitleaks", "trufflehog", "semgrep"} {
		if args, ok, err := providerArgs(name, root); err != nil || !ok || len(args) == 0 {
			t.Fatalf("missing provider args for %s: %#v ok=%t err=%v", name, args, ok, err)
		}
	}
	for _, name := range []string{"trivy", "osv-scanner", "socket"} {
		if args, ok, err := providerArgs(name, root); err != nil || !ok || len(args) == 0 {
			t.Fatalf("missing online-capable provider args for %s: %#v ok=%t err=%v", name, args, ok, err)
		}
	}
	assertStringSlice(t, "trivy args", []string{"filesystem", "--format", "json", "--scanners", "vuln,secret,misconfig", "--skip-version-check", root}, mustProviderArgs(t, "trivy", root))
	assertStringSlice(t, "osv-scanner args", []string{"scan", "source", "-r", "--format", "json", root}, mustProviderArgs(t, "osv-scanner", root))
	assertStringSlice(t, "socket args", []string{"scan", "create", root, "--json"}, mustProviderArgs(t, "socket", root))
	if _, ok, err := providerArgs("semgrep", t.TempDir()); !ok || err == nil {
		t.Fatalf("semgrep should require a local config, ok=%t err=%v", ok, err)
	}
	externalConfigDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(externalConfigDir, "semgrep.yml"), []byte("rules: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	symlinkRoot := t.TempDir()
	if err := os.Symlink(filepath.Join(externalConfigDir, "semgrep.yml"), filepath.Join(symlinkRoot, "semgrep.yml")); err == nil {
		if _, ok := localSemgrepConfig(symlinkRoot); ok {
			t.Fatal("semgrep config symlink outside workspace should be ignored")
		}
	}
	if got := semgrepSeverity("WARNING"); got != inventory.RiskMedium {
		t.Fatalf("unexpected warning severity: %s", got)
	}
	if got := semgrepSeverity("INFO"); got != inventory.RiskLow {
		t.Fatalf("unexpected info severity: %s", got)
	}
	if got := semgrepSeverity("unknown"); got != inventory.RiskMedium {
		t.Fatalf("unexpected default severity: %s", got)
	}

	var limited limitedBuffer
	limited.limit = 5
	if n, err := limited.Write([]byte("123456789")); err != nil || n != 9 {
		t.Fatalf("unexpected limited write n=%d err=%v", n, err)
	}
	if got := limited.String(); !strings.Contains(got, "12345") || !strings.Contains(got, "truncated") {
		t.Fatalf("unexpected limited buffer output: %q", got)
	}
	if got := firstProviderLine("API_TOKEN=super-secret-value\nsecond"); strings.Contains(got, "super-secret-value") || !strings.Contains(got, "[redacted]") {
		t.Fatalf("stderr was not redacted: %q", got)
	}
	if got := providerRecommendation("semgrep", CategoryExecution); !strings.Contains(got, "semgrep") {
		t.Fatalf("unexpected provider recommendation: %q", got)
	}
}

func TestRunOnlineProviderTimeoutIsReportedAndRedacted(t *testing.T) {
	dir := t.TempDir()
	workspace := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "trivy"), `#!/bin/sh
echo 'API_TOKEN=super-secret-value still running' >&2
/bin/sleep 1
`)
	t.Setenv("PATH", dir)
	oldTimeout := providerTimeout
	providerTimeout = 10 * time.Millisecond
	t.Cleanup(func() { providerTimeout = oldTimeout })

	report := inventory.NewWorkspaceScanner(workspace).Scan()
	out := Run(report, Options{Mode: "workspace", Workspace: workspace, With: []string{"trivy"}, Online: true})
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "provider_execution_failed") || !strings.Contains(text, "timed out") {
		t.Fatalf("timeout was not reported as a provider failure: %s", text)
	}
	if strings.Contains(text, "super-secret-value") {
		t.Fatalf("timeout failure leaked provider stderr: %s", text)
	}
}

func TestExplainRelativePathAndHumanProviderSummary(t *testing.T) {
	report := Run(inventory.Report{
		Findings: []inventory.Finding{
			{
				ID:             "mcp_shell_command-111111111111",
				Tool:           "Codex",
				Path:           "/tmp/work/config.toml",
				Rule:           "mcp_shell_command",
				Severity:       inventory.RiskMedium,
				Message:        "Shell wrapper",
				Recommendation: "Use a direct command.",
			},
		},
	}, Options{})

	if len(report.Signals) != 1 {
		t.Fatalf("expected one signal: %#v", report.Signals)
	}
	fullID := report.Signals[0].ID
	if signal, ok := Explain(report, fullID); !ok || signal.ID != fullID {
		t.Fatalf("full signal explain failed: %#v ok=%t", signal, ok)
	}
	if signal, ok := Explain(report, fullID[:8]); !ok || signal.ID != fullID {
		t.Fatalf("prefix signal explain failed: %#v ok=%t", signal, ok)
	}
	if _, ok := Explain(report, "missing"); ok {
		t.Fatal("unexpected match for missing signal")
	}
	if got := RelativeWorkspacePath("/tmp/work", "/tmp/work/config.toml"); got != "config.toml" {
		t.Fatalf("unexpected relative path: %s", got)
	}
	if got := RelativeWorkspacePath("/tmp/work", "/tmp/else/config.toml"); got != "/tmp/else/config.toml" {
		t.Fatalf("unexpected path outside workspace: %s", got)
	}

	enabled := HumanProviderSummary(ProviderStatus{Provider: Provider{Name: "gitleaks"}, Enabled: true, Available: true, Status: "ready"})
	disabled := HumanProviderSummary(ProviderStatus{Provider: Provider{Name: "socket"}, Enabled: false, Available: false, Status: "missing"})
	if !strings.Contains(enabled, "enabled ready (available)") || !strings.Contains(disabled, "disabled missing") {
		t.Fatalf("unexpected provider summaries: %q / %q", enabled, disabled)
	}
}

func writeExecutable(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0700); err != nil {
		t.Fatal(err)
	}
}

func mustProviderArgs(t *testing.T, name, root string) []string {
	t.Helper()
	args, ok, err := providerArgs(name, root)
	if err != nil || !ok {
		t.Fatalf("providerArgs(%s) ok=%t err=%v", name, ok, err)
	}
	return args
}

func assertStringSlice(t *testing.T, name string, want, got []string) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("%s length mismatch\nwant=%#v\ngot=%#v", name, want, got)
	}
	for i := range want {
		if want[i] != got[i] {
			t.Fatalf("%s mismatch at %d\nwant=%#v\ngot=%#v", name, i, want, got)
		}
	}
}

func TestAnalysisItemCategoriesAndRiskRanks(t *testing.T) {
	report := inventory.Report{Items: []inventory.Item{
		{ID: "machine", Classification: inventory.MachineLocal, Path: "/tmp/socket", Risk: inventory.RiskMedium},
		{ID: "app", Classification: inventory.AppOwned, Path: "/tmp/app.db", Risk: inventory.RiskLow},
		{ID: "portable", Classification: inventory.Portable, Path: "/tmp/config", Risk: inventory.RiskInfo},
	}}
	out := Run(report, Options{})
	if out.Summary.TotalSignals != 2 {
		t.Fatalf("expected machine-local and app-owned signals only: %#v", out.Summary)
	}
	if out.Summary.SignalsByCategory[CategoryLocality] != 1 || out.Summary.SignalsByCategory[CategoryAppState] != 1 {
		t.Fatalf("unexpected item categories: %#v", out.Summary.SignalsByCategory)
	}

	for rule, want := range map[string]SignalCategory{
		"mcp_unpinned_package":  CategorySupplyChain,
		"mcp_secret_header":     CategorySecrets,
		"mcp_broad_filesystem":  CategoryFilesystem,
		"mcp_local_endpoint":    CategoryNetwork,
		"mcp_unknown_command":   CategoryExecution,
		"mcp_server_review":     CategoryExecution,
		"nightward/custom_rule": CategoryUnknown,
	} {
		if got := categoryForRule(rule); got != want {
			t.Fatalf("categoryForRule(%q)=%q, want %q", rule, got, want)
		}
	}
	if riskRank(inventory.RiskCritical) <= riskRank(inventory.RiskHigh) || riskRank(inventory.RiskInfo) != 1 {
		t.Fatal("unexpected risk ranking")
	}
}
