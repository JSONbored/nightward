# Getting Started

Run a read-only scan from the release launcher:

```sh
npx @jsonbored/nightward scan
```

That is the shortest proof that Nightward can see your local AI-tool and dotfiles surface. It prints redacted findings and does not write to your configs.

Install the CLI/TUI for repeated use:

```sh
npm install -g @jsonbored/nightward
nw
```

For local development from source:

```sh
git clone https://github.com/JSONbored/nightward.git
cd nightward
make install-local
```

## First Pass

Start with the status and scan commands:

```sh
nw doctor --json
nw scan --json
nw findings list
nw fix plan
```

For a repository or dotfiles workspace:

```sh
nw scan --workspace . --json
nw policy check --workspace . --include-analysis --strict --json
```

> [!TIP]
> Start with `doctor`, then `scan`, then `fix plan`. Do not sync anything until findings and machine-local paths are reviewed.

## What To Review First

| Result | Meaning | Typical next action |
| --- | --- | --- |
| `critical` or `high` MCP finding | A server may expose a secret, run an unpinned package, mount broad filesystem paths, or depend on a local-only endpoint. | Open `nw` or `nw findings explain <id>` and review the exact redacted evidence. |
| `portable` inventory | A path appears safe to consider for private dotfiles after review. | Include only after checking secrets and machine-specific assumptions. |
| `machine-local` inventory | The path probably belongs to one machine. | Keep it out of shared dotfiles or document a local overlay. |
| `secret-auth`, `runtime-cache`, `app-owned` | Nightward thinks the path should not be synced. | Exclude it unless you have a very specific, reviewed reason. |

MCP findings are review prompts, not proof that a server is malicious. The useful output is the exact place to look, the risk pattern, and the plan-only remediation path.
