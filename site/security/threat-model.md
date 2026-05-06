# Threat Model

Nightward's primary asset is local AI/devtool state: config files, MCP server definitions, skills, rules, commands, editor settings, report exports, and scheduled scan output.

## Trust boundaries

- Local filesystem input is untrusted.
- Config files may be malformed, hostile, huge, symlinked, or privacy-sensitive.
- Optional providers may execute local tools and, if explicitly allowed, contact external services. Trivy, Grype, OSV-Scanner, OpenSSF Scorecard, and Socket are online-capable.
- GitHub Actions and Trunk integrations treat repository contents and PR input as untrusted.
- Release automation and npm publishing are privileged publishing boundaries.

## Key mitigations

- Read-only scanner and remediation planner by default; writes flow through confirmed shared action-registry actions.
- Redaction across JSON, SARIF, Markdown, TUI, and Raycast output.
- No default network calls.
- Explicit online-provider opt-in.
- MCP is read-only: it can list and preview registry actions, but cannot accept disclosure or apply local writes.
- MCP tool inputs are validated server-side, and explicit workspace/report paths are scoped under `NIGHTWARD_HOME` with no-symlink regular-file/directory checks.
- GitHub Actions pinned by full SHA.
- Signed release checksums and SBOMs.
- No-postinstall npm launcher with checksum verification, archive-entry validation, and optional strict Sigstore verification.

See the repository `docs/threat-model.md` for the full maintainer-facing model.
