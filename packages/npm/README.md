# nightward

NPM launcher for Nightward.

Nightward is a local-first TUI/CLI for auditing AI agent state, MCP config, and dotfiles backup safety.

```sh
npx nightward --help
npm install -g nightward
nw scan --json
```

This package is intentionally a thin launcher for the Go release binaries from <https://github.com/JSONbored/nightward>. It does not use a `postinstall` script. On first run, it downloads the matching GitHub Release archive, verifies the archive SHA-256 against `checksums.txt`, caches the extracted binaries, and then executes `nightward` or `nw`.

For the strongest release verification, use the GitHub Release artifacts and verify the signed checksum file with Cosign.
