# Testing

Nightward treats read-only behavior, redaction, and policy stability as release gates.

## Local Checks

Start with the developer toolchain doctor:

```sh
make doctor
```

If `cargo` is installed but your shell cannot find it, load the repo-local path helper:

```sh
. scripts/dev-env.sh
```

Optional Cargo security/coverage tools can be installed with:

```sh
make install-dev-tools
```

This installs pinned tool versions. Some optional tools have newer Rust MSRVs
than the project runtime, so the target installs them with an auxiliary Rust
toolchain while keeping Nightward itself on the repo-pinned toolchain.

Use the suite aliases first. They are intentionally shaped around the same surfaces CI and release workflows gate, so local failures are caught before a branch is pushed.

```sh
make test-fast
make test-security
make test-ux
make test-release
make test-prepush
```

- `make test-fast` runs Rust unit tests plus npm launcher and Raycast unit tests.
- `make test-security` runs static analysis, secret/vulnerability checks, and npm audits.
- `make test-ux` runs the Raycast and VitePress site validation paths.
- `make test-release` runs release helper tests, npm package verification, Raycast/site builds, and a Rust archive snapshot.
- `make test-prepush` is the full local gate and is equivalent to `make verify`.
- `make doctor` verifies required local tools and checks common optional tools used by release, UX, and security evidence paths.

After a package is published, verify the install path explicitly:

```sh
make test-release-install VERSION=0.1.4
```

The lower-level targets remain available for focused iteration:

```sh
make test
make fmt
make clippy
make gitleaks
make cargo-audit
make cargo-deny
make coverage-check
make fuzz-smoke
make test-junit
make trunk-flaky-validate
make trunk-check
make ci-scripts-test
make raycast-verify
make npm-package-verify
make docs-qa
make site-verify
make release-snapshot
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
- Redaction tests must cover scan JSON, policy output, SARIF, Markdown exports, fix exports, and TUI text.
- Badge artifact tests must cover pass/fail shape, policy summary fields, optional SARIF URL, and no-write stdout mode.
- Golden-style tests should stay stable for JSON/SARIF shape, not timestamps or host-specific paths. Scan-summary goldens must keep item buckets separate from finding buckets.
- MCP fixture tests should cover command servers, URL-shaped servers, sensitive headers, local endpoints, and unsupported shapes.
- Parser fuzz harnesses live under `fuzz/` and cover MCP JSON/TOML/YAML parsing, URL/header redaction, symlink traversal, huge-file handling, and malformed config cases. Run a bounded local fuzz check with `make fuzz-smoke`; run a single target directly with `cargo fuzz run mcp_config_formats -- -runs=1024`.
- Provider contract tests use `testdata/providers/*` fixtures for `gitleaks`, `trufflehog`, `semgrep`, `trivy`, `osv-scanner`, and `socket`.
- Scheduler tests verify generated launchd, systemd user timer, and cron text without installing schedules.
- TUI tests cover fixed terminal rendering behavior, redaction boundaries, and embedded OpenTUI layout helpers.
- Scheduler tests cover report history ordering, finding counts, non-report filtering, and symlink skipping without installing timers.
- Raycast extension tests cover pure redaction/formatting helpers and safe command execution wrappers.
- `cargo fmt`, `cargo clippy -D warnings`, `cargo test`, optional `cargo audit`/`cargo deny`, Gitleaks, and coverage checks are part of the local verification bar.
- `make coverage-check` enforces the practical coverage target when `cargo-llvm-cov` is available, and always runs the Rust workspace tests.
- `make ci-scripts-test` verifies repository-maintained CI helper scripts such as DCO checking, action path validation, and release-script input validation.
- Raycast dependency audits run with `npm audit --audit-level=moderate`.
- The npm launcher tests run with `make npm-package-verify`, including unit tests, `npm audit`, and `npm pack --dry-run`.
- `make docs-qa` verifies generated CLI/provider/rule/config references and fails on stale release-version placeholders in public docs.

## Trunk Flaky Tests

`make test-junit` writes:

- `reports/junit/raycast.xml` from the Raycast extension Node tests

`make trunk-flaky-validate` runs:

```sh
trunk flakytests validate --junit-paths reports/junit/raycast.xml
```

CI validates that the JUnit report is parseable for every pull request. Trunk uploads are gated on `TRUNK_ORG_URL_SLUG` and `TRUNK_API_TOKEN`, so contributors do not need Trunk credentials.

## Release Snapshot

`make release-snapshot` builds the Rust release binaries, creates a local archive, writes `checksums.txt`, and emits a lightweight SBOM placeholder for archive-shape validation. Real release signing remains restricted to the tag-driven release workflow.

## Raycast Extension

The extension has its own npm package under `integrations/raycast`.

```sh
cd integrations/raycast
npm ci
npm test
npm run lint
npm run build
```

`npm run dev` is the manual Raycast development path when the Raycast CLI is available. Do not run `npm run publish` unless release/publish scope is explicit.

Manual smoke and screenshots must use fixture `Home Override` data only. Keep the evidence table in `docs/screenshots.md` current before broader promotion or Raycast store metadata work.

## NPM Launcher

The release-gated npm package lives under `packages/npm`.

```sh
cd packages/npm
npm ci
npm test
npm audit --audit-level=moderate
npm run pack:dry-run
```

## Documentation Site

The public docs/marketing site lives under `site/` and uses VitePress with local search.

```sh
cd site
npm ci
npm audit --audit-level=moderate
npm run build
```

`make site-verify` also runs `make docs-qa` from the repository root. The site should not add analytics or third-party runtime scripts by default. If validating the public website analytics path, build once without Umami env values and confirm no tracker script exists, then build once with `NIGHTWARD_UMAMI_SCRIPT_URL` and `NIGHTWARD_UMAMI_WEBSITE_ID` set and confirm the script is scoped to `nightward.aethereal.dev`, respects Do Not Track, and excludes search/hash data.

The launcher must remain dependency-light, avoid `postinstall`, and verify downloaded GitHub Release archives against `checksums.txt` before extraction.

## Intentional Manual Or Post-Release Checks

Most repository checks are centralized behind `make verify` and the suite aliases above. The remaining loose commands are intentionally not part of the default gate because they require a browser, a published release, or manual UI evidence:

- `make demo-assets` regenerates fixture-only sample JSON, HTML, report screenshot, and social preview assets. It requires Chrome, Chromium, Brave, or `NIGHTWARD_CHROME`.
- `make tui-media` regenerates fixture-only TUI PNGs, the walkthrough GIF, and the homepage WebM loop from the embedded Rust TUI. It requires `vhs` and `ffmpeg`, uses `site/public/demo/nightward-sample-scan.json`, and must not be run against live HOME data.
- `make test-release-install VERSION=<version>` verifies a published GitHub/npm release after artifacts exist.
- `npm run dev` under `integrations/raycast` is the local Raycast UI smoke path and should be paired with fixture-only evidence in `docs/screenshots.md`.

New validation scripts should be wired into `make verify`, a suite alias, or this section with a clear reason they are manual.
