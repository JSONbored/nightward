# MCP Security

MCP config is an executable trust boundary. Nightward treats MCP server definitions as reviewable local code entrypoints.

## Findings

Nightward checks for:

- Package executors without pinned versions.
- Package-name impersonation or direct remote package sources.
- Shell wrappers.
- Docker/socket or privileged host-control exposure.
- Sensitive env keys.
- Sensitive header keys.
- Local/private endpoints.
- Broad filesystem mounts.
- Local token paths.
- Stale configs and app-owned state that should not be synced as portable dotfiles.
- Parse failures.
- Unknown server shapes.

## URL-shaped servers

Remote MCP servers using `url`, `type`, `transport`, and `headers` are recognized as URL-shaped servers. Nightward redacts URL evidence to scheme and host and never prints header values.

## What a finding means

A finding means "review this before syncing." It does not prove a server is malicious.

Use:

```sh
nw findings explain <finding-id>
nw analyze finding <finding-id> --json
nw fix plan --finding <finding-id>
```

## MCP action approvals

Nightward MCP clients can request bounded action tickets, but they cannot approve their own requests or accept the beta responsibility disclosure, Nightward's local one-time acknowledgement that write-capable beta actions are user-authorized. Review the exact action, command, writes, risk, and expiry in the TUI, Raycast, or CLI:

```sh
nw approvals list --json
nw approvals approve <approval-id> --reason "reviewed locally"
```

After local approval, the MCP client can apply only that exact one-time ticket once. If the action preview changes, the ticket expires, or the client tries to replay it, Nightward blocks the application.
