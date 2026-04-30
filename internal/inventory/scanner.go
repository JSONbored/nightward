package inventory

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Scanner struct {
	Home      string
	Workspace string
	Hostname  string
	Now       func() time.Time
}

type pathSpec struct {
	Tool           string
	RelPath        string
	KindHint       string
	Classification Classification
	Risk           RiskLevel
	Reason         string
	Recommendation string
	CheckMCP       bool
}

type adapterSpec struct {
	Name        string
	Description string
	Paths       []pathSpec
}

func NewScanner(home string) Scanner {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	host, _ := os.Hostname()
	return Scanner{Home: home, Hostname: host, Now: time.Now}
}

func NewWorkspaceScanner(root string) Scanner {
	if root == "" {
		root = "."
	}
	abs, err := filepath.Abs(root)
	if err == nil {
		root = abs
	}
	host, _ := os.Hostname()
	return Scanner{Workspace: root, Hostname: host, Now: time.Now}
}

func (s Scanner) Scan() Report {
	now := s.now()
	if s.Workspace != "" {
		return s.scanWorkspace(now)
	}
	report := Report{
		GeneratedAt: now,
		Hostname:    s.Hostname,
		Home:        s.Home,
		ScanMode:    "home",
		Summary: Summary{
			ItemsByClassification: map[Classification]int{},
			ItemsByRisk:           map[RiskLevel]int{},
			ItemsByTool:           map[string]int{},
			FindingsBySeverity:    map[RiskLevel]int{},
			FindingsByRule:        map[string]int{},
			FindingsByTool:        map[string]int{},
		},
	}

	for _, adapter := range adapters() {
		status := AdapterStatus{
			Name:        adapter.Name,
			Description: adapter.Description,
		}

		for _, spec := range adapter.Paths {
			path := s.expand(spec.RelPath)
			status.Checked = append(status.Checked, path)
			item, ok := inspectPath(path, spec)
			if !ok {
				continue
			}
			item.ID = stableID(item.Tool, item.Path)
			report.Items = append(report.Items, item)
			status.Available = true
			status.Found = append(status.Found, path)
			report.Findings = append(report.Findings, inspectMCP(item, spec)...)
		}

		sort.Strings(status.Checked)
		sort.Strings(status.Found)
		report.Adapters = append(report.Adapters, status)
	}

	sort.Slice(report.Items, func(i, j int) bool {
		if report.Items[i].Tool == report.Items[j].Tool {
			return report.Items[i].Path < report.Items[j].Path
		}
		return report.Items[i].Tool < report.Items[j].Tool
	})
	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].Severity == report.Findings[j].Severity {
			return report.Findings[i].Path < report.Findings[j].Path
		}
		return severityRank(report.Findings[i].Severity) > severityRank(report.Findings[j].Severity)
	})

	report.Summary.TotalItems = len(report.Items)
	report.Summary.TotalFindings = len(report.Findings)
	for _, item := range report.Items {
		report.Summary.ItemsByClassification[item.Classification]++
		report.Summary.ItemsByRisk[item.Risk]++
		report.Summary.ItemsByTool[item.Tool]++
	}
	for _, finding := range report.Findings {
		report.Summary.FindingsBySeverity[finding.Severity]++
		report.Summary.FindingsByRule[finding.Rule]++
		report.Summary.FindingsByTool[finding.Tool]++
	}

	return report
}

func (s Scanner) scanWorkspace(now time.Time) Report {
	report := Report{
		GeneratedAt: now,
		Hostname:    s.Hostname,
		Home:        "",
		Workspace:   s.Workspace,
		ScanMode:    "workspace",
		Summary: Summary{
			ItemsByClassification: map[Classification]int{},
			ItemsByRisk:           map[RiskLevel]int{},
			ItemsByTool:           map[string]int{},
			FindingsBySeverity:    map[RiskLevel]int{},
			FindingsByRule:        map[string]int{},
			FindingsByTool:        map[string]int{},
		},
	}

	for _, adapter := range workspaceAdapters() {
		status := AdapterStatus{
			Name:        adapter.Name,
			Description: adapter.Description,
		}
		for _, spec := range adapter.Paths {
			path := filepath.Join(s.Workspace, filepath.FromSlash(spec.RelPath))
			status.Checked = append(status.Checked, path)
			item, ok := inspectPath(path, spec)
			if !ok {
				continue
			}
			item.ID = stableID("workspace", item.Tool, item.Path)
			if item.Metadata == nil {
				item.Metadata = map[string]string{}
			}
			item.Metadata["source"] = "nightward-workspace"
			report.Items = append(report.Items, item)
			status.Available = true
			status.Found = append(status.Found, path)
			report.Findings = append(report.Findings, inspectMCP(item, spec)...)
		}
		sort.Strings(status.Checked)
		sort.Strings(status.Found)
		report.Adapters = append(report.Adapters, status)
	}

	finalizeReport(&report)
	return report
}

func (s Scanner) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s Scanner) expand(rel string) string {
	if strings.HasPrefix(rel, "~/") {
		return filepath.Join(s.Home, strings.TrimPrefix(rel, "~/"))
	}
	if rel == "~" {
		return s.Home
	}
	return rel
}

func finalizeReport(report *Report) {
	sort.Slice(report.Items, func(i, j int) bool {
		if report.Items[i].Tool == report.Items[j].Tool {
			return report.Items[i].Path < report.Items[j].Path
		}
		return report.Items[i].Tool < report.Items[j].Tool
	})
	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].Severity == report.Findings[j].Severity {
			if report.Findings[i].Rule == report.Findings[j].Rule {
				return report.Findings[i].Path < report.Findings[j].Path
			}
			return report.Findings[i].Rule < report.Findings[j].Rule
		}
		return severityRank(report.Findings[i].Severity) > severityRank(report.Findings[j].Severity)
	})

	report.Summary.TotalItems = len(report.Items)
	report.Summary.TotalFindings = len(report.Findings)
	for _, item := range report.Items {
		report.Summary.ItemsByClassification[item.Classification]++
		report.Summary.ItemsByRisk[item.Risk]++
		report.Summary.ItemsByTool[item.Tool]++
	}
	for _, finding := range report.Findings {
		report.Summary.FindingsBySeverity[finding.Severity]++
		report.Summary.FindingsByRule[finding.Rule]++
		report.Summary.FindingsByTool[finding.Tool]++
	}
}

func inspectPath(path string, spec pathSpec) (Item, bool) {
	info, err := os.Lstat(path)
	if err != nil {
		return Item{}, false
	}

	kind := spec.KindHint
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		kind = "symlink"
	case info.IsDir():
		kind = "directory"
	case kind == "":
		kind = "file"
	}

	mod := info.ModTime().UTC()
	return Item{
		Tool:           spec.Tool,
		Path:           path,
		Kind:           kind,
		Classification: spec.Classification,
		Risk:           spec.Risk,
		Reason:         spec.Reason,
		Recommendation: spec.Recommendation,
		Exists:         true,
		SizeBytes:      info.Size(),
		ModTime:        &mod,
		Metadata: map[string]string{
			"source": "nightward-adapter",
		},
	}, true
}

func stableID(parts ...string) string {
	hash := sha1.Sum([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(hash[:])[:12]
}

func severityRank(r RiskLevel) int {
	switch r {
	case RiskCritical:
		return 5
	case RiskHigh:
		return 4
	case RiskMedium:
		return 3
	case RiskLow:
		return 2
	default:
		return 1
	}
}

func adapters() []adapterSpec {
	return []adapterSpec{
		{
			Name:        "Codex",
			Description: "OpenAI Codex CLI and desktop agent state.",
			Paths: []pathSpec{
				portable("Codex", "~/.codex/config.toml", "Codex preferences and MCP server definitions.", true),
				portable("Codex", "~/.codex/AGENTS.md", "Global Codex instruction file.", false),
				portableDir("Codex", "~/.codex/skills", "User skill directories. Runtime-managed hidden directories should stay excluded."),
				machineLocal("Codex", "~/.codex/automations", "Local automation schedules and state should be reviewed before syncing."),
				secret("Codex", "~/.codex/auth.json", "Codex authentication material must never be copied to dotfiles."),
				runtime("Codex", "~/.codex/cache", "Runtime cache is generated by Codex."),
				runtime("Codex", "~/.codex/sessions", "Conversation/session state is runtime data."),
				appOwned("Codex", "~/.codex/sqlite", "Local application database is app-owned state."),
			},
		},
		{
			Name:        "Claude",
			Description: "Claude Code and Claude Desktop configuration.",
			Paths: []pathSpec{
				machineLocalMCP("Claude", "~/.claude.json", "Claude Code global state often mixes settings, projects, and local paths."),
				portable("Claude", "~/.claude/settings.json", "Claude Code settings are potentially portable after review.", false),
				portableDir("Claude", "~/.claude/commands", "Claude Code custom commands are usually portable."),
				portableDir("Claude", "~/.claude/agents", "Claude Code agents are usually portable."),
				appOwned("Claude", "~/.claude/projects", "Project conversation state is app-owned runtime data."),
				secret("Claude", "~/.claude/.credentials.json", "Claude credentials must never be copied to dotfiles."),
				machineLocalMCP("Claude", "~/Library/Application Support/Claude/claude_desktop_config.json", "Claude Desktop MCP config can contain local commands, paths, and env references."),
				appOwned("Claude", "~/Library/Application Support/Claude/Code Cache", "Electron cache is generated runtime state."),
			},
		},
		{
			Name:        "Cursor",
			Description: "Cursor editor settings, rules, projects, and MCP state.",
			Paths: []pathSpec{
				portable("Cursor", "~/.cursor/mcp.json", "Cursor MCP config should be reviewed before syncing.", true),
				portableDir("Cursor", "~/.cursor/rules", "Cursor rules are usually portable project/editor guidance."),
				portableDir("Cursor", "~/.cursor/skills-cursor", "Cursor skills are usually portable after review."),
				appOwned("Cursor", "~/.cursor/projects", "Cursor project databases are app-owned state."),
				appOwned("Cursor", "~/.cursor/extensions", "Installed extension binaries are app-owned; export IDs instead."),
				appOwned("Cursor", "~/.cursor/ai-tracking", "AI usage tracking databases are app-owned local data."),
				portable("Cursor", "~/Library/Application Support/Cursor/User/settings.json", "Cursor user settings are potentially portable.", false),
				portable("Cursor", "~/Library/Application Support/Cursor/User/keybindings.json", "Cursor keybindings are portable editor settings.", false),
				portable("Cursor", "~/Library/Application Support/Cursor/User/mcp.json", "Cursor user MCP config should be reviewed before syncing.", true),
			},
		},
		{
			Name:        "Windsurf",
			Description: "Windsurf editor settings and MCP state.",
			Paths: []pathSpec{
				portable("Windsurf", "~/.windsurf/mcp.json", "Windsurf MCP config should be reviewed before syncing.", true),
				portableDir("Windsurf", "~/.windsurf/rules", "Windsurf rules are usually portable guidance."),
				portable("Windsurf", "~/Library/Application Support/Windsurf/User/settings.json", "Windsurf user settings are potentially portable.", false),
				portable("Windsurf", "~/Library/Application Support/Windsurf/User/keybindings.json", "Windsurf keybindings are portable editor settings.", false),
				portable("Windsurf", "~/Library/Application Support/Windsurf/User/mcp.json", "Windsurf user MCP config should be reviewed before syncing.", true),
			},
		},
		{
			Name:        "VS Code",
			Description: "VS Code settings, extension exports, and MCP state.",
			Paths: []pathSpec{
				portable("VS Code", "~/.vscode/vscode-extensions", "Plain extension ID export is portable.", false),
				appOwned("VS Code", "~/.vscode/extensions", "Installed extension binaries are app-owned; export IDs instead."),
				portable("VS Code", "~/Library/Application Support/Code/User/settings.json", "VS Code user settings are potentially portable.", false),
				portable("VS Code", "~/Library/Application Support/Code/User/keybindings.json", "VS Code keybindings are portable editor settings.", false),
				portable("VS Code", "~/Library/Application Support/Code/User/mcp.json", "VS Code MCP config should be reviewed before syncing.", true),
				portableDir("VS Code", "~/Library/Application Support/Code/User/snippets", "VS Code snippets are usually portable."),
			},
		},
		{
			Name:        "Raycast",
			Description: "Raycast extension and local app state.",
			Paths: []pathSpec{
				machineLocal("Raycast", "~/.config/raycast", "Raycast config may be portable, but extension preferences can include local paths."),
				appOwned("Raycast", "~/Library/Application Support/com.raycast.macos", "Raycast encrypted databases and runtime state are app-owned."),
				appOwned("Raycast", "~/Library/Application Support/com.raycast.shared", "Shared Raycast runtime state is app-owned."),
			},
		},
		{
			Name:        "JetBrains",
			Description: "JetBrains IDE settings, plugins, and AI assistant state.",
			Paths: []pathSpec{
				machineLocal("JetBrains", "~/Library/Application Support/JetBrains", "JetBrains settings can mix portable IDE preferences with local paths, plugins, and account state."),
				appOwned("JetBrains", "~/Library/Caches/JetBrains", "JetBrains caches are generated runtime state."),
				appOwned("JetBrains", "~/Library/Logs/JetBrains", "JetBrains logs are app-owned runtime data."),
			},
		},
		{
			Name:        "Zed",
			Description: "Zed editor settings, keymaps, snippets, and assistant state.",
			Paths: []pathSpec{
				portable("Zed", "~/.config/zed/settings.json", "Zed settings are potentially portable after reviewing local model/provider assumptions.", false),
				portable("Zed", "~/.config/zed/keymap.json", "Zed keymaps are usually portable editor settings.", false),
				portableDir("Zed", "~/.config/zed/snippets", "Zed snippets are usually portable."),
				appOwned("Zed", "~/Library/Application Support/Zed", "Zed application support can include app-owned databases, sessions, and extension state."),
			},
		},
		{
			Name:        "Continue",
			Description: "Continue.dev assistant configuration and local indexes.",
			Paths: []pathSpec{
				portable("Continue", "~/.continue/config.json", "Continue config is potentially portable after provider, model, and path review.", true),
				portable("Continue", "~/.continue/config.yaml", "Continue config is potentially portable after provider, model, and path review.", true),
				portable("Continue", "~/.continue/config.ts", "Continue TypeScript config may be portable, but executable config needs review.", false),
				runtime("Continue", "~/.continue/index", "Continue local indexes are generated runtime data."),
				appOwned("Continue", "~/.continue/sessions", "Continue session history is app-owned state."),
			},
		},
		{
			Name:        "Cline/Roo",
			Description: "Cline and Roo Code extension storage inside VS Code-compatible editors.",
			Paths: []pathSpec{
				appOwned("Cline/Roo", "~/Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev", "Cline extension storage may contain prompts, secrets, and app-owned state."),
				appOwned("Cline/Roo", "~/Library/Application Support/Code/User/globalStorage/rooveterinaryinc.roo-cline", "Roo Code extension storage may contain prompts, secrets, and app-owned state."),
				appOwned("Cline/Roo", "~/Library/Application Support/Cursor/User/globalStorage/saoudrizwan.claude-dev", "Cursor Cline extension storage may contain prompts, secrets, and app-owned state."),
				appOwned("Cline/Roo", "~/Library/Application Support/Cursor/User/globalStorage/rooveterinaryinc.roo-cline", "Cursor Roo Code extension storage may contain prompts, secrets, and app-owned state."),
			},
		},
		{
			Name:        "Aider",
			Description: "Aider CLI preferences, model settings, and generated caches.",
			Paths: []pathSpec{
				portable("Aider", "~/.aider.conf.yml", "Aider config is potentially portable after provider and path review.", false),
				portable("Aider", "~/.aider.model.settings.yml", "Aider model settings are potentially portable after provider review.", false),
				appOwned("Aider", "~/.aider.chat.history.md", "Aider chat history is app-owned local session state."),
				runtime("Aider", "~/.aider.tags.cache.v4", "Aider tag cache is generated runtime data."),
			},
		},
		{
			Name:        "OpenCode",
			Description: "OpenCode configuration and local state.",
			Paths: []pathSpec{
				portable("OpenCode", "~/.opencode.json", "OpenCode config is potentially portable after provider and path review.", true),
				portable("OpenCode", "~/.config/opencode/opencode.json", "OpenCode config is potentially portable after provider and path review.", true),
				appOwned("OpenCode", "~/.local/share/opencode", "OpenCode local share data is app-owned state."),
			},
		},
		{
			Name:        "Goose",
			Description: "Block Goose agent configuration and local state.",
			Paths: []pathSpec{
				portable("Goose", "~/.config/goose/config.yaml", "Goose config is potentially portable after provider and extension review.", true),
				machineLocal("Goose", "~/.config/goose", "Goose config directories can include local extensions, paths, and provider assumptions."),
				appOwned("Goose", "~/.local/share/goose", "Goose local share data is app-owned state."),
			},
		},
		{
			Name:        "LM Studio",
			Description: "LM Studio model, server, and application state.",
			Paths: []pathSpec{
				machineLocal("LM Studio", "~/.lmstudio/config.json", "LM Studio config can include local model and server assumptions."),
				appOwned("LM Studio", "~/.lmstudio/models", "LM Studio model files are app-owned heavy runtime assets."),
				appOwned("LM Studio", "~/Library/Application Support/LM Studio", "LM Studio app support is app-owned runtime state."),
			},
		},
		{
			Name:        "Ollama/Open WebUI",
			Description: "Local model runtime and Open WebUI state.",
			Paths: []pathSpec{
				machineLocal("Ollama/Open WebUI", "~/.ollama/config.json", "Ollama config can include local runtime assumptions."),
				secret("Ollama/Open WebUI", "~/.ollama/id_ed25519", "Ollama identity material must not be copied to dotfiles."),
				appOwned("Ollama/Open WebUI", "~/.ollama/models", "Ollama model blobs are app-owned heavy runtime assets."),
				appOwned("Ollama/Open WebUI", "~/.open-webui", "Open WebUI local state can include databases, uploads, and credentials."),
				machineLocal("Ollama/Open WebUI", "~/.config/open-webui", "Open WebUI config can include machine-local service assumptions."),
			},
		},
		{
			Name:        "Neovim",
			Description: "Neovim configuration and AI plugin state.",
			Paths: []pathSpec{
				portableDir("Neovim", "~/.config/nvim", "Neovim config is usually portable, but plugin-managed AI credentials must stay external."),
				appOwned("Neovim", "~/.local/share/nvim", "Neovim plugin state and package data are app-owned local state."),
				runtime("Neovim", "~/.cache/nvim", "Neovim cache is generated runtime data."),
			},
		},
		{
			Name:        "Generic MCP",
			Description: "Common standalone MCP config files.",
			Paths: []pathSpec{
				portable("Generic MCP", "~/.mcp.json", "Standalone MCP config should be reviewed before syncing.", true),
				portable("Generic MCP", "~/.config/mcp/mcp.json", "Standalone MCP config should be reviewed before syncing.", true),
			},
		},
	}
}

func workspaceAdapters() []adapterSpec {
	return []adapterSpec{
		{
			Name:        "Workspace Codex",
			Description: "Codex project instructions and repo-local MCP config.",
			Paths: []pathSpec{
				portable("Codex", ".codex/config.toml", "Repo-local Codex MCP config should be reviewed before CI or dotfile sync.", true),
				portable("Codex", "AGENTS.md", "Project instruction files are portable but can encode machine assumptions.", false),
				portableDir("Codex", ".codex/skills", "Repo-local Codex skills are portable after secret and path review."),
			},
		},
		{
			Name:        "Workspace Claude",
			Description: "Claude project settings, agents, commands, and MCP config.",
			Paths: []pathSpec{
				portable("Claude", ".claude/settings.json", "Claude settings are portable after provider and path review.", false),
				portable("Claude", ".claude/mcp.json", "Claude MCP config should be reviewed before CI or dotfile sync.", true),
				portableDir("Claude", ".claude/commands", "Claude commands are usually portable project guidance."),
				portableDir("Claude", ".claude/agents", "Claude agents are usually portable project guidance."),
			},
		},
		{
			Name:        "Workspace Cursor",
			Description: "Cursor repo rules, skills, settings, and MCP config.",
			Paths: []pathSpec{
				portable("Cursor", ".cursor/mcp.json", "Cursor MCP config should be reviewed before CI or dotfile sync.", true),
				portable("Cursor", ".cursor/settings.json", "Cursor repo settings are portable after provider and path review.", false),
				portableDir("Cursor", ".cursor/rules", "Cursor rules are usually portable project guidance."),
				portableDir("Cursor", ".cursor/skills-cursor", "Cursor skills are usually portable after review."),
			},
		},
		{
			Name:        "Workspace Editors",
			Description: "Editor and assistant configs commonly committed to repos.",
			Paths: []pathSpec{
				portable("VS Code", ".vscode/mcp.json", "VS Code MCP config should be reviewed before CI or dotfile sync.", true),
				portable("VS Code", ".vscode/settings.json", "VS Code repo settings are portable after path review.", false),
				portable("Windsurf", ".windsurf/mcp.json", "Windsurf MCP config should be reviewed before CI or dotfile sync.", true),
				portable("Zed", ".zed/settings.json", "Zed repo settings are portable after provider and path review.", false),
				portable("Continue", ".continue/config.json", "Continue config is portable after provider, model, and path review.", true),
				portable("OpenCode", ".opencode.json", "OpenCode config is portable after provider and path review.", true),
				portable("Generic MCP", ".mcp.json", "Standalone MCP config should be reviewed before CI or dotfile sync.", true),
				portable("Generic MCP", "mcp.json", "Standalone MCP config should be reviewed before CI or dotfile sync.", true),
			},
		},
		{
			Name:        "Workspace Secrets",
			Description: "Credential-like files that should not be committed or synced.",
			Paths: []pathSpec{
				secret("Secrets", ".env", "Environment files commonly contain credentials and should stay local-only."),
				secret("Secrets", ".env.local", "Local environment files commonly contain credentials and should stay local-only."),
				secret("Secrets", ".npmrc", "npm config can contain registry tokens."),
				secret("Secrets", ".netrc", "netrc files contain machine credentials."),
				secret("Secrets", ".pypirc", "Python package registry config can contain publish tokens."),
				secret("Secrets", ".git-credentials", "Git credential files must not be committed or synced."),
				secret("Secrets", ".ssh", "SSH keys and config are machine-local credential material."),
				secret("Secrets", ".aws", "AWS credentials and profiles are machine-local credential material."),
				secret("Secrets", ".config/gh/hosts.yml", "GitHub CLI host tokens are machine-local credential material."),
				secret("Secrets", ".docker/config.json", "Docker client config can contain registry tokens."),
			},
		},
	}
}

func portable(tool, rel, reason string, mcp bool) pathSpec {
	return pathSpec{Tool: tool, RelPath: rel, Classification: Portable, Risk: RiskLow, Reason: reason, Recommendation: "Review and sync through a private dotfiles repo if paths are portable.", CheckMCP: mcp}
}

func portableDir(tool, rel, reason string) pathSpec {
	spec := portable(tool, rel, reason, false)
	spec.KindHint = "directory"
	return spec
}

func machineLocal(tool, rel, reason string) pathSpec {
	return pathSpec{Tool: tool, RelPath: rel, Classification: MachineLocal, Risk: RiskMedium, Reason: reason, Recommendation: "Keep local-only unless reviewed and templated for portability."}
}

func machineLocalMCP(tool, rel, reason string) pathSpec {
	spec := machineLocal(tool, rel, reason)
	spec.CheckMCP = true
	return spec
}

func secret(tool, rel, reason string) pathSpec {
	return pathSpec{Tool: tool, RelPath: rel, Classification: SecretAuth, Risk: RiskCritical, Reason: reason, Recommendation: "Exclude from backups and store credentials in a password manager or secret store."}
}

func runtime(tool, rel, reason string) pathSpec {
	return pathSpec{Tool: tool, RelPath: rel, Classification: RuntimeCache, Risk: RiskInfo, Reason: reason, Recommendation: "Exclude; this is generated runtime state."}
}

func appOwned(tool, rel, reason string) pathSpec {
	return pathSpec{Tool: tool, RelPath: rel, Classification: AppOwned, Risk: RiskHigh, Reason: reason, Recommendation: "Exclude unless the app documents a safe export format."}
}

func findingID(rule, tool, path, evidence string) string {
	return fmt.Sprintf("%s-%s", rule, stableID(tool, path, evidence))
}
