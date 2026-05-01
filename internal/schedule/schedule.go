package schedule

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jsonbored/nightward/internal/inventory"
)

const Label = "dev.nightward.scan"

var execCommand = exec.Command

type Plan struct {
	SchemaVersion int             `json:"schema_version"`
	Preset        string          `json:"preset"`
	Platform      string          `json:"platform"`
	ReportDir     string          `json:"report_dir"`
	LogDir        string          `json:"log_dir"`
	Command       []string        `json:"command"`
	Files         []GeneratedFile `json:"files"`
	Notes         []string        `json:"notes,omitempty"`
	Installed     bool            `json:"installed"`
	LastReport    string          `json:"last_report,omitempty"`
	LastRun       *time.Time      `json:"last_run,omitempty"`
	LastFindings  int             `json:"last_findings,omitempty"`
	History       []ReportRecord  `json:"history,omitempty"`
}

type GeneratedFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Mode    uint32 `json:"mode"`
}

type ReportRecord struct {
	Path               string                      `json:"path"`
	ModTime            time.Time                   `json:"mod_time"`
	Findings           int                         `json:"findings"`
	HighestSeverity    inventory.RiskLevel         `json:"highest_severity,omitempty"`
	FindingsBySeverity map[inventory.RiskLevel]int `json:"findings_by_severity,omitempty"`
	SizeBytes          int64                       `json:"size_bytes"`
	ReportName         string                      `json:"report_name"`
}

func BuildPlan(home, executable, preset string) (Plan, error) {
	return BuildPlanForOS(runtime.GOOS, home, executable, preset)
}

func BuildPlanForOS(goos, home, executable, preset string) (Plan, error) {
	if preset == "" {
		preset = "nightly"
	}
	if preset != "nightly" {
		return Plan{}, fmt.Errorf("unsupported preset %q", preset)
	}
	if executable == "" {
		executable = "nightward"
	}

	reportDir := filepath.Join(home, ".local", "state", "nightward", "reports")
	logDir := filepath.Join(home, ".local", "state", "nightward", "logs")
	command := []string{executable, "scan", "--json", "--output-dir", reportDir}
	plan := Plan{
		SchemaVersion: 1,
		Preset:        preset,
		Platform:      goos,
		ReportDir:     reportDir,
		LogDir:        logDir,
		Command:       command,
	}

	switch goos {
	case "darwin":
		path := filepath.Join(home, "Library", "LaunchAgents", Label+".plist")
		content, err := launchdPlist(command, logDir)
		if err != nil {
			return Plan{}, err
		}
		plan.Files = append(plan.Files, GeneratedFile{Path: path, Content: content, Mode: 0644})
	case "linux":
		servicePath := filepath.Join(home, ".config", "systemd", "user", Label+".service")
		timerPath := filepath.Join(home, ".config", "systemd", "user", Label+".timer")
		plan.Files = append(plan.Files,
			GeneratedFile{Path: servicePath, Content: systemdService(command, logDir), Mode: 0644},
			GeneratedFile{Path: timerPath, Content: systemdTimer(), Mode: 0644},
		)
	default:
		plan.Files = append(plan.Files, GeneratedFile{Path: "crontab", Content: cronLine(command), Mode: 0644})
		plan.Notes = append(plan.Notes, "Cron is generated as text only; v1 does not mutate crontab automatically.")
	}

	status := Status(home)
	plan.Installed = status.Installed
	plan.LastReport = status.LastReport
	plan.LastRun = status.LastRun
	plan.LastFindings = status.LastFindings
	plan.History = status.History
	return plan, nil
}

func Status(home string) Plan {
	reportDir := filepath.Join(home, ".local", "state", "nightward", "reports")
	status := Plan{
		SchemaVersion: 1,
		Preset:        "nightly",
		Platform:      runtime.GOOS,
		ReportDir:     reportDir,
		LogDir:        filepath.Join(home, ".local", "state", "nightward", "logs"),
	}
	switch runtime.GOOS {
	case "darwin":
		status.Installed = fileExists(filepath.Join(home, "Library", "LaunchAgents", Label+".plist"))
	case "linux":
		status.Installed = fileExists(filepath.Join(home, ".config", "systemd", "user", Label+".timer"))
	}
	status.LastReport, status.LastRun, status.LastFindings = lastReport(reportDir)
	status.History = ReportHistory(reportDir, 5)
	return status
}

func Install(home, executable, preset string) (Plan, error) {
	plan, err := BuildPlan(home, executable, preset)
	if err != nil {
		return Plan{}, err
	}
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return plan, errors.New("automatic schedule install is only supported for launchd and systemd user timers in v1")
	}
	if err := os.MkdirAll(plan.ReportDir, 0700); err != nil {
		return plan, err
	}
	if err := os.MkdirAll(plan.LogDir, 0700); err != nil {
		return plan, err
	}
	for _, file := range plan.Files {
		if err := os.MkdirAll(filepath.Dir(file.Path), 0700); err != nil {
			return plan, err
		}
		if err := os.WriteFile(file.Path, []byte(file.Content), os.FileMode(file.Mode)); err != nil {
			return plan, err
		}
	}
	if runtime.GOOS == "darwin" {
		uid := strconv.Itoa(os.Getuid())
		_ = execCommand("launchctl", "bootout", "gui/"+uid, plan.Files[0].Path).Run()                       // #nosec G204 -- fixed system command, generated user LaunchAgent path, no shell.
		if err := execCommand("launchctl", "bootstrap", "gui/"+uid, plan.Files[0].Path).Run(); err != nil { // #nosec G204 -- fixed system command, generated user LaunchAgent path, no shell.
			return plan, fmt.Errorf("wrote launchd plist but failed to bootstrap: %w", err)
		}
	} else if runtime.GOOS == "linux" {
		_ = execCommand("systemctl", "--user", "daemon-reload").Run()
		if err := execCommand("systemctl", "--user", "enable", "--now", Label+".timer").Run(); err != nil {
			return plan, fmt.Errorf("wrote systemd timer but failed to enable: %w", err)
		}
	}
	plan.Installed = true
	return plan, nil
}

func Remove(home string) (Plan, error) {
	plan := Status(home)
	switch runtime.GOOS {
	case "darwin":
		path := filepath.Join(home, "Library", "LaunchAgents", Label+".plist")
		uid := strconv.Itoa(os.Getuid())
		_ = execCommand("launchctl", "bootout", "gui/"+uid, path).Run() // #nosec G204 -- fixed system command, generated user LaunchAgent path, no shell.
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return plan, err
		}
	case "linux":
		_ = execCommand("systemctl", "--user", "disable", "--now", Label+".timer").Run()
		for _, path := range []string{
			filepath.Join(home, ".config", "systemd", "user", Label+".service"),
			filepath.Join(home, ".config", "systemd", "user", Label+".timer"),
		} {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return plan, err
			}
		}
		_ = execCommand("systemctl", "--user", "daemon-reload").Run()
	default:
		return plan, errors.New("automatic schedule remove is only supported for launchd and systemd user timers in v1")
	}
	plan.Installed = false
	return plan, nil
}

func launchdPlist(command []string, logDir string) (string, error) {
	type key string
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.WriteString("<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n")
	buf.WriteString("<plist version=\"1.0\">\n<dict>\n")
	writeKey := func(k key) { buf.WriteString(fmt.Sprintf("  <key>%s</key>\n", k)) }
	writeKey("Label")
	buf.WriteString(fmt.Sprintf("  <string>%s</string>\n", Label))
	writeKey("ProgramArguments")
	buf.WriteString("  <array>\n")
	for _, arg := range command {
		escaped, err := xmlEscape(arg)
		if err != nil {
			return "", err
		}
		buf.WriteString(fmt.Sprintf("    <string>%s</string>\n", escaped))
	}
	buf.WriteString("  </array>\n")
	writeKey("StartCalendarInterval")
	buf.WriteString("  <dict>\n    <key>Hour</key>\n    <integer>2</integer>\n    <key>Minute</key>\n    <integer>17</integer>\n  </dict>\n")
	writeKey("StandardOutPath")
	buf.WriteString(fmt.Sprintf("  <string>%s</string>\n", filepath.Join(logDir, "nightward.out.log")))
	writeKey("StandardErrorPath")
	buf.WriteString(fmt.Sprintf("  <string>%s</string>\n", filepath.Join(logDir, "nightward.err.log")))
	writeKey("RunAtLoad")
	buf.WriteString("  <false/>\n</dict>\n</plist>\n")
	return buf.String(), nil
}

func systemdService(command []string, logDir string) string {
	return fmt.Sprintf(`[Unit]
Description=Nightward AI agent state scan

[Service]
Type=oneshot
ExecStart=%s
Environment=NIGHTWARD_LOG_DIR=%s
`, shellJoin(command), logDir)
}

func systemdTimer() string {
	return `[Unit]
Description=Nightly Nightward AI agent state scan

[Timer]
OnCalendar=*-*-* 02:17:00
Persistent=true

[Install]
WantedBy=timers.target
`
}

func cronLine(command []string) string {
	return fmt.Sprintf("17 2 * * * %s\n", shellJoin(command))
}

func shellJoin(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.ContainsAny(arg, " \t\n'\"") {
			quoted = append(quoted, "'"+strings.ReplaceAll(arg, "'", "'\\''")+"'")
		} else {
			quoted = append(quoted, arg)
		}
	}
	return strings.Join(quoted, " ")
}

func xmlEscape(value string) (string, error) {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(value)); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func lastReport(reportDir string) (string, *time.Time, int) {
	history := ReportHistory(reportDir, 1)
	if len(history) == 0 {
		return "", nil, 0
	}
	mod := history[0].ModTime
	return history[0].Path, &mod, history[0].Findings
}

func ReportHistory(reportDir string, limit int) []ReportRecord {
	entries, err := os.ReadDir(reportDir)
	if err != nil {
		return nil
	}
	var history []ReportRecord
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(reportDir, entry.Name())
		count, bySeverity, highest := reportFindingSummary(path)
		history = append(history, ReportRecord{
			Path:               path,
			ModTime:            info.ModTime().UTC(),
			Findings:           count,
			HighestSeverity:    highest,
			FindingsBySeverity: bySeverity,
			SizeBytes:          info.Size(),
			ReportName:         entry.Name(),
		})
	}
	sort.Slice(history, func(i, j int) bool {
		return history[i].ModTime.After(history[j].ModTime)
	})
	if limit > 0 && len(history) > limit {
		return history[:limit]
	}
	return history
}

func countFindings(path string) int {
	count, _, _ := reportFindingSummary(path)
	return count
}

func reportFindingSummary(path string) (int, map[inventory.RiskLevel]int, inventory.RiskLevel) {
	contents, err := os.ReadFile(filepath.Clean(path)) // #nosec G304 -- path is selected from the private Nightward report directory.
	if err != nil {
		return 0, nil, ""
	}
	var parsed struct {
		Summary struct {
			TotalFindings      int                         `json:"total_findings"`
			FindingsBySeverity map[inventory.RiskLevel]int `json:"findings_by_severity"`
		} `json:"summary"`
		Findings []struct {
			Severity inventory.RiskLevel `json:"severity"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(contents, &parsed); err == nil {
		bySeverity := parsed.Summary.FindingsBySeverity
		if bySeverity == nil {
			bySeverity = map[inventory.RiskLevel]int{}
			for _, finding := range parsed.Findings {
				if finding.Severity != "" {
					bySeverity[finding.Severity]++
				}
			}
		}
		count := parsed.Summary.TotalFindings
		if count == 0 {
			for _, value := range bySeverity {
				count += value
			}
			if count == 0 {
				count = len(parsed.Findings)
			}
		}
		return count, bySeverity, highestSeverity(bySeverity)
	}
	return strings.Count(string(contents), `"severity"`), nil, ""
}

func highestSeverity(counts map[inventory.RiskLevel]int) inventory.RiskLevel {
	for _, severity := range []inventory.RiskLevel{inventory.RiskCritical, inventory.RiskHigh, inventory.RiskMedium, inventory.RiskLow, inventory.RiskInfo} {
		if counts[severity] > 0 {
			return severity
		}
	}
	return ""
}
