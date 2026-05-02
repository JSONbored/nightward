# TUI

Run the TUI with:

```sh
nw
```

## Screens

- Dashboard: scan counts and schedule status.
- Inventory: discovered paths by tool, classification, and risk.
- Findings: severity/tool/rule filters plus detail pane.
- Analysis: offline risk signals and provider warnings.
- Fix Plan: safe, review-required, and blocked remediation groups.
- Backup Plan: private-dotfiles dry-run preview.

## Shortcuts

- `1`-`6`: switch tabs.
- `p`: open the command palette.
- Arrow keys or `h`/`j`/`k`/`l`: navigate.
- `/`: search findings.
- `s`, `t`, `r`: cycle severity, tool, and rule filters.
- `x`: clear filters.
- `c`: copy selected path, recommendation, or fix step.
- `e`: export redacted fix plan.
- `o`: open remediation docs.
- `?`: help.
- `q` or `esc`: quit.

> [!NOTE]
> The TUI remains read-only except explicit redacted export actions.

## Command Palette

The command palette exposes the main actions without memorizing shortcuts: switch tabs, copy the current selection, export a redacted fix plan, open docs for the selected finding or fix, search findings, cycle filters, and clear filters. Palette actions stay plan-only and do not mutate agent configs.
