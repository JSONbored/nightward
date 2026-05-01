# Raycast

Nightward's Raycast extension is a read-only macOS companion for checking AI-agent, MCP, and dotfiles risk without opening a terminal.

## Commands

- Dashboard.
- Findings.
- Analysis.
- Provider Doctor with provider enable/disable controls for Raycast Analysis.
- Explain Finding.
- Explain Signal.
- Export Fix Plan.
- Export Analysis.
- Open Reports.
- Menu-bar status.

The extension shells out to `nw` or `nightward`, renders redacted output, and never mutates agent configs.

The dashboard and menu-bar status include scheduled report counts when `nw doctor --json` reports history. They can also reveal or open the latest report when one exists, so users can move from the menu-bar counter to the underlying local JSON quickly.

## Preferences

- `Nightward Command`: command name or absolute path, defaulting to `nw`.
- `Home Override`: typed `NIGHTWARD_HOME` path for fixture homes or test profiles.
- `Allow Online Providers`: enables selected online-capable providers in Raycast Analysis. Leave it off for local-only behavior.

Provider Doctor can select `gitleaks`, `trufflehog`, `semgrep`, `trivy`, `osv-scanner`, and `socket` for the Analysis command. Online-capable selections stay blocked until `Allow Online Providers` is enabled; Socket creates a remote scan artifact when it runs.
