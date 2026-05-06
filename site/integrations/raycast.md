# Raycast

Nightward’s [Raycast](https://www.raycast.com/) extension is a macOS companion for AI-agent, Model Context Protocol (MCP), provider, and dotfiles risk review. It shells out to `nw` or `nightward`, renders redacted output, and uses the shared Nightward action registry for confirmation-gated local writes.

## Command Surface

| Command | Use it for | Writes |
| --- | --- | --- |
| Nightward Dashboard | Scan counts, adapters, schedule state, top findings, and fix-plan summary. | No |
| Nightward Status | Compact menu-bar finding count with a structured dropdown. | No |
| Nightward Findings | Browse findings, copy redacted evidence, export finding/rule fix plans, and copy reviewed-ignore snippets. | Clipboard only |
| Nightward Analysis | Browse built-in and selected-provider analysis signals. | No |
| Nightward Provider Doctor | Check provider availability, choose providers for Raycast Analysis, and preview/apply known provider install actions. | Raycast preference or confirmed action-registry provider install |
| Nightward Actions | Preview and apply confirmed provider, policy, schedule, backup, cleanup, and setup actions. | Confirmation-gated local writes |
| Nightward MCP Approvals | Approve or deny exact MCP-requested action tickets. | Approval state only; MCP applies the approved ticket |
| Explain Finding / Explain Signal | Jump directly to one known ID. | No |
| Export Fix Plan / Export Analysis | Copy redacted Markdown for review. | Clipboard only |
| Open Nightward Reports | Open the local report folder in Finder. | Finder open only |

The menu-bar title stays intentionally small: icon plus the current finding count. The dropdown carries severity, analysis, provider-warning, schedule, open, and action detail so it does not read like one long paragraph.

## Preferences

| Preference | Purpose |
| --- | --- |
| `Nightward Command` | Command name or absolute path. Defaults to `nw`. |
| `Home Override` | Typed `NIGHTWARD_HOME` path for fixture homes, QA profiles, or demos. |
| `Allow Online Providers` | Allows selected online-capable providers in Raycast Analysis. Leave off for local-only behavior. |

Provider selection is separate from execution. If a provider is missing, Provider Doctor and Nightward Actions offer confirmation-gated install actions for known package-manager provider CLIs through the shared Nightward action registry, not through ad hoc shell execution. MCP Approvals shows pending tickets from AI clients; approving one ticket does not approve disclosure, hidden edits, or any other action.

## Responsibility

Nightward is beta operator tooling. Actions can change local package-manager state, scheduled jobs, settings, backup files, or Nightward-owned report/cache files. MCP approval tickets can let an AI client apply the exact approved action once. Review every confirmation, approval, command, and write target before applying. Nightward is provided without warranty; maintainers are not liable for broken configs, lost data, exposed secrets, or third-party tool side effects.

## Providers

Provider Doctor can select `gitleaks`, `trufflehog`, `semgrep`, `syft`, `trivy`, `osv-scanner`, `grype`, `scorecard`, and `socket` for the Analysis command.

- Local providers run only after they are selected.
- Online-capable providers stay blocked until `Allow Online Providers` is enabled.
- Socket creates a remote scan artifact when it runs.
- Missing providers show explicit install guidance rather than silently failing.

## Store Submission

The extension now has fixture-only metadata screenshots for Dashboard, Findings, and Provider Doctor. Before a store PR, rerun the package checks from the extension directory:

```sh
cd integrations/raycast
npm ci
npm test
npm run lint
npm run build
npm run store-check:strict
```

Raycast’s public publishing flow runs from an extension directory with `npm run build` for validation and `npm run publish` to open a PR against [`raycast/extensions`](https://github.com/raycast/extensions). Their [store preparation guide](https://developers.raycast.com/basics/prepare-an-extension-for-store) expects npm lockfiles, local lint/build validation, clear metadata, and 2000x1250 PNG screenshots. Nightward should not run `npm run publish` until the `raycast/extensions` fork is synced and a maintainer is ready to open the draft submission PR.

Before a draft PR:

1. Sync the `raycast/extensions` fork.
2. Copy the self-contained package into `extensions/nightward`.
3. Re-capture fixture-only metadata screenshots from `ray develop` if the UI changed.
4. Confirm icon, README, CHANGELOG, categories, command descriptions, and preferences match the store package.
5. Run `npm run store-check:strict`, `npm run lint`, `npm run build`, and `npm test`.
6. Open a draft PR and link Nightward’s GitHub repo, docs, and fixture evidence.
