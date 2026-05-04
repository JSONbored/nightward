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
- Do not publish a screenshot, GIF, or Raycast store metadata surface until the matching evidence row below is filled from fixture runs.

## Suggested Fixture Run

```sh
target/debug/nw tui --input site/public/demo/nightward-sample-scan.json
vhs docs/demo/nightward-tui.tape
```

For Raycast, set the extension `Home Override` preference to the same fixture home before `npm run dev`.

## Release Evidence Table

| Surface | Fixture-only source | Required evidence | Status |
| --- | --- | --- | --- |
| Static HTML report | `testdata/homes/policy` | Scrubbed scan JSON, filterable static HTML report, and PNG screenshot generated from fixture output | captured in `site/public/demo/` with `node scripts/generate-demo-assets.mjs`; no local paths or secret values found |
| TUI dashboard | `site/public/demo/nightward-sample-scan.json` | PNG shows the embedded Rust OpenTUI dashboard with severity colors, summary panels, findings, and next action | fixture PNG in `site/public/demo/nightward-opentui.png`; no live HOME data used |
| TUI review flows | `site/public/demo/nightward-sample-scan.json` | GIF shows findings, analysis, fix plan, inventory, backup preview, search, and severity filtering once regenerated from the Rust TUI | fixture GIF in `site/public/demo/nightward-opentui.gif`; no `/Users`, username, or secret fixture values allowed |
| Raycast Dashboard | `/tmp/nightward-raycast-home` copied from `testdata/homes/policy` | Screenshot shows counts and top findings from synthetic data | captured in `integrations/raycast/metadata/dashboard.png` via `npm run dev`; no live HOME data visible |
| Raycast Findings/Analysis | `/tmp/nightward-raycast-home` copied from `testdata/homes/policy` | Screenshots show redacted evidence and no config mutation actions | findings captured in `integrations/raycast/metadata/findings.png` via `npm run dev`; analysis command passed automated tests |
| Raycast Provider Doctor | fixture `Home Override` plus local provider availability | Screenshot shows provider status without implying online scans | captured in `integrations/raycast/metadata/providers.png` via `npm run dev`; online-capable providers remain blocked without opt-in |

Record the commit SHA, fixture path, command used, and reviewer initials beside the captured assets before linking them from README or store metadata.
