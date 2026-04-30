package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/shadowbook/nightward/internal/backupplan"
	"github.com/shadowbook/nightward/internal/inventory"
	"github.com/shadowbook/nightward/internal/schedule"
	"github.com/shadowbook/nightward/internal/tui"
)

const version = "0.1.0"

type Check struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

type DoctorReport struct {
	GeneratedAt time.Time                 `json:"generated_at"`
	Version     string                    `json:"version"`
	Home        string                    `json:"home"`
	Executable  string                    `json:"executable"`
	Checks      []Check                   `json:"checks"`
	Schedule    schedule.Plan             `json:"schedule"`
	Adapters    []inventory.AdapterStatus `json:"adapters"`
}

func Run(args []string, stdout, stderr io.Writer) int {
	home, err := os.UserHomeDir()
	if err != nil {
		return fail(stderr, "cannot determine home directory: %v", err)
	}

	if len(args) == 0 {
		report := inventory.NewScanner(home).Scan()
		if err := tui.Run(report, schedule.Status(home)); err != nil {
			return fail(stderr, "tui failed: %v", err)
		}
		return 0
	}

	switch args[0] {
	case "-h", "--help", "help":
		printHelp(stdout)
	case "--version", "version":
		fmt.Fprintln(stdout, version)
	case "scan":
		return runScan(home, args[1:], stdout, stderr)
	case "doctor":
		return runDoctor(home, args[1:], stdout, stderr)
	case "plan":
		return runPlan(home, args[1:], stdout, stderr)
	case "adapters":
		return runAdapters(home, args[1:], stdout, stderr)
	case "schedule":
		return runSchedule(home, args[1:], stdout, stderr)
	default:
		return fail(stderr, "unknown command %q\n\nRun `nightward --help` for usage.", args[0])
	}
	return 0
}

func runScan(home string, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print JSON output")
	output := fs.String("output", "", "write JSON report to a file")
	outputDir := fs.String("output-dir", "", "write JSON report to a timestamped file in this directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	report := inventory.NewScanner(home).Scan()
	if err := maybeWriteReport(report, *output, *outputDir); err != nil {
		return fail(stderr, "failed to write scan report: %v", err)
	}
	if *jsonOut {
		return writeJSON(stdout, report, stderr)
	}
	printScan(stdout, report)
	return 0
}

func runDoctor(home string, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print JSON output")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	report := doctor(home)
	if *jsonOut {
		return writeJSON(stdout, report, stderr)
	}
	printDoctor(stdout, report)
	return 0
}

func runPlan(home string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "backup" {
		return fail(stderr, "usage: nightward plan backup --target <repo> [--json]")
	}
	fs := flag.NewFlagSet("plan backup", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", "", "private dotfiles repo or backup target root")
	jsonOut := fs.Bool("json", false, "print JSON output")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if *target == "" {
		return fail(stderr, "missing required --target")
	}
	absTarget := expandHome(home, *target)
	report := inventory.NewScanner(home).Scan()
	plan := backupplan.Build(report, absTarget)
	if *jsonOut {
		return writeJSON(stdout, plan, stderr)
	}
	printBackupPlan(stdout, plan)
	return 0
}

func runAdapters(home string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "list" {
		return fail(stderr, "usage: nightward adapters list [--json]")
	}
	fs := flag.NewFlagSet("adapters list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print JSON output")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	adapters := inventory.NewScanner(home).Scan().Adapters
	if *jsonOut {
		return writeJSON(stdout, adapters, stderr)
	}
	for _, adapter := range adapters {
		status := "missing"
		if adapter.Available {
			status = "found"
		}
		fmt.Fprintf(stdout, "%-12s %s - %s\n", adapter.Name, status, adapter.Description)
	}
	return 0
}

func runSchedule(home string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return fail(stderr, "usage: nightward schedule <plan|install|remove> [flags]")
	}
	exe := executablePath()
	switch args[0] {
	case "plan":
		fs := flag.NewFlagSet("schedule plan", flag.ContinueOnError)
		fs.SetOutput(stderr)
		preset := fs.String("preset", "nightly", "schedule preset")
		jsonOut := fs.Bool("json", false, "print JSON output")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		plan, err := schedule.BuildPlan(home, exe, *preset)
		if err != nil {
			return fail(stderr, "failed to build schedule plan: %v", err)
		}
		if *jsonOut {
			return writeJSON(stdout, plan, stderr)
		}
		printSchedulePlan(stdout, plan)
	case "install":
		fs := flag.NewFlagSet("schedule install", flag.ContinueOnError)
		fs.SetOutput(stderr)
		preset := fs.String("preset", "nightly", "schedule preset")
		dryRun := fs.Bool("dry-run", false, "print generated schedule files without writing")
		jsonOut := fs.Bool("json", false, "print JSON output")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if *dryRun {
			plan, err := schedule.BuildPlan(home, exe, *preset)
			if err != nil {
				return fail(stderr, "failed to build schedule plan: %v", err)
			}
			if *jsonOut {
				return writeJSON(stdout, plan, stderr)
			}
			printSchedulePlan(stdout, plan)
			return 0
		}
		plan, err := schedule.Install(home, exe, *preset)
		if err != nil {
			return fail(stderr, "failed to install schedule: %v", err)
		}
		if *jsonOut {
			return writeJSON(stdout, plan, stderr)
		}
		fmt.Fprintln(stdout, "Nightward schedule installed.")
		printSchedulePlan(stdout, plan)
	case "remove":
		fs := flag.NewFlagSet("schedule remove", flag.ContinueOnError)
		fs.SetOutput(stderr)
		dryRun := fs.Bool("dry-run", false, "print what would be removed without writing")
		jsonOut := fs.Bool("json", false, "print JSON output")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		plan, err := schedule.BuildPlan(home, exe, "nightly")
		if err != nil {
			return fail(stderr, "failed to build schedule plan: %v", err)
		}
		if *dryRun {
			if *jsonOut {
				return writeJSON(stdout, plan, stderr)
			}
			fmt.Fprintln(stdout, "Would remove:")
			for _, file := range plan.Files {
				fmt.Fprintf(stdout, "  %s\n", file.Path)
			}
			return 0
		}
		removed, err := schedule.Remove(home)
		if err != nil {
			return fail(stderr, "failed to remove schedule: %v", err)
		}
		if *jsonOut {
			return writeJSON(stdout, removed, stderr)
		}
		fmt.Fprintln(stdout, "Nightward schedule removed.")
	default:
		return fail(stderr, "usage: nightward schedule <plan|install|remove> [flags]")
	}
	return 0
}

func doctor(home string) DoctorReport {
	report := inventory.NewScanner(home).Scan()
	exe := executablePath()
	checks := []Check{
		commandCheck("git", "optional private-repo workflow and future Git integration"),
		commandCheck("launchctl", "macOS user scheduling"),
		commandCheck("systemctl", "Linux user scheduling"),
		commandCheck("crontab", "fallback schedule text"),
		pathCheck("home", home, true),
		pathCheck("state_dir", filepath.Join(home, ".local", "state", "nightward"), false),
	}
	return DoctorReport{
		GeneratedAt: time.Now().UTC(),
		Version:     version,
		Home:        home,
		Executable:  exe,
		Checks:      checks,
		Schedule:    schedule.Status(home),
		Adapters:    report.Adapters,
	}
}

func commandCheck(name, detail string) Check {
	path, err := exec.LookPath(name)
	if err != nil {
		return Check{ID: "command_" + name, Status: "warn", Message: name + " not found", Detail: detail}
	}
	return Check{ID: "command_" + name, Status: "ok", Message: name + " found", Detail: path}
}

func pathCheck(id, path string, required bool) Check {
	info, err := os.Stat(path)
	if err != nil {
		status := "info"
		if required {
			status = "warn"
		}
		return Check{ID: id, Status: status, Message: "path missing", Detail: path}
	}
	if !info.IsDir() {
		return Check{ID: id, Status: "warn", Message: "path is not a directory", Detail: path}
	}
	return Check{ID: id, Status: "ok", Message: "path available", Detail: path}
}

func maybeWriteReport(report inventory.Report, output, outputDir string) error {
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return err
		}
		output = filepath.Join(outputDir, "nightward-scan-"+report.GeneratedAt.Format("20060102-150405Z")+".json")
	}
	if output == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(output, data, 0600)
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, `Nightward watches local AI agent state before it leaks into dotfiles.

Usage:
  nightward                                Open the TUI
  nightward scan [--json] [--output FILE] [--output-dir DIR]
  nightward doctor [--json]
  nightward plan backup --target <repo> [--json]
  nightward adapters list [--json]
  nightward schedule plan --preset nightly [--json]
  nightward schedule install --preset nightly --dry-run [--json]
  nightward schedule remove --dry-run [--json]

V1 is read-only except explicit schedule install/remove commands.
`)
}

func printScan(w io.Writer, report inventory.Report) {
	fmt.Fprintf(w, "Nightward scan: %d items, %d findings\n", report.Summary.TotalItems, report.Summary.TotalFindings)
	for class, count := range report.Summary.ByClassification {
		fmt.Fprintf(w, "  %-14s %d\n", class, count)
	}
	if len(report.Findings) > 0 {
		fmt.Fprintln(w, "\nTop findings:")
		for i, finding := range report.Findings {
			if i >= 8 {
				break
			}
			fmt.Fprintf(w, "  [%s] %s: %s\n", finding.Severity, finding.Rule, finding.Message)
		}
	}
}

func printDoctor(w io.Writer, report DoctorReport) {
	fmt.Fprintf(w, "Nightward doctor %s\n", report.Version)
	for _, check := range report.Checks {
		fmt.Fprintf(w, "  %-4s %-18s %s\n", check.Status, check.ID, check.Detail)
	}
	fmt.Fprintf(w, "\nSchedule: installed=%t report_dir=%s", report.Schedule.Installed, report.Schedule.ReportDir)
	if report.Schedule.LastReport != "" {
		fmt.Fprintf(w, " last_report=%s", report.Schedule.LastReport)
	}
	fmt.Fprintln(w)
}

func printBackupPlan(w io.Writer, plan backupplan.Plan) {
	fmt.Fprintf(w, "Backup dry-run plan for %s\n", plan.TargetRoot)
	fmt.Fprintf(w, "  include: %d  review: %d  exclude: %d\n", plan.Summary.Included, plan.Summary.Review, plan.Summary.Excluded)
	for _, entry := range plan.Entries {
		fmt.Fprintf(w, "  %-7s %-12s %s -> %s\n", entry.Action, entry.Tool, entry.Source, entry.Target)
	}
}

func printSchedulePlan(w io.Writer, plan schedule.Plan) {
	fmt.Fprintf(w, "Schedule preset: %s (%s)\n", plan.Preset, plan.Platform)
	fmt.Fprintf(w, "Command: %s\n", strings.Join(plan.Command, " "))
	fmt.Fprintf(w, "Reports: %s\n", plan.ReportDir)
	for _, file := range plan.Files {
		fmt.Fprintf(w, "\n# %s\n%s", file.Path, file.Content)
		if !strings.HasSuffix(file.Content, "\n") {
			fmt.Fprintln(w)
		}
	}
	for _, note := range plan.Notes {
		fmt.Fprintf(w, "\nNote: %s\n", note)
	}
}

func writeJSON(w io.Writer, value any, stderr io.Writer) int {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fail(stderr, "failed to encode JSON: %v", err)
	}
	return 0
}

func fail(stderr io.Writer, format string, args ...any) int {
	fmt.Fprintf(stderr, "nightward: "+format+"\n", args...)
	return 1
}

func expandHome(home, path string) string {
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

func executablePath() string {
	exe, err := os.Executable()
	if err == nil && exe != "" {
		if strings.Contains(exe, "go-build") {
			if path, err := exec.LookPath("nightward"); err == nil {
				return path
			}
			return "nightward"
		}
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			return resolved
		}
		return exe
	}
	return "nightward"
}
