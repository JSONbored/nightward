# Nightward OpenTUI

This package is Nightward's interactive TUI built with [OpenTUI](https://github.com/anomalyco/opentui).

The Go binary remains the scanner and policy engine. It writes a private review bundle and launches the compiled `nightward-tui` sidecar beside `nightward` and `nw`.

## Run

```sh
bun install
bun run demo
```

Run against a scan JSON file:

```sh
bun src/main.ts --input ../../site/public/demo/nightward-sample-scan.json
```

Run against the installed `nightward` binary:

```sh
bun src/main.ts
```

Set `NIGHTWARD_BIN=/path/to/nightward` to point at a specific binary.

## Validate

```sh
bun test
bun run build
bun run compile
make opentui-demo
```

The generated GIF lives at `site/public/demo/nightward-opentui.gif`.

## Packaging Notes

Release archives include `nightward-tui` for the archive OS/architecture. The npm launcher downloads and verifies the same GitHub Release archive, so `npx nightward` gets the sidecar automatically. A raw `go install` build can still run non-interactive commands, but the interactive TUI needs `nightward-tui` beside the Go binary or `NIGHTWARD_TUI_BIN` set explicitly.
