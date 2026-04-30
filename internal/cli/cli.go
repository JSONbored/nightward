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
	"github.com/shadowbook/nightward/internal/fixplan"
	"github.com/shadowbook/nightward/internal/inventory"
	"github.com/shadowbook/nightward/internal/policy"
	"github.com/shadowbook/nightward/internal/schedule"
	"github.com/shadowbook/nightward/internal/snapshot"
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
	return RunWithName("nightward", args, stdout, stderr)
}

func RunWithName(commandName string, args []string, stdout, stderr io.Writer) int {
	if commandName == "" {
		commandName = "nightward"
	}
	home := os.Getenv("NIGHTWARD_HOME")
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return fail(stderr, "cannot determine home directory: %v", err)
		}
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
		printHelp(stdout, commandName)
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
	case "findings":
		return runFindings(home, args[1:], stdout, stderr)
	case "fix":
		return runFix(home, args[1:], stdout, stderr)
	case "policy":
		return runPolicy(home, args[1:], stdout, stderr)
	case "snapshot":
		return runSnapshot(home, args[1:], stdout, stderr)
	case "schedule":
		return runSchedule(home, args[1:], stdout, stderr)
	default:
		return fail(stderr, "unknown command %q\n\nRun `%s --help` for usage.", args[0], commandName)
	}
	return 0
}

func runSnapshot(home string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return fail(stderr, "usage: nightward snapshot <plan|diff> [flags]")
	}
	switch args[0] {
	case "plan":
		fs := flag.NewFlagSet("snapshot plan", flag.ContinueOnError)
		fs.SetOutput(stderr)
		target := fs.String("target", "", "snapshot target root")
		jsonOut := fs.Bool("json", false, "print JSON output")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if *target == "" {
			return fail(stderr, "missing required --target")
		}
		report := inventory.NewScanner(home).Scan()
		plan := snapshot.Build(report, expandHome(home, *target))
		if *jsonOut {
			return writeJSON(stdout, plan, stderr)
		}
		printSnapshotPlan(stdout, plan)
	case "diff":
		fs := flag.NewFlagSet("snapshot diff", flag.ContinueOnError)
		fs.SetOutput(stderr)
		fromPath := fs.String("from", "", "previous snapshot plan JSON")
		toPath := fs.String("to", "", "new snapshot plan JSON")
		jsonOut := fs.Bool("json", false, "print JSON output")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if *fromPath == "" || *toPath == "" {
			return fail(stderr, "missing required --from and --to")
		}
		from, err := snapshot.Load(expandHome(home, *fromPath))
		if err != nil {
			return fail(stderr, "failed to load --from snapshot: %v", err)
		}
		to, err := snapshot.Load(expandHome(home, *toPath))
		if err != nil {
			return fail(stderr, "failed to load --to snapshot: %v", err)
		}
		diff := snapshot.Compare(*fromPath, *toPath, from, to)
		if *jsonOut {
			return writeJSON(stdout, diff, stderr)
		}
		printSnapshotDiff(stdout, diff)
	default:
		return fail(stderr, "usage: nightward snapshot <plan|diff> [flags]")
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

func runFindings(home string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return fail(stderr, "usage: nightward findings <list|explain> [flags]")
	}
	report := inventory.NewScanner(home).Scan()
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("findings list", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jsonOut := fs.Bool("json", false, "print JSON output")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if *jsonOut {
			return writeJSON(stdout, report.Findings, stderr)
		}
		printFindingsList(stdout, report.Findings)
	case "explain":
		fs := flag.NewFlagSet("findings explain", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jsonOut := fs.Bool("json", false, "print JSON output")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		remaining := fs.Args()
		if len(remaining) != 1 {
			return fail(stderr, "usage: nightward findings explain <finding-id> [--json]")
		}
		finding, ok := fixplan.Find(report, remaining[0])
		if !ok {
			return fail(stderr, "finding not found: %s", remaining[0])
		}
		if *jsonOut {
			return writeJSON(stdout, finding, stderr)
		}
		printFindingExplain(stdout, finding)
	default:
		return fail(stderr, "usage: nightward findings <list|explain> [flags]")
	}
	return 0
}

func runFix(home string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return fail(stderr, "usage: nightward fix <plan|preview|export> [flags]")
	}
	report := inventory.NewScanner(home).Scan()
	switch args[0] {
	case "plan":
		fs := flag.NewFlagSet("fix plan", flag.ContinueOnError)
		fs.SetOutput(stderr)
		findingID := fs.String("finding", "", "limit to a finding ID or unique prefix")
		rule := fs.String("rule", "", "limit to a rule ID")
		all := fs.Bool("all", false, "include all findings")
		jsonOut := fs.Bool("json", false, "print JSON output")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		selector, err := fixSelector(report, *findingID, *rule, *all)
		if err != nil {
			return fail(stderr, err.Error())
		}
		plan := fixplan.Build(report, selector)
		if *jsonOut {
			return writeJSON(stdout, plan, stderr)
		}
		printFixPlan(stdout, plan)
	case "preview":
		fs := flag.NewFlagSet("fix preview", flag.ContinueOnError)
		fs.SetOutput(stderr)
		format := fs.String("format", "diff", "preview format: diff, markdown, or json")
		findingID := fs.String("finding", "", "limit to a finding ID or unique prefix")
		rule := fs.String("rule", "", "limit to a rule ID")
		all := fs.Bool("all", false, "include all findings")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		selector, err := fixSelector(report, *findingID, *rule, *all)
		if err != nil {
			return fail(stderr, err.Error())
		}
		preview := fixplan.BuildPreview(report, selector)
		switch *format {
		case "json":
			return writeJSON(stdout, preview, stderr)
		case "markdown", "md":
			fmt.Fprint(stdout, fixplan.PreviewMarkdown(preview))
		case "diff":
			fmt.Fprint(stdout, fixplan.PreviewDiff(preview))
		default:
			return fail(stderr, "unsupported fix preview format %q", *format)
		}
	case "export":
		fs := flag.NewFlagSet("fix export", flag.ContinueOnError)
		fs.SetOutput(stderr)
		format := fs.String("format", "markdown", "export format: markdown or json")
		findingID := fs.String("finding", "", "limit to a finding ID or unique prefix")
		rule := fs.String("rule", "", "limit to a rule ID")
		all := fs.Bool("all", false, "include all findings")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		selector, err := fixSelector(report, *findingID, *rule, *all)
		if err != nil {
			return fail(stderr, err.Error())
		}
		plan := fixplan.Build(report, selector)
		switch *format {
		case "json":
			return writeJSON(stdout, plan, stderr)
		case "markdown", "md":
			fmt.Fprint(stdout, fixplan.Markdown(plan))
		default:
			return fail(stderr, "unsupported fix export format %q", *format)
		}
	default:
		return fail(stderr, "usage: nightward fix <plan|preview|export> [flags]")
	}
	return 0
}

func runPolicy(home string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return fail(stderr, "usage: nightward policy <init|explain|check|sarif> [flags]")
	}
	switch args[0] {
	case "init":
		fs := flag.NewFlagSet("policy init", flag.ContinueOnError)
		fs.SetOutput(stderr)
		dryRun := fs.Bool("dry-run", false, "print default policy config without writing")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if !*dryRun {
			return fail(stderr, "policy init is dry-run only in this release; rerun with --dry-run")
		}
		fmt.Fprint(stdout, policy.DefaultConfigYAML())
	case "explain":
		fs := flag.NewFlagSet("policy explain", flag.ContinueOnError)
		fs.SetOutput(stderr)
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		fmt.Fprint(stdout, policy.Explain())
	case "check":
		fs := flag.NewFlagSet("policy check", flag.ContinueOnError)
		fs.SetOutput(stderr)
		strict := fs.Bool("strict", false, "fail on medium or higher findings")
		jsonOut := fs.Bool("json", false, "print JSON output")
		configPath := fs.String("config", "", "optional .nightward.yml policy config")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		config, err := policy.LoadConfig(expandConfigPath(home, *configPath))
		if err != nil {
			return fail(stderr, "failed to load policy config: %v", err)
		}
		report := inventory.NewScanner(home).Scan()
		policyReport := policy.CheckWithOptions(report, policy.Options{Strict: *strict, Config: config})
		if *jsonOut {
			code := writeJSON(stdout, policyReport, stderr)
			if code != 0 {
				return code
			}
		} else {
			printPolicy(stdout, policyReport)
		}
		if !policyReport.Passed {
			return 1
		}
	case "sarif":
		fs := flag.NewFlagSet("policy sarif", flag.ContinueOnError)
		fs.SetOutput(stderr)
		output := fs.String("output", "nightward.sarif", "write SARIF report to this path")
		configPath := fs.String("config", "", "optional .nightward.yml policy config")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		config, err := policy.LoadConfig(expandConfigPath(home, *configPath))
		if err != nil {
			return fail(stderr, "failed to load policy config: %v", err)
		}
		report := inventory.NewScanner(home).Scan()
		if err := policy.WriteSARIFWithConfig(report, *output, config); err != nil {
			return fail(stderr, "failed to write SARIF: %v", err)
		}
		fmt.Fprintf(stdout, "Wrote SARIF policy report to %s\n", *output)
	default:
		return fail(stderr, "usage: nightward policy <init|explain|check|sarif> [flags]")
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

func printHelp(w io.Writer, commandName string) {
	fmt.Fprintf(w, `Nightward watches local AI agent state before it leaks into dotfiles.

Usage:
  %[1]s                                Open the TUI
  %[1]s scan [--json] [--output FILE] [--output-dir DIR]
  %[1]s doctor [--json]
  %[1]s plan backup --target <repo> [--json]
  %[1]s adapters list [--json]
  %[1]s findings list [--json]
  %[1]s findings explain <finding-id> [--json]
  %[1]s fix plan [--finding <id>|--rule <rule>|--all] [--json]
  %[1]s fix preview [--finding <id>|--rule <rule>|--all] [--format diff|json|markdown]
  %[1]s fix export --format markdown|json
  %[1]s policy init --dry-run
  %[1]s policy explain
  %[1]s policy check [--config .nightward.yml] [--strict] [--json]
  %[1]s policy sarif [--config .nightward.yml] --output nightward.sarif
  %[1]s snapshot plan --target <dir> [--json]
  %[1]s snapshot diff --from <plan.json> --to <plan.json> [--json]
  %[1]s schedule plan --preset nightly [--json]
  %[1]s schedule install --preset nightly --dry-run [--json]
  %[1]s schedule remove --dry-run [--json]

Nightward does not mutate agent configs. It only writes explicit report/SARIF outputs and schedule install/remove files.

Canonical command: nightward
Short alias: nw
`, commandName)
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

func printFindingsList(w io.Writer, findings []inventory.Finding) {
	if len(findings) == 0 {
		fmt.Fprintln(w, "No findings.")
		return
	}
	for _, finding := range findings {
		fmt.Fprintf(w, "%s  %-8s %-22s %-12s %s\n", finding.ID, finding.Severity, finding.Rule, finding.Tool, finding.Message)
	}
}

func printFindingExplain(w io.Writer, finding inventory.Finding) {
	fmt.Fprintf(w, "%s\n", finding.ID)
	fmt.Fprintf(w, "  rule:      %s\n", finding.Rule)
	fmt.Fprintf(w, "  severity:  %s\n", finding.Severity)
	fmt.Fprintf(w, "  tool:      %s\n", finding.Tool)
	fmt.Fprintf(w, "  path:      %s\n", finding.Path)
	fmt.Fprintf(w, "  message:   %s\n", finding.Message)
	if finding.Evidence != "" {
		fmt.Fprintf(w, "  evidence:  %s\n", finding.Evidence)
	}
	if finding.Impact != "" {
		fmt.Fprintf(w, "\nImpact: %s\n", finding.Impact)
	}
	if finding.Why != "" {
		fmt.Fprintf(w, "Why this matters: %s\n", finding.Why)
	}
	if finding.FixAvailable {
		fmt.Fprintf(w, "\nSuggested fix (%s, confidence=%s, risk=%s):\n", finding.FixKind, finding.Confidence, finding.Risk)
		fmt.Fprintf(w, "  %s\n", finding.FixSummary)
		for i, step := range finding.FixSteps {
			fmt.Fprintf(w, "  %d. %s\n", i+1, step)
		}
		if finding.RequiresReview {
			fmt.Fprintln(w, "  review: required before applying manually")
		}
	} else {
		fmt.Fprintf(w, "\nRecommendation: %s\n", finding.Recommendation)
	}
}

func printFixPlan(w io.Writer, plan fixplan.Plan) {
	fmt.Fprintf(w, "Fix plan: total=%d safe=%d review=%d blocked=%d\n", plan.Summary.Total, plan.Summary.Safe, plan.Summary.Review, plan.Summary.Blocked)
	for _, fix := range plan.Fixes {
		fmt.Fprintf(w, "  %-7s %-20s %-22s %s\n", fix.Status, fix.FixKind, fix.Rule, fix.FindingID)
		if fix.Summary != "" {
			fmt.Fprintf(w, "    %s\n", fix.Summary)
		}
	}
}

func printPolicy(w io.Writer, report policy.Report) {
	status := "passed"
	if !report.Passed {
		status = "failed"
	}
	fmt.Fprintf(w, "Nightward policy %s: threshold=%s violations=%d total_findings=%d\n", status, report.Threshold, report.Summary.Violations, report.Summary.TotalFindings)
	for _, finding := range report.Violations {
		fmt.Fprintf(w, "  [%s] %s %s\n", finding.Severity, finding.Rule, finding.ID)
	}
}

func printSnapshotPlan(w io.Writer, plan snapshot.Plan) {
	fmt.Fprintf(w, "Snapshot dry-run plan for %s\n", plan.TargetRoot)
	fmt.Fprintf(w, "  total: %d  include: %d  review: %d  exclude: %d\n", plan.Summary.Total, plan.Summary.Include, plan.Summary.Review, plan.Summary.Excluded)
	for _, entry := range plan.Entries {
		fmt.Fprintf(w, "  %-7s %-12s %s -> %s\n", entry.Action, entry.Tool, entry.Source, entry.Target)
	}
}

func printSnapshotDiff(w io.Writer, diff snapshot.Diff) {
	fmt.Fprintf(w, "Snapshot diff: added=%d removed=%d changed=%d\n", diff.Summary.Added, diff.Summary.Removed, diff.Summary.Changed)
	for _, entry := range diff.Added {
		fmt.Fprintf(w, "  added   %-12s %s\n", entry.Tool, entry.Source)
	}
	for _, entry := range diff.Removed {
		fmt.Fprintf(w, "  removed %-12s %s\n", entry.Tool, entry.Source)
	}
	for _, change := range diff.Changed {
		fmt.Fprintf(w, "  changed %-12s %s (%s -> %s)\n", change.After.Tool, change.Source, change.Before.Action, change.After.Action)
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

func fixSelector(report inventory.Report, findingID, rule string, all bool) (fixplan.Selector, error) {
	selected := 0
	if findingID != "" {
		selected++
	}
	if rule != "" {
		selected++
	}
	if all {
		selected++
	}
	if selected > 1 {
		return fixplan.Selector{}, fmt.Errorf("choose only one of --finding, --rule, or --all")
	}
	if selected == 0 {
		all = true
	}
	if findingID != "" {
		finding, ok := fixplan.Find(report, findingID)
		if !ok {
			return fixplan.Selector{}, fmt.Errorf("finding not found or ambiguous: %s", findingID)
		}
		findingID = finding.ID
	}
	return fixplan.Selector{FindingID: findingID, Rule: rule, All: all}, nil
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

func expandConfigPath(home, path string) string {
	if path == "" {
		return ""
	}
	return expandHome(home, path)
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
