# TUI

Run Nightward without arguments to open the interactive OpenTUI app:

```sh
nw
```

![Nightward OpenTUI fixture walkthrough](/demo/nightward-opentui.gif)

## Sections

- Overview: risk posture, severity bars, recent findings, and next action.
- Findings: searchable finding list with redacted detail panes.
- Analysis: normalized offline signals and provider-warning context.
- Fix Plan: plan-only remediation groups and review steps.
- Inventory: discovered AI-tool paths by tool, classification, and risk.
- Backup: dry-run dotfiles backup choices.
- Help: key bindings and safety reminders.

The Go scanner remains the source of truth. The CLI writes a private review bundle, then launches the compiled `nightward-tui` OpenTUI sidecar. Release archives, local installs, and npm-downloaded archives include that sidecar beside `nightward` and `nw`.

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
make opentui-verify
make opentui-demo
```

`make opentui-demo` regenerates the fixture-only GIF and PNG used by the README and website. Use fixture media for public docs; do not capture a real workstation.
