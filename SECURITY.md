# Security Policy

Nightward is local-first security tooling for AI agent and devtool state. It should be held to a conservative standard because it inspects paths that may sit near credentials.

## Supported Versions

Nightward is pre-1.0. Security fixes target `main` until tagged releases exist.

## Runtime Security Posture

- No telemetry.
- No cloud dashboard.
- No network calls from the Nightward runtime.
- No secret values in JSON, Markdown, SARIF, or TUI output.
- No agent config mutation from scan, doctor, policy, backup-plan, snapshot, findings, fix-plan, or fix-preview commands.
- Explicit output flags may write redacted report or SARIF artifacts.
- Scheduling only writes explicit user-level scheduler files through `schedule install`.
- Policy ignores require documented reasons so suppressions are reviewable.

## Reporting a Vulnerability

Open a private security advisory through [GitHub Security Advisories](https://github.com/JSONbored/nightward/security/advisories/new) if available. If that path is unavailable, open a public issue with sensitive details removed and say that you need a private disclosure channel.

Do not include real tokens, auth files, private MCP configs, or personal paths in reports. Redact values and keep only the minimum config shape needed to reproduce the issue.

Maintainers aim to acknowledge vulnerability reports within 14 days. If no public response is appropriate, acknowledgement may happen privately in the GitHub Security Advisory thread.

## Vulnerability Handling

Maintainers triage reports by practical impact:

- Critical: secret disclosure, unsafe write/default mutation, release compromise, or exploitable code execution.
- High: reliable policy bypass, redaction failure in a common output path, or CI behavior that could publish untrusted artifacts.
- Medium: parser misclassification, unsafe remediation guidance, or denial-of-service through malformed local config.
- Low: documentation gaps, unclear warnings, or hardening improvements without direct exploitability.

Confirmed vulnerabilities are tracked privately until a fix is ready or disclosure is safe. Fixes should include regression tests for the affected output surface, parser shape, or workflow path. Release notes must call out publicly known project vulnerabilities that had a CVE, GHSA, or similar public identifier when the release was created.

Reporters may be credited in release notes or advisories unless they request otherwise. Nightward does not pay bug bounties.

## What Counts

- Secret value disclosure in any output surface.
- Unexpected writes outside explicit schedule install/remove or requested output files.
- Unsafe remediation guidance that would leak credentials or broaden permissions.
- CI/release workflow behavior that could let untrusted code publish artifacts.
- Parser behavior that misclassifies known high-risk MCP config as safe.

## Threat Model

The active threat model is maintained in [docs/threat-model.md](docs/threat-model.md).
