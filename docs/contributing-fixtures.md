# Contributing Fixtures

Nightward needs real-world config shapes, but contributions must stay synthetic and redacted.

## Good Fixture Rules

- Use fake home paths such as `/tmp/nightward-fixture-home` or `~/example`.
- Use fake tokens such as `super-secret-value`, `Bearer example-token`, or `sk-example-redacted`.
- Preserve key names, nesting, command names, URL shape, and file extension.
- Include the expected finding rule, severity, and redaction behavior in the test or issue.
- Cover one behavior per fixture when possible.

## Do Not Include

- Real API keys, bearer tokens, cookies, private keys, or OAuth material.
- Real usernames, project names, internal hostnames, private IPs, or customer names.
- Full app-owned state, runtime caches, chat transcripts, model caches, or logs.
- Screenshots of real local config.

## Useful Fixture Requests

- New MCP client config shapes for Codex, Claude Code, Cursor, VS Code, Cline/Roo, Goose, OpenCode, and generic MCP tools.
- Malformed JSON/TOML/YAML cases that should produce safe parse findings.
- Huge-file, symlink, local endpoint, credential path, broad filesystem, URL/header, and redaction edge cases.
- Provider output samples from `gitleaks`, `trufflehog`, `semgrep`, `trivy`, `osv-scanner`, `grype`, `syft`, `scorecard`, and `socket` with all secret material replaced.

Open an adapter or rule request issue when the fixture is not ready as a pull request.
