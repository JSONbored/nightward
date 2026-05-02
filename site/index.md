---
layout: home

hero:
  name: Nightward
  text: Find AI-tool risks before you sync.
  tagline: Scan agent configs, MCP servers, and dotfiles for secrets, broad local access, and machine-only state. Local by default. Review-first by design.
  image:
    src: /logo.svg
    alt: Nightward logo
  actions:
    - theme: brand
      text: Install Nightward
      link: /guide/install
    - theme: alt
      text: View sample report
      link: /demo/nightward-sample-report.html
    - theme: alt
      text: GitHub
      link: https://github.com/JSONbored/nightward

features:
  - title: Local-first by default
    details: No telemetry, no default network calls, no cloud dashboard, and no live config mutation.
  - title: Built for MCP-heavy machines
    details: Scans Codex, Claude, Cursor, VS Code, Cline/Roo, Goose, OpenCode, Raycast, generic MCP files, and workspace AI config.
  - title: Reviewable output
    details: Emits redacted JSON, searchable static HTML reports, report diffs, policy badges, SARIF, Trunk output, MCP context, and Raycast status.
---

<!-- markdownlint-disable MD041 MD033 -->

<section class="nw-install-strip" aria-label="Install Nightward">
  <div class="nw-install-copy">
    <p class="nw-eyebrow">Start now</p>
    <h2>Run a local AI-tool audit in one command.</h2>
    <p>No account, no telemetry, no default network calls, and no config mutation.</p>
  </div>
  <div class="nw-install-command" aria-label="Recommended install command">
    <span>Recommended</span>
    <code>npx @jsonbored/nightward</code>
  </div>
</section>

Prefer a persistent CLI? Use [npm, GitHub Releases, or `go install`](/guide/install), then run `nw scan`.

## Pick A Path

- [Before syncing dotfiles](/start/before-syncing-dotfiles): classify what belongs in a private repo and what should stay machine-local.
- [Audit an MCP-heavy workstation](/start/audit-mcp-workstation): review command execution, broad filesystem access, local endpoints, and credential exposure.
- [Run in CI](/start/run-in-ci): fail a workflow on policy violations and upload SARIF to code scanning.
- [Use Raycast](/integrations/raycast): keep a menu-bar status surface and jump into findings without opening a terminal.
- [Use MCP](/integrations/mcp-server): expose local Nightward context to AI clients without giving them write tools.
- [Verify a release](/security/release-verification): check signatures, checksums, npm provenance, and install behavior.

## Real Fixture Output

The sample report below is generated from the committed `testdata/homes/policy` fixture home. Hostname, HOME, local paths, timestamps, and secret-looking fixture values are scrubbed before publication.

[![Scrubbed Nightward HTML report showing fixture MCP findings](/demo/nightward-sample-report.png)](/demo/nightward-sample-report.html)

[Sample scan JSON](/demo/nightward-sample-scan.json) · [Static HTML report](/demo/nightward-sample-report.html) · [TUI GIF](/demo/nightward-tui.gif) · [Provider reference](/reference/providers) · [Output surfaces](/reference/output-surfaces)

![Nightward TUI fixture walkthrough](/demo/nightward-tui.gif)

## What Nightward Checks

| Area | What you get |
| --- | --- |
| Inventory | Portable, machine-local, secret-auth, runtime-cache, app-owned, and unknown state across HOME or a workspace. |
| MCP security | Findings for unpinned package executors, shell wrappers, broad filesystem mounts, sensitive env/header exposure, local endpoints, token paths, symlinks, parse failures, and unknown server shapes. |
| Report history | Compare scan JSON files, inspect latest-report status, render filterable diff-aware HTML, and generate a static local report index. |
| Policy and CI | Reason-required ignores, policy badges, SARIF output, GitHub Action mode, and Trunk plugin support. |
| Providers | Local `gitleaks`, `trufflehog`, and `semgrep`; online-gated `trivy`, `osv-scanner`, and remote Socket scan creation. |
| MCP server | Read-only stdio tools/resources for local AI clients; no network listener, no mutation tools, and no online providers in v1. |

## Trust Posture

Nightward v0.1.4 ships through signed GitHub Releases and a no-`postinstall` npm launcher that verifies GitHub Release checksums before running a cached binary. The project keeps OpenSSF evidence in-repo, runs CodeQL/Scorecard/Gitleaks/govulncheck/gosec/staticcheck, and keeps online-capable providers blocked until explicitly enabled.

Nightward does not copy secrets, push to Git, restore configs, sync machines, or apply live mutations in v1.
