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

<script setup>
import { withBase } from "vitepress";
</script>

<section class="nw-install-strip" aria-label="Install Nightward">
  <div class="nw-install-copy">
    <p class="nw-eyebrow">Start now</p>
    <h2>Run a local AI-tool audit in one command.</h2>
    <p>No account, no telemetry, no default network calls, and no config mutation.</p>
  </div>
  <div class="nw-install-command" aria-label="Recommended install command">
    <span>Start with a read-only scan</span>
    <code>npx @jsonbored/nightward scan</code>
  </div>
</section>

Prefer a persistent CLI? Use [npm, GitHub Releases, or a source build](/guide/install), then run `nw`.

## Pick A Path

- [Before syncing dotfiles](/start/before-syncing-dotfiles): classify what belongs in a private repo and what should stay machine-local.
- [Audit an MCP-heavy workstation](/start/audit-mcp-workstation): review command execution, broad filesystem access, local endpoints, and credential exposure.
- [Run in CI](/start/run-in-ci): fail a workflow on policy violations and upload SARIF to code scanning.
- [Use Raycast](/integrations/raycast): keep a menu-bar status surface and jump into findings without opening a terminal.
- [Use MCP](/integrations/mcp-server): expose local Nightward context and bounded action workflows to Claude, Cursor, Codex, Antigravity, Windsurf, and other MCP clients.
- [Verify a release](/security/release-verification): check signatures, checksums, npm provenance, and install behavior.

## Real Fixture Output

The sample report below is generated from the committed `testdata/homes/policy` fixture home. Hostname, HOME, local paths, timestamps, and secret-looking fixture values are scrubbed before publication.

[![Scrubbed Nightward HTML report showing fixture MCP findings](/demo/nightward-sample-report.png)](/demo/nightward-sample-report.html)

[Sample scan JSON](/demo/nightward-sample-scan.json) · [Static HTML report](/demo/nightward-sample-report.html) · [OpenTUI gallery](/guide/tui) · [OpenTUI GIF](/demo/nightward-opentui.gif) · [Provider reference](/reference/providers) · [Output surfaces](/reference/output-surfaces)

<section id="tui-media" class="nw-tui-media" aria-labelledby="nw-tui-media-title">
  <div class="nw-tui-media__copy">
    <p class="nw-eyebrow">Terminal review flow</p>
    <h2 id="nw-tui-media-title">Move from posture to evidence without leaving the terminal.</h2>
    <p>The loop uses the scrubbed fixture report: overview, findings, offline analysis, plan-only fixes, inventory, backup choices, and safety reminders.</p>
    <p class="nw-tui-media__links">
      <a :href="withBase('/guide/tui')">Open the TUI guide</a>
      <a href="https://github.com/JSONbored/nightward">Star on GitHub</a>
    </p>
  </div>
  <a class="nw-tui-media__frame" :href="withBase('/guide/tui')" aria-label="Open the Nightward TUI guide">
    <video class="nw-tui-media__video" autoplay muted loop playsinline :poster="withBase('/demo/tui/overview.png')">
      <source :src="withBase('/demo/tui/nightward-opentui.webm')" type="video/webm">
    </video>
    <img class="nw-tui-media__fallback" :src="withBase('/demo/tui/overview.png')" alt="Nightward OpenTUI dashboard from scrubbed fixture output">
  </a>
</section>

## What Nightward Checks

| Area | What you get |
| --- | --- |
| Inventory | Portable, machine-local, secret-auth, runtime-cache, app-owned, and unknown state across HOME or a workspace. |
| MCP security | Findings for unpinned package executors, package-name impersonation risk, remote package sources, shell wrappers, Docker/socket exposure, broad filesystem mounts, sensitive env/header exposure, local endpoints, token paths, stale configs, symlinks, parse failures, and unknown server shapes. |
| Report history | Compare scan JSON files, inspect latest-report status, render filterable diff-aware HTML, and generate a static local report index. |
| Policy and CI | Reason-required ignores, policy badges, SARIF output, GitHub Action mode, and Trunk plugin support. |
| Providers | Local [Gitleaks](https://github.com/gitleaks/gitleaks), [TruffleHog](https://github.com/trufflesecurity/trufflehog), [Semgrep](https://semgrep.dev/), and [Syft](https://oss.anchore.com/syft/); online-gated [Trivy](https://trivy.dev/), [OSV-Scanner](https://google.github.io/osv-scanner/), [Grype](https://oss.anchore.com/grype/), [OpenSSF Scorecard](https://github.com/ossf/scorecard), and remote [Socket](https://socket.dev/) scan creation. |
| MCP server | Exposes tools, resources, and prompts over stdio for local AI clients; includes action previews, approval workflows, and controlled application of approved actions through the shared local action registry. |

## Trust Posture

Nightward ships through signed GitHub Releases and a no-`postinstall` npm launcher that verifies GitHub Release checksums, validates archive entries, and can require Sigstore verification before running a cached Rust binary. The project keeps OpenSSF evidence in-repo, runs CodeQL/Scorecard/Gitleaks/OSV/Clippy, and keeps online-capable providers blocked until explicitly enabled.

Nightward does not copy secrets, push to Git, restore configs, sync machines, or rewrite live MCP/agent configs in v1. Confirmed actions are limited to provider setup/settings, user-level scheduled scans, and local portable backup snapshots.
