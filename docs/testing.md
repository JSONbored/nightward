# Testing

Nightward treats read-only behavior, redaction, and policy stability as release gates.

## Local Checks

```sh
make test
make test-race
make test-junit
make trunk-flaky-validate
make trunk-check
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
- Golden-style tests should stay stable for JSON/SARIF shape, not timestamps or host-specific paths.
- Scheduler tests verify generated launchd, systemd user timer, and cron text without installing schedules.

## Trunk Flaky Tests

`make test-junit` writes `reports/go-tests.xml` with `gotestsum`.

`make trunk-flaky-validate` runs:

```sh
trunk flakytests validate --junit-paths reports/go-tests.xml
```

CI validates that the JUnit report is parseable for every pull request. Trunk uploads are gated on `TRUNK_ORG_URL_SLUG` and `TRUNK_API_TOKEN`, so contributors do not need Trunk credentials.
