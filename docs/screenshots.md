# Screenshot And GIF Capture Plan

Nightward should have visual proof before broader promotion.

## Targets

- Dashboard showing scan counts and schedule status.
- Findings tab with severity/tool/rule filters.
- Finding detail pane with evidence, impact, suggested fix, and why it matters.
- Fix Plan tab grouped into safe, review, and blocked fixes.
- Backup Plan preview.

## Capture Rules

- Use fixture homes only.
- Never capture real local paths, tokens, auth files, project names, or private MCP servers.
- Keep terminal width around 120 columns for README screenshots.
- Prefer short GIFs that show filtering and detail navigation, not long demos.

## Suggested Fixture Run

```sh
NIGHTWARD_HOME="$PWD/testdata/homes/policy" go run ./cmd/nw
```

After capture, review images manually for private state before committing.
