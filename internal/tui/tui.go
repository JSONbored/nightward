package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shadowbook/nightward/internal/backupplan"
	"github.com/shadowbook/nightward/internal/fixplan"
	"github.com/shadowbook/nightward/internal/inventory"
	"github.com/shadowbook/nightward/internal/schedule"
)

type model struct {
	report    inventory.Report
	schedule  schedule.Plan
	tab       int
	cursor    int
	width     int
	height    int
	severity  string
	tool      string
	rule      string
	search    string
	status    string
	searching bool
	showHelp  bool
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

var tabs = []string{"Dashboard", "Inventory", "Findings", "Fix Plan", "Backup Plan"}

func Run(report inventory.Report, scheduleStatus schedule.Plan) error {
	_, err := tea.NewProgram(model{report: report, schedule: scheduleStatus}, tea.WithAltScreen()).Run()
	return err
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if m.searching {
			switch msg.String() {
			case "enter":
				m.searching = false
				m.status = "search: " + filterLabel(m.search)
			case "esc":
				m.searching = false
			case "backspace":
				if len(m.search) > 0 {
					m.search = m.search[:len(m.search)-1]
					m.cursor = 0
				}
			case "ctrl+u":
				m.search = ""
				m.cursor = 0
			default:
				if len(msg.Runes) > 0 {
					m.search += string(msg.Runes)
					m.cursor = 0
				}
			}
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
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
				m.status = "type search, enter to keep, esc to cancel"
			}
		case "x":
			if m.tab == 2 {
				m.search = ""
				m.severity = ""
				m.tool = ""
				m.rule = ""
				m.cursor = 0
				m.status = "filters cleared"
			}
		case "c":
			if finding, ok := m.currentFinding(); ok && len(finding.FixSteps) > 0 {
				m.status = "copy: " + finding.FixSteps[0]
			}
		case "e":
			m.status = "export: nw fix export --format markdown"
		case "o":
			if finding, ok := m.currentFinding(); ok && finding.DocsURL != "" {
				m.status = "open docs: " + finding.DocsURL
			} else {
				m.status = "open docs: no docs URL for selected finding"
			}
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

	tabLine := lipgloss.JoinHorizontal(lipgloss.Top, titleStyle.Render("nightward"), m.renderTabs())
	bodyText := m.renderBody(bodyWidth-6, bodyHeight-2)
	if m.showHelp {
		bodyText = m.help(bodyWidth - 6)
	}
	body := panelStyle.Width(bodyWidth).Height(bodyHeight).Render(bodyText)
	footerText := "1-5 tabs  arrows/hjkl navigate  / search  s/t/r filters  x clear  ? help  q quit"
	if m.searching {
		footerText = "search: " + m.search
	}
	if m.status != "" {
		footerText += "  " + m.status
	}
	footer := footerStyle.Render(footerText)
	return baseStyle.Render(lipgloss.JoinVertical(lipgloss.Left, tabLine, body, footer))
}

func (m model) renderTabs() string {
	rendered := make([]string, 0, len(tabs))
	for i, tab := range tabs {
		label := fmt.Sprintf("%d %s", i+1, tab)
		if i == m.tab {
			rendered = append(rendered, activeTabStyle.Render(label))
		} else {
			rendered = append(rendered, tabStyle.Render(label))
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
		if count := m.report.Summary.ByClassification[class]; count > 0 {
			lines = append(lines, fmt.Sprintf("%-14s %d", class, count))
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
	return fitLines(lines, width)
}

func (m model) inventory(width, height int) string {
	lines := []string{section("Inventory"), ""}
	items := m.report.Items
	if len(items) == 0 {
		return "No known AI agent/devtool config paths found yet."
	}
	visible := max(1, height-4)
	start := clampCursor(m.cursor, len(items), visible)
	for i := start; i < len(items) && len(lines) < visible+2; i++ {
		item := items[i]
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		line := fmt.Sprintf("%s%-10s %-14s %-12s %s", prefix, item.Tool, item.Classification, item.Risk, item.Path)
		lines = append(lines, line)
	}
	return fitLines(lines, width)
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
	visible := max(1, height-5)
	if m.cursor >= len(findings) {
		m.cursor = len(findings) - 1
	}
	start := clampCursor(m.cursor, len(findings), visible)
	for i := start; i < len(findings) && len(lines) < visible+2; i++ {
		finding := findings[i]
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		line := fmt.Sprintf("%s%-8s %-22s %s", prefix, finding.Severity, finding.Rule, finding.Message)
		lines = append(lines, line)
	}
	left := fitLines(lines, listWidth)
	if detailWidth <= 0 {
		return left
	}
	detail := findingDetail(findings[m.cursor], detailWidth, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", detail)
}

func (m model) fixPlan(width, height int) string {
	plan := fixplan.Build(m.report, fixplan.Selector{All: true})
	lines := []string{
		section("Fix Plan"),
		fmt.Sprintf("Safe: %d  Review: %d  Blocked: %d", plan.Summary.Safe, plan.Summary.Review, plan.Summary.Blocked),
		"",
	}
	if len(plan.Fixes) == 0 {
		return "No fix plans available."
	}
	visible := max(1, height-5)
	if m.cursor >= len(plan.Fixes) {
		m.cursor = len(plan.Fixes) - 1
	}
	start := clampCursor(m.cursor, len(plan.Fixes), visible)
	for i := start; i < len(plan.Fixes) && len(lines) < visible+3; i++ {
		fix := plan.Fixes[i]
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%-7s %-20s %-22s %s", prefix, fix.Status, fix.FixKind, fix.Rule, fix.Summary))
	}
	lines = append(lines, "", "Use `nw fix plan --all --json` or `nw fix export --format markdown` for full steps.")
	return fitLines(lines, width)
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
	visible := max(1, height-6)
	start := clampCursor(m.cursor, len(plan.Entries), visible)
	for i := start; i < len(plan.Entries) && len(lines) < visible+4; i++ {
		entry := plan.Entries[i]
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%-7s %-12s %s", prefix, entry.Action, entry.Tool, entry.Source))
	}
	lines = append(lines, "", "Use `nightward plan backup --target <repo>` for exact JSON or shell output.")
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
		"1-5 or tab: switch tabs",
		"arrows or h/j/k/l: navigate rows",
		"s/t/r: cycle severity, tool, and rule filters in Findings",
		"/: search findings",
		"x: clear finding filters",
		"c: show first suggested command or step for selected finding",
		"e: show export command for fix plan",
		"o: show docs URL for selected finding",
		"?: toggle this help",
		"q or esc: quit",
		"",
		"Nightward TUI actions do not mutate agent configs.",
	}
	return fitLines(lines, width)
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

func findingDetail(finding inventory.Finding, width, height int) string {
	lines := []string{
		section("Detail"),
		finding.ID,
		fmt.Sprintf("%s / %s / %s", finding.Tool, finding.Severity, finding.Rule),
		"",
		finding.Message,
	}
	if finding.Evidence != "" {
		lines = append(lines, "", section("Evidence"), finding.Evidence)
	}
	if finding.Impact != "" {
		lines = append(lines, "", section("Impact"), finding.Impact)
	}
	if finding.FixAvailable {
		lines = append(lines, "", section("Suggested Fix"), fmt.Sprintf("%s  confidence=%s  risk=%s", finding.FixKind, finding.Confidence, finding.Risk), finding.FixSummary)
		for i, step := range finding.FixSteps {
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, step))
		}
	}
	if finding.Why != "" {
		lines = append(lines, "", section("Why"), finding.Why)
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return fitLines(lines, width)
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
		out = append(out, truncate(line, width))
	}
	return strings.Join(out, "\n")
}

func truncate(line string, width int) string {
	if width <= 4 || lipgloss.Width(line) <= width {
		return line
	}
	runes := []rune(line)
	if len(runes) <= width-1 {
		return line
	}
	return string(runes[:width-1]) + "..."
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
