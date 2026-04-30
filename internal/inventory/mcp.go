package inventory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type mcpServer struct {
	Name    string
	Command string
	Args    []string
	EnvKeys []string
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
	for name, raw := range group {
		serverMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		server := mcpServer{Name: name, Raw: serverMap}
		server.Command = stringValue(serverMap["command"])
		server.Args = stringSlice(serverMap["args"])
		if env, ok := serverMap["env"].(map[string]any); ok {
			for key := range env {
				server.EnvKeys = append(server.EnvKeys, key)
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
	}
}

func commandFindings(item Item, server mcpServer) []Finding {
	command := strings.ToLower(filepath.Base(server.Command))
	var findings []Finding

	if command == "npx" || command == "uvx" || command == "pipx" {
		if !hasPinnedPackage(server.Args) {
			findings = append(findings, Finding{
				ID:             findingID("mcp_unpinned_package", item.Tool, item.Path, server.Name),
				Tool:           item.Tool,
				Path:           item.Path,
				Severity:       RiskHigh,
				Rule:           "mcp_unpinned_package",
				Message:        fmt.Sprintf("MCP server %q runs a package executor without an obvious pinned package version.", server.Name),
				Evidence:       fmt.Sprintf("command=%s args=%s", redact(server.Command), redact(strings.Join(server.Args, " "))),
				Recommendation: "Pin package versions or replace with a trusted local binary before syncing.",
			})
		}
	}

	if command == "sh" || command == "bash" || command == "zsh" || command == "cmd" || command == "powershell" || command == "pwsh" || hasArg(server.Args, "-c") {
		findings = append(findings, Finding{
			ID:             findingID("mcp_shell_command", item.Tool, item.Path, server.Name),
			Tool:           item.Tool,
			Path:           item.Path,
			Severity:       RiskHigh,
			Rule:           "mcp_shell_command",
			Message:        fmt.Sprintf("MCP server %q executes through a shell.", server.Name),
			Evidence:       fmt.Sprintf("command=%s args=%s", redact(server.Command), redact(strings.Join(server.Args, " "))),
			Recommendation: "Prefer direct executable invocation and review the command before syncing.",
		})
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
		})
	}

	return findings
}

func envFindings(item Item, server mcpServer) []Finding {
	var findings []Finding
	for _, key := range server.EnvKeys {
		if secretKeyPattern.MatchString(key) {
			findings = append(findings, Finding{
				ID:             findingID("mcp_secret_env", item.Tool, item.Path, server.Name+key),
				Tool:           item.Tool,
				Path:           item.Path,
				Severity:       RiskCritical,
				Rule:           "mcp_secret_env",
				Message:        fmt.Sprintf("MCP server %q references a sensitive environment key.", server.Name),
				Evidence:       fmt.Sprintf("env_key=%s", key),
				Recommendation: "Keep secret values outside dotfiles and document required env names only.",
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
		})
	}
	return findings
}

func hasPinnedPackage(args []string) bool {
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") || strings.Contains(arg, "/") || strings.Contains(arg, "\\") {
			continue
		}
		if strings.Contains(arg, "@") && !strings.HasPrefix(arg, "@") {
			return true
		}
		if strings.Count(arg, "@") >= 2 {
			return true
		}
	}
	return false
}

func hasArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
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
