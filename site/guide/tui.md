# TUI

Run the TUI with:

```sh
nw
```

![Nightward TUI fixture walkthrough](/demo/nightward-tui.gif)

## Screens

- Dashboard: scan counts and schedule status.
- Inventory: discovered paths by tool, classification, and risk.
- Findings: severity/tool/rule filters plus detail pane.
- Analysis: offline risk signals and provider warnings.
- Fix Plan: safe, review-required, and blocked remediation groups.
- Backup Plan: private-dotfiles dry-run preview.

Nightward uses Bubble Tea with Bubbles table, viewport, help, and text-input components. The current interface is still read-only, but list rendering, filter input, detail panes, footer help, and tab-specific accent colors are now component-backed instead of ad hoc terminal text.

This is the current production TUI, not the final visual ceiling. The remaining polish track is better table density, richer report-history comparison, mouse affordances, and fixture-backed screenshot checks for no-overlap rendering across common terminal sizes.

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

## Visual QA

TUI screenshots and GIFs are generated from fixture homes, not a real workstation:

```sh
make tui-demo
make demo-assets
```

Use these before changing README or site media. For interactive changes, also run the TUI against a fixture home and a real HOME scan, then check findings, analysis, fix plan, backup plan, empty states, and terminal widths around 80, 120, and 160 columns.
