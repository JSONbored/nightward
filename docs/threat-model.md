# Threat Model

Nightward inspects local AI agent and devtool state, so its primary risk is accidental disclosure or mutation of private local data.

## Assets

- MCP server configs and executable arguments.
- Agent/editor settings, skills, rules, and local workflow files.
- Credentials, auth files, tokens, headers, env files, and private local paths.
- Redacted reports, SARIF files, fix-plan exports, and planned scheduled scan reports.
- Release artifacts, GitHub Actions workflows, and package metadata.

## Trust Boundaries

- Local filesystem input is untrusted. Config files may be malformed, hostile, huge, symlinked, or privacy-sensitive.
- CLI/TUI/Raycast output is a disclosure boundary. Secret values must not cross it.
- Optional providers are execution boundaries. They are discovered, selected, and installed only through explicit action paths, unselected providers are skipped, online-capable providers are blocked until explicitly allowed, and provider timeout/output-cap failures are surfaced as warnings instead of clean results. Trivy, Grype, OSV-Scanner, OpenSSF Scorecard, and Socket are treated as online-capable when their normal operation can contact external services.
- MCP clients are agent boundaries. `nw mcp serve` exposes local context and bounded action previews through stdio, so returned tool/resource/prompt content must stay redacted and the server must not perform local writes.
- GitHub Actions and Trunk integrations treat repository contents and PR input as untrusted.
- Scheduler install/remove is explicit, confirmation-gated, and user-level only.
- Release automation and npm publishing are privileged publishing boundaries.

## Threats And Controls

- Secret disclosure: redact env/header values, secret-looking args, token-like strings, and Markdown/SARIF/TUI exports; test every output surface.
- Unexpected mutation: scan, doctor, findings, fix, policy, backup-plan, snapshot-plan, analysis, MCP, and GitHub Action policy paths stay read-only except explicit output files. TUI, Raycast, and CLI writes must flow through the shared action registry with disclosure acceptance and confirmation; cleanup actions are limited to Nightward-owned report, log, and cache paths.
- Unsafe portability: classify secret-auth, app-owned, runtime-cache, machine-local, and unknown state conservatively.
- MCP execution ambiguity: flag shell wrappers, broad filesystem access, unpinned package execution, package-name impersonation risk, remote package sources, Docker/socket exposure, local/private endpoints, sensitive headers/env, token paths, stale configs, app-owned state, and unknown shapes.
- Supply-chain compromise: pin GitHub Actions by full SHA, use Renovate, run Gitleaks/OSV/CodeQL/Clippy/Trunk, keep release artifacts signed, and keep the npm package as a no-postinstall launcher that verifies archive checksums, rejects unsafe archive entries, and can require Sigstore verification in strict environments.
- Malformed config denial-of-service: keep parser/fuzz coverage for MCP JSON/TOML/YAML, URL/header redaction, symlink traversal, huge-file handling, and malformed configs.
- Agent overreach through MCP: keep MCP read-only. It can list and preview registry actions, but it cannot accept the responsibility disclosure or apply local writes because MCP tool arguments are not an out-of-band user confirmation channel. Tool inputs are validated against strict server-side schemas, and explicit workspace/report paths are scoped under `NIGHTWARD_HOME` with no-symlink regular-file/directory checks. Do not expose live MCP/agent config mutation, restore, sync, HTTP listener behavior, or local write apply through MCP v1.

## Non-Goals

Nightward does not prove a tool, package, MCP server, or URL is safe. It reports local structure, known risky signals, and review guidance.

Nightward can create local portable backup snapshots, but it does not restore, sync, push to Git, or mutate agent configs in v1.

## Review Triggers

Update this model before adding live MCP/agent config mutation, restore, encrypted sync, hosted dashboards, release/npm publishing changes, MCP write tools, or new writable integrations.
