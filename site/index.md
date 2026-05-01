---
layout: home

hero:
  name: Nightward
  text: Audit AI agent state before it leaks into dotfiles.
  tagline: Local-first TUI and CLI for MCP security, agent config inventory, policy output, and dry-run backup planning.
  image:
    src: /logo.svg
    alt: Nightward logo
  actions:
    - theme: brand
      text: Get started
      link: /guide/getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/JSONbored/nightward

features:
  - title: Local-first by default
    details: No telemetry, no default network calls, and no live agent-config mutation.
  - title: Built for AI tool sprawl
    details: Scans Codex, Claude, Cursor, Windsurf, VS Code, Raycast, JetBrains, Zed, MCP configs, and workspace AI state.
  - title: CI-ready security output
    details: Emits redacted JSON, policy reports, SARIF, Trunk plugin output, and GitHub Action results.
---

<!-- markdownlint-disable MD041 -->

```sh
npx @jsonbored/nightward --help
nw doctor --json
nw scan --workspace . --json
nw policy sarif --workspace . --include-analysis --output nightward.sarif
```

## Real Fixture Output

The sample report below is generated from the committed `testdata/homes/policy` fixture home. Hostname, HOME, local paths, timestamps, and secret-looking fixture values are scrubbed before publication.

[![Scrubbed Nightward HTML report showing one fixture item and four MCP findings](/demo/nightward-sample-report.png)](/demo/nightward-sample-report.html)

_Static HTML report rendered from scrubbed Nightward JSON._

[Scrubbed sample JSON](/demo/nightward-sample-scan.json) · [Open the static HTML report](/demo/nightward-sample-report.html)

## Why Nightward exists

AI coding tools leave useful state in config files, MCP server definitions, rules, commands, skills, editor settings, caches, and local databases. Some of that belongs in a private dotfiles repo. Some of it is machine-local. Some of it is app-owned. Some of it is credential material.

Nightward gives users a redacted, reviewable answer before anything is synced.

- Inventory: find agent and devtool state across HOME or a workspace, then classify it as portable, machine-local, secret-auth, runtime-cache, app-owned, or unknown.
- MCP security: flag unpinned package executors, shell wrappers, broad filesystem mounts, sensitive env/header exposure, local endpoints, token paths, and unknown server shapes.
- Plan-only fixes: generate remediation plans and patch previews without mutating live agent configs.
- Distribution-ready: signed release artifacts, npm launcher, GitHub Action, Trunk plugin, Raycast companion, and OpenSSF-focused project hygiene.

> [!IMPORTANT]
> Nightward does not copy secrets, push to Git, restore configs, or apply live mutations in v1.

## Current release posture

Nightward is pre-1.0. The first release channel is signed GitHub Releases, followed by a no-postinstall npm launcher that verifies release checksums before running a cached binary.

See [install](/guide/install), [release verification](/security/release-verification), and [policy/SARIF](/guide/policy-and-sarif).
