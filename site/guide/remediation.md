# Remediation

Nightward fix plans are plan-only in v1.

## Fix kinds

- `pin-package`: pin a package executor to a reviewed version.
- `externalize-secret`: move inline sensitive values out of config.
- `replace-shell-wrapper`: prefer direct executable invocation.
- `narrow-filesystem`: replace broad filesystem scope with explicit paths.
- `manual-review`: inspect ambiguous or high-risk config manually.
- `ignore-with-reason`: document why an advisory finding is expected.

## Commands

```sh
nw fix plan --json
nw fix plan --rule mcp_secret_env
nw fix export --format markdown
```

> [!WARNING]
> Nightward does not apply fixes to live agent configs. Fix exports are redacted review artifacts.
