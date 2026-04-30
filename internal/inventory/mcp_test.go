package inventory

import "testing"

func TestPackageNameStripsMovingLatestTag(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{args: []string{"-y", "shadcn@latest"}, want: "shadcn"},
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
