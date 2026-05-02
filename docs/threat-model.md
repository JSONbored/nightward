# Threat Model

Nightward inspects local AI agent and devtool state, so its primary risk is accidental disclosure or mutation of private local data.

## Assets

- MCP server configs and executable arguments.
- Agent/editor settings, skills, rules, and local workflow files.
- Credentials, auth files, tokens, headers, env files, and private local paths.
- Redacted reports, SARIF files, fix-plan exports, and scheduled scan reports.
- Release artifacts, GitHub Actions workflows, and package metadata.

## Trust Boundaries

- Local filesystem input is untrusted. Config files may be malformed, hostile, huge, symlinked, or privacy-sensitive.
- CLI/TUI/Raycast output is a disclosure boundary. Secret values must not cross it.
- Optional providers are execution boundaries. They are discovered but not installed or run online by default. Socket is treated as online-capable because it creates a remote scan artifact from dependency manifest metadata.
- MCP clients are agent boundaries. `nw mcp serve` exposes read-only local context through stdio, so returned tool/resource content must stay redacted and bounded.
- GitHub Actions and Trunk integrations treat repository contents and PR input as untrusted.
- Scheduler install/remove is an explicit write boundary and must stay user-level.
- Release automation and npm publishing are privileged publishing boundaries.

## Threats And Controls

- Secret disclosure: redact env/header values, secret-looking args, token-like strings, and Markdown/SARIF/TUI exports; test every output surface.
- Unexpected mutation: scan, doctor, findings, fix, policy, backup-plan, snapshot, analysis, TUI, Raycast, and GitHub Action policy paths stay read-only except explicit output files.
- Unsafe portability: classify secret-auth, app-owned, runtime-cache, machine-local, and unknown state conservatively.
- MCP execution ambiguity: flag shell wrappers, broad filesystem access, unpinned package execution, local endpoints, sensitive headers/env, token paths, and unknown shapes.
- Supply-chain compromise: pin GitHub Actions by full SHA, use Renovate, run Gitleaks/govulncheck/OSV/CodeQL/gosec/staticcheck/Trunk, keep release artifacts signed, and keep the npm package as a no-postinstall launcher that verifies archive checksums.
- Malformed config denial-of-service: keep parser/fuzz tests for MCP JSON/TOML/YAML and add size/symlink hardening as the scanner expands.
- Agent overreach through MCP: do not expose write tools, schedule install/remove, HTTP listeners, live config mutation, or online provider execution through the MCP server in v1.

## Non-Goals

Nightward does not prove a tool, package, MCP server, or URL is safe. It reports local structure, known risky signals, and review guidance.

Nightward does not back up, restore, sync, push to Git, or mutate agent configs in v1.

## Review Triggers

Update this model before adding live config mutation, restore, encrypted sync, online provider execution, hosted dashboards, release/npm publishing changes, MCP write tools, or new writable integrations.
