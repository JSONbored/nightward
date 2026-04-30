package inventory

import (
	"strings"
	"testing"
)

func TestPackageNameStripsMovingLatestTag(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{args: []string{"-y", "shadcn@latest"}, want: "shadcn"},
		{args: []string{"--package", "@modelcontextprotocol/server-filesystem", "mcp-server"}, want: "@modelcontextprotocol/server-filesystem"},
		{args: []string{"@playwright/mcp@latest"}, want: "@playwright/mcp"},
		{args: []string{"@modelcontextprotocol/server-filesystem"}, want: "@modelcontextprotocol/server-filesystem"},
	} {
		got, ok := packageName(tc.args)
		if !ok || got != tc.want {
			t.Fatalf("packageName(%v) = %q, %t; want %q, true", tc.args, got, ok, tc.want)
		}
	}
}

func TestHasPinnedPackageRejectsLatest(t *testing.T) {
	if hasPinnedPackage([]string{"shadcn@latest"}) {
		t.Fatal("@latest should not count as a pinned package")
	}
	if !hasPinnedPackage([]string{"@playwright/mcp@1.2.3"}) {
		t.Fatal("scoped package with explicit version should count as pinned")
	}
}

func TestRedactArgsRedactsSecretFlagValues(t *testing.T) {
	got := redactArgs([]string{"server", "--api-key", "super-secret-value", "--token=another-secret", "--path", "/tmp/project"})
	for _, leaked := range []string{"super-secret-value", "another-secret"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("redaction leaked %q in %q", leaked, got)
		}
	}
	if !strings.Contains(got, "/tmp/project") {
		t.Fatalf("redaction removed non-secret path: %q", got)
	}
}
