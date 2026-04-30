## Summary

-

## What Changed

-

## Safety

- [ ] No agent config mutation was added outside an explicit apply/write path.
- [ ] New output surfaces redact secret values.
- [ ] New adapters classify portable, machine-local, secret-auth, runtime-cache, app-owned, or unknown state.
- [ ] Workflow changes use least-privilege permissions and pinned third-party actions.

## Validation

- [ ] `go test ./...`
- [ ] `actionlint`
- [ ] `gitleaks detect --source . --no-git --redact`
- [ ] `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`

## Notes

-
