# Contributing

Nightward should stay useful to AI power users without becoming a tool that copies private state by accident.

## Development

```sh
make test-fast
make test-security
make test-ux
make test-release
make test-prepush
```

`make test-prepush` is the full local gate and should pass before pushing release-sensitive branches. The lower-level commands below remain useful for focused iteration:

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
- Major user-facing, security, parser, policy, adapter, and output changes need tests or a documented reason in the PR.

## Developer Certificate Of Origin

Nightward uses the Developer Certificate of Origin (DCO) for contributions. Every non-merge commit in a pull request must include a sign-off line:

```text
Signed-off-by: Name <email@example.com>
```

Use `git commit -s` for new commits, or `git commit --amend -s --no-edit` to add the sign-off to the latest commit. The sign-off means you certify that you have the right to submit the contribution under the project license.

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

Maintainer-authored changes should still flow through pull requests and human review unless there is an emergency security fix or repository administration task that cannot wait. Emergency exceptions should be documented after the fact.
