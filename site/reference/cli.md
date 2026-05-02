# CLI Reference

This page is generated from `go run ./cmd/nw --help`.

```text
Nightward finds AI-tool risks before you sync.

Usage:
  nw                                Open the TUI
  nw scan [--json] [--workspace DIR] [--output FILE|-] [--output-dir DIR]
  nw doctor [--json]
  nw doctor fix-hints [--workspace DIR] [--json]
  nw plan backup [--target <repo>] [--json]
  nw adapters list [--workspace DIR] [--json]
  nw adapters explain <adapter-name> [--workspace DIR] [--json]
  nw adapters template <adapter-name> [--workspace DIR] [--json]
  nw findings list [--json]
  nw findings explain <finding-id> [--json]
  nw fix plan [--finding <id>|--rule <rule>|--all] [--json]
  nw fix preview [--finding <id>|--rule <rule>|--all] [--format diff|json|markdown]
  nw fix export --format markdown|json
  nw analyze [--all] [--workspace DIR] [--with providers] [--online] [--json]
  nw analyze finding <finding-id> [--workspace DIR] [--json]
  nw analyze package <package> [--with providers] [--online] [--json]
  nw trust explain <finding-id> [--workspace DIR] [--json]
  nw providers list [--json]
  nw providers doctor [--with providers] [--online] [--json]
  nw rules list [--json]
  nw rules explain <rule-id> [--json]
  nw report html [--input scan.json] [--output report.html] [--previous previous.json]
  nw report diff [--from previous.json --to current.json] [--dir reports] [--json]
  nw report changes [--dir reports] [--json]
  nw report history [--dir reports] [--limit 10] [--json]
  nw report latest [--dir reports] [--json]
  nw report index [--dir reports] --output index.html [--limit 50]
  nw policy init [--dry-run]
  nw policy explain
  nw policy check [--config .nightward.yml] [--workspace DIR] [--include-analysis] [--strict] [--json]
  nw policy sarif [--config .nightward.yml] [--workspace DIR] [--include-analysis] --output nightward.sarif|-
  nw policy badge [--config .nightward.yml] [--workspace DIR] [--include-analysis] [--sarif-url URL] --output badge.json|-
  nw mcp serve
  nw snapshot plan --target <dir> [--json]
  nw snapshot diff --from <plan.json> --to <plan.json> [--json]
  nw schedule plan --preset nightly [--json]
  nw schedule install --preset nightly --dry-run [--json]
  nw schedule remove --dry-run [--json]

Nightward does not mutate agent configs. It only writes explicit report/SARIF outputs and schedule install/remove files.

Canonical command: nightward
Short alias: nw
```
