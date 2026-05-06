# TUI

<!-- markdownlint-disable MD033 -->

<script setup>
import { withBase } from "vitepress";
</script>

Run Nightward without arguments to open the interactive terminal app:

```sh
nw
```

Review a saved report without scanning again:

```sh
nw tui --input scan.json
```

<section class="nw-tui-media" aria-labelledby="tui-loop">
  <div class="nw-tui-media__copy">
    <p class="nw-eyebrow">Fixture walkthrough</p>
    <h2 id="tui-loop">Seven screens from one scrubbed report.</h2>
    <p>The homepage loop and gallery below are generated from `site/public/demo/nightward-sample-scan.json`, not from a live workstation scan.</p>
    <p class="nw-tui-media__links">
      <a :href="withBase('/demo/nightward-opentui.gif')">Open GIF</a>
      <a :href="withBase('/demo/tui/nightward-opentui.webm')">Open WebM</a>
    </p>
  </div>
  <a class="nw-tui-media__frame" :href="withBase('/demo/tui/overview.png')" aria-label="Open the fixture dashboard PNG">
    <video class="nw-tui-media__video" autoplay muted loop playsinline :poster="withBase('/demo/tui/overview.png')">
      <source :src="withBase('/demo/tui/nightward-opentui.webm')" type="video/webm">
    </video>
    <img class="nw-tui-media__fallback" :src="withBase('/demo/tui/overview.png')" alt="Nightward TUI fixture overview">
  </a>
</section>

## Screen Gallery

<div class="nw-tui-gallery">
  <figure>
    <img :src="withBase('/demo/tui/overview.png')" alt="Nightward TUI fixture overview">
    <figcaption><strong>Overview.</strong> Severity posture, recent findings, safe defaults, and the first next action.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/demo/tui/findings.png')" alt="Nightward TUI fixture findings">
    <figcaption><strong>Findings.</strong> Searchable finding list beside redacted evidence and plan-only next action detail.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/demo/tui/analysis.png')" alt="Nightward TUI fixture analysis">
    <figcaption><strong>Analysis.</strong> Offline normalized signals grouped by category without provider network calls.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/demo/tui/fix-plan.png')" alt="Nightward TUI fixture fix plan">
    <figcaption><strong>Fix Plan.</strong> Review groups and explicit manual remediation steps, with no live config mutation.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/demo/tui/inventory.png')" alt="Nightward TUI fixture inventory">
    <figcaption><strong>Inventory.</strong> Tool paths by classification and risk so sync candidates stay separate from local-only state.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/demo/tui/backup.png')" alt="Nightward TUI fixture backup">
    <figcaption><strong>Backup.</strong> Dry-run portable candidates and never-sync exclusions from the fixture home model.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/demo/tui/help.png')" alt="Nightward TUI fixture help">
    <figcaption><strong>Help.</strong> Keyboard controls and the confirmed-action safety model shown inside the app.</figcaption>
  </figure>
</div>

## Sections

- Overview: risk posture, severity bars, recent findings, and next action.
- Findings: searchable finding list with redacted detail panes.
- Analysis: normalized offline signals and provider-warning context.
- Fix Plan: plan-only remediation groups and review steps.
- Inventory: discovered AI-tool paths by tool, classification, and risk.
- Backup: dry-run dotfiles backup choices.
- Actions: preview and confirm bounded provider, policy, schedule, backup, cleanup, and setup actions.
- Help: key bindings and safety reminders.

The Rust CLI is the source of truth. The TUI uses embedded `opentui_rust` rendering for the colored dashboard; there is no Bun package or `nightward-tui` sidecar.

## Shortcuts

- `1`-`8`: switch sections.
- `tab`, `right`, or `l`: next section.
- `left` or `h`: previous section.
- `up`, `down`, `j`, or `k`: move selection.
- `enter`: confirm the selected action in the Actions section.
- `y` / `n`: apply or cancel the pending action.
- `/`: search.
- `s`: cycle severity.
- `x`: clear filters.
- `q` or `esc`: quit.

> [!NOTE]
> The TUI is read-only until the user accepts the beta responsibility disclosure and confirms a specific action. High-risk MCP edits remain review-first; bounded provider, policy, schedule, backup, cleanup, and setup actions can be applied from the Actions section.

## Local Development

```sh
cargo run --bin nw
cargo run --bin nw -- tui --input site/public/demo/nightward-sample-scan.json
make demo-assets
make tui-media
```

Use fixture media for public docs; do not capture a real workstation. `make tui-media` requires `vhs` and `ffmpeg`, writes the seven gallery PNGs under `site/public/demo/tui/`, refreshes the legacy TUI PNG/GIF, and builds the homepage WebM loop.
