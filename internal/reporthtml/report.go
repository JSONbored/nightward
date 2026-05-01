package reporthtml

import (
	"bytes"
	"html/template"
	"sort"
	"strconv"
	"time"

	"github.com/jsonbored/nightward/internal/inventory"
	"github.com/jsonbored/nightward/internal/reportdiff"
	"github.com/jsonbored/nightward/internal/schedule"
)

type Options struct {
	Diff *reportdiff.Diff
}

type viewModel struct {
	GeneratedAt      string
	Hostname         string
	Home             string
	Workspace        string
	SchemaVersion    int
	Summary          inventory.Summary
	Items            []inventory.Item
	Findings         []inventory.Finding
	FindingGroups    []findingGroup
	FixGroups        []fixGroup
	FilterTools      []string
	FilterRules      []string
	FilterFixes      []string
	CriticalFindings int
	HighFindings     int
	MediumFindings   int
	LowFindings      int
	InfoFindings     int
	Diff             *reportdiff.Diff
}

type findingGroup struct {
	Severity inventory.RiskLevel
	Findings []inventory.Finding
}

type fixGroup struct {
	Label    string
	Findings []inventory.Finding
}

type indexModel struct {
	GeneratedAt string
	Records     []indexRecord
}

type indexRecord struct {
	schedule.ReportRecord
	DeltaLabel      string
	SeverityLabel   string
	SeverityClass   string
	SeveritySummary []severityCount
}

type severityCount struct {
	Label inventory.RiskLevel
	Count int
	Class string
}

func Render(report inventory.Report) (string, error) {
	return RenderWithOptions(report, Options{})
}

func RenderWithOptions(report inventory.Report, options Options) (string, error) {
	generatedAt := report.GeneratedAt.Format(time.RFC3339)
	if report.GeneratedAt.IsZero() {
		generatedAt = "unknown"
	}
	model := viewModel{
		GeneratedAt:      generatedAt,
		Hostname:         report.Hostname,
		Home:             report.Home,
		Workspace:        report.Workspace,
		SchemaVersion:    report.SchemaVersion,
		Summary:          report.Summary,
		Items:            limitItems(report.Items, 200),
		Findings:         report.Findings,
		FindingGroups:    groupFindings(report.Findings),
		FixGroups:        groupFixes(report.Findings),
		FilterTools:      uniqueFindingStrings(report.Findings, func(f inventory.Finding) string { return f.Tool }),
		FilterRules:      uniqueFindingStrings(report.Findings, func(f inventory.Finding) string { return f.Rule }),
		FilterFixes:      uniqueFindingStrings(report.Findings, fixLabel),
		CriticalFindings: report.Summary.FindingsBySeverity[inventory.RiskCritical],
		HighFindings:     report.Summary.FindingsBySeverity[inventory.RiskHigh],
		MediumFindings:   report.Summary.FindingsBySeverity[inventory.RiskMedium],
		LowFindings:      report.Summary.FindingsBySeverity[inventory.RiskLow],
		InfoFindings:     report.Summary.FindingsBySeverity[inventory.RiskInfo],
		Diff:             options.Diff,
	}
	var buf bytes.Buffer
	if err := reportTemplate.Execute(&buf, model); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func RenderIndex(records []schedule.ReportRecord) (string, error) {
	model := indexModel{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Records:     buildIndexRecords(records),
	}
	var buf bytes.Buffer
	if err := indexTemplate.Execute(&buf, model); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func limitItems(items []inventory.Item, limit int) []inventory.Item {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func groupFindings(findings []inventory.Finding) []findingGroup {
	var groups []findingGroup
	for _, severity := range []inventory.RiskLevel{inventory.RiskCritical, inventory.RiskHigh, inventory.RiskMedium, inventory.RiskLow, inventory.RiskInfo} {
		group := findingGroup{Severity: severity}
		for _, finding := range findings {
			if finding.Severity == severity {
				group.Findings = append(group.Findings, finding)
			}
		}
		if len(group.Findings) > 0 {
			groups = append(groups, group)
		}
	}
	return groups
}

func groupFixes(findings []inventory.Finding) []fixGroup {
	grouped := map[string][]inventory.Finding{}
	for _, finding := range findings {
		label := fixLabel(finding)
		grouped[label] = append(grouped[label], finding)
	}
	labels := make([]string, 0, len(grouped))
	for label := range grouped {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	out := make([]fixGroup, 0, len(labels))
	for _, label := range labels {
		out = append(out, fixGroup{Label: label, Findings: grouped[label]})
	}
	return out
}

func uniqueFindingStrings(findings []inventory.Finding, value func(inventory.Finding) string) []string {
	seen := map[string]bool{}
	var out []string
	for _, finding := range findings {
		text := value(finding)
		if text == "" || seen[text] {
			continue
		}
		seen[text] = true
		out = append(out, text)
	}
	sort.Strings(out)
	return out
}

func fixLabel(finding inventory.Finding) string {
	if finding.FixAvailable && finding.FixKind != "" {
		return string(finding.FixKind)
	}
	return "manual review"
}

func buildIndexRecords(records []schedule.ReportRecord) []indexRecord {
	out := make([]indexRecord, 0, len(records))
	previous := -1
	for i, record := range records {
		row := indexRecord{
			ReportRecord:    record,
			DeltaLabel:      "latest",
			SeverityLabel:   "none",
			SeverityClass:   "severity-none",
			SeveritySummary: severitySummary(record.FindingsBySeverity),
		}
		if record.HighestSeverity != "" {
			row.SeverityLabel = string(record.HighestSeverity)
			row.SeverityClass = "severity-" + string(record.HighestSeverity)
		}
		if i > 0 {
			change := record.Findings - previous
			switch {
			case change > 0:
				row.DeltaLabel = "+" + intString(change) + " vs newer"
			case change < 0:
				row.DeltaLabel = intString(change) + " vs newer"
			default:
				row.DeltaLabel = "no change vs newer"
			}
		}
		previous = record.Findings
		out = append(out, row)
	}
	return out
}

func severitySummary(counts map[inventory.RiskLevel]int) []severityCount {
	var out []severityCount
	for _, severity := range []inventory.RiskLevel{inventory.RiskCritical, inventory.RiskHigh, inventory.RiskMedium, inventory.RiskLow, inventory.RiskInfo} {
		if count := counts[severity]; count > 0 {
			out = append(out, severityCount{Label: severity, Count: count, Class: "severity-" + string(severity)})
		}
	}
	return out
}

func intString(value int) string {
	return strconv.Itoa(value)
}

var reportTemplate = template.Must(template.New("report").Funcs(template.FuncMap{"fixLabel": fixLabel}).Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Nightward Report</title>
  <style>
    :root { color-scheme: dark; font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #071014; color: #e6fffb; }
    body { margin: 0; padding: 32px; }
    main { max-width: 1180px; margin: 0 auto; }
    h1, h2, h3 { letter-spacing: 0; }
    .meta, .panel { border: 1px solid #1f3b3d; border-radius: 8px; background: #0b171b; padding: 16px; margin: 16px 0; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(170px, 1fr)); gap: 12px; }
    .metric { border: 1px solid #1f3b3d; border-radius: 8px; padding: 12px; background: #091317; }
    .metric strong { display: block; color: #5eead4; font-size: 28px; }
    .toolbar { display: flex; flex-wrap: wrap; gap: 8px; margin: 12px 0; }
    .pill { border: 1px solid #315559; border-radius: 999px; color: #c7f9f4; padding: 5px 10px; text-decoration: none; }
    .pill:hover { border-color: #5eead4; }
    .filters { display: grid; grid-template-columns: minmax(220px, 1.6fr) repeat(4, minmax(130px, 1fr)); gap: 10px; margin: 12px 0 4px; }
    .filters label { color: #99f6e4; font-size: 12px; font-weight: 700; text-transform: uppercase; }
    .filters input, .filters select { box-sizing: border-box; width: 100%; margin-top: 5px; border: 1px solid #315559; border-radius: 8px; background: #071014; color: #e6fffb; padding: 9px 10px; font: inherit; }
    .empty { border: 1px dashed #315559; border-radius: 8px; color: #b7c9ca; padding: 12px; }
    table { width: 100%; border-collapse: collapse; margin: 12px 0; }
    th, td { border-bottom: 1px solid #1f3b3d; padding: 10px; text-align: left; vertical-align: top; }
    th { color: #99f6e4; font-weight: 700; }
    code { color: #c7f9f4; overflow-wrap: anywhere; }
    details { border: 1px solid #1f3b3d; border-radius: 8px; margin: 12px 0; padding: 10px 12px; background: #091317; }
    details[open] { background: #0b171b; }
    summary { cursor: pointer; font-weight: 700; }
    .finding { border-left: 4px solid #5eead4; padding-left: 12px; }
    .severity-high, .severity-critical { border-left-color: #fb7185; }
    .severity-medium { border-left-color: #facc15; }
    .severity-low { border-left-color: #7dd3fc; }
    .note { color: #b7c9ca; }
    .diff-added { color: #86efac; }
    .diff-removed { color: #fca5a5; }
    .diff-changed { color: #fde68a; }
    [hidden] { display: none !important; }
    @media (max-width: 860px) { body { padding: 18px; } .filters { grid-template-columns: 1fr 1fr; } }
    @media (max-width: 560px) { .filters { grid-template-columns: 1fr; } }
  </style>
</head>
<body>
<main>
  <h1>Nightward Report</h1>
  <p class="note">Generated {{ .GeneratedAt }}. This static report is rendered from redacted Nightward JSON.</p>
  <section class="meta">
    <div><strong>Schema:</strong> <code>{{ .SchemaVersion }}</code></div>
    <div><strong>Host:</strong> {{ .Hostname }}</div>
    <div><strong>Home:</strong> <code>{{ .Home }}</code></div>
    {{ if .Workspace }}<div><strong>Workspace:</strong> <code>{{ .Workspace }}</code></div>{{ end }}
  </section>
  <section class="grid">
    <div class="metric"><strong>{{ .Summary.TotalItems }}</strong> items</div>
    <div class="metric"><strong>{{ .Summary.TotalFindings }}</strong> findings</div>
    <div class="metric"><strong>{{ .CriticalFindings }}</strong> critical findings</div>
    <div class="metric"><strong>{{ .HighFindings }}</strong> high findings</div>
    <div class="metric"><strong>{{ .MediumFindings }}</strong> medium findings</div>
  </section>
  <nav class="toolbar" aria-label="Report sections">
    <a class="pill" href="#findings">Findings</a>
    <a class="pill" href="#remediation">Remediation groups</a>
    <a class="pill" href="#inventory">Inventory sample</a>
    {{ if .Diff }}<a class="pill" href="#changes">Changes since previous scan</a>{{ end }}
  </nav>
  {{ if .Diff }}
  <section class="panel" id="changes">
    <h2>Changes Since Previous Scan</h2>
    <p class="note"><code>{{ .Diff.From }}</code> to <code>{{ .Diff.To }}</code></p>
    <div class="grid">
      <div class="metric"><strong class="diff-added">{{ .Diff.Summary.Added }}</strong> added</div>
      <div class="metric"><strong class="diff-removed">{{ .Diff.Summary.Removed }}</strong> removed</div>
      <div class="metric"><strong class="diff-changed">{{ .Diff.Summary.Changed }}</strong> changed</div>
      <div class="metric"><strong>{{ .Diff.Summary.Unchanged }}</strong> unchanged</div>
    </div>
    {{ if .Diff.Added }}
      <h3>Added</h3>
      {{ range .Diff.Added }}<p><code>{{ .Finding.ID }}</code> {{ .Finding.Rule }}: {{ .Finding.Message }}</p>{{ end }}
    {{ end }}
    {{ if .Diff.Changed }}
      <h3>Changed</h3>
      {{ range .Diff.Changed }}<p><code>{{ .Key }}</code> {{ .After.Rule }} changed fields: <code>{{ range $i, $field := .Fields }}{{ if $i }}, {{ end }}{{ $field }}{{ end }}</code></p>{{ end }}
    {{ end }}
    {{ if .Diff.Removed }}
      <h3>Removed</h3>
      {{ range .Diff.Removed }}<p><code>{{ .Finding.ID }}</code> {{ .Finding.Rule }}: {{ .Finding.Message }}</p>{{ end }}
    {{ end }}
  </section>
  {{ end }}
  <section class="panel" id="findings">
    <h2>Findings</h2>
    {{ if .FindingGroups }}
      <div class="filters" aria-label="Finding filters">
        <label>Search
          <input id="finding-search" type="search" placeholder="rule, path, tool, evidence, server">
        </label>
        <label>Severity
          <select id="severity-filter">
            <option value="">All severities</option>
            {{ range .FindingGroups }}<option value="{{ .Severity }}">{{ .Severity }}</option>{{ end }}
          </select>
        </label>
        <label>Tool
          <select id="tool-filter">
            <option value="">All tools</option>
            {{ range .FilterTools }}<option value="{{ . }}">{{ . }}</option>{{ end }}
          </select>
        </label>
        <label>Rule
          <select id="rule-filter">
            <option value="">All rules</option>
            {{ range .FilterRules }}<option value="{{ . }}">{{ . }}</option>{{ end }}
          </select>
        </label>
        <label>Fix type
          <select id="fix-filter">
            <option value="">All fixes</option>
            {{ range .FilterFixes }}<option value="{{ . }}">{{ . }}</option>{{ end }}
          </select>
        </label>
      </div>
      <p class="note"><span id="finding-count">{{ len .Findings }}</span> of {{ len .Findings }} findings shown. Filters run locally in this file; no data leaves the browser.</p>
      <p class="empty" id="finding-empty" hidden>No findings match the active filters.</p>
      <div class="toolbar">
      {{ range .FindingGroups }}<a class="pill" href="#severity-{{ .Severity }}">{{ .Severity }} ({{ len .Findings }})</a>{{ end }}
      </div>
      {{ range .FindingGroups }}
      <section id="severity-{{ .Severity }}" data-severity-section>
        <h3>{{ .Severity }}</h3>
        {{ range .Findings }}
        <details class="finding severity-{{ .Severity }}" data-finding-card data-severity="{{ .Severity }}" data-tool="{{ .Tool }}" data-rule="{{ .Rule }}" data-fix="{{ fixLabel . }}" data-text="{{ .ID }} {{ .Tool }} {{ .Path }} {{ .Server }} {{ .Severity }} {{ .Rule }} {{ .Message }} {{ .Evidence }} {{ .Recommendation }} {{ .Impact }} {{ .FixSummary }}">
          <summary>{{ .Rule }}: {{ .Message }}</summary>
          <p><strong>Severity:</strong> {{ .Severity }} · <strong>Tool:</strong> {{ .Tool }}</p>
          <p><code>{{ .Path }}</code></p>
          {{ if .Server }}<p><strong>Server:</strong> <code>{{ .Server }}</code></p>{{ end }}
          {{ if .Evidence }}<p><strong>Evidence:</strong> <code>{{ .Evidence }}</code></p>{{ end }}
          {{ if .Impact }}<p><strong>Impact:</strong> {{ .Impact }}</p>{{ end }}
          <p><strong>Recommendation:</strong> {{ .Recommendation }}</p>
          {{ if .FixAvailable }}
          <p><strong>Plan-only remediation:</strong> {{ .FixKind }} · confidence {{ .Confidence }} · risk {{ .Risk }}</p>
          {{ if .FixSummary }}<p>{{ .FixSummary }}</p>{{ end }}
          {{ if .FixSteps }}<ol>{{ range .FixSteps }}<li>{{ . }}</li>{{ end }}</ol>{{ end }}
          {{ end }}
        </details>
        {{ end }}
      </section>
      {{ end }}
    {{ else }}
      <p>No findings.</p>
    {{ end }}
  </section>
  <section class="panel" id="remediation">
    <h2>Remediation Groups</h2>
    {{ if .FixGroups }}
      {{ range .FixGroups }}
      <details>
        <summary>{{ .Label }} ({{ len .Findings }})</summary>
        <ul>{{ range .Findings }}<li><code>{{ .ID }}</code> {{ .Rule }}: {{ .Message }}</li>{{ end }}</ul>
      </details>
      {{ end }}
    {{ else }}
      <p>No remediation groups.</p>
    {{ end }}
  </section>
  <section class="panel" id="inventory">
    <h2>Inventory Sample</h2>
    <table>
      <thead><tr><th>Tool</th><th>Classification</th><th>Risk</th><th>Path</th></tr></thead>
      <tbody>
      {{ range .Items }}
        <tr><td>{{ .Tool }}</td><td>{{ .Classification }}</td><td>{{ .Risk }}</td><td><code>{{ .Path }}</code></td></tr>
      {{ end }}
      </tbody>
    </table>
  </section>
</main>
<script>
(() => {
  const cards = Array.from(document.querySelectorAll("[data-finding-card]"));
  const sections = Array.from(document.querySelectorAll("[data-severity-section]"));
  const search = document.getElementById("finding-search");
  const severity = document.getElementById("severity-filter");
  const tool = document.getElementById("tool-filter");
  const rule = document.getElementById("rule-filter");
  const fix = document.getElementById("fix-filter");
  const count = document.getElementById("finding-count");
  const empty = document.getElementById("finding-empty");
  const value = (element) => (element && element.value ? element.value : "");
  const update = () => {
    const query = value(search).trim().toLowerCase();
    let visible = 0;
    for (const card of cards) {
      const text = (card.dataset.text || "").toLowerCase();
      const show = (!query || text.includes(query)) &&
        (!value(severity) || card.dataset.severity === value(severity)) &&
        (!value(tool) || card.dataset.tool === value(tool)) &&
        (!value(rule) || card.dataset.rule === value(rule)) &&
        (!value(fix) || card.dataset.fix === value(fix));
      card.hidden = !show;
      if (show) visible++;
    }
    for (const section of sections) {
      section.hidden = section.querySelectorAll("[data-finding-card]:not([hidden])").length === 0;
    }
    if (count) count.textContent = String(visible);
    if (empty) empty.hidden = visible !== 0;
  };
  for (const control of [search, severity, tool, rule, fix]) {
    if (control) control.addEventListener("input", update);
  }
  update();
})();
</script>
</body>
</html>
`))

var indexTemplate = template.Must(template.New("index").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Nightward Report History</title>
  <style>
    :root { color-scheme: dark; font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #071014; color: #e6fffb; }
    body { margin: 0; padding: 32px; }
    main { max-width: 980px; margin: 0 auto; }
    table { width: 100%; border-collapse: collapse; margin: 16px 0; }
    th, td { border-bottom: 1px solid #1f3b3d; padding: 10px; text-align: left; vertical-align: top; }
    th { color: #99f6e4; }
    code { color: #c7f9f4; overflow-wrap: anywhere; }
    .note { color: #b7c9ca; }
    .badge { display: inline-block; border: 1px solid #315559; border-radius: 999px; color: #c7f9f4; padding: 3px 8px; margin: 2px 4px 2px 0; white-space: nowrap; }
    .severity-critical, .severity-high { border-color: #fb7185; color: #fecdd3; }
    .severity-medium { border-color: #facc15; color: #fef08a; }
    .severity-low { border-color: #7dd3fc; color: #bae6fd; }
    .severity-info, .severity-none { border-color: #64748b; color: #cbd5e1; }
  </style>
</head>
<body>
<main>
  <h1>Nightward Report History</h1>
  <p class="note">Generated {{ .GeneratedAt }} from local report files.</p>
  <table>
    <thead><tr><th>Report</th><th>Modified</th><th>Findings</th><th>Highest</th><th>Delta</th><th>Size</th></tr></thead>
    <tbody>
    {{ range .Records }}
      <tr>
        <td><strong>{{ .ReportName }}</strong><br><code>{{ .Path }}</code></td>
        <td>{{ .ModTime }}</td>
        <td>
          <strong>{{ .Findings }}</strong>
          {{ range .SeveritySummary }}<span class="badge {{ .Class }}">{{ .Label }} {{ .Count }}</span>{{ end }}
        </td>
        <td><span class="badge {{ .SeverityClass }}">{{ .SeverityLabel }}</span></td>
        <td>{{ .DeltaLabel }}</td>
        <td>{{ .SizeBytes }}</td>
      </tr>
    {{ else }}
      <tr><td colspan="6">No JSON reports found.</td></tr>
    {{ end }}
    </tbody>
  </table>
</main>
</body>
</html>
`))
