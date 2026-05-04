# Website And Docs Plan

Nightward's public website lives under `site/` and uses VitePress. This mirrors Beszel's static-docs model while keeping Nightward's docs fully repo-owned.

The site currently uses the latest VitePress 2 alpha because VitePress 1.x depends on a Vite/esbuild development-server chain with an unresolved moderate npm advisory. The site is static, analytics-free by default, and CI builds it with `npm audit --audit-level=moderate`.

The public deployment target is <https://nightward.aethereal.dev/>. Use `NIGHTWARD_SITE_URL` and `NIGHTWARD_SITE_BASE` when building an alternate target, such as the fallback GitHub Pages project URL:

```sh
NIGHTWARD_SITE_URL=https://jsonbored.github.io/nightward/ \
NIGHTWARD_SITE_BASE=/nightward/ \
npm run build
```

Public sample output lives under `site/public/demo/` and is generated from committed fixture data:

```sh
make demo-assets
```

The generator rewrites hostname, HOME, local paths, timestamps, and secret-looking fixture values before writing the sample scan JSON, static HTML report, and PNG screenshot.
It also renders the static Open Graph preview image used by the website metadata.
Screenshot capture requires Chrome, Chromium, Brave, or `NIGHTWARD_CHROME=/path/to/browser`.

TUI media is generated separately because it needs VHS and ffmpeg:

```sh
make tui-media
```

That target uses the scrubbed sample scan plus `NIGHTWARD_TUI_VIEW` to write seven gallery PNGs under `site/public/demo/tui/`, refresh the legacy TUI PNG/GIF, and build `site/public/demo/tui/nightward-opentui.webm` for the homepage animation. Review generated frames for `/Users`, username, hostname, private MCP names, real project paths, and secret-looking values before committing.

## Site Goals

- Explain the problem in the first viewport.
- Make the install path obvious.
- Keep favicon, social preview metadata, and release-current copy in the repo.
- Show the local-first privacy stance clearly.
- Use fixture-only terminal media on the homepage, with a reduced-motion static fallback.
- Document CLI, TUI, policy, integrations, security, and release verification.
- Document the read-only MCP server as an AI-client integration, not a write/control surface.
- Avoid runtime analytics, telemetry, or hosted-docs dependencies by default.
- Allow the deployed public website to use explicitly configured, self-hosted Umami for aggregate visitor analytics.

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

## Analytics Boundary

Nightward runtime surfaces stay no-telemetry and no-default-network: CLI, TUI, MCP server, Raycast, npm launcher, and local docs preview do not emit analytics.

The public website can load self-hosted Umami only when the Pages build receives both `NIGHTWARD_UMAMI_SCRIPT_URL` and `NIGHTWARD_UMAMI_WEBSITE_ID`. The tracker is configured with:

- `data-domains="nightward.aethereal.dev"`
- `data-do-not-track="true"`
- `data-exclude-search="true"`
- `data-exclude-hash="true"`

Do not commit Umami credentials or make analytics mandatory for local builds.

## Deployment

The current deployment target is GitHub Pages via `.github/workflows/pages.yml`: <https://nightward.aethereal.dev/>. The custom domain uses a DNS-only `nightward.aethereal.dev` CNAME to `jsonbored.github.io`, with GitHub Pages enforcing HTTPS after certificate issuance.
