package reporthtml

import (
	"bytes"
	"html/template"
	"sort"
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
	Records     []schedule.ReportRecord
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
		Records:     records,
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
		label := "manual review"
		if finding.FixAvailable && finding.FixKind != "" {
			label = string(finding.FixKind)
		}
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

var reportTemplate = template.Must(template.New("report").Parse(`<!doctype html>
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
      <div class="toolbar">
      {{ range .FindingGroups }}<a class="pill" href="#severity-{{ .Severity }}">{{ .Severity }} ({{ len .Findings }})</a>{{ end }}
      </div>
      {{ range .FindingGroups }}
      <section id="severity-{{ .Severity }}">
        <h3>{{ .Severity }}</h3>
        {{ range .Findings }}
        <details class="finding severity-{{ .Severity }}">
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
  </style>
</head>
<body>
<main>
  <h1>Nightward Report History</h1>
  <p class="note">Generated {{ .GeneratedAt }} from local report files.</p>
  <table>
    <thead><tr><th>Report</th><th>Modified</th><th>Findings</th><th>Size</th></tr></thead>
    <tbody>
    {{ range .Records }}
      <tr><td><code>{{ .Path }}</code></td><td>{{ .ModTime }}</td><td>{{ .Findings }}</td><td>{{ .SizeBytes }}</td></tr>
    {{ else }}
      <tr><td colspan="4">No JSON reports found.</td></tr>
    {{ end }}
    </tbody>
  </table>
</main>
</body>
</html>
`))
