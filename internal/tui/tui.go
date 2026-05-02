package tui

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	bubblekey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jsonbored/nightward/internal/analysis"
	"github.com/jsonbored/nightward/internal/backupplan"
	"github.com/jsonbored/nightward/internal/fixplan"
	"github.com/jsonbored/nightward/internal/inventory"
	"github.com/jsonbored/nightward/internal/schedule"
)

type model struct {
	report        inventory.Report
	schedule      schedule.Plan
	tab           int
	cursor        int
	width         int
	height        int
	severity      string
	tool          string
	rule          string
	search        string
	status        string
	searching     bool
	showHelp      bool
	palette       bool
	paletteCursor int
	helpModel     help.Model
	searchInput   textinput.Model
}

type actionMsg struct {
	status string
}

var (
	bg        = lipgloss.Color("#0b1020")
	ink       = lipgloss.Color("#d7dde8")
	muted     = lipgloss.Color("#7d8799")
	blue      = lipgloss.Color("#7aa2f7")
	cyan      = lipgloss.Color("#7dcfff")
	amber     = lipgloss.Color("#e0af68")
	red       = lipgloss.Color("#f7768e")
	green     = lipgloss.Color("#9ece6a")
	purple    = lipgloss.Color("#bb9af7")
	panelLine = lipgloss.Color("#26314a")

	baseStyle  = lipgloss.NewStyle().Foreground(ink).Background(bg)
	titleStyle = lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true).
			Padding(0, 1)
	tabStyle = lipgloss.NewStyle().
			Foreground(muted).
			Padding(0, 1)
	activeTabStyle = lipgloss.NewStyle().
			Foreground(bg).
			Background(cyan).
			Bold(true).
			Padding(0, 1)
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(panelLine).
			Padding(1, 2)
	footerStyle = lipgloss.NewStyle().
			Foreground(muted).
			Padding(0, 1)
)

var tabs = []string{"Dashboard", "Inventory", "Findings", "Analysis", "Fix Plan", "Backup Plan"}
var compactTabs = []string{"Dash", "Inv", "Find", "Analysis", "Fix", "Backup"}
var tinyTabs = []string{"D", "I", "F", "A", "X", "B"}
var tabAccents = []lipgloss.Color{cyan, blue, red, purple, amber, green}

type tuiKeyMap struct {
	Tabs    bubblekey.Binding
	Palette bubblekey.Binding
	Search  bubblekey.Binding
	Filters bubblekey.Binding
	Copy    bubblekey.Binding
	Export  bubblekey.Binding
	Docs    bubblekey.Binding
	Help    bubblekey.Binding
	Quit    bubblekey.Binding
}

func (k tuiKeyMap) ShortHelp() []bubblekey.Binding {
	return []bubblekey.Binding{k.Tabs, k.Palette, k.Search, k.Filters, k.Copy, k.Export, k.Help, k.Quit}
}

func (k tuiKeyMap) FullHelp() [][]bubblekey.Binding {
	return [][]bubblekey.Binding{
		{k.Tabs, k.Palette, k.Search, k.Filters},
		{k.Copy, k.Export, k.Docs, k.Help, k.Quit},
	}
}

var tuiKeys = tuiKeyMap{
	Tabs:    bubblekey.NewBinding(bubblekey.WithKeys("tab", "1-6"), bubblekey.WithHelp("tab/1-6", "tabs")),
	Palette: bubblekey.NewBinding(bubblekey.WithKeys("p"), bubblekey.WithHelp("p", "palette")),
	Search:  bubblekey.NewBinding(bubblekey.WithKeys("/"), bubblekey.WithHelp("/", "search")),
	Filters: bubblekey.NewBinding(bubblekey.WithKeys("s", "t", "r", "x"), bubblekey.WithHelp("s/t/r/x", "filters")),
	Copy:    bubblekey.NewBinding(bubblekey.WithKeys("c"), bubblekey.WithHelp("c", "copy")),
	Export:  bubblekey.NewBinding(bubblekey.WithKeys("e"), bubblekey.WithHelp("e", "export")),
	Docs:    bubblekey.NewBinding(bubblekey.WithKeys("o"), bubblekey.WithHelp("o", "docs")),
	Help:    bubblekey.NewBinding(bubblekey.WithKeys("?"), bubblekey.WithHelp("?", "help")),
	Quit:    bubblekey.NewBinding(bubblekey.WithKeys("q", "esc"), bubblekey.WithHelp("q/esc", "quit")),
}

type paletteCommand struct {
	Title  string
	Detail string
	Action string
}

const remediationDocsURL = "https://github.com/JSONbored/nightward/blob/main/docs/remediation.md"
const analysisDocsURL = "https://github.com/JSONbored/nightward/blob/main/docs/analysis.md"

var tuiSecretAssignmentPattern = regexp.MustCompile(`(?i)((?:token|secret|password|passwd|api[_-]?key|auth|credential|private[_-]?key)[\w.-]*\s*[:=]\s*)(["']?)[^"',\s}]+`)
var tuiLongSecretPattern = regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{12,}\b`)

func Run(report inventory.Report, scheduleStatus schedule.Plan) error {
	_, err := tea.NewProgram(newModel(report, scheduleStatus), tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	return err
}

func newModel(report inventory.Report, scheduleStatus schedule.Plan) model {
	return model{
		report:      report,
		schedule:    scheduleStatus,
		helpModel:   newHelpModel(),
		searchInput: newSearchInput(""),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case actionMsg:
		m.status = msg.status
	case tea.KeyMsg:
		if m.searching {
			switch msg.String() {
			case "enter":
				m.search = m.searchInputValue()
				m.searching = false
				m.status = "search: " + filterLabel(m.search)
			case "esc":
				m.searching = false
				m.searchInput = newSearchInput(m.search)
			case "backspace":
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInputModel().Update(msg)
				m.search = m.searchInput.Value()
				m.cursor = 0
				return m, cmd
			case "ctrl+u":
				m.search = ""
				m.searchInput = newSearchInput("")
				m.cursor = 0
			default:
				if len(msg.Runes) > 0 || msg.Type == tea.KeySpace {
					var cmd tea.Cmd
					m.searchInput, cmd = m.searchInputModel().Update(msg)
					m.search = m.searchInput.Value()
					m.cursor = 0
					return m, cmd
				}
			}
			return m, nil
		}
		if m.palette {
			switch msg.String() {
			case "esc", "p":
				m.palette = false
			case "enter":
				commands := m.paletteCommands()
				if len(commands) == 0 {
					m.palette = false
					return m, nil
				}
				if m.paletteCursor >= len(commands) {
					m.paletteCursor = len(commands) - 1
				}
				selected := commands[m.paletteCursor]
				m.palette = false
				return m.applyPaletteCommand(selected)
			case "up", "k":
				if m.paletteCursor > 0 {
					m.paletteCursor--
				}
			case "down", "j":
				if m.paletteCursor < len(m.paletteCommands())-1 {
					m.paletteCursor++
				}
			}
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
		case "p":
			m.palette = true
			m.paletteCursor = 0
			m.showHelp = false
			m.status = "command palette"
		case "tab", "right", "l":
			m.tab = (m.tab + 1) % len(tabs)
			m.cursor = 0
		case "shift+tab", "left", "h":
			m.tab--
			if m.tab < 0 {
				m.tab = len(tabs) - 1
			}
			m.cursor = 0
		case "1":
			m.tab = 0
		case "2":
			m.tab = 1
		case "3":
			m.tab = 2
		case "4":
			m.tab = 3
		case "5":
			m.tab = 4
		case "6":
			m.tab = 5
		case "s":
			if m.tab == 2 {
				m.severity = cycle(m.severity, riskOptions(m.report.Findings))
				m.cursor = 0
			}
		case "t":
			if m.tab == 2 {
				m.tool = cycle(m.tool, toolOptions(m.report.Findings))
				m.cursor = 0
			}
		case "r":
			if m.tab == 2 {
				m.rule = cycle(m.rule, ruleOptions(m.report.Findings))
				m.cursor = 0
			}
		case "/":
			if m.tab == 2 {
				m.searching = true
				m.searchInput = newSearchInput(m.search)
				m.searchInput.Focus()
				m.status = "type search, enter to keep, esc to cancel"
			}
		case "x":
			if m.tab == 2 {
				m.search = ""
				m.searchInput = newSearchInput("")
				m.severity = ""
				m.tool = ""
				m.rule = ""
				m.cursor = 0
				m.status = "filters cleared"
			}
		case "c":
			value, label, ok := m.copySelection()
			if !ok {
				m.status = "copy: nothing selected"
				return m, nil
			}
			m.status = "copying " + label + "..."
			return m, copyToClipboardCmd(value, label)
		case "e":
			m.status = "exporting fix plan..."
			return m, exportFixPlanCmd(m.report)
		case "o":
			docsURL, ok := m.currentDocsURL()
			if !ok {
				m.status = "open docs: no docs URL for selected row"
				return m, nil
			}
			m.status = "opening docs..."
			return m, openURLCmd(docsURL)
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			m.cursor++
		}
	}
	return m, nil
}

func (m model) View() string {
	width := m.width
	if width <= 0 {
		width = 96
	}
	bodyWidth := max(40, width-4)
	bodyHeight := max(12, m.height-7)

	tabLine := lipgloss.JoinHorizontal(lipgloss.Top, titleStyle.Render("nightward"), m.renderTabs(max(16, bodyWidth-12)))
	tabLine = lipgloss.NewStyle().Width(bodyWidth).MaxWidth(bodyWidth).Render(tabLine)
	bodyText := m.renderBody(bodyWidth-6, bodyHeight-2)
	if m.showHelp {
		bodyText = m.help(bodyWidth - 6)
	}
	if m.palette {
		bodyText = m.commandPalette(bodyWidth-6, bodyHeight-2)
	}
	body := panelStyle.BorderForeground(tabAccent(m.tab)).Width(bodyWidth).Height(bodyHeight).Render(bodyText)
	footerText := m.footerHelp(bodyWidth - 2)
	if m.searching {
		input := m.searchInputModel()
		footerText = "search: " + input.View()
	}
	if m.palette {
		footerText = "palette: enter run  arrows navigate  esc close"
	}
	if m.status != "" {
		footerText += "  " + m.status
	}
	footer := footerStyle.Width(bodyWidth).Render(truncate(footerText, bodyWidth-2))
	return baseStyle.Render(lipgloss.JoinVertical(lipgloss.Left, tabLine, body, footer))
}

func (m model) renderTabs(width int) string {
	labels := tabs
	if width < 44 {
		labels = tinyTabs
	} else if width < 84 {
		labels = compactTabs
	}
	rendered := make([]string, 0, len(tabs))
	for i, tab := range labels {
		label := fmt.Sprintf("%d %s", i+1, tab)
		accent := tabAccent(i)
		if i == m.tab {
			rendered = append(rendered, activeTabStyle.Background(accent).Render(label))
		} else {
			rendered = append(rendered, tabStyle.Foreground(accent).Render(label))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func (m model) renderBody(width, height int) string {
	switch m.tab {
	case 0:
		return m.dashboard(width)
	case 1:
		return m.inventory(width, height)
	case 2:
		return m.findings(width, height)
	case 3:
		return m.analysis(width, height)
	case 4:
		return m.fixPlan(width, height)
	default:
		return m.backupPlan(width, height)
	}
}

func (m model) dashboard(width int) string {
	lines := []string{
		section("Scan"),
		fmt.Sprintf("Generated: %s", m.report.GeneratedAt.Local().Format("2006-01-02 15:04:05")),
		fmt.Sprintf("Host:      %s", m.report.Hostname),
		fmt.Sprintf("Home:      %s", m.report.Home),
		"",
		metricLine("Items", m.report.Summary.TotalItems, blue),
		metricLine("Findings", m.report.Summary.TotalFindings, severityColor(maxRisk(m.report.Findings))),
		"",
		section("Classifications"),
	}
	for _, class := range []inventory.Classification{inventory.Portable, inventory.MachineLocal, inventory.SecretAuth, inventory.RuntimeCache, inventory.AppOwned, inventory.Unknown} {
		if count := m.report.Summary.ItemsByClassification[class]; count > 0 {
			lines = append(lines, fmt.Sprintf("%-14s %d", class, count))
		}
	}
	lines = append(lines, "", section("Finding Severity"))
	for _, risk := range []inventory.RiskLevel{inventory.RiskCritical, inventory.RiskHigh, inventory.RiskMedium, inventory.RiskLow, inventory.RiskInfo} {
		if count := m.report.Summary.FindingsBySeverity[risk]; count > 0 {
			lines = append(lines, fmt.Sprintf("%-14s %d", risk, count))
		}
	}
	lines = append(lines, "", section("Schedule"))
	installed := "not installed"
	if m.schedule.Installed {
		installed = "installed"
	}
	lines = append(lines, fmt.Sprintf("Nightly scan: %s", installed))
	lines = append(lines, fmt.Sprintf("Report dir:   %s", m.schedule.ReportDir))
	if m.schedule.LastReport != "" {
		lines = append(lines, fmt.Sprintf("Last report:  %s", m.schedule.LastReport))
	}
	if len(m.schedule.History) > 0 {
		lines = append(lines, "", section("Recent Reports"))
		limit := len(m.schedule.History)
		if limit > 3 {
			limit = 3
		}
		for _, record := range m.schedule.History[:limit] {
			lines = append(lines, fmt.Sprintf("%s  findings=%d  size=%s  %s", record.ModTime.Local().Format("2006-01-02 15:04"), record.Findings, byteSize(record.SizeBytes), record.ReportName))
		}
		if delta := reportDelta(m.schedule.History); delta != "" {
			lines = append(lines, "Latest delta: "+delta)
			if severity := reportSeverityDelta(m.schedule.History); severity != "" {
				lines = append(lines, "Severity delta: "+severity)
			}
		}
	}
	lines = append(lines, "", section("What Next"))
	lines = append(lines, m.nextActions()...)
	return fitLines(lines, width)
}

func (m model) inventory(width, height int) string {
	items := m.report.Items
	if len(items) == 0 {
		return "No known AI agent/devtool config paths found yet."
	}
	rows := make([]table.Row, 0, len(items))
	for _, item := range items {
		rows = append(rows, table.Row{item.Tool, string(item.Classification), string(item.Risk), item.Path})
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		section("Inventory"),
		renderTable(
			[]table.Column{{Title: "Tool", Width: 12}, {Title: "Class", Width: 16}, {Title: "Risk", Width: 10}, {Title: "Path", Width: max(12, width-46)}},
			rows,
			m.cursor,
			width,
			max(3, height-2),
			tabAccent(m.tab),
		),
	)
}

func (m model) findings(width, height int) string {
	findings := m.filteredFindings()
	if len(findings) == 0 {
		return "No findings match the current filters."
	}
	listWidth := width
	detailWidth := 0
	if width >= 94 {
		listWidth = width/2 - 1
		detailWidth = width - listWidth - 3
	}
	lines := []string{
		section("Findings"),
		fmt.Sprintf("severity=%s  tool=%s  rule=%s  search=%s", filterLabel(m.severity), filterLabel(m.tool), filterLabel(m.rule), filterLabel(m.search)),
		"",
	}
	if m.cursor >= len(findings) {
		m.cursor = len(findings) - 1
	}
	rows := make([]table.Row, 0, len(findings))
	for _, finding := range findings {
		rows = append(rows, table.Row{string(finding.Severity), finding.Rule, finding.Message})
	}
	lines = append(lines, renderTable(
		[]table.Column{{Title: "Risk", Width: 9}, {Title: "Rule", Width: 24}, {Title: "Message", Width: max(12, listWidth-39)}},
		rows,
		m.cursor,
		listWidth,
		max(3, height-4),
		tabAccent(m.tab),
	))
	left := fitLines(lines, listWidth)
	if detailWidth <= 0 {
		return left
	}
	detail := findingDetail(findings[m.cursor], detailWidth, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", detail)
}

func (m model) fixPlan(width, height int) string {
	plan := fixplan.Build(m.report, fixplan.Selector{All: true})
	listWidth := width
	detailWidth := 0
	if width >= 94 {
		listWidth = width/2 - 1
		detailWidth = width - listWidth - 3
	}
	lines := []string{
		section("Fix Plan"),
		fmt.Sprintf("Safe: %d  Review: %d  Blocked: %d", plan.Summary.Safe, plan.Summary.Review, plan.Summary.Blocked),
		"",
	}
	if len(plan.Fixes) == 0 {
		return "No fix plans available."
	}
	if m.cursor >= len(plan.Fixes) {
		m.cursor = len(plan.Fixes) - 1
	}
	rows := make([]table.Row, 0, len(plan.Fixes))
	for _, fix := range plan.Fixes {
		rows = append(rows, table.Row{string(fix.Status), string(fix.FixKind), fix.Rule, fix.Summary})
	}
	lines = append(lines, renderTable(
		[]table.Column{{Title: "Status", Width: 9}, {Title: "Kind", Width: 20}, {Title: "Rule", Width: 22}, {Title: "Summary", Width: max(12, listWidth-57)}},
		rows,
		m.cursor,
		listWidth,
		max(3, height-6),
		tabAccent(m.tab),
	))
	lines = append(lines, "", "Use `nw fix plan --json` or `nw fix export --format markdown` for full steps.")
	left := fitLines(lines, listWidth)
	if detailWidth <= 0 {
		return left
	}
	detail := fixDetail(plan.Fixes[m.cursor], detailWidth, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", detail)
}

func (m model) analysis(width, height int) string {
	report := analysis.Run(m.report, analysis.Options{Mode: m.report.ScanMode, Workspace: m.report.Workspace})
	listWidth := width
	detailWidth := 0
	if width >= 94 {
		listWidth = width/2 - 1
		detailWidth = width - listWidth - 3
	}
	lines := []string{
		section("Analysis"),
		fmt.Sprintf("Signals: %d  Subjects: %d  Highest: %s  Provider warnings: %d", report.Summary.TotalSignals, report.Summary.TotalSubjects, report.Summary.HighestSeverity, report.Summary.ProviderWarnings),
		"",
	}
	if len(report.Signals) == 0 {
		lines = append(lines, "No known risky signals from enabled providers.")
		return fitLines(lines, width)
	}
	if m.cursor >= len(report.Signals) {
		m.cursor = len(report.Signals) - 1
	}
	rows := make([]table.Row, 0, len(report.Signals))
	for _, signal := range report.Signals {
		rows = append(rows, table.Row{string(signal.Severity), signal.Rule, signal.Message})
	}
	lines = append(lines, renderTable(
		[]table.Column{{Title: "Risk", Width: 9}, {Title: "Rule", Width: 30}, {Title: "Message", Width: max(12, listWidth-43)}},
		rows,
		m.cursor,
		listWidth,
		max(3, height-6),
		tabAccent(m.tab),
	))
	lines = append(lines, "", "Use `nw analyze --json` or `nw providers doctor --json` for full details.")
	left := fitLines(lines, listWidth)
	if detailWidth <= 0 {
		return left
	}
	detail := signalDetail(report.Signals[m.cursor], detailWidth, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", detail)
}

func (m model) backupPlan(width, height int) string {
	target := filepath.Join(m.report.Home, "dotfiles")
	plan := backupplan.Build(m.report, target)
	lines := []string{
		section("Backup Plan Preview"),
		fmt.Sprintf("Target:  %s", target),
		fmt.Sprintf("Include: %d  Review: %d  Exclude: %d", plan.Summary.Included, plan.Summary.Review, plan.Summary.Excluded),
		"",
	}
	rows := make([]table.Row, 0, len(plan.Entries))
	for _, entry := range plan.Entries {
		rows = append(rows, table.Row{string(entry.Action), entry.Tool, entry.Source})
	}
	lines = append(lines, renderTable(
		[]table.Column{{Title: "Action", Width: 9}, {Title: "Tool", Width: 14}, {Title: "Source", Width: max(12, width-27)}},
		rows,
		m.cursor,
		width,
		max(3, height-8),
		tabAccent(m.tab),
	))
	lines = append(lines, "", "Use `nw plan backup --json` for exact JSON output, or add `--target` for a custom repo.")
	return fitLines(lines, width)
}

func (m model) filteredFindings() []inventory.Finding {
	filtered := make([]inventory.Finding, 0, len(m.report.Findings))
	for _, finding := range m.report.Findings {
		if m.severity != "" && string(finding.Severity) != m.severity {
			continue
		}
		if m.tool != "" && finding.Tool != m.tool {
			continue
		}
		if m.rule != "" && finding.Rule != m.rule {
			continue
		}
		if m.search != "" && !findingMatchesSearch(finding, m.search) {
			continue
		}
		filtered = append(filtered, finding)
	}
	return filtered
}

func findingMatchesSearch(finding inventory.Finding, query string) bool {
	query = strings.ToLower(query)
	haystack := strings.ToLower(strings.Join([]string{
		finding.ID,
		finding.Tool,
		finding.Path,
		finding.Server,
		finding.Rule,
		finding.Message,
		finding.Evidence,
		finding.Recommendation,
		finding.FixSummary,
	}, "\n"))
	return strings.Contains(haystack, query)
}

func (m model) help(width int) string {
	lines := []string{
		section("Help"),
		"1-6 or tab: switch tabs",
		"p: open command palette",
		"arrows or h/j/k/l: navigate rows",
		"s/t/r: cycle severity, tool, and rule filters in Findings",
		"/: search findings",
		"x: clear finding filters",
		"c: copy selected path, recommendation, or fix step to clipboard",
		"e: export a redacted fix plan to ~/.local/state/nightward/exports",
		"o: open docs URL for the selected finding or fix",
		"?: toggle this help",
		"q or esc: quit",
		"",
		"Nightward TUI actions do not mutate agent configs.",
	}
	return fitLines(lines, width)
}

func (m model) footerHelp(width int) string {
	helpModel := m.helpModel
	if helpModel.Width == 0 {
		helpModel = newHelpModel()
	}
	helpModel.Width = width
	return helpModel.View(tuiKeys)
}

func (m model) searchInputModel() textinput.Model {
	input := m.searchInput
	if input.Width == 0 && input.Placeholder == "" {
		input = newSearchInput(m.search)
	}
	input.Focus()
	return input
}

func (m model) searchInputValue() string {
	return strings.TrimSpace(m.searchInputModel().Value())
}

func newHelpModel() help.Model {
	h := help.New()
	h.ShortSeparator = "  "
	return h
}

func newSearchInput(value string) textinput.Model {
	input := textinput.New()
	input.Placeholder = "rule, path, tool, server, or ID"
	input.Prompt = ""
	input.CharLimit = 160
	input.Width = 42
	input.TextStyle = lipgloss.NewStyle().Foreground(ink)
	input.PlaceholderStyle = lipgloss.NewStyle().Foreground(muted)
	input.SetValue(value)
	return input
}

func (m model) commandPalette(width, height int) string {
	commands := m.paletteCommands()
	lines := []string{
		section("Command Palette"),
		"Choose an action. Nothing here mutates agent config.",
		"",
	}
	visible := max(1, height-4)
	if m.paletteCursor >= len(commands) {
		m.paletteCursor = len(commands) - 1
	}
	if m.paletteCursor < 0 {
		m.paletteCursor = 0
	}
	start := clampCursor(m.paletteCursor, len(commands), visible)
	for i := start; i < len(commands) && len(lines) < visible+3; i++ {
		command := commands[i]
		prefix := "  "
		if i == m.paletteCursor {
			prefix = "> "
		}
		line := fmt.Sprintf("%s%-24s %s", prefix, command.Title, command.Detail)
		lines = append(lines, line)
	}
	return fitLines(lines, width)
}

func (m model) paletteCommands() []paletteCommand {
	commands := []paletteCommand{
		{Title: "Dashboard", Detail: "show scan summary and next actions", Action: "tab:0"},
		{Title: "Inventory", Detail: "review discovered config paths", Action: "tab:1"},
		{Title: "Findings", Detail: "filter and inspect MCP/config risks", Action: "tab:2"},
		{Title: "Analysis", Detail: "review normalized signals", Action: "tab:3"},
		{Title: "Fix Plan", Detail: "review plan-only remediation", Action: "tab:4"},
		{Title: "Backup Plan", Detail: "preview dotfiles backup choices", Action: "tab:5"},
		{Title: "Copy Selection", Detail: "copy selected redacted action or path", Action: "copy"},
		{Title: "Export Fix Plan", Detail: "write redacted markdown review material", Action: "export"},
	}
	if _, ok := m.currentDocsURL(); ok {
		commands = append(commands, paletteCommand{Title: "Open Docs", Detail: "open docs for selected row", Action: "docs"})
	}
	if m.tab == 2 {
		commands = append(commands,
			paletteCommand{Title: "Search Findings", Detail: "filter by rule, path, server, or evidence", Action: "search"},
			paletteCommand{Title: "Cycle Severity", Detail: "advance severity filter", Action: "filter:severity"},
			paletteCommand{Title: "Cycle Tool", Detail: "advance tool filter", Action: "filter:tool"},
			paletteCommand{Title: "Cycle Rule", Detail: "advance rule filter", Action: "filter:rule"},
		)
		if m.search != "" || m.severity != "" || m.tool != "" || m.rule != "" {
			commands = append(commands, paletteCommand{Title: "Clear Filters", Detail: "reset finding filters and search", Action: "filters:clear"})
		}
	}
	return commands
}

func (m model) applyPaletteCommand(command paletteCommand) (model, tea.Cmd) {
	switch command.Action {
	case "tab:0", "tab:1", "tab:2", "tab:3", "tab:4", "tab:5":
		tab := int(command.Action[len(command.Action)-1] - '0')
		if tab >= 0 && tab < len(tabs) {
			m.tab = tab
			m.cursor = 0
			m.status = "opened " + tabs[tab]
		}
	case "copy":
		value, label, ok := m.copySelection()
		if !ok {
			m.status = "copy: nothing selected"
			return m, nil
		}
		m.status = "copying " + label + "..."
		return m, copyToClipboardCmd(value, label)
	case "export":
		m.status = "exporting fix plan..."
		return m, exportFixPlanCmd(m.report)
	case "docs":
		docsURL, ok := m.currentDocsURL()
		if !ok {
			m.status = "open docs: no docs URL for selected row"
			return m, nil
		}
		m.status = "opening docs..."
		return m, openURLCmd(docsURL)
	case "search":
		m.searching = true
		m.status = "type search, enter to keep, esc to cancel"
	case "filter:severity":
		m.severity = cycle(m.severity, riskOptions(m.report.Findings))
		m.cursor = 0
		m.status = "severity: " + filterLabel(m.severity)
	case "filter:tool":
		m.tool = cycle(m.tool, toolOptions(m.report.Findings))
		m.cursor = 0
		m.status = "tool: " + filterLabel(m.tool)
	case "filter:rule":
		m.rule = cycle(m.rule, ruleOptions(m.report.Findings))
		m.cursor = 0
		m.status = "rule: " + filterLabel(m.rule)
	case "filters:clear":
		m.search = ""
		m.searchInput = newSearchInput("")
		m.severity = ""
		m.tool = ""
		m.rule = ""
		m.cursor = 0
		m.status = "filters cleared"
	}
	return m, nil
}

func (m model) currentSignal() (analysis.Signal, bool) {
	report := analysis.Run(m.report, analysis.Options{Mode: m.report.ScanMode, Workspace: m.report.Workspace})
	if len(report.Signals) == 0 {
		return analysis.Signal{}, false
	}
	cursor := m.cursor
	if cursor >= len(report.Signals) {
		cursor = len(report.Signals) - 1
	}
	return report.Signals[cursor], true
}

func (m model) currentFinding() (inventory.Finding, bool) {
	findings := m.filteredFindings()
	if len(findings) == 0 {
		return inventory.Finding{}, false
	}
	cursor := m.cursor
	if cursor >= len(findings) {
		cursor = len(findings) - 1
	}
	return findings[cursor], true
}

func (m model) currentItem() (inventory.Item, bool) {
	if len(m.report.Items) == 0 {
		return inventory.Item{}, false
	}
	cursor := m.cursor
	if cursor >= len(m.report.Items) {
		cursor = len(m.report.Items) - 1
	}
	return m.report.Items[cursor], true
}

func (m model) currentFix() (fixplan.Fix, bool) {
	plan := fixplan.Build(m.report, fixplan.Selector{All: true})
	if len(plan.Fixes) == 0 {
		return fixplan.Fix{}, false
	}
	cursor := m.cursor
	if cursor >= len(plan.Fixes) {
		cursor = len(plan.Fixes) - 1
	}
	return plan.Fixes[cursor], true
}

func (m model) currentBackupEntry() (backupplan.Entry, bool) {
	plan := backupplan.Build(m.report, filepath.Join(m.report.Home, "dotfiles"))
	if len(plan.Entries) == 0 {
		return backupplan.Entry{}, false
	}
	cursor := m.cursor
	if cursor >= len(plan.Entries) {
		cursor = len(plan.Entries) - 1
	}
	return plan.Entries[cursor], true
}

func (m model) copySelection() (string, string, bool) {
	switch m.tab {
	case 0:
		if len(m.schedule.History) > 0 && m.schedule.History[0].Path != "" {
			return m.schedule.History[0].Path, "latest report", true
		}
		if m.schedule.ReportDir != "" {
			return m.schedule.ReportDir, "report path", true
		}
	case 1:
		if item, ok := m.currentItem(); ok {
			return item.Path, "path", true
		}
	case 2:
		if finding, ok := m.currentFinding(); ok {
			return findingCopyText(finding), "finding action", true
		}
	case 3:
		if signal, ok := m.currentSignal(); ok {
			return signal.Recommendation, "analysis recommendation", true
		}
	case 4:
		if fix, ok := m.currentFix(); ok {
			return fixCopyText(fix), "fix step", true
		}
	case 5:
		if entry, ok := m.currentBackupEntry(); ok {
			return entry.Source, "backup source path", true
		}
	}
	return "", "", false
}

func (m model) nextActions() []string {
	if m.report.Summary.TotalFindings == 0 && len(m.report.Findings) == 0 {
		if m.schedule.ReportDir != "" {
			return []string{"Run `nw scan --output-dir " + m.schedule.ReportDir + "` before syncing shared dotfiles."}
		}
		return []string{"Run `nw scan --json` before syncing shared dotfiles."}
	}
	if m.report.Summary.FindingsBySeverity[inventory.RiskCritical] > 0 || m.report.Summary.FindingsBySeverity[inventory.RiskHigh] > 0 || maxRisk(m.report.Findings) == inventory.RiskCritical || maxRisk(m.report.Findings) == inventory.RiskHigh {
		return []string{
			"Review Findings and Fix Plan before syncing or publishing config.",
			"Export a redacted fix plan with `e` for review material.",
		}
	}
	if !m.schedule.Installed {
		return []string{"Preview a nightly schedule with `nw schedule plan --json`."}
	}
	if len(m.schedule.History) > 1 {
		return []string{"Compare recent reports before publishing screenshots or store metadata."}
	}
	return []string{"Run explicit local providers with `nw analyze --with gitleaks,trufflehog,semgrep --json`."}
}

func reportDelta(history []schedule.ReportRecord) string {
	if len(history) < 2 {
		return ""
	}
	delta := history[0].Findings - history[1].Findings
	switch {
	case delta > 0:
		return fmt.Sprintf("+%d findings since previous report", delta)
	case delta < 0:
		return fmt.Sprintf("%d findings since previous report", delta)
	default:
		return "no finding change since previous report"
	}
}

func reportSeverityDelta(history []schedule.ReportRecord) string {
	if len(history) < 2 {
		return ""
	}
	var parts []string
	for _, severity := range []inventory.RiskLevel{inventory.RiskCritical, inventory.RiskHigh, inventory.RiskMedium, inventory.RiskLow, inventory.RiskInfo} {
		delta := history[0].FindingsBySeverity[severity] - history[1].FindingsBySeverity[severity]
		if delta == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %+d", severity, delta))
	}
	return strings.Join(parts, ", ")
}

func (m model) currentDocsURL() (string, bool) {
	switch m.tab {
	case 2:
		if finding, ok := m.currentFinding(); ok {
			if finding.DocsURL != "" {
				return finding.DocsURL, true
			}
			return ruleDocsURL(finding.Rule), true
		}
	case 3:
		return analysisDocsURL, true
	case 4:
		if fix, ok := m.currentFix(); ok && fix.Rule != "" {
			return ruleDocsURL(fix.Rule), true
		}
	}
	return "", false
}

func ruleDocsURL(rule string) string {
	if rule == "" {
		return remediationDocsURL
	}
	return remediationDocsURL + "#" + url.QueryEscape(strings.ToLower(rule))
}

func findingCopyText(finding inventory.Finding) string {
	if len(finding.FixSteps) > 0 {
		return redactTUIText(finding.FixSteps[0])
	}
	if finding.Recommendation != "" {
		return redactTUIText(finding.Recommendation)
	}
	if finding.Evidence != "" {
		return redactTUIText(finding.Evidence)
	}
	return finding.Path
}

func fixCopyText(fix fixplan.Fix) string {
	if len(fix.Steps) > 0 {
		return redactTUIText(fix.Steps[0])
	}
	if fix.Summary != "" {
		return redactTUIText(fix.Summary)
	}
	return fix.Path
}

func findingDetail(finding inventory.Finding, width, height int) string {
	lines := []string{
		section("Detail"),
		finding.ID,
		fmt.Sprintf("%s / %s / %s", finding.Tool, finding.Severity, finding.Rule),
		"",
		redactTUIText(finding.Message),
	}
	if finding.Evidence != "" {
		lines = append(lines, "", section("Evidence"), redactTUIText(finding.Evidence))
	}
	if finding.Impact != "" {
		lines = append(lines, "", section("Impact"), redactTUIText(finding.Impact))
	}
	if finding.FixAvailable {
		lines = append(lines, "", section("Suggested Fix"), fmt.Sprintf("%s  confidence=%s  risk=%s", finding.FixKind, finding.Confidence, finding.Risk), redactTUIText(finding.FixSummary))
		for i, step := range finding.FixSteps {
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, redactTUIText(step)))
		}
	}
	if finding.Why != "" {
		lines = append(lines, "", section("Why"), redactTUIText(finding.Why))
	}
	return renderViewport(lines, width, height)
}

func signalDetail(signal analysis.Signal, width, height int) string {
	lines := []string{
		section("Signal Detail"),
		signal.ID,
		fmt.Sprintf("%s / %s / %s", signal.Provider, signal.Severity, signal.Rule),
		"",
		redactTUIText(signal.Message),
	}
	if signal.Evidence != "" {
		lines = append(lines, "", section("Evidence"), redactTUIText(signal.Evidence))
	}
	lines = append(lines, "", section("Recommendation"), redactTUIText(signal.Recommendation))
	lines = append(lines, "", section("Review"), fmt.Sprintf("confidence=%s  category=%s", signal.Confidence, signal.Category))
	if signal.Why != "" {
		lines = append(lines, "", section("Why"), redactTUIText(signal.Why))
	}
	return renderViewport(lines, width, height)
}

func fixDetail(fix fixplan.Fix, width, height int) string {
	lines := []string{
		section("Fix Detail"),
		fix.FindingID,
		fmt.Sprintf("%s / %s / %s", fix.Tool, fix.Status, fix.Rule),
		"",
		redactTUIText(fix.Summary),
	}
	if fix.Evidence != "" {
		lines = append(lines, "", section("Evidence"), redactTUIText(fix.Evidence))
	}
	if fix.Impact != "" {
		lines = append(lines, "", section("Impact"), redactTUIText(fix.Impact))
	}
	lines = append(lines, "", section("Review"), fmt.Sprintf("kind=%s  confidence=%s  risk=%s  requires_review=%t", fix.FixKind, fix.Confidence, fix.Risk, fix.RequiresReview))
	if len(fix.Steps) > 0 {
		lines = append(lines, "", section("Steps"))
		for i, step := range fix.Steps {
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, redactTUIText(step)))
		}
	}
	if fix.Why != "" {
		lines = append(lines, "", section("Why"), redactTUIText(fix.Why))
	}
	return renderViewport(lines, width, height)
}

func copyToClipboardCmd(value, label string) tea.Cmd {
	value = redactTUIText(value)
	cmd, err := clipboardCommand(value)
	if err != nil {
		return actionStatusCmd(fmt.Sprintf("copy failed: %v", err))
	}
	return runActionCommand(cmd, "copied "+label)
}

func openURLCmd(target string) tea.Cmd {
	cmd, err := openURLCommand(target)
	if err != nil {
		return actionStatusCmd(fmt.Sprintf("open docs failed: %v", err))
	}
	return runActionCommand(cmd, "opened docs")
}

func exportFixPlanCmd(report inventory.Report) tea.Cmd {
	return func() tea.Msg {
		path, err := exportFixPlan(report, time.Now().UTC())
		if err != nil {
			return actionMsg{status: fmt.Sprintf("export failed: %v", err)}
		}
		return actionMsg{status: "exported fix plan: " + path}
	}
}

func actionStatusCmd(status string) tea.Cmd {
	return func() tea.Msg {
		return actionMsg{status: status}
	}
}

func runActionCommand(cmd *exec.Cmd, success string) tea.Cmd {
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return actionMsg{status: fmt.Sprintf("%s failed: %v", success, err)}
		}
		return actionMsg{status: success}
	})
}

func clipboardCommand(value string) (*exec.Cmd, error) {
	return clipboardCommandFor(runtime.GOOS, value, exec.LookPath)
}

func clipboardCommandFor(goos, value string, lookPath func(string) (string, error)) (*exec.Cmd, error) {
	var cmd *exec.Cmd
	switch goos {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if path, err := lookPath("wl-copy"); err == nil {
			cmd = exec.Command(path) // #nosec G204 -- clipboard helper resolved by PATH and invoked without a shell.
		} else if path, err := lookPath("xclip"); err == nil {
			cmd = exec.Command(path, "-selection", "clipboard") // #nosec G204 -- clipboard helper resolved by PATH and invoked without a shell.
		} else if path, err := lookPath("xsel"); err == nil {
			cmd = exec.Command(path, "--clipboard", "--input") // #nosec G204 -- clipboard helper resolved by PATH and invoked without a shell.
		} else {
			return nil, errors.New("no clipboard command found: install wl-copy, xclip, or xsel")
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return nil, fmt.Errorf("clipboard unsupported on %s", goos)
	}
	cmd.Stdin = strings.NewReader(value)
	return cmd, nil
}

func openURLCommand(target string) (*exec.Cmd, error) {
	return openURLCommandFor(runtime.GOOS, target)
}

func openURLCommandFor(goos, target string) (*exec.Cmd, error) {
	parsed, err := url.Parse(target)
	if err != nil || parsed == nil || parsed.Host == "" {
		return nil, fmt.Errorf("invalid URL %q", target)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return nil, fmt.Errorf("unsupported URL scheme %q", parsed.Scheme)
	}
	switch goos {
	case "darwin":
		return exec.Command("open", target), nil // #nosec G204 -- validated http(s) documentation URL, invoked without a shell.
	case "linux":
		return exec.Command("xdg-open", target), nil // #nosec G204 -- validated http(s) documentation URL, invoked without a shell.
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", target), nil // #nosec G204 -- validated http(s) documentation URL, invoked without a shell.
	default:
		return nil, fmt.Errorf("opening URLs unsupported on %s", goos)
	}
}

func exportFixPlan(report inventory.Report, now time.Time) (string, error) {
	if report.Home == "" {
		return "", errors.New("home directory missing from report")
	}
	dir := filepath.Join(report.Home, ".local", "state", "nightward", "exports")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	plan := fixplan.Build(report, fixplan.Selector{All: true})
	markdown := redactTUIText(fixplan.Markdown(plan))
	name := "fix-plan-" + now.UTC().Format("20060102T150405Z") + ".md"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(markdown), 0600); err != nil {
		return "", err
	}
	return path, nil
}

func redactTUIText(value string) string {
	value = tuiSecretAssignmentPattern.ReplaceAllString(value, "$1$2[redacted]")
	return tuiLongSecretPattern.ReplaceAllString(value, "[redacted]")
}

func riskOptions(findings []inventory.Finding) []string {
	values := make([]string, 0, len(findings))
	for _, finding := range findings {
		values = append(values, string(finding.Severity))
	}
	return unique(values)
}

func toolOptions(findings []inventory.Finding) []string {
	values := make([]string, 0, len(findings))
	for _, finding := range findings {
		values = append(values, finding.Tool)
	}
	return unique(values)
}

func ruleOptions(findings []inventory.Finding) []string {
	values := make([]string, 0, len(findings))
	for _, finding := range findings {
		values = append(values, finding.Rule)
	}
	return unique(values)
}

func unique(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func cycle(current string, options []string) string {
	if len(options) == 0 {
		return ""
	}
	if current == "" {
		return options[0]
	}
	for i, option := range options {
		if option == current {
			if i == len(options)-1 {
				return ""
			}
			return options[i+1]
		}
	}
	return ""
}

func filterLabel(value string) string {
	if value == "" {
		return "all"
	}
	return value
}

func section(label string) string {
	return lipgloss.NewStyle().Foreground(cyan).Bold(true).Render(label)
}

func metricLine(label string, value int, color lipgloss.Color) string {
	return lipgloss.NewStyle().Foreground(color).Bold(true).Render(fmt.Sprintf("%-10s %d", label, value))
}

func renderTable(cols []table.Column, rows []table.Row, cursor, width, height int, accent lipgloss.Color) string {
	if len(rows) == 0 {
		return "No rows."
	}
	if accent == "" {
		accent = cyan
	}
	styles := table.DefaultStyles()
	styles.Header = lipgloss.NewStyle().
		Foreground(accent).
		Bold(true).
		Padding(0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(panelLine)
	styles.Cell = lipgloss.NewStyle().Foreground(ink).Padding(0, 1)
	styles.Selected = lipgloss.NewStyle().Foreground(bg).Background(accent).Bold(true).Padding(0, 1)
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithWidth(width),
		table.WithHeight(max(3, height)),
		table.WithStyles(styles),
	)
	t.SetCursor(cursor)
	return t.View()
}

func tabAccent(index int) lipgloss.Color {
	if index < 0 || index >= len(tabAccents) {
		return cyan
	}
	return tabAccents[index]
}

func renderViewport(lines []string, width, height int) string {
	view := viewport.New(max(1, width), max(1, height))
	view.SetContent(fitLines(lines, width))
	return view.View()
}

func byteSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(size)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(size)/(1024*1024))
}

func severityColor(risk inventory.RiskLevel) lipgloss.Color {
	switch risk {
	case inventory.RiskCritical, inventory.RiskHigh:
		return red
	case inventory.RiskMedium:
		return amber
	case inventory.RiskLow:
		return blue
	default:
		return green
	}
}

func maxRisk(findings []inventory.Finding) inventory.RiskLevel {
	max := inventory.RiskInfo
	for _, finding := range findings {
		if rank(finding.Severity) > rank(max) {
			max = finding.Severity
		}
	}
	return max
}

func rank(risk inventory.RiskLevel) int {
	switch risk {
	case inventory.RiskCritical:
		return 5
	case inventory.RiskHigh:
		return 4
	case inventory.RiskMedium:
		return 3
	case inventory.RiskLow:
		return 2
	default:
		return 1
	}
}

func fitLines(lines []string, width int) string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		for _, segment := range strings.Split(line, "\n") {
			out = append(out, truncate(segment, width))
		}
	}
	return strings.Join(out, "\n")
}

func truncate(line string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(line) <= width {
		return line
	}
	suffix := "..."
	if width <= len(suffix) {
		suffix = ""
	}
	var b strings.Builder
	for _, r := range line {
		next := b.String() + string(r)
		if lipgloss.Width(next+suffix) > width {
			break
		}
		b.WriteRune(r)
	}
	return b.String() + suffix
}

func clampCursor(cursor, total, visible int) int {
	if total == 0 || cursor < visible {
		return 0
	}
	if cursor >= total {
		cursor = total - 1
	}
	start := cursor - visible + 1
	if start < 0 {
		return 0
	}
	return start
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
