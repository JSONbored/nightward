# Install

Nightward uses signed GitHub Releases as the canonical distribution channel. Convenience installers point back to those release artifacts.

## Source install

```sh
make install-local
```

This installs `nightward` and `nw` into `~/.local/bin` by default.

## GitHub Releases

Use GitHub Releases as the canonical binary source:

1. Download the archive for your platform.
2. Download `checksums.txt` and `checksums.txt.sig`.
3. Verify the signed checksum file.
4. Verify the archive checksum.
5. Place `nightward` and `nw` on `PATH`.

## NPM

The npm package is a launcher, not a JavaScript rewrite:

```sh
npx @jsonbored/nightward --help
npm install -g @jsonbored/nightward
nw scan --json
```

The launcher has no `postinstall` script. On first run it downloads the matching GitHub Release archive, verifies its SHA-256 from `checksums.txt`, caches the extracted binaries, and executes `nightward` or `nw`.

## Channels

1. GitHub Releases: shipped.
2. npm launcher: shipped.
3. `go install`: shipped.
4. Trunk plugin import: shipped.
5. GitHub Action tags: shipped.
6. Homebrew tap: planned next.
7. Nix, Scoop, WinGet, mise, and aqua: later.

Docker is deferred until Nightward has a useful local report browser. Docker is not a good default for scanning a user's HOME directory safely.
