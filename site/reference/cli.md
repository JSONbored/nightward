# CLI Reference

This page is generated from `cargo run --bin nw -- --help`.

```text
Nightward audits AI agent state, MCP config, and dotfiles sync risk.

USAGE:
  nightward                         Open the TUI
  nightward tui --input scan.json   Review a saved report in the TUI
  nightward scan --json             Scan HOME
  nightward scan --workspace . --json
  nightward analyze --all --with gitleaks --json
  nightward providers doctor --with trivy --online --json
  nightward fix plan --all --json
  nightward report html --input scan.json --output report.html
  nightward policy check --json
  nightward mcp serve

Nightward is local-first, read-only by default, and never enables online providers without --online.
```
