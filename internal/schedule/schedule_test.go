package schedule

import (
	"os"
	"path/filepath"
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
	limited := ReportHistory(reportDir, 1)
	if len(limited) != 1 || limited[0].Path != newReport {
		t.Fatalf("unexpected limited report history: %#v", limited)
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
