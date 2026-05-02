# Raycast

Nightward’s [Raycast](https://www.raycast.com/) extension is a read-only macOS companion for AI-agent, MCP, provider, and dotfiles risk review. It shells out to `nw` or `nightward`, renders redacted output, and does not mutate local agent configs.

## Command Surface

| Command | Use it for | Writes |
| --- | --- | --- |
| Nightward Dashboard | Scan counts, adapters, schedule state, top findings, and fix-plan summary. | No |
| Nightward Status | Compact menu-bar risk count with a structured dropdown. | No |
| Nightward Findings | Browse findings, copy redacted evidence, export finding/rule fix plans, and copy reviewed-ignore snippets. | Clipboard only |
| Nightward Analysis | Browse built-in and selected-provider analysis signals. | No |
| Nightward Provider Doctor | Check provider availability and choose providers for Raycast Analysis. | Raycast local preference only |
| Explain Finding / Explain Signal | Jump directly to one known ID. | No |
| Export Fix Plan / Export Analysis | Copy redacted Markdown for review. | Clipboard only |
| Open Nightward Reports | Open the local report folder in Finder. | Finder open only |

The menu-bar title stays intentionally small: icon plus the highest-risk count. The dropdown carries the detail, split into Findings, Analysis, Schedule, Open, and Actions sections so it does not read like one long paragraph.

## Preferences

| Preference | Purpose |
| --- | --- |
| `Nightward Command` | Command name or absolute path. Defaults to `nw`. |
| `Home Override` | Typed `NIGHTWARD_HOME` path for fixture homes, QA profiles, or demos. |
| `Allow Online Providers` | Allows selected online-capable providers in Raycast Analysis. Leave off for local-only behavior. |

Provider selection is separate from installation. If a provider is missing, Provider Doctor offers the install command and upstream install docs, but it does not run package managers for you.

## Providers

Provider Doctor can select `gitleaks`, `trufflehog`, `semgrep`, `trivy`, `osv-scanner`, and `socket` for the Analysis command.

- Local providers run only after they are selected.
- Online-capable providers stay blocked until `Allow Online Providers` is enabled.
- Socket creates a remote scan artifact when it runs.
- Missing providers show explicit install guidance rather than silently failing.

## Store Submission

The extension is close to store-ready, but submission still needs manual evidence:

```sh
cd integrations/raycast
npm ci
npm test
npm run lint
npm run build
npm run store-check:strict
```

Raycast’s public publishing flow runs from an extension directory with `npm run build` for validation and `npm run publish` to open a PR against [`raycast/extensions`](https://github.com/raycast/extensions). Their [store preparation guide](https://developers.raycast.com/basics/prepare-an-extension-for-store) also expects npm lockfiles, local lint/build validation, clear metadata, and store screenshots. Nightward should not run `npm run publish` until the fixture screenshot set and PR metadata are ready.

Before a draft PR:

1. Sync the `raycast/extensions` fork.
2. Copy the self-contained package into `extensions/nightward`.
3. Capture at least three fixture-only metadata screenshots from `ray develop`.
4. Confirm icon, README, CHANGELOG, categories, command descriptions, and preferences match the store package.
5. Run `npm run store-check:strict`, `npm run lint`, `npm run build`, and `npm test`.
6. Open a draft PR and link Nightward’s GitHub repo, docs, and fixture evidence.
