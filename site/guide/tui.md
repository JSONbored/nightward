# TUI

Run Nightward without arguments to open the interactive terminal app:

```sh
nw
```

Review a saved report without scanning again:

```sh
nw tui --input scan.json
```

![Nightward OpenTUI fixture dashboard](/demo/nightward-opentui.png)

[Open the fixture walkthrough GIF](/demo/nightward-opentui.gif)

## Sections

- Overview: risk posture, severity bars, recent findings, and next action.
- Findings: searchable finding list with redacted detail panes.
- Analysis: normalized offline signals and provider-warning context.
- Fix Plan: plan-only remediation groups and review steps.
- Inventory: discovered AI-tool paths by tool, classification, and risk.
- Backup: dry-run dotfiles backup choices.
- Help: key bindings and safety reminders.

The Rust CLI is the source of truth. The TUI uses embedded `opentui_rust` rendering for the colored dashboard; there is no Bun package or `nightward-tui` sidecar.

## Shortcuts

- `1`-`7`: switch sections.
- `tab`, `right`, or `l`: next section.
- `left` or `h`: previous section.
- `up`, `down`, `j`, or `k`: move selection.
- `/`: search.
- `s`: cycle severity.
- `x`: clear filters.
- `q` or `esc`: quit.

> [!NOTE]
> The TUI is read-only. It does not mutate MCP, agent, or dotfiles config. Fixes remain plan-only review material.

## Local Development

```sh
cargo run --bin nw
cargo run --bin nw -- tui --input site/public/demo/nightward-sample-scan.json
make demo-assets
```

Use fixture media for public docs; do not capture a real workstation.
