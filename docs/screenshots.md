# Screenshot And GIF Capture Plan

Nightward should have visual proof before broader promotion.

## Targets

- Dashboard showing scan counts and schedule status.
- Dashboard showing report history and the "What Next" flow.
- Findings tab with severity/tool/rule filters.
- Finding detail pane with evidence, impact, suggested fix, and why it matters.
- Fix Plan tab grouped into safe, review, and blocked fixes.
- Backup Plan preview.
- Raycast Dashboard, Findings, Analysis, and Provider Doctor commands.

## Capture Rules

- Use fixture homes only.
- Never capture real local paths, tokens, auth files, project names, or private MCP servers.
- Keep terminal width around 120 columns for README screenshots.
- Prefer short GIFs that show filtering and detail navigation, not long demos.
- Review every captured frame for private state before committing it.
- Do not publish screenshots, GIFs, or Raycast store metadata until the evidence table below is filled from fixture runs.

## Suggested Fixture Run

```sh
NIGHTWARD_HOME="$PWD/testdata/homes/policy" go run ./cmd/nw
```

For Raycast, set the extension `Home Override` preference to the same fixture home before `npm run dev`.

## Release Evidence Table

| Surface | Fixture-only source | Required evidence | Status |
| --- | --- | --- | --- |
| TUI dashboard | `testdata/homes/policy` or another committed fixture home | Screenshot shows counts, schedule status, report history, and what-next text | pending manual capture |
| TUI findings/fix plan | committed MCP/security fixtures only | GIF or screenshots show filters, detail pane, redacted evidence, and fix plan | pending manual capture |
| Raycast Dashboard | fixture `Home Override` | Screenshot shows counts and top findings from synthetic data | pending `ray develop` smoke |
| Raycast Findings/Analysis | fixture `Home Override` | Screenshots show redacted evidence and no config mutation actions | pending `ray develop` smoke |
| Raycast Provider Doctor | fixture/default local environment | Screenshot shows provider status without implying online scans | pending `ray develop` smoke |

Record the commit SHA, fixture path, command used, and reviewer initials beside the captured assets before linking them from README or store metadata.
