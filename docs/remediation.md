# Remediation

Nightward remediation is plan-only. It recommends changes, explains risk, and can preview redacted patch hunks, but it does not mutate agent configs.

## Fix Kinds

- `pin-package`: pin package-executor MCP servers such as `npx`, `uvx`, or `pipx`.
- `externalize-secret`: move inline secret values into local environment, keychain, or a secret manager.
- `replace-shell-wrapper`: replace simple shell passthroughs with direct executable invocation.
- `narrow-filesystem`: reduce broad filesystem mounts or path arguments.
- `manual-review`: inspect unsupported or ambiguous config shapes.
- `ignore-with-reason`: accept advisory findings only with documented reasoning.

## Preview Rules

`nw fix preview` generates redacted patch previews only when Nightward can parse the config and target a specific MCP server. It does not show raw file diffs because raw config diffs can leak inline secrets.

Package pinning does not guess versions. Choose a reviewed version from the upstream registry or release notes, then edit the package token manually.

## Policy Ignores

Policy ignores must include a reason:

```yaml
ignore_rules:
  - rule: mcp_server_review
    reason: Advisory review findings are accepted for this private fixture.
```

Reasonless ignores fail config validation so accidental blanket suppression is visible.
