# Changelog

## Unreleased

- Use a compact menu-bar finding count and move severity/provider detail into the dropdown.
- Add scoped finding and rule fix-plan copy actions plus reviewed-policy-ignore snippets.
- Redact additional provider token-shaped values in Raycast-rendered text.

## 0.1.0

- Add read-only Dashboard, Findings, Explain Finding, Export Fix Plan, and Open Reports commands.
- Add a read-only menu-bar status command for finding, analysis, provider-warning, and scheduled-report counters.
- Add latest-report actions to dashboard and menu-bar status when scheduled report history is available.
- Shell out with `execFile` and render only redacted Nightward JSON/Markdown output.
