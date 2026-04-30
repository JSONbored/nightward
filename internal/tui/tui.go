package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shadowbook/nightward/internal/backupplan"
	"github.com/shadowbook/nightward/internal/inventory"
	"github.com/shadowbook/nightward/internal/schedule"
)

type model struct {
	report   inventory.Report
	schedule schedule.Plan
	tab      int
	cursor   int
	width    int
	height   int
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

var tabs = []string{"Dashboard", "Inventory", "MCP/Security", "Backup Plan"}

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
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
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
	body := panelStyle.Width(bodyWidth).Height(bodyHeight).Render(m.renderBody(bodyWidth-6, bodyHeight-2))
	footer := footerStyle.Render("1-4 tabs  arrows/hjkl navigate  q quit")
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
	lines := []string{section("MCP/Security Findings"), ""}
	findings := m.report.Findings
	if len(findings) == 0 {
		return "No MCP/security findings found in discovered configs."
	}
	visible := max(1, height-4)
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
