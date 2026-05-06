# CLI Reference

This page is generated from `cargo run --bin nw -- --help`.

```text
Nightward audits AI agent state, MCP config, and dotfiles sync risk.

USAGE:
  nightward                         Open the TUI
  nightward tui --input scan.json   Review a saved report in the TUI
  nightward tui --from old.json --to new.json
  nightward scan --json             Scan HOME
  nightward scan --workspace . --json
  nightward analyze --all --with gitleaks --json
  nightward providers doctor --with trivy --online --json
  nightward providers enable gitleaks --confirm
  nightward providers install gitleaks --confirm
  nightward disclosure accept
  nightward fix plan --all --json
  nightward backup create --confirm
  nightward schedule install --confirm
  nightward actions list --json
  nightward actions apply backup.snapshot --confirm
  nightward actions apply reports.cleanup --confirm
  nightward actions apply cache.cleanup --confirm
  nightward actions apply policy.ignore --finding <id> --reason "reviewed" --confirm
  nightward report html --input scan.json --output report.html
  nightward report html --from old.json --to new.json --output report.html
  nightward policy check --json
  nightward mcp serve

Nightward is local-first and read-only by default. Write-capable actions require disclosure acceptance and explicit confirmation.
```
