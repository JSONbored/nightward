package schedule

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBuildPlanForDarwin(t *testing.T) {
	plan, err := BuildPlanForOS("darwin", "/Users/test", "/usr/local/bin/nightward", "nightly")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Files) != 1 {
		t.Fatalf("expected one launchd file, got %d", len(plan.Files))
	}
	if !strings.Contains(plan.Files[0].Path, "Library/LaunchAgents/dev.nightward.scan.plist") {
		t.Fatalf("unexpected launchd path: %s", plan.Files[0].Path)
	}
	if !strings.Contains(plan.Files[0].Content, "--output-dir") {
		t.Fatal("launchd plist does not write reports to output dir")
	}
}

func TestBuildPlanForLinux(t *testing.T) {
	plan, err := BuildPlanForOS("linux", "/home/test", "/usr/local/bin/nightward", "nightly")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Files) != 2 {
		t.Fatalf("expected service and timer files, got %d", len(plan.Files))
	}
	if !strings.Contains(plan.Files[1].Content, "OnCalendar=*-*-* 02:17:00") {
		t.Fatal("systemd timer does not use the nightly schedule")
	}
}

func TestBuildPlanForUnsupportedOSReturnsCronTextOnly(t *testing.T) {
	plan, err := BuildPlanForOS("freebsd", "/home/test", "nightward", "nightly")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Files) != 1 || plan.Files[0].Path != "crontab" {
		t.Fatalf("expected crontab text fallback, got %#v", plan.Files)
	}
	if len(plan.Notes) == 0 {
		t.Fatal("expected cron mutation warning note")
	}
}

func TestBuildPlanRejectsUnsupportedPreset(t *testing.T) {
	if _, err := BuildPlanForOS("linux", "/home/test", "nightward", "hourly"); err == nil {
		t.Fatal("expected unsupported preset error")
	}
}

func TestInstallAndRemoveUseGeneratedFilesWithoutSystemMutation(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("automatic install/remove only supports launchd or systemd")
	}
	home := t.TempDir()
	originalExecCommand := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		commandArgs := append([]string{"-test.run=TestScheduleHelperProcess", "--", name}, args...)
		cmd := exec.Command(os.Args[0], commandArgs...)
		cmd.Env = append(os.Environ(), "NIGHTWARD_SCHEDULE_HELPER=1")
		return cmd
	}
	t.Cleanup(func() { execCommand = originalExecCommand })

	plan, err := Install(home, "nightward", "nightly")
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Installed {
		t.Fatal("expected installed plan")
	}
	if _, err := os.Stat(plan.ReportDir); err != nil {
		t.Fatalf("expected report dir: %v", err)
	}
	if _, err := os.Stat(plan.LogDir); err != nil {
		t.Fatalf("expected log dir: %v", err)
	}
	for _, file := range plan.Files {
		if _, err := os.Stat(file.Path); err != nil {
			t.Fatalf("expected generated schedule file %s: %v", file.Path, err)
		}
	}

	removed, err := Remove(home)
	if err != nil {
		t.Fatal(err)
	}
	if removed.Installed {
		t.Fatal("expected removed plan to report not installed")
	}
	for _, file := range plan.Files {
		if _, err := os.Stat(file.Path); !os.IsNotExist(err) {
			t.Fatalf("expected generated schedule file to be removed, path=%s err=%v", file.Path, err)
		}
	}
}

func TestScheduleHelperProcess(t *testing.T) {
	if os.Getenv("NIGHTWARD_SCHEDULE_HELPER") != "1" {
		return
	}
	os.Exit(0)
}

func TestStatusReadsLatestReportAndFindingCount(t *testing.T) {
	home := t.TempDir()
	reportDir := filepath.Join(home, ".local", "state", "nightward", "reports")
	if err := os.MkdirAll(reportDir, 0700); err != nil {
		t.Fatal(err)
	}
	oldReport := filepath.Join(reportDir, "old.json")
	newReport := filepath.Join(reportDir, "new.json")
	if err := os.WriteFile(oldReport, []byte(`{"findings":[{"severity":"low"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newReport, []byte(`{"findings":[{"severity":"medium"},{"severity":"high"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Date(2026, 4, 30, 1, 0, 0, 0, time.UTC)
	newTime := time.Date(2026, 4, 30, 2, 0, 0, 0, time.UTC)
	if err := os.Chtimes(oldReport, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newReport, newTime, newTime); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "notes.txt"), []byte("not a report"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(newReport, filepath.Join(reportDir, "linked.json")); err != nil {
		t.Logf("skipping symlink fixture: %v", err)
	}

	status := Status(home)
	if status.LastReport != newReport || status.LastRun == nil || !status.LastRun.Equal(newTime) {
		t.Fatalf("unexpected latest report status: %#v", status)
	}
	if status.LastFindings != 2 {
		t.Fatalf("expected two findings, got %d", status.LastFindings)
	}
	if len(status.History) != 2 || status.History[0].ReportName != "new.json" || status.History[1].ReportName != "old.json" {
		t.Fatalf("unexpected report history: %#v", status.History)
	}
	if status.History[0].Findings != 2 || status.History[1].Findings != 1 {
		t.Fatalf("unexpected report history finding counts: %#v", status.History)
	}
	if status.History[0].HighestSeverity != "high" || status.History[0].FindingsBySeverity["high"] != 1 {
		t.Fatalf("unexpected report history severity summary: %#v", status.History[0])
	}
	limited := ReportHistory(reportDir, 1)
	if len(limited) != 1 || limited[0].Path != newReport {
		t.Fatalf("unexpected limited report history: %#v", limited)
	}
}

func TestReportFindingSummaryFallbacks(t *testing.T) {
	dir := t.TempDir()
	summaryOnly := filepath.Join(dir, "summary.json")
	if err := os.WriteFile(summaryOnly, []byte(`{"summary":{"total_findings":3,"findings_by_severity":{"critical":1,"info":2}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	count, bySeverity, highest := reportFindingSummary(summaryOnly)
	if count != 3 || bySeverity["critical"] != 1 || highest != "critical" || countFindings(summaryOnly) != 3 {
		t.Fatalf("unexpected summary-only report parsing count=%d bySeverity=%#v highest=%s", count, bySeverity, highest)
	}

	malformed := filepath.Join(dir, "malformed.json")
	if err := os.WriteFile(malformed, []byte(`{"severity":"high"}{"severity":"medium"}`), 0600); err != nil {
		t.Fatal(err)
	}
	count, bySeverity, highest = reportFindingSummary(malformed)
	if count != 2 || bySeverity != nil || highest != "" {
		t.Fatalf("unexpected malformed report fallback count=%d bySeverity=%#v highest=%s", count, bySeverity, highest)
	}

	count, bySeverity, highest = reportFindingSummary(filepath.Join(dir, "missing.json"))
	if count != 0 || bySeverity != nil || highest != "" {
		t.Fatalf("unexpected missing report summary count=%d bySeverity=%#v highest=%s", count, bySeverity, highest)
	}
}

func TestEscapingHelpers(t *testing.T) {
	joined := shellJoin([]string{"nightward", "scan", "--path", "/tmp/has space/it's"})
	if !strings.Contains(joined, `'/tmp/has space/it'\''s'`) {
		t.Fatalf("unexpected shell join: %s", joined)
	}
	escaped, err := xmlEscape(`a&b"c`)
	if err != nil {
		t.Fatal(err)
	}
	if escaped != `a&amp;b&#34;c` {
		t.Fatalf("unexpected xml escape: %s", escaped)
	}
}
