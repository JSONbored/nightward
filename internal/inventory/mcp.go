package inventory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type mcpServer struct {
	Name    string
	Command string
	Args    []string
	Env     map[string]string
	Raw     map[string]any
}

var secretKeyPattern = regexp.MustCompile(`(?i)(token|secret|password|passwd|api[_-]?key|auth|credential|private[_-]?key)`)

func inspectMCP(item Item, spec pathSpec) []Finding {
	if !spec.CheckMCP || item.Kind == "directory" {
		return nil
	}

	servers, err := readMCPServers(item.Path)
	if err != nil {
		return []Finding{{
			ID:             findingID("mcp_parse_failed", item.Tool, item.Path, err.Error()),
			Tool:           item.Tool,
			Path:           item.Path,
			Severity:       RiskMedium,
			Rule:           "mcp_parse_failed",
			Message:        "MCP config could not be parsed.",
			Evidence:       err.Error(),
			Recommendation: "Review this file manually before syncing or scheduling automated backups.",
			Impact:         "Nightward cannot reason about server commands, environment handling, or filesystem scope in this config.",
			Why:            "Unreadable agent config can hide local paths, shell wrappers, or credential material that should not be synced blindly.",
			FixAvailable:   true,
			FixKind:        FixManualReview,
			Confidence:     "high",
			Risk:           RiskLow,
			RequiresReview: true,
			FixSummary:     "Open the MCP config, correct the syntax, and rerun `nw scan --json`.",
			FixSteps: []string{
				"Validate the file as JSON or TOML, depending on its extension.",
				"Confirm each MCP server has an explicit command and reviewed arguments.",
				"Rerun `nw findings list --json` after the file parses cleanly.",
			},
		}}
	}

	var findings []Finding
	for _, server := range servers {
		findings = append(findings, reviewFinding(item, server))
		findings = append(findings, commandFindings(item, server)...)
		findings = append(findings, envFindings(item, server)...)
		findings = append(findings, argFindings(item, server)...)
	}
	return findings
}

func readMCPServers(path string) ([]mcpServer, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".toml":
		return readTOMLServers(contents)
	default:
		return readJSONServers(contents)
	}
}

func readJSONServers(contents []byte) ([]mcpServer, error) {
	var doc map[string]any
	if err := json.Unmarshal(contents, &doc); err != nil {
		return nil, err
	}

	var servers []mcpServer
	for _, key := range []string{"mcpServers", "servers"} {
		if group, ok := doc[key].(map[string]any); ok {
			servers = append(servers, mapServers(group)...)
		}
	}
	return servers, nil
}

func readTOMLServers(contents []byte) ([]mcpServer, error) {
	var doc map[string]any
	if err := toml.Unmarshal(contents, &doc); err != nil {
		return nil, err
	}

	var servers []mcpServer
	for _, key := range []string{"mcp_servers", "mcpServers", "servers"} {
		if group, ok := doc[key].(map[string]any); ok {
			servers = append(servers, mapServers(group)...)
		}
	}
	return servers, nil
}

func mapServers(group map[string]any) []mcpServer {
	var servers []mcpServer
	names := make([]string, 0, len(group))
	for name := range group {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		raw := group[name]
		serverMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		server := mcpServer{Name: name, Raw: serverMap, Env: map[string]string{}}
		server.Command = stringValue(serverMap["command"])
		server.Args = stringSlice(serverMap["args"])
		if env, ok := serverMap["env"].(map[string]any); ok {
			for key, value := range env {
				server.Env[key] = stringValue(value)
			}
		}
		servers = append(servers, server)
	}
	return servers
}

func reviewFinding(item Item, server mcpServer) Finding {
	command := server.Command
	if command == "" {
		command = "unknown"
	}
	return Finding{
		ID:             findingID("mcp_server_review", item.Tool, item.Path, server.Name),
		Tool:           item.Tool,
		Path:           item.Path,
		Severity:       RiskInfo,
		Rule:           "mcp_server_review",
		Message:        fmt.Sprintf("Review MCP server %q before syncing this config.", server.Name),
		Evidence:       fmt.Sprintf("command=%s", redact(command)),
		Recommendation: "Confirm the server source, permissions, and local path assumptions.",
		Impact:         "This server may execute local code or expose local files when an AI client invokes it.",
		Why:            "MCP configs are executable trust boundaries, so portable backups should preserve intent without hiding local-only risk.",
		FixAvailable:   true,
		FixKind:        FixIgnoreWithReason,
		Confidence:     "medium",
		Risk:           RiskLow,
		RequiresReview: true,
		FixSummary:     "Keep the server only if its source, permissions, and local assumptions are understood.",
		FixSteps: []string{
			"Identify the server package, binary, or script owner.",
			"Confirm the command and arguments are expected for this machine.",
			"Document why this server is safe to keep or exclude it from portable dotfiles.",
		},
	}
}

func commandFindings(item Item, server mcpServer) []Finding {
	command := strings.ToLower(filepath.Base(server.Command))
	var findings []Finding

	if command == "npx" || command == "uvx" || command == "pipx" {
		if !hasPinnedPackage(server.Args) {
			pkg, pkgOK := packageName(server.Args)
			finding := Finding{
				ID:             findingID("mcp_unpinned_package", item.Tool, item.Path, server.Name),
				Tool:           item.Tool,
				Path:           item.Path,
				Severity:       RiskHigh,
				Rule:           "mcp_unpinned_package",
				Message:        fmt.Sprintf("MCP server %q runs a package executor without an obvious pinned package version.", server.Name),
				Evidence:       fmt.Sprintf("command=%s args=%s", redact(server.Command), redact(strings.Join(server.Args, " "))),
				Recommendation: "Pin package versions or replace with a trusted local binary before syncing.",
				Impact:         "A future package publish or dependency change can alter code that the AI client executes locally.",
				Why:            "Unpinned package execution makes config restores non-reproducible and widens the supply-chain attack surface.",
				FixAvailable:   true,
				FixKind:        FixManualReview,
				Confidence:     "medium",
				Risk:           RiskMedium,
				RequiresReview: true,
				FixSummary:     "Identify the package name and pin it to a reviewed version.",
				FixSteps: []string{
					"Locate the package token in the MCP server args.",
					"Choose a reviewed version from the package registry or project release notes.",
					"Replace the package token with an explicit pinned version, then rerun `nw policy check --strict --json`.",
				},
			}
			if pkgOK {
				finding.FixKind = FixPinPackage
				finding.Confidence = "high"
				finding.FixSummary = fmt.Sprintf("Pin %s to an explicit version before syncing this MCP config.", pkg)
				finding.FixSteps = []string{
					fmt.Sprintf("Choose a reviewed version for %s.", pkg),
					fmt.Sprintf("Change the package arg from %q to %q.", pkg, pkg+"@<version>"),
					"Commit only the pinned package reference, not any local credential or cache files.",
				}
			}
			findings = append(findings, finding)
		}
	}

	if command == "sh" || command == "bash" || command == "zsh" || command == "cmd" || command == "powershell" || command == "pwsh" || hasArg(server.Args, "-c") {
		directCommand, directArgs, simple := simpleShellPassthrough(server)
		finding := Finding{
			ID:             findingID("mcp_shell_command", item.Tool, item.Path, server.Name),
			Tool:           item.Tool,
			Path:           item.Path,
			Severity:       RiskHigh,
			Rule:           "mcp_shell_command",
			Message:        fmt.Sprintf("MCP server %q executes through a shell.", server.Name),
			Evidence:       fmt.Sprintf("command=%s args=%s", redact(server.Command), redact(strings.Join(server.Args, " "))),
			Recommendation: "Prefer direct executable invocation and review the command before syncing.",
			Impact:         "Shell wrappers can hide compound commands, environment expansion, and shell-specific behavior.",
			Why:            "Direct executable invocation is easier to audit and less likely to preserve unsafe local shell assumptions in dotfiles.",
			FixAvailable:   true,
			FixKind:        FixManualReview,
			Confidence:     "medium",
			Risk:           RiskMedium,
			RequiresReview: true,
			FixSummary:     "Review the shell wrapper and replace it with a direct command when possible.",
			FixSteps: []string{
				"Inspect the full shell command locally.",
				"Confirm it is not chaining commands, expanding secrets, or depending on shell startup files.",
				"Replace the shell command with a direct executable invocation if the wrapper is only a passthrough.",
			},
		}
		if simple {
			finding.FixKind = FixReplaceShellWrapper
			finding.Confidence = "high"
			finding.FixSummary = fmt.Sprintf("Replace the shell wrapper with direct command %q.", directCommand)
			step := fmt.Sprintf("Set command to %q.", directCommand)
			if len(directArgs) > 0 {
				step = fmt.Sprintf("Set command to %q and args to [%s].", directCommand, quoteList(directArgs))
			}
			finding.FixSteps = []string{
				step,
				"Keep any environment variable references as references, not inline values.",
				"Rerun `nw findings explain " + finding.ID + "` to confirm the shell-wrapper finding is gone.",
			}
		}
		findings = append(findings, finding)
	}

	if server.Command == "" {
		findings = append(findings, Finding{
			ID:             findingID("mcp_unknown_command", item.Tool, item.Path, server.Name),
			Tool:           item.Tool,
			Path:           item.Path,
			Severity:       RiskMedium,
			Rule:           "mcp_unknown_command",
			Message:        fmt.Sprintf("MCP server %q does not declare a command Nightward can inspect.", server.Name),
			Recommendation: "Review this server manually before syncing.",
			Impact:         "Nightward cannot determine what executable would run for this MCP server.",
			Why:            "Missing or unsupported server shapes block reliable policy checks and backup decisions.",
			FixAvailable:   true,
			FixKind:        FixManualReview,
			Confidence:     "high",
			Risk:           RiskLow,
			RequiresReview: true,
			FixSummary:     "Add a standard command field or open an adapter issue with a redacted config shape.",
			FixSteps: []string{
				"Confirm whether the server uses a supported MCP config shape.",
				"Add an explicit command field if the client supports it.",
				"If the shape is valid but unsupported, file a Nightward adapter issue with secret values removed.",
			},
		})
	}

	return findings
}

func envFindings(item Item, server mcpServer) []Finding {
	var findings []Finding
	keys := make([]string, 0, len(server.Env))
	for key := range server.Env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if secretKeyPattern.MatchString(key) {
			value := server.Env[key]
			referenceOnly := value == "" || looksEnvReference(value)
			severity := RiskCritical
			risk := RiskHigh
			message := fmt.Sprintf("MCP server %q stores a sensitive environment key.", server.Name)
			summary := fmt.Sprintf("Move %s out of this config and into a local environment or secret manager.", key)
			steps := []string{
				fmt.Sprintf("Remove the inline value for %s from the MCP config.", key),
				fmt.Sprintf("Set %s in your shell profile, launchd environment, password manager CLI, or another local secret source.", key),
				"Keep only the env key name or documented setup prerequisite in portable dotfiles.",
			}
			confidence := "high"
			if referenceOnly {
				severity = RiskMedium
				risk = RiskLow
				message = fmt.Sprintf("MCP server %q references a sensitive environment key.", server.Name)
				summary = fmt.Sprintf("Document %s as a local prerequisite without committing its value.", key)
				steps = []string{
					fmt.Sprintf("Confirm %s is only referenced by name or environment interpolation.", key),
					"Document how to provide the secret locally without adding the value to dotfiles.",
					"Keep the real value in a password manager, OS keychain, or machine-local env file excluded from Git.",
				}
				confidence = "medium"
			}
			findings = append(findings, Finding{
				ID:             findingID("mcp_secret_env", item.Tool, item.Path, server.Name+key),
				Tool:           item.Tool,
				Path:           item.Path,
				Severity:       severity,
				Rule:           "mcp_secret_env",
				Message:        message,
				Evidence:       fmt.Sprintf("env_key=%s", key),
				Recommendation: "Keep secret values outside dotfiles and document required env names only.",
				Impact:         "Credential material in agent config can leak through dotfiles, backups, screenshots, or support bundles.",
				Why:            "Agent tools often bridge local files and remote models, so secrets should stay in dedicated local secret stores.",
				FixAvailable:   true,
				FixKind:        FixExternalizeSecret,
				Confidence:     confidence,
				Risk:           risk,
				RequiresReview: true,
				FixSummary:     summary,
				FixSteps:       steps,
			})
		}
	}
	return findings
}

func argFindings(item Item, server mcpServer) []Finding {
	joined := strings.Join(server.Args, " ")
	var findings []Finding
	if referencesBroadPath(server.Args) {
		findings = append(findings, Finding{
			ID:             findingID("mcp_broad_filesystem", item.Tool, item.Path, server.Name),
			Tool:           item.Tool,
			Path:           item.Path,
			Severity:       RiskMedium,
			Rule:           "mcp_broad_filesystem",
			Message:        fmt.Sprintf("MCP server %q appears to reference broad filesystem access.", server.Name),
			Evidence:       redact(joined),
			Recommendation: "Narrow filesystem access to explicit project/config paths where possible.",
			Impact:         "A broad mount can expose unrelated personal files or credentials to an MCP server.",
			Why:            "Least-privilege filesystem scope reduces accidental disclosure and makes portable configs easier to review.",
			FixAvailable:   true,
			FixKind:        FixNarrowFilesystem,
			Confidence:     "medium",
			Risk:           RiskMedium,
			RequiresReview: true,
			FixSummary:     "Replace broad filesystem arguments with the smallest project or config paths that server actually needs.",
			FixSteps: []string{
				"List the exact directories this MCP server needs for your workflow.",
				"Replace broad paths such as $HOME, ~, /, or full user roots with those explicit directories.",
				"Do not guess missing paths; rerun the tool workflow after narrowing scope.",
			},
		})
	}
	if referencesTokenPath(server.Args) {
		findings = append(findings, Finding{
			ID:             findingID("mcp_local_token_path", item.Tool, item.Path, server.Name),
			Tool:           item.Tool,
			Path:           item.Path,
			Severity:       RiskHigh,
			Rule:           "mcp_local_token_path",
			Message:        fmt.Sprintf("MCP server %q appears to reference local credential paths.", server.Name),
			Evidence:       redact(joined),
			Recommendation: "Keep credential paths local-only and avoid committing them to dotfiles.",
			Impact:         "Credential file paths are machine-local assumptions and can reveal where sensitive material is stored.",
			Why:            "Portable dotfiles should not encode local credential locations unless they are intentionally templated and ignored.",
			FixAvailable:   true,
			FixKind:        FixManualReview,
			Confidence:     "medium",
			Risk:           RiskMedium,
			RequiresReview: true,
			FixSummary:     "Move credential-path assumptions into machine-local config or documented setup steps.",
			FixSteps: []string{
				"Confirm whether the credential path is required by the server.",
				"Prefer environment references, keychain integration, or a machine-local ignored overlay.",
				"Exclude any file containing credential paths from public or shared dotfiles unless redacted.",
			},
		})
	}
	return findings
}

func packageName(args []string) (string, bool) {
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "" || arg == "--" {
			continue
		}
		if strings.HasPrefix(arg, "-") {
			if flagLikelyHasValue(arg) && i+1 < len(args) {
				i++
			}
			continue
		}
		if strings.Contains(arg, "/") && !strings.HasPrefix(arg, "@") {
			continue
		}
		if strings.Contains(arg, "\\") || strings.Contains(arg, "$") {
			continue
		}
		if hasPinnedPackage([]string{arg}) {
			continue
		}
		return packageBaseName(arg), true
	}
	return "", false
}

func packageBaseName(arg string) string {
	if strings.HasSuffix(arg, "@latest") {
		return strings.TrimSuffix(arg, "@latest")
	}
	return arg
}

func flagLikelyHasValue(arg string) bool {
	if strings.Contains(arg, "=") {
		return false
	}
	switch arg {
	case "-p", "--package", "--from", "--python", "--spec":
		return true
	default:
		return false
	}
}

func hasPinnedPackage(args []string) bool {
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") || strings.Contains(arg, "\\") {
			continue
		}
		if pinnedPackageArg(arg) {
			return true
		}
	}
	return false
}

func pinnedPackageArg(arg string) bool {
	at := strings.LastIndex(arg, "@")
	if at <= 0 {
		return false
	}
	if strings.HasPrefix(arg, "@") {
		at = strings.LastIndex(arg[1:], "@")
		if at < 0 {
			return false
		}
		at++
	}
	version := strings.TrimSpace(arg[at+1:])
	return version != "" && version != "latest"
}

func hasArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func simpleShellPassthrough(server mcpServer) (string, []string, bool) {
	command := strings.ToLower(filepath.Base(server.Command))
	if command != "sh" && command != "bash" && command != "zsh" {
		return "", nil, false
	}
	var script string
	for i, arg := range server.Args {
		if arg == "-c" && i+1 < len(server.Args) {
			script = strings.TrimSpace(server.Args[i+1])
			break
		}
	}
	if script == "" || strings.ContainsAny(script, "|;&<>`") || strings.Contains(script, "$(") {
		return "", nil, false
	}
	fields := strings.Fields(script)
	if len(fields) == 0 || strings.Contains(fields[0], "=") {
		return "", nil, false
	}
	return fields[0], fields[1:], true
}

func quoteList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return strings.Join(quoted, ", ")
}

func looksEnvReference(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return true
	}
	if strings.HasPrefix(trimmed, "$") {
		return true
	}
	if strings.Contains(trimmed, "${") && strings.Contains(trimmed, "}") {
		return true
	}
	return false
}

func referencesBroadPath(args []string) bool {
	for _, arg := range args {
		normalized := strings.TrimSpace(arg)
		if normalized == "~" || normalized == "$HOME" || normalized == "/" || strings.HasPrefix(normalized, "/Users/") {
			return true
		}
		if strings.Contains(normalized, "--mount") || strings.Contains(normalized, "--volume") {
			return true
		}
	}
	return false
}

func referencesTokenPath(args []string) bool {
	for _, arg := range args {
		lower := strings.ToLower(arg)
		if strings.Contains(lower, ".ssh") || strings.Contains(lower, ".aws") || strings.Contains(lower, ".npmrc") || strings.Contains(lower, ".netrc") || strings.Contains(lower, ".git-credentials") || strings.Contains(lower, "id_rsa") || strings.Contains(lower, "token") {
			return true
		}
	}
	return false
}

func stringValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func stringSlice(v any) []string {
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

func redact(value string) string {
	if value == "" {
		return value
	}
	parts := strings.Fields(value)
	for i, part := range parts {
		if secretKeyPattern.MatchString(part) || strings.Contains(part, "=") && secretKeyPattern.MatchString(strings.SplitN(part, "=", 2)[0]) {
			parts[i] = "[redacted]"
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	return value
}
