# Getting Started

Try the release launcher or install from source:

```sh
npx @jsonbored/nightward --help
```

For local development:

```sh
git clone https://github.com/JSONbored/nightward.git
cd nightward
make install-local
```

Open the TUI:

```sh
nw
```

Run the first safe CLI checks:

```sh
nw doctor --json
nw scan --json
nw findings list
nw fix plan --all
```

For a repository or dotfiles workspace:

```sh
nw scan --workspace . --json
nw policy check --workspace . --include-analysis --strict --json
```

> [!TIP]
> Start with `doctor`, then `scan`, then `fix plan`. Do not sync anything until findings and machine-local paths are reviewed.

## Common first outcomes

- `portable` items can usually be copied into a private repo after review.
- `machine-local` items need local overlays or per-machine config.
- `secret-auth`, `runtime-cache`, and `app-owned` items are excluded by default.
- MCP findings are review prompts, not proof that a server is malicious.
