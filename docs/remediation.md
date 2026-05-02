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

`nw fix plan` groups repeated review work where Nightward can identify a shared
root cause. For example, repeated `mcp-remote` unpinned-package findings are
summarized as one grouped review item so the same reviewed version can be
applied consistently instead of triaging dozens of identical hints.

Advisory `mcp_server_review` findings are collapsed when the same server already
has a higher-severity finding. The stronger finding remains the review anchor,
which keeps Raycast, HTML reports, MCP output, and policy results focused on the
actionable risk.

## Rule Guidance

### mcp_secret_env

Move inline secret values out of agent config. Keep only env key names, setup prerequisites, or environment interpolation references in portable files.

### mcp_secret_header

Move sensitive remote MCP header values out of agent config. Keep only header names, setup prerequisites, or environment interpolation references in portable files.

### mcp_local_endpoint

Move loopback, private-network, or machine-local MCP URLs into local-only overlays unless the endpoint is intentionally portable and documented.

### mcp_local_token_path

Move local token, credential, and secret-file paths into local-only overlays. Portable config should document the prerequisite without committing the machine-specific path.

### mcp_symlink_config

Review symlinked MCP config targets before syncing. A portable dotfiles repo should not depend on a symlink that resolves to private or machine-local state.

### mcp_unpinned_package

Pin package-executor commands such as `npx`, `uvx`, or `pipx` to reviewed package versions before syncing MCP config.

### mcp_shell_command

Replace simple shell passthrough wrappers with direct executable invocation. Review compound shell commands manually because they may depend on local shell startup files or expand secrets.

### mcp_broad_filesystem

Replace broad filesystem mounts with explicit project or data paths after confirming the server's real access requirement.

### mcp_unknown_command

Review unsupported MCP server shapes manually. Add an explicit command where the client supports it, or open an adapter issue with a redacted config example.

## Policy Ignores

Policy ignores must include a reason:

```yaml
ignore_rules:
  - rule: mcp_server_review
    reason: Advisory review findings are accepted for this private fixture.
```

Reasonless ignores fail config validation so accidental blanket suppression is visible.
