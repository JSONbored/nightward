# Distribution

Nightward is distributed through signed GitHub Releases and the npm launcher.

## Current Channels

| Channel | Status | Notes |
| --- | --- | --- |
| [GitHub Releases](https://github.com/JSONbored/nightward/releases) | Shipped | Canonical signed artifacts, checksums, and release notes. |
| [npm launcher](https://www.npmjs.com/package/@jsonbored/nightward) | Shipped | No `postinstall`; verifies GitHub Release checksums, validates archive entries, and can require Sigstore verification before caching binaries. |
| Cargo source build | Development | Useful for local development and branch comparison; release users should prefer signed archives or npm. |
| [GitHub Action](/integrations/github-action) | Shipped | Uses release tags for CI policy/SARIF workflows. |
| [Trunk plugin import](/integrations/trunk) | Shipped | Imports the in-repo plugin from release tags. |
| [Raycast extension](/integrations/raycast) | Development-ready | Local Raycast extension commands and menu-bar status; store PR still pending. |
| [MCP server](/integrations/mcp-server) | Shipped in CLI | Stdio tools/resources/prompts plus bounded action preview, approval request/status, and approved-ticket apply. Registry metadata lives in `server.json`. |

## Later Channels

Homebrew is the next packaging target. Nix, Scoop, WinGet, mise, and aqua should follow once release artifacts prove stable across a few tags. Docker is deferred because scanning a user's HOME from a container is awkward and easy to misconfigure.

## Homebrew Path

Homebrew should be a small tap-backed formula generated from the signed GitHub Release archive and checksum data. The formula should install both `nightward` and `nw`, include a lightweight `nightward --version` test, and point users back to the release-verification docs.
