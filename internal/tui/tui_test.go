package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jsonbored/nightward/internal/inventory"
	"github.com/jsonbored/nightward/internal/schedule"
)

func TestRunExecutesConfiguredOpenTUISidecar(t *testing.T) {
	dir := t.TempDir()
	argsPath := filepath.Join(dir, "args.txt")
	reportPath := filepath.Join(dir, "report.json")
	sidecar := fakeSidecar(t, dir, argsPath, reportPath)
	t.Setenv(sidecarEnv, sidecar)

	report := inventory.Report{
		Summary: inventory.Summary{TotalFindings: 1},
		Findings: []inventory.Finding{
			{ID: "finding-1", Rule: "mcp_secret_env", Severity: inventory.RiskCritical},
		},
	}
	if err := Run(report, schedule.Plan{}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	args := readFile(t, argsPath)
	if !strings.Contains(args, "--input") {
		t.Fatalf("sidecar did not receive input flag: %q", args)
	}
	var got Bundle
	if err := json.Unmarshal([]byte(readFile(t, reportPath)), &got); err != nil {
		t.Fatalf("sidecar did not copy valid report JSON: %v", err)
	}
	if got.Scan.Summary.TotalFindings != 1 || got.Scan.Findings[0].ID != "finding-1" {
		t.Fatalf("unexpected report payload: %#v", got)
	}
	if got.Analysis.Summary.TotalSignals == 0 || got.FixPlan.Summary.Total != 1 {
		t.Fatalf("expected bundled analysis and fix plan: %#v", got)
	}
}

func TestCommandReportsMissingOpenTUISidecar(t *testing.T) {
	t.Setenv(sidecarEnv, "")
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", "")
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })

	cwd := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldCwd) })

	_, err = Command(filepath.Join(cwd, "scan.json"))
	if err == nil || !strings.Contains(err.Error(), "OpenTUI sidecar not found") {
		t.Fatalf("expected missing sidecar error, got %v", err)
	}
}

func fakeSidecar(t *testing.T, dir, argsPath, reportPath string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell sidecar fixture is POSIX-only")
	}
	path := filepath.Join(dir, "nightward-tui")
	script := `#!/usr/bin/env sh
set -eu
printf '%s\n' "$*" > "$1"
shift
if [ "${1:-}" != "--input" ]; then
  exit 64
fi
cp "$2" "$3"
`
	wrapper := filepath.Join(dir, "wrapper")
	if err := os.WriteFile(wrapper, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	shim := `#!/usr/bin/env sh
exec "` + wrapper + `" "` + argsPath + `" "$@" "` + reportPath + `"
`
	if err := os.WriteFile(path, []byte(shim), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
