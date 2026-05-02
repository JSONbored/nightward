# Current Priorities

Nightward’s roadmap is intentionally conservative. The next releases should make local AI-tool review easier to understand, easier to trust, and easier to extend before adding live mutation, restore, or sync behavior.

## Shipped

- Local inventory and MCP security review.
- Redacted JSON, SARIF, policy output, and TUI workflows.
- Plan-only remediation and fix previews.
- GitHub Action, Trunk plugin definition, Raycast extension, and npm launcher.
- Signed v0.1.x releases with npm provenance and release-checksum verification.
- Explicit provider execution for local providers and online-gated provider runs.
- Static HTML reports with local finding filters, report diffs, report history, and sample fixture assets.
- Read-only stdio MCP server for local AI-client integration.
- Bubbles-backed TUI tables, detail panes, footer help, and search input.
- OpenSSF-oriented governance, coverage, DCO, threat model, and release hardening.
- Generated CLI, provider, rule, and config reference pages.

## Next Release Focus

- Report-history comparison across TUI and Raycast, building on the CLI and HTML report diff flow.
- Provider-warning summaries and policy status in HTML reports.
- Generated docs contracts for every public JSON schema and policy example.
- Contributor fixture templates.
- Homebrew tap.
- Fixture-only Raycast screenshots, sample SARIF screenshots, and store-ready Raycast metadata.
- MCP Registry metadata once the package target is settled.
- Raycast Store draft PR after store screenshots and upstream fork sync are complete.

## Later Milestones

- Nix, Scoop, WinGet, mise, and aqua packages.
- Local report browser.
- Encrypted snapshots.
- Cross-machine diff.
- Private dotfiles integration.
- Restore workflow after preview, rollback, and secret-safety controls exist.

## Not Planned For v1

- Telemetry.
- Cloud dashboards.
- Default network calls.
- Live mutation of MCP, agent, or dotfiles config.
- Secret syncing.
