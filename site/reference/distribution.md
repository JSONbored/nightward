# Distribution

Nightward v0.1.4 is distributed through signed GitHub Releases and the npm launcher.

## Current Channels

| Channel | Status | Notes |
| --- | --- | --- |
| [GitHub Releases](https://github.com/JSONbored/nightward/releases) | Shipped | Canonical signed artifacts, checksums, and release notes. |
| [npm launcher](https://www.npmjs.com/package/@jsonbored/nightward) | Shipped | No `postinstall`; verifies GitHub Release checksums, validates archive entries, and can require Sigstore verification before caching binaries. |
| Cargo source build | Development | Useful for local development and branch comparison; release users should prefer signed archives or npm. |
| [GitHub Action](/integrations/github-action) | Shipped | Uses release tags for CI policy/SARIF workflows. |
| [Trunk plugin import](/integrations/trunk) | Shipped | Imports the in-repo plugin from release tags. |
| [Raycast extension](/integrations/raycast) | Development-ready | Local Raycast extension commands and menu-bar status; store PR still pending. |
| [MCP server](/integrations/mcp-server) | Shipped in CLI | Stdio tools/resources/prompts plus bounded read-only action list/preview. Registry metadata lives in `server.json`. |

## Later Channels

Homebrew tap publication is the next packaging target. Nix, Scoop, WinGet, mise, and aqua should follow once release artifacts prove stable across a few tags. Docker is deferred because scanning a user's HOME from a container is awkward and easy to misconfigure.

## Homebrew Support

`scripts/generate-homebrew-formula.mjs` now generates a tap-ready formula from `checksums.txt`. The formula uses the existing signed release archive names, installs both `nightward` and `nw`, and tests both command names with `--version`. The release verifier runs this generation after Cosign and checksum verification, so the formula stays tied to the canonical GitHub Release artifacts.

There is not yet a published Homebrew tap command in public docs.
