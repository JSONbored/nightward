package fixplan

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jsonbored/nightward/internal/inventory"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

type Preview struct {
	GeneratedAt time.Time      `json:"generated_at"`
	Summary     PreviewSummary `json:"summary"`
	Patches     []PatchPreview `json:"patches"`
}

type PreviewSummary struct {
	Total     int `json:"total"`
	Patchable int `json:"patchable"`
	Review    int `json:"review"`
	Blocked   int `json:"blocked"`
}

type PatchPreview struct {
	FindingID      string              `json:"finding_id"`
	Tool           string              `json:"tool"`
	Path           string              `json:"path"`
	Server         string              `json:"server,omitempty"`
	Rule           string              `json:"rule"`
	FixKind        inventory.FixKind   `json:"fix_kind,omitempty"`
	Severity       inventory.RiskLevel `json:"severity"`
	RequiresReview bool                `json:"requires_review"`
	PatchAvailable bool                `json:"patch_available"`
	Reason         string              `json:"reason"`
	Diff           string              `json:"diff,omitempty"`
	Steps          []string            `json:"steps,omitempty"`
}

type parsedMCP struct {
	server map[string]any
}

var previewSecretKeyPattern = regexp.MustCompile(`(?i)(token|secret|password|passwd|api[_-]?key|auth|credential|private[_-]?key)`)

func BuildPreview(report inventory.Report, selector Selector) Preview {
	preview := Preview{GeneratedAt: report.GeneratedAt}
	for _, finding := range report.Findings {
		if !matches(finding, selector) {
			continue
		}
		patch := previewFinding(finding)
		preview.Patches = append(preview.Patches, patch)
		preview.Summary.Total++
		switch {
		case patch.PatchAvailable:
			preview.Summary.Patchable++
		case patch.RequiresReview:
			preview.Summary.Review++
		default:
			preview.Summary.Blocked++
		}
	}
	sort.Slice(preview.Patches, func(i, j int) bool {
		if preview.Patches[i].PatchAvailable == preview.Patches[j].PatchAvailable {
			return preview.Patches[i].FindingID < preview.Patches[j].FindingID
		}
		return preview.Patches[i].PatchAvailable
	})
	return preview
}

func PreviewMarkdown(preview Preview) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Nightward Fix Preview\n\n")
	fmt.Fprintf(&b, "Generated: %s\n\n", preview.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "Total: %d  Patchable: %d  Review: %d  Blocked: %d\n\n", preview.Summary.Total, preview.Summary.Patchable, preview.Summary.Review, preview.Summary.Blocked)
	if len(preview.Patches) == 0 {
		b.WriteString("No findings matched the selected preview.\n")
		return b.String()
	}
	for _, patch := range preview.Patches {
		fmt.Fprintf(&b, "## %s\n\n", patch.FindingID)
		fmt.Fprintf(&b, "- Tool: `%s`\n", patch.Tool)
		fmt.Fprintf(&b, "- Path: `%s`\n", patch.Path)
		if patch.Server != "" {
			fmt.Fprintf(&b, "- Server: `%s`\n", patch.Server)
		}
		fmt.Fprintf(&b, "- Rule: `%s`\n", patch.Rule)
		fmt.Fprintf(&b, "- Patch available: `%t`\n", patch.PatchAvailable)
		fmt.Fprintf(&b, "- Requires review: `%t`\n", patch.RequiresReview)
		fmt.Fprintf(&b, "- Reason: %s\n", patch.Reason)
		if patch.Diff != "" {
			fmt.Fprintf(&b, "\n```diff\n%s```\n", patch.Diff)
		}
		if len(patch.Steps) > 0 {
			b.WriteString("\nSteps:\n")
			for i, step := range patch.Steps {
				fmt.Fprintf(&b, "%d. %s\n", i+1, step)
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func PreviewDiff(preview Preview) string {
	var b strings.Builder
	for _, patch := range preview.Patches {
		if patch.Diff == "" {
			continue
		}
		b.WriteString(patch.Diff)
		if !strings.HasSuffix(patch.Diff, "\n") {
			b.WriteByte('\n')
		}
	}
	if b.Len() == 0 {
		return "# No redacted patch previews are available for the selected findings.\n"
	}
	return b.String()
}

func previewFinding(finding inventory.Finding) PatchPreview {
	patch := PatchPreview{
		FindingID:      finding.ID,
		Tool:           finding.Tool,
		Path:           finding.Path,
		Server:         finding.Server,
		Rule:           finding.Rule,
		FixKind:        finding.FixKind,
		Severity:       finding.Severity,
		RequiresReview: finding.RequiresReview,
		Reason:         "No structured patch hint is available for this finding.",
		Steps:          finding.FixSteps,
	}
	if finding.PatchHint == nil {
		return patch
	}
	if finding.Server == "" {
		patch.Reason = "Nightward cannot target this patch because the MCP server name is unknown."
		return patch
	}
	if finding.PatchHint.Kind == inventory.FixPinPackage {
		patch.Reason = "Patch preview blocked: Nightward will not guess a package version. Choose a reviewed version and edit the package token manually."
		return patch
	}
	parsed, err := parseMCPServer(finding.Path, finding.Server)
	if err != nil {
		patch.Reason = "Patch preview blocked: " + err.Error()
		return patch
	}
	switch finding.PatchHint.Kind {
	case inventory.FixExternalizeSecret:
		return previewExternalizeSecret(patch, parsed, finding.PatchHint)
	case inventory.FixReplaceShellWrapper:
		return previewReplaceShellWrapper(patch, parsed, finding.PatchHint)
	case inventory.FixNarrowFilesystem:
		return previewNarrowFilesystem(patch, parsed, finding.PatchHint)
	case inventory.FixManualReview:
		return previewManualReview(patch, parsed, finding, finding.PatchHint)
	default:
		return patch
	}
}

func previewExternalizeSecret(patch PatchPreview, parsed parsedMCP, hint *inventory.PatchHint) PatchPreview {
	if !hint.InlineSecret {
		patch.Reason = "No patch needed: this finding references a sensitive env key without an inline value."
		return patch
	}
	if hint.EnvKey == "" {
		patch.Reason = "Patch preview blocked: env key is unknown."
		return patch
	}
	if hint.HeaderKey != "" {
		return previewExternalizeHeader(patch, parsed, hint)
	}
	env, ok := parsed.server["env"].(map[string]any)
	if !ok {
		patch.Reason = "Patch preview blocked: server env map is missing or unsupported."
		return patch
	}
	if _, ok := env[hint.EnvKey]; !ok {
		patch.Reason = "Patch preview blocked: env key was not found in the parsed server."
		return patch
	}
	patch.PatchAvailable = true
	patch.Reason = "Redacted preview replaces the inline secret with an environment reference. Review before editing the config."
	patch.Diff = redactedHunk(patch.Path, patch.Server,
		[]string{fmt.Sprintf("env.%s = %q", hint.EnvKey, "[redacted]")},
		[]string{fmt.Sprintf("env.%s = %q", hint.EnvKey, "${"+hint.EnvKey+"}")},
	)
	return patch
}

func previewExternalizeHeader(patch PatchPreview, parsed parsedMCP, hint *inventory.PatchHint) PatchPreview {
	headers, ok := parsed.server["headers"].(map[string]any)
	if !ok {
		patch.Reason = "Patch preview blocked: server headers map is missing or unsupported."
		return patch
	}
	if _, ok := headers[hint.HeaderKey]; !ok {
		patch.Reason = "Patch preview blocked: header key was not found in the parsed server."
		return patch
	}
	patch.PatchAvailable = true
	patch.Reason = "Redacted preview replaces the inline header secret with an environment reference. Review before editing the config."
	patch.Diff = redactedHunk(patch.Path, patch.Server,
		[]string{fmt.Sprintf("headers.%s = %q", hint.HeaderKey, "[redacted]")},
		[]string{fmt.Sprintf("headers.%s = %q", hint.HeaderKey, "${"+hint.EnvKey+"}")},
	)
	return patch
}

func previewReplaceShellWrapper(patch PatchPreview, parsed parsedMCP, hint *inventory.PatchHint) PatchPreview {
	if hint.DirectCommand == "" {
		patch.Reason = "Patch preview blocked: direct command is unknown."
		return patch
	}
	currentCommand := redactString(stringValue(parsed.server["command"]))
	currentArgs := redactArgs(stringSlice(parsed.server["args"]))
	nextArgs := redactArgs(hint.DirectArgs)
	oldLines := []string{fmt.Sprintf("command = %q", currentCommand)}
	if len(currentArgs) > 0 {
		oldLines = append(oldLines, fmt.Sprintf("args = [%s]", quoteStrings(currentArgs)))
	}
	newLines := []string{fmt.Sprintf("command = %q", hint.DirectCommand)}
	if len(nextArgs) > 0 {
		newLines = append(newLines, fmt.Sprintf("args = [%s]", quoteStrings(nextArgs)))
	} else {
		newLines = append(newLines, "args = []")
	}
	patch.PatchAvailable = true
	patch.Reason = "Redacted preview replaces a simple shell passthrough with direct executable invocation. Review before editing the config."
	patch.Diff = redactedHunk(patch.Path, patch.Server, oldLines, newLines)
	return patch
}

func previewNarrowFilesystem(patch PatchPreview, parsed parsedMCP, hint *inventory.PatchHint) PatchPreview {
	currentArgs := redactArgs(stringSlice(parsed.server["args"]))
	nextArgs := redactArgs(hint.DirectArgs)
	if len(currentArgs) == 0 || len(nextArgs) == 0 {
		patch.Reason = "Review-only preview blocked: argument shape is missing or unsupported."
		return patch
	}
	patch.Reason = "Review-only preview narrows broad filesystem arguments with placeholders. Choose exact paths before editing."
	patch.Diff = redactedHunk(patch.Path, patch.Server,
		[]string{fmt.Sprintf("args = [%s]", quoteStrings(currentArgs))},
		[]string{fmt.Sprintf("args = [%s]", quoteStrings(nextArgs))},
	)
	return patch
}

func previewManualReview(patch PatchPreview, parsed parsedMCP, finding inventory.Finding, hint *inventory.PatchHint) PatchPreview {
	switch finding.Rule {
	case "mcp_local_endpoint":
		currentURL := redactString(stringValue(parsed.server["url"]))
		if currentURL == "" {
			patch.Reason = "Review-only preview blocked: server URL is missing or unsupported."
			return patch
		}
		replacement := hint.Replacement
		if replacement == "" {
			replacement = "<reviewed-portable-or-local-overlay-url>"
		}
		patch.Reason = "Review-only preview marks the local endpoint as a portability decision. Keep machine-local URLs out of shared dotfiles."
		patch.Diff = redactedHunk(patch.Path, patch.Server,
			[]string{fmt.Sprintf("url = %q", currentURL)},
			[]string{fmt.Sprintf("url = %q", replacement)},
		)
	case "mcp_local_token_path":
		currentArgs := redactArgs(stringSlice(parsed.server["args"]))
		nextArgs := redactArgs(hint.DirectArgs)
		if len(currentArgs) == 0 || len(nextArgs) == 0 {
			patch.Reason = "Review-only preview blocked: credential path arguments are missing or unsupported."
			return patch
		}
		replacement := hint.Replacement
		if replacement == "" {
			replacement = "<local-secret-path-kept-out-of-dotfiles>"
		}
		for i, arg := range nextArgs {
			lower := strings.ToLower(arg)
			if strings.Contains(lower, "token") || strings.Contains(lower, "credential") || strings.Contains(lower, "secret") {
				nextArgs[i] = replacement
			}
		}
		patch.Reason = "Review-only preview keeps credential path assumptions machine-local. Choose an ignored local overlay before editing."
		patch.Diff = redactedHunk(patch.Path, patch.Server,
			[]string{fmt.Sprintf("args = [%s]", quoteStrings(currentArgs))},
			[]string{fmt.Sprintf("args = [%s]", quoteStrings(nextArgs))},
		)
	}
	return patch
}

func parseMCPServer(path, serverName string) (parsedMCP, error) {
	contents, err := os.ReadFile(filepath.Clean(path)) // #nosec G304 -- preview reads scanner-discovered local MCP config without mutating it.
	if err != nil {
		return parsedMCP{}, err
	}
	var doc map[string]any
	switch strings.ToLower(filepath.Ext(path)) {
	case ".toml":
		if err := toml.Unmarshal(contents, &doc); err != nil {
			return parsedMCP{}, err
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(contents, &doc); err != nil {
			return parsedMCP{}, err
		}
	default:
		if err := json.Unmarshal(contents, &doc); err != nil {
			return parsedMCP{}, err
		}
	}
	for _, groupName := range []string{"mcpServers", "mcp_servers", "servers"} {
		group, ok := doc[groupName].(map[string]any)
		if !ok {
			continue
		}
		server, ok := group[serverName].(map[string]any)
		if !ok {
			continue
		}
		return parsedMCP{server: server}, nil
	}
	return parsedMCP{}, fmt.Errorf("server %q was not found in a supported MCP config shape", serverName)
}

func redactedHunk(path, server string, oldLines, newLines []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "--- %s\n", path)
	fmt.Fprintf(&b, "+++ %s\n", path)
	fmt.Fprintf(&b, "@@ MCP server %q @@\n", server)
	for _, line := range oldLines {
		fmt.Fprintf(&b, "-%s\n", line)
	}
	for _, line := range newLines {
		fmt.Fprintf(&b, "+%s\n", line)
	}
	return b.String()
}

func stringValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func stringSlice(v any) []string {
	if values, ok := v.([]string); ok {
		return append([]string(nil), values...)
	}
	values, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if s, ok := value.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func redactArgs(values []string) []string {
	out := append([]string(nil), values...)
	redactNext := false
	for i, value := range values {
		if redactNext {
			out[i] = "[redacted]"
			redactNext = false
			continue
		}
		if secretFlag(value) {
			out[i] = "[redacted]"
			redactNext = true
			continue
		}
		if secretAssignment(value) || previewSecretKeyPattern.MatchString(value) {
			out[i] = "[redacted]"
		}
	}
	return out
}

func redactString(value string) string {
	return strings.Join(redactArgs(strings.Fields(value)), " ")
}

func secretAssignment(value string) bool {
	if !strings.Contains(value, "=") {
		return false
	}
	key := strings.TrimLeft(strings.SplitN(value, "=", 2)[0], "-")
	return previewSecretKeyPattern.MatchString(key)
}

func secretFlag(value string) bool {
	if strings.Contains(value, "=") {
		return false
	}
	trimmed := strings.TrimLeft(value, "-")
	return trimmed != "" && previewSecretKeyPattern.MatchString(trimmed)
}

func quoteStrings(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, strconv.Quote(value))
	}
	return strings.Join(quoted, ", ")
}
