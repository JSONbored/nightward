# Contribute Adapters And Rules

Nightward contributions should be fixture-first. A new adapter or rule is not complete until a future maintainer can run the fixture and see why the detection exists.

## Good First Adapter

Use this path when a tool stores MCP or agent config in a new location or shape.

Required:

- Smallest realistic fixture home or workspace.
- Parser support for the config shape.
- Expected inventory and finding behavior.
- Redaction proof for secret-looking values.
- Malformed sample if the parser changes.
- Docs update in the support matrix.

Suggested layout:

```text
testdata/homes/<tool-name>/
  .config/<tool>/config.json
  expected.md
```

Starter commands:

```sh
nw adapters list --json
nw adapters explain Codex --json
nw adapters template "New Tool"
```

## Good First Rule

Use this path when the adapter already sees the config but Nightward needs a new finding, severity, recommendation, or fix-plan hint.

Required:

- Rule metadata with severity, impact, recommendation, and docs link.
- Fixture that triggers the rule.
- Negative fixture or test proving normal config is not flagged.
- Redaction test when evidence can include secrets, headers, paths, or URLs.
- Reference-doc regeneration if generated rule output changes.

Starter commands:

```sh
nw rules list --json
nw rules explain mcp_secret_header --json
make docs-reference
```

## Provider Fixtures

Provider contributions must use scrubbed JSON samples from the provider’s real output shape. Do not commit raw provider output from a private repo.

Cover:

- valid JSON with one finding;
- malformed JSON;
- command failure stderr redaction;
- output cap behavior when relevant;
- no-write regression around the scanned workspace.

## Validation

Run the focused tests first, then the repo gate:

```sh
cargo test --workspace
make docs-reference-check
make test-prepush
```

Use `good first adapter` for narrow config-shape additions and `good first rule` for isolated detection improvements with fixtures.
