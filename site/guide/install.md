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
| Go install | Go users who want to build from module source | `go install github.com/jsonbored/nightward/cmd/nw@latest` |
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

The launcher has no `postinstall` script. On first run it downloads the matching GitHub Release archive, verifies its SHA-256 from `checksums.txt`, caches the extracted binaries, and executes `nightward` or `nw`.

## GitHub Releases

Use GitHub Releases when you want to verify everything yourself:

1. Download the archive for your platform.
2. Download `checksums.txt` and `checksums.txt.sig`.
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
| npm launcher | Shipped | No `postinstall`; verifies GitHub Release checksums. |
| `go install` | Shipped | Useful for Go-native workflows. |
| Trunk plugin import | Shipped | Pin to a Nightward release tag or SHA. |
| GitHub Action tags | Shipped | Use for policy/SARIF checks in CI. |
| Homebrew tap | Planned next | Best next macOS distribution channel. |
| Nix, Scoop, WinGet, mise, aqua | Later | Add after release artifact behavior stays stable. |

Docker is deferred. It is awkward as a primary scanner because Nightward’s most useful mode audits user HOME and local AI-tool config, which should not be casually mounted into containers.
