# Contributing

Nightward should stay useful to AI power users without becoming a tool that copies private state by accident.

## Development

```sh
go test ./...
go run ./cmd/nw --help
go run ./cmd/nw scan --json
go run ./cmd/nw fix plan --all --json
```

Use fixtures with temporary homes for adapter and policy tests. Do not add real local config files, tokens, credentials, shell history, app databases, or personal paths.

## Design Rules

- Scanner, doctor, findings, fix, policy, and backup-plan commands must not mutate agent configs.
- Explicit output flags may write redacted report or SARIF artifacts.
- Fix plans recommend steps; they do not edit agent configs.
- Any new output surface must preserve redaction.
- New adapters need classification, risk rationale, and no-write tests when practical.
- MCP/security rules need fixture coverage and a safe remediation posture.

## CI And Action Pinning

GitHub Actions must use least-privilege permissions and pin third-party actions by full commit SHA. Keep the human-readable upstream version in a comment next to the SHA.

Current security checks:

- Go tests
- Gitleaks
- govulncheck
- OSV Scanner
- Nightward SARIF upload
- OpenSSF Scorecard

GoReleaser, Cosign signing, and Homebrew tap automation should wait until CI is stable and release credentials can be protected behind branch/tag rules.

## Pull Requests

Use focused PRs. A good PR explains:

- what changed
- what user-facing behavior changed
- what was validated
- any remaining risk or follow-up
