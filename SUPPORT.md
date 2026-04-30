# Support

Nightward is early-stage OSS. Use GitHub issues for reproducible bugs, adapter requests, and feature proposals.

## Before Opening An Issue

Run:

```sh
nw doctor --json
nw scan --json
nw findings list
```

When sharing output, remove private usernames, project names, tokens, credential paths, and real local MCP server details unless they are essential and already public.

## Good Issue Inputs

- Nightward version or commit.
- Operating system and shell.
- Command run.
- Redacted config shape or fixture.
- Expected result.
- Actual result.

## Do Not Post

- API keys, tokens, passwords, private keys, session files, auth JSON, cookies, `.env` values, or full local agent configs.
- Private company/client/project paths.
- Screenshots that include private state.

For suspected secret disclosure or unsafe mutation, follow [SECURITY.md](SECURITY.md).
