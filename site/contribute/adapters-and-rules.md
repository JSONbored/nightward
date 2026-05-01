# Contribute Adapters And Rules

Nightward is easiest to extend with fixture-first changes.

## Adapter Fixtures

When adding an adapter or config shape, include:

- A fixture home or workspace with the smallest realistic config file.
- Expected findings or classification behavior.
- Secret-looking values that prove redaction without containing real credentials.
- Malformed or edge-case samples when the parser changes.

Suggested fixture layout:

```text
testdata/homes/<tool-name>/
  .config/<tool>/config.json
  expected.md
```

Generate a starter checklist from the current adapter catalog:

```sh
nw adapters template Codex
nw adapters template "Workspace Cursor" --workspace .
```

## Rule Changes

Use the rule reference and `nw rules explain` to keep docs grounded:

```sh
nw rules list --json
nw rules explain mcp_secret_header --json
```

Rules should include severity, impact, recommendation, docs link, and whether any plan-only remediation metadata is available.

## Validation

```sh
go test ./internal/inventory ./internal/analysis ./internal/cli
make docs-reference-check
make test-prepush
```

Use `good first adapter` for narrow config-shape additions and `good first rule` for isolated detection improvements with fixtures.
