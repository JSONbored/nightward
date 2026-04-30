# Contributing

Nightward should stay useful to AI power users without becoming a tool that copies private state by accident.

## Development

```sh
make test
make test-junit
make trunk-flaky-validate
make trunk-check
go run ./cmd/nw --help
go run ./cmd/nw scan --json
go run ./cmd/nw fix plan --all --json
go run ./cmd/nw fix preview --all --format markdown
```

Use fixtures with temporary homes for adapter and policy tests. Do not add real local config files, tokens, credentials, shell history, app databases, or personal paths.

## Design Rules

- Scanner, doctor, findings, fix, policy, backup-plan, and snapshot commands must not mutate agent configs.
- Explicit output flags may write redacted report or SARIF artifacts.
- Fix plans and fix previews recommend steps; they do not edit agent configs.
- Policy ignores must include a reason.
- Any new output surface must preserve redaction.
- New adapters need classification, risk rationale, and no-write tests when practical.
- MCP/security rules need fixture coverage and a safe remediation posture.

## CI And Action Pinning

GitHub Actions must use least-privilege permissions and pin third-party actions by full commit SHA. Keep the human-readable upstream version in a comment next to the SHA.

Nightward uses Renovate instead of Dependabot. Dependency PRs should use Conventional Commit titles under `chore(deps):`, preserve pinned action digests, and pass the same validation as feature PRs.

Current security checks:

- Go tests
- Trunk Check
- Trunk Flaky Tests JUnit validation
- Gitleaks
- govulncheck
- OSV Scanner
- Nightward SARIF upload
- OpenSSF Scorecard
- Renovate dependency maintenance

Homebrew tap automation should wait until the first tagged release proves stable.

## Pull Requests

Use focused PRs. A good PR explains:

- what changed
- what user-facing behavior changed
- what was validated
- any remaining risk or follow-up
