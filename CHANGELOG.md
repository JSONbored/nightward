# Changelog

Nightward uses Conventional Commit-style PR titles so release notes stay readable.

Formal changelog entries begin with the first tagged release.

## Unreleased

- Replaced pre-1.0 deterministic SHA-1 IDs with SHA-256-backed IDs while preserving the short ID shape.
- Added OpenSSF Silver-ready governance, threat model, DCO, coverage, and release snapshot hardening.
- Added a release-gated npm launcher package that downloads GitHub Release binaries on first run without `postinstall`.
