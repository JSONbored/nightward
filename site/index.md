---
layout: home

hero:
  name: Nightward
  text: Audit AI agent state before it leaks into dotfiles.
  tagline: Local-first TUI and CLI for MCP security, AI devtool inventory, report history, policy output, and plan-only remediation.
  image:
    src: /logo.svg
    alt: Nightward logo
  actions:
    - theme: brand
      text: Start with dotfiles
      link: /start/before-syncing-dotfiles
    - theme: alt
      text: Verify a release
      link: /security/release-verification
    - theme: alt
      text: GitHub
      link: https://github.com/JSONbored/nightward

features:
  - title: Local-first by default
    details: No telemetry, no default network calls, no cloud dashboard, and no live config mutation.
  - title: Built for MCP-heavy machines
    details: Scans Codex, Claude, Cursor, VS Code, Cline/Roo, Goose, OpenCode, Raycast, generic MCP files, and workspace AI config.
  - title: Reviewable output
    details: Emits redacted JSON, static HTML reports, report diffs, policy badges, SARIF, Trunk output, and Raycast status.
---

<!-- markdownlint-disable MD041 -->

```sh
npx @jsonbored/nightward --help
nw doctor --json
nw scan --workspace . --json --output nightward-scan.json
nw report html --input nightward-scan.json --output nightward-report.html
```

## Pick A Path

- [Before syncing dotfiles](/start/before-syncing-dotfiles): classify what belongs in a private repo and what should stay machine-local.
- [Audit an MCP-heavy workstation](/start/audit-mcp-workstation): review command execution, broad filesystem access, local endpoints, and credential exposure.
- [Run in CI](/start/run-in-ci): fail a workflow on policy violations and upload SARIF to code scanning.
- [Use Raycast](/integrations/raycast): keep a menu-bar status surface and jump into findings without opening a terminal.
- [Verify a release](/security/release-verification): check signatures, checksums, npm provenance, and install behavior.

## Real Fixture Output

The sample report below is generated from the committed `testdata/homes/policy` fixture home. Hostname, HOME, local paths, timestamps, and secret-looking fixture values are scrubbed before publication.

[![Scrubbed Nightward HTML report showing fixture MCP findings](/demo/nightward-sample-report.png)](/demo/nightward-sample-report.html)

[Sample scan JSON](/demo/nightward-sample-scan.json) · [Static HTML report](/demo/nightward-sample-report.html) · [Provider reference](/reference/providers) · [Output surfaces](/reference/output-surfaces)

## What Nightward Checks

| Area | What you get |
| --- | --- |
| Inventory | Portable, machine-local, secret-auth, runtime-cache, app-owned, and unknown state across HOME or a workspace. |
| MCP security | Findings for unpinned package executors, shell wrappers, broad filesystem mounts, sensitive env/header exposure, local endpoints, token paths, symlinks, parse failures, and unknown server shapes. |
| Report history | Compare scan JSON files, render diff-aware HTML, and generate a static local report index. |
| Policy and CI | Reason-required ignores, policy badges, SARIF output, GitHub Action mode, and Trunk plugin support. |
| Providers | Local `gitleaks`, `trufflehog`, and `semgrep`; online-gated `trivy`, `osv-scanner`, and remote Socket scan creation. |

## Trust Posture

Nightward v0.1.4 ships through signed GitHub Releases and a no-`postinstall` npm launcher that verifies GitHub Release checksums before running a cached binary. The project keeps OpenSSF evidence in-repo, runs CodeQL/Scorecard/Gitleaks/govulncheck/gosec/staticcheck, and keeps online-capable providers blocked until explicitly enabled.

Nightward does not copy secrets, push to Git, restore configs, sync machines, or apply live mutations in v1.
