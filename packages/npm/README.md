# @jsonbored/nightward

NPM launcher for Nightward.

Nightward shows what AI tools can read, run, and accidentally sync before
configs leave your machine.

```sh
npx @jsonbored/nightward --help
npm install -g @jsonbored/nightward
nw scan --json
```

This package is intentionally a thin launcher for the release binaries from
<https://github.com/JSONbored/nightward>. It does not use a `postinstall`
script. On first run, it downloads the matching GitHub Release archive, verifies
the archive SHA-256 against `checksums.txt`, caches the extracted `nightward`
and `nw` Rust binaries, and then executes `nightward` or `nw`.

Supported launcher platforms are macOS arm64/amd64, Linux arm64/amd64, and
Windows amd64. The embedded terminal TUI currently requires a Unix-like
terminal; use JSON, policy, provider, and HTML report commands on Windows.

Release notes are published with GitHub Releases:

<https://github.com/JSONbored/nightward/releases>

For the strongest release verification, use the GitHub Release artifacts and
verify the signed checksum file with Cosign.
