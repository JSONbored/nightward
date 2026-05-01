package reporthtml

import (
	"bytes"
	"html/template"
	"time"

	"github.com/jsonbored/nightward/internal/inventory"
)

type viewModel struct {
	GeneratedAt    string
	Hostname       string
	Home           string
	Workspace      string
	Summary        inventory.Summary
	Items          []inventory.Item
	Findings       []inventory.Finding
	HighFindings   int
	MediumFindings int
}

func Render(report inventory.Report) (string, error) {
	generatedAt := report.GeneratedAt.Format(time.RFC3339)
	if report.GeneratedAt.IsZero() {
		generatedAt = "unknown"
	}
	model := viewModel{
		GeneratedAt:    generatedAt,
		Hostname:       report.Hostname,
		Home:           report.Home,
		Workspace:      report.Workspace,
		Summary:        report.Summary,
		Items:          limitItems(report.Items, 200),
		Findings:       report.Findings,
		HighFindings:   report.Summary.FindingsBySeverity[inventory.RiskHigh],
		MediumFindings: report.Summary.FindingsBySeverity[inventory.RiskMedium],
	}
	var buf bytes.Buffer
	if err := reportTemplate.Execute(&buf, model); err != nil {
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

var reportTemplate = template.Must(template.New("report").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Nightward Report</title>
  <style>
    :root { color-scheme: dark; font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #071014; color: #e6fffb; }
    body { margin: 0; padding: 32px; }
    main { max-width: 1120px; margin: 0 auto; }
    h1, h2 { letter-spacing: 0; }
    .meta, .card { border: 1px solid #1f3b3d; border-radius: 8px; background: #0b171b; padding: 16px; margin: 16px 0; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 12px; }
    .metric { border: 1px solid #1f3b3d; border-radius: 8px; padding: 12px; background: #091317; }
    .metric strong { display: block; color: #5eead4; font-size: 28px; }
    table { width: 100%; border-collapse: collapse; margin: 12px 0; }
    th, td { border-bottom: 1px solid #1f3b3d; padding: 10px; text-align: left; vertical-align: top; }
    th { color: #99f6e4; font-weight: 700; }
    code { color: #c7f9f4; overflow-wrap: anywhere; }
    .finding { border-left: 4px solid #5eead4; padding-left: 12px; margin: 14px 0; }
    .severity-high, .severity-critical { border-left-color: #fb7185; }
    .severity-medium { border-left-color: #facc15; }
    .note { color: #b7c9ca; }
  </style>
</head>
<body>
<main>
  <h1>Nightward Report</h1>
  <p class="note">Generated {{ .GeneratedAt }}. This static report is rendered from redacted Nightward JSON.</p>
  <section class="meta">
    <div><strong>Host:</strong> {{ .Hostname }}</div>
    <div><strong>Home:</strong> <code>{{ .Home }}</code></div>
    {{ if .Workspace }}<div><strong>Workspace:</strong> <code>{{ .Workspace }}</code></div>{{ end }}
  </section>
  <section class="grid">
    <div class="metric"><strong>{{ .Summary.TotalItems }}</strong> items</div>
    <div class="metric"><strong>{{ .Summary.TotalFindings }}</strong> findings</div>
    <div class="metric"><strong>{{ .HighFindings }}</strong> high findings</div>
    <div class="metric"><strong>{{ .MediumFindings }}</strong> medium findings</div>
  </section>
  <section class="card">
    <h2>Findings</h2>
    {{ if .Findings }}
      {{ range .Findings }}
      <article class="finding severity-{{ .Severity }}">
        <h3>{{ .Rule }}: {{ .Message }}</h3>
        <p><strong>Severity:</strong> {{ .Severity }} · <strong>Tool:</strong> {{ .Tool }}</p>
        <p><code>{{ .Path }}</code></p>
        {{ if .Evidence }}<p><strong>Evidence:</strong> <code>{{ .Evidence }}</code></p>{{ end }}
        <p><strong>Recommendation:</strong> {{ .Recommendation }}</p>
      </article>
      {{ end }}
    {{ else }}
      <p>No findings.</p>
    {{ end }}
  </section>
  <section class="card">
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
