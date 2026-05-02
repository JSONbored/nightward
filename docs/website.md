# Website And Docs Plan

Nightward's public website lives under `site/` and uses VitePress. This mirrors Beszel's static-docs model while keeping Nightward's docs fully repo-owned.

The site currently uses the latest VitePress 2 alpha because VitePress 1.x depends on a Vite/esbuild development-server chain with an unresolved moderate npm advisory. The site is static, analytics-free by default, and CI builds it with `npm audit --audit-level=moderate`.

Public sample output lives under `site/public/demo/` and is generated from committed fixture data:

```sh
make demo-assets
```

The generator rewrites hostname, HOME, local paths, timestamps, and secret-looking fixture values before writing the sample scan JSON, static HTML report, and PNG screenshot.
The TUI GIF is generated from `docs/demo/nightward-tui.tape` with VHS using scrubbed fixture output.
It also renders the static Open Graph preview image used by the website metadata.
Screenshot capture requires Chrome, Chromium, Brave, or `NIGHTWARD_CHROME=/path/to/browser`.

## Site Goals

- Explain the problem in the first viewport.
- Make the install path obvious.
- Keep favicon, social preview metadata, and release-current copy in the repo.
- Show the local-first privacy stance clearly.
- Document CLI, TUI, policy, integrations, security, and release verification.
- Document the read-only MCP server as an AI-client integration, not a write/control surface.
- Avoid analytics, telemetry, or hosted-docs dependencies by default.

## Pages

- `/`
- `/start/before-syncing-dotfiles`
- `/start/audit-mcp-workstation`
- `/start/run-in-ci`
- `/guide/what-is-nightward`
- `/guide/getting-started`
- `/guide/install`
- `/guide/privacy-model`
- `/guide/tui`
- `/guide/cli`
- `/guide/mcp-security`
- `/guide/remediation`
- `/guide/policy-and-sarif`
- `/use/provider-execution`
- `/use/report-history`
- `/use/troubleshooting`
- `/integrations/github-action`
- `/integrations/trunk`
- `/integrations/raycast`
- `/reference/cli`
- `/reference/config`
- `/reference/providers`
- `/reference/json-output`
- `/reference/rules`
- `/reference/support-matrix`
- `/reference/output-surfaces`
- `/reference/distribution`
- `/security/threat-model`
- `/security/release-verification`
- `/contribute/adapters-and-rules`
- `/roadmap`

## Stitch Design Brief

Design one polished landing page first. Keep it dark, operator-focused, and concrete. Use terminal/TUI imagery, trust badges, a visible install command, and restrained security-tool styling. Avoid vague AI gradients, generic SaaS hero cards, and claims that Nightward mutates or backs up secrets.

## Deployment

The current deployment target is GitHub Pages via `.github/workflows/pages.yml`: <https://jsonbored.github.io/nightward/>. A custom domain can be added later without changing the framework.
