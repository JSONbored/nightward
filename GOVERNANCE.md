# Governance

Nightward is maintained as a small, security-sensitive open source project. The project favors local custody, explicit user consent, and conservative defaults over broad automation.

## Decision Making

Maintainers make decisions in public issues and pull requests when practical. Security-sensitive reports may be handled privately through GitHub Security Advisories until disclosure is safe.

Project changes should preserve these boundaries:

- no telemetry
- no default network calls
- no live agent-config mutation without an explicit future apply workflow
- no secret values in reports, SARIF, TUI, Raycast, or docs examples
- reviewed, reasoned exceptions for policy suppressions

## Roles

- Maintainers review and merge changes, manage releases, triage security reports, and protect project boundaries.
- Contributors propose code, docs, tests, adapters, and issue reports through GitHub.
- Security reporters may use private advisories for sensitive issues.

## Review Policy

Normal changes should land through pull requests with CI passing and human review. Maintainer-authored changes are not exempt from review once additional maintainers are available. Emergency security or repository administration exceptions should be documented after merge.

## Access Continuity

At least one maintainer must retain admin access to the GitHub repository, release configuration, OpenSSF badge entry, and package publishing surfaces. Additional maintainers should be added only after a history of useful, security-aware contributions.

Maintainers are expected to use 2FA on GitHub and protect signing, release, and package credentials.

## Scope Changes

Large scope changes, especially anything involving config mutation, restore, sync, online providers, or hosted services, should start as an issue or design note before implementation.
