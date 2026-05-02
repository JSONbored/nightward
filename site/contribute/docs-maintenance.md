# Docs Maintenance

Nightward docs are split between human-authored guides and generated references. The goal is to keep public pages readable while making drift visible in CI.

## Source Of Truth

| Surface | Source | Check |
| --- | --- | --- |
| CLI reference | `internal/cli` and `scripts/generate-reference-docs.mjs` | `make docs-reference-check` |
| Rules reference | rule metadata emitted by the generator | `make docs-reference-check` |
| Provider reference | analysis provider metadata | `make docs-reference-check` |
| Config examples | repo docs and fixture policies | `make docs-freshness` |
| Public guides | `site/**/*.md` | `make site-verify` |
| Screenshots and samples | committed fixture homes | `make demo-assets` and manual review |

Run the full docs gate before opening a docs PR:

```sh
make site-verify
make docs-qa
```

`make test-prepush` includes the docs gates plus the normal code, security, Raycast, npm, and site checks.

## Writing Rules

- Keep guides human-readable and task-led.
- Link to the actual upstream tool or service when a page names one.
- Label behavior precisely: `read-only`, `explicit write`, `online-capable`, `plan-only`, or `future/not shipped`.
- Do not claim a provider, adapter, integration, or distribution channel is published until it is publicly available.
- Prefer generated reference updates over hand-editing command, rule, provider, or JSON schema tables.
- Keep examples redacted and fixture-backed when they show findings.

## Updating Pages

1. Change the implementation or source metadata first.
2. Run `make docs-reference` when CLI/rule/provider/reference output changed.
3. Edit public guide pages for the user-facing explanation.
4. Run `make docs-qa` and `make site-verify`.
5. If screenshots changed, regenerate fixture-only assets and update `docs/screenshots.md`.

## Future Automation

The next improvement is a docs contract check that compares public snippets against real command output. The useful shape is:

- run documented commands against fixture homes;
- parse fenced command snippets with stable labels;
- fail when generated references are stale;
- fail when public pages mention future channels as shipped;
- optionally check outbound docs links on a scheduled workflow.

That keeps the docs living without turning every page into generated text.
