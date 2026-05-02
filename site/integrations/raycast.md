# Raycast

Nightward's Raycast extension is a read-only macOS companion for checking AI-agent, MCP, and dotfiles risk without opening a terminal.

## Commands

- Dashboard.
- Findings with scoped fix-plan exports and reviewed-policy-ignore snippets.
- Analysis.
- Provider Doctor with provider enable/disable controls for Raycast Analysis.
- Explain Finding.
- Explain Signal.
- Export Fix Plan.
- Export Analysis.
- Open Reports.
- Menu-bar status.

The extension shells out to `nw` or `nightward`, renders redacted output, and never mutates agent configs.

The dashboard and menu-bar status include scheduled report counts when `nw doctor --json` reports history. The menu-bar title stays compact (`3C`, `18H`, or a total count), while the dropdown shows critical/high/total counts, provider warnings, and latest-report links.

## Preferences

- `Nightward Command`: command name or absolute path, defaulting to `nw`.
- `Home Override`: typed `NIGHTWARD_HOME` path for fixture homes or test profiles.
- `Allow Online Providers`: enables selected online-capable providers in Raycast Analysis. Leave it off for local-only behavior.

Provider Doctor can select `gitleaks`, `trufflehog`, `semgrep`, `trivy`, `osv-scanner`, and `socket` for the Analysis command. Online-capable selections stay blocked until `Allow Online Providers` is enabled; Socket creates a remote scan artifact when it runs.

Findings actions remain plan-only: users can copy a redacted fix plan for the
selected finding, copy a grouped rule-level fix plan, or copy a policy-ignore
snippet with a reason placeholder. The extension does not write policy files or
mutate MCP configs.

## Store Readiness

The extension is development-ready, but store submission still requires fixture-only Raycast screenshots captured from `ray develop`.

```sh
cd integrations/raycast
npm ci
npm test
npm run lint
npm run build
npm run store-check
```

Before opening a draft PR to `raycast/extensions`, sync the fork, copy the self-contained package into `extensions/nightward`, capture at least three metadata screenshots with fixture `Home Override`, then run:

```sh
npm run store-check:strict
```
