package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/jsonbored/nightward/internal/analysis"
	"github.com/jsonbored/nightward/internal/backupplan"
	"github.com/jsonbored/nightward/internal/fixplan"
	"github.com/jsonbored/nightward/internal/inventory"
	"github.com/jsonbored/nightward/internal/schedule"
)

const (
	sidecarEnv = "NIGHTWARD_TUI_BIN"
	binEnv     = "NIGHTWARD_BIN"
)

type Bundle struct {
	SchemaVersion int              `json:"schema_version"`
	Scan          inventory.Report `json:"scan"`
	Analysis      analysis.Report  `json:"analysis"`
	FixPlan       fixplan.Plan     `json:"fix_plan"`
	BackupPlan    backupplan.Plan  `json:"backup_plan"`
	Schedule      schedule.Plan    `json:"schedule"`
}

// Run starts the OpenTUI renderer with a private JSON report snapshot.
func Run(report inventory.Report, scheduleStatus schedule.Plan) error {
	reportPath, cleanup, err := writeBundle(report, scheduleStatus)
	if err != nil {
		return err
	}
	defer cleanup()

	cmd, err := Command(reportPath)
	if err != nil {
		return err
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Command returns the sidecar command used by tests and the CLI launcher.
func Command(reportPath string) (*exec.Cmd, error) {
	launcher, args, err := resolveLauncher(reportPath)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(launcher, args...) // #nosec G204 -- launcher is resolved from explicit env, adjacent release binary, PATH, or repo-local dev source.
	if exe, err := os.Executable(); err == nil && os.Getenv(binEnv) == "" {
		cmd.Env = append(os.Environ(), binEnv+"="+exe)
	}
	return cmd, nil
}

func writeBundle(report inventory.Report, scheduleStatus schedule.Plan) (string, func(), error) {
	dir, err := os.MkdirTemp("", "nightward-tui-*")
	if err != nil {
		return "", nil, fmt.Errorf("create TUI temp dir: %w", err)
	}
	cleanup := func() {
		_ = os.RemoveAll(dir)
	}
	path := filepath.Join(dir, "scan.json")
	data, err := json.MarshalIndent(newBundle(report, scheduleStatus), "", "  ")
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("encode TUI report: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0600); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("write TUI report: %w", err)
	}
	return path, cleanup, nil
}

func newBundle(report inventory.Report, scheduleStatus schedule.Plan) Bundle {
	mode := report.ScanMode
	if mode == "" {
		mode = "home"
	}
	target := filepath.Join(report.Home, "dotfiles")
	return Bundle{
		SchemaVersion: 1,
		Scan:          report,
		Analysis: analysis.Run(report, analysis.Options{
			Mode:      mode,
			Workspace: report.Workspace,
		}),
		FixPlan:    fixplan.Build(report, fixplan.Selector{All: true}),
		BackupPlan: backupplan.Build(report, target),
		Schedule:   scheduleStatus,
	}
}

func resolveLauncher(reportPath string) (string, []string, error) {
	if override := os.Getenv(sidecarEnv); override != "" {
		return override, []string{"--input", reportPath}, nil
	}

	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), sidecarName())
		if isExecutable(candidate) {
			return candidate, []string{"--input", reportPath}, nil
		}
	}

	if sidecar, err := exec.LookPath(sidecarName()); err == nil {
		return sidecar, []string{"--input", reportPath}, nil
	}

	if bun, source, ok := repoOpenTUI(); ok {
		return bun, []string{source, "--input", reportPath}, nil
	}

	return "", nil, errors.New("OpenTUI sidecar not found; run `make install-local`, install a release archive, or set NIGHTWARD_TUI_BIN")
}

func repoOpenTUI() (string, string, bool) {
	bun, err := exec.LookPath("bun")
	if err != nil {
		return "", "", false
	}
	for _, root := range repoRoots() {
		source := filepath.Join(root, "packages", "tui", "src", "main.ts")
		if info, err := os.Stat(source); err == nil && !info.IsDir() {
			return bun, source, true
		}
	}
	return "", "", false
}

func repoRoots() []string {
	var roots []string
	if cwd, err := os.Getwd(); err == nil {
		roots = append(roots, cwd)
		for dir := cwd; ; {
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			roots = append(roots, parent)
			dir = parent
		}
	}
	return roots
}

func sidecarName() string {
	if runtime.GOOS == "windows" {
		return "nightward-tui.exe"
	}
	return "nightward-tui"
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0111 != 0
}
