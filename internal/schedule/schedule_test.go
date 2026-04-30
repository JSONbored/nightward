package schedule

import (
	"strings"
	"testing"
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
