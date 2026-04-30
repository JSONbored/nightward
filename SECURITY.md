# Security Policy

Nightward is local-first security tooling for AI agent and devtool state. It should be held to a conservative standard because it inspects paths that may sit near credentials.

## Supported Versions

Nightward is pre-1.0. Security fixes target `main` until tagged releases exist.

## Runtime Security Posture

- No telemetry.
- No cloud dashboard.
- No network calls from the Nightward runtime.
- No secret values in JSON, Markdown, SARIF, or TUI output.
- No config mutation from scan, doctor, policy, backup-plan, findings, or fix-plan commands.
- Scheduling only writes explicit user-level scheduler files through `schedule install`.

## Reporting a Vulnerability

Open a private security advisory on GitHub if available. If not, open a public issue with sensitive details removed and say that you need a private disclosure channel.

Do not include real tokens, auth files, private MCP configs, or personal paths in reports. Redact values and keep only the minimum config shape needed to reproduce the issue.

## What Counts

- Secret value disclosure in any output surface.
- Unexpected writes outside explicit schedule install/remove or requested output files.
- Unsafe remediation guidance that would leak credentials or broaden permissions.
- CI/release workflow behavior that could let untrusted code publish artifacts.
- Parser behavior that misclassifies known high-risk MCP config as safe.
