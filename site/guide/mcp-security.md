# MCP Security

MCP config is an executable trust boundary. Nightward treats MCP server definitions as reviewable local code entrypoints.

## Findings

Nightward checks for:

- Package executors without pinned versions.
- Shell wrappers.
- Sensitive env keys.
- Sensitive header keys.
- Local/private endpoints.
- Broad filesystem mounts.
- Local token paths.
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
