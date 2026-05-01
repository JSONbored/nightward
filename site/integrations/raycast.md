# Raycast

Nightward's Raycast extension is a read-only macOS companion.

## Commands

- Dashboard.
- Findings.
- Analysis.
- Provider Doctor.
- Explain Finding.
- Explain Signal.
- Export Fix Plan.
- Export Analysis.
- Open Reports.
- Menu-bar status.

The extension shells out to `nw` or `nightward`, renders redacted output, and never mutates agent configs.

The dashboard and menu-bar status include scheduled report counts when `nw doctor --json` reports history. Use this to notice when new findings appear after a scheduled scan.
