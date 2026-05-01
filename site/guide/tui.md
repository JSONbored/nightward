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
