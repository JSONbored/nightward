# Pull Request

## Summary

-

## What Changed

-

## Safety

- [ ] Every non-merge commit includes a DCO `Signed-off-by:` line.
- [ ] No agent config mutation was added outside an explicit apply/write path.
- [ ] New output surfaces redact secret values.
- [ ] New adapters classify portable, machine-local, secret-auth, runtime-cache, app-owned, or unknown state.
- [ ] Workflow changes use least-privilege permissions and pinned third-party actions.
- [ ] Maintainer PRs received human review, or an emergency/admin exception is explained below.

## Validation

- [ ] `go test ./...`
- [ ] `make coverage-check`
- [ ] `make test-junit`
- [ ] `make trunk-flaky-validate`
- [ ] `trunk check --show-existing --all`
- [ ] `make gitleaks`
- [ ] `make govulncheck`
- [ ] `make fuzz-smoke`
- [ ] `make release-snapshot` when release/build behavior changed

## Notes

-
