# Testing

Nightward treats read-only behavior, redaction, and policy stability as release gates.

## Local Checks

```sh
make test
make test-race
make test-junit
make trunk-flaky-validate
make trunk-check
make raycast-verify
make verify
```

Use `make trunk-fix` for the local repair path:

```sh
make trunk-fix
make verify
```

## Test Coverage Expectations

- Adapter tests use temporary HOME directories and fixture config files.
- CLI no-write tests prove read-only commands do not mutate HOME.
- Redaction tests must cover scan JSON, policy output, SARIF, Markdown exports, fix previews, and TUI text.
- Golden-style tests should stay stable for JSON/SARIF shape, not timestamps or host-specific paths. Scan-summary goldens must keep item buckets separate from finding buckets.
- MCP fixture tests should cover command servers, URL-shaped servers, sensitive headers, local endpoints, and unsupported shapes.
- Scheduler tests verify generated launchd, systemd user timer, and cron text without installing schedules.
- TUI action tests cover clipboard/open command construction and private redacted fix-plan exports.
- Raycast extension tests cover pure redaction/formatting helpers and safe command execution wrappers.

## Trunk Flaky Tests

`make test-junit` writes:

- `reports/go-tests.xml` from Go tests with `gotestsum`
- `reports/junit/raycast.xml` from the Raycast extension Node tests

`make trunk-flaky-validate` runs:

```sh
trunk flakytests validate --junit-paths reports/go-tests.xml,reports/junit/raycast.xml
```

CI validates that the JUnit report is parseable for every pull request. Trunk uploads are gated on `TRUNK_ORG_URL_SLUG` and `TRUNK_API_TOKEN`, so contributors do not need Trunk credentials.

## Raycast Extension

The extension has its own npm package under `integrations/raycast`.

```sh
cd integrations/raycast
npm ci
npm test
npm run lint
npm run build
```

`npm run dev` is the manual smoke path when the Raycast CLI is available. Do not run `npm run publish` unless release/publish scope is explicit.
