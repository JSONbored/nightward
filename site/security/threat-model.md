# Threat Model

Nightward's primary asset is local AI/devtool state: config files, MCP server definitions, skills, rules, commands, editor settings, report exports, and scheduled scan output.

## Trust boundaries

- Local filesystem input is untrusted.
- Config files may be malformed, hostile, huge, symlinked, or privacy-sensitive.
- Optional providers may execute local tools and, if explicitly allowed, contact external services; Socket creates a remote scan artifact from dependency manifest metadata.
- GitHub Actions and Trunk integrations treat repository contents and PR input as untrusted.
- Release automation and npm publishing are privileged publishing boundaries.

## Key mitigations

- Read-only scanner and remediation planner by default.
- Redaction across JSON, SARIF, Markdown, TUI, and Raycast output.
- No default network calls.
- Explicit online-provider opt-in.
- GitHub Actions pinned by full SHA.
- Signed release checksums and SBOMs.
- No-postinstall npm launcher with checksum verification.

See the repository `docs/threat-model.md` for the full maintainer-facing model.
