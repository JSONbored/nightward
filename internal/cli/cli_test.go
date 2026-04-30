package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadOnlyCommandsDoNotMutateHome(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, filepath.Join(home, ".mcp.json"), `{"mcpServers":{"demo":{"command":"node","args":["server.js"]}}}`)
	t.Setenv("NIGHTWARD_HOME", home)

	before := listTestFiles(t, home)
	commands := [][]string{
		{"scan", "--json"},
		{"doctor", "--json"},
		{"findings", "list", "--json"},
		{"fix", "plan", "--all", "--json"},
		{"fix", "export", "--format", "markdown"},
		{"policy", "check", "--json"},
	}
	for _, args := range commands {
		var stdout, stderr bytes.Buffer
		if code := RunWithName("nw", args, &stdout, &stderr); code != 0 {
			t.Fatalf("%s failed with %d: %s", strings.Join(args, " "), code, stderr.String())
		}
	}
	after := listTestFiles(t, home)
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatalf("read-only commands mutated home\nbefore=%v\nafter=%v", before, after)
	}
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
}

func listTestFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
