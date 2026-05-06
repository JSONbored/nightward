# Install

The fastest path is the npm launcher:

```sh
npx @jsonbored/nightward scan
```

That runs a read-only HOME scan, prints a redacted summary, and does not mutate local config. For daily use, install the launcher globally:

```sh
npm install -g @jsonbored/nightward
nw
```

Nightward uses signed [GitHub Releases](https://github.com/JSONbored/nightward/releases) as the canonical binary source. Convenience installers point back to those release artifacts.

## Recommended Paths

| Path | Best for | Command |
| --- | --- | --- |
| npm launcher | Fast trial, `npx`, Raycast users, JavaScript-heavy machines | `npx @jsonbored/nightward scan` |
| npm global | Everyday CLI/TUI use | `npm install -g @jsonbored/nightward` |
| GitHub Releases | Manual verification and pinned binary installs | Download from [Releases](https://github.com/JSONbored/nightward/releases) |
| Cargo source build | Rust users who want a local source build | `make install-local` |
| Source checkout | Nightward development | `make install-local` |

The installed commands are:

- `nightward`: canonical project command.
- `nw`: short alias for frequent terminal/TUI use.

## npm Launcher

The npm package is a launcher, not a JavaScript rewrite:

```sh
npx @jsonbored/nightward --help
npm install -g @jsonbored/nightward
nw scan --json
```

The launcher has no `postinstall` script. On first run it downloads the matching GitHub Release archive, verifies its SHA-256 from `checksums.txt`, rejects unsafe archive entries before extraction, caches the extracted binaries, and executes `nightward` or `nw`. Set `NIGHTWARD_NPM_REQUIRE_SIGSTORE=1` to require Cosign verification of `checksums.txt.sigstore.json` before the launcher trusts the checksum file.

Published release archives currently cover macOS arm64/amd64, Linux arm64/amd64, and Windows amd64. Windows ARM64 remains deferred until the release matrix includes a validated Rust build.

## GitHub Releases

Use GitHub Releases when you want to verify everything yourself:

1. Download the archive for your platform.
2. Download `checksums.txt` and `checksums.txt.sigstore.json`.
3. Verify the signed checksum file.
4. Verify the archive checksum.
5. Place `nightward` and `nw` on `PATH`.

See [Release verification](/security/release-verification) for the full command set.

## Source Install

```sh
git clone https://github.com/JSONbored/nightward.git
cd nightward
make install-local
```

This installs `nightward` and `nw` into `~/.local/bin` by default.

## Channels

| Channel | Status | Notes |
| --- | --- | --- |
| GitHub Releases | Shipped | Canonical signed release artifacts. |
| npm launcher | Shipped | No `postinstall`; verifies GitHub Release checksums and validates archive entries. |
| Cargo source build | Development | Useful for local Nightward development and branch comparison. |
| Trunk plugin import | Shipped | Pin to a Nightward release tag or SHA. |
| GitHub Action tags | Shipped | Use for policy/SARIF checks in CI. |
| Homebrew tap | Planned next | Best next macOS distribution channel. |
| Nix, Scoop, WinGet, mise, aqua | Later | Add after release artifact behavior stays stable. |

Docker is deferred. It is awkward as a primary scanner because Nightward’s most useful mode audits user HOME and local AI-tool config, which should not be casually mounted into containers.
