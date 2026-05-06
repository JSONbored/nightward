# Output Surfaces

| Surface | Command | Write behavior |
| --- | --- | --- |
| Scan JSON | `nw scan --json` or `--output` | stdout unless an explicit output path is provided |
| HTML report | `nw report html` or `--input scan.json --output report.html` | private default report file or explicit local HTML file |
| Report diff | `nw report diff --from previous.json --to current.json` | stdout only |
| Latest report | `nw report latest` | stdout only |
| Report history | `nw report history` | stdout only |
| History index | `nw report index --output index.html` | explicit local HTML file |
| Policy report | `nw policy check --json` | stdout only |
| SARIF | `nw policy sarif --output nightward.sarif` | explicit SARIF file or stdout with `-` |
| Badge JSON | `nw policy badge --output badge.json` | explicit badge file or stdout with `-` |
| Fix plan | `nw fix plan --json` | stdout only |
| Fix export | `nw fix export --format markdown` | stdout only |
| Actions list/preview | `nw actions list --json`, `nw actions preview <id> --json` | stdout only |
| Actions apply | `nw actions apply <id> --confirm` | disclosure-accepted, confirmation-gated provider, policy, schedule, backup, or settings writes |
| MCP server | `nw mcp serve` | stdio JSON-RPC only; read tools plus shared action-registry list/preview, no local writes |
| Schedule install/remove | `nw schedule install --confirm`, `nw schedule remove --confirm` | user-level launchd/systemd files only |
| Backup snapshot | `nw backup create --confirm` | local snapshot under Nightward state |

Labels used in docs:

- `read-only`: reads local config and writes only stdout.
- `explicit write`: writes only the path requested by the user.
- `online-capable`: can invoke provider behavior that contacts a network service.
- `plan-only`: generates review material without mutating live config.
- `confirmed action`: mutates only after explicit preview and confirmation.
- `mcp action preview`: shows shared Nightward action write targets and risks without applying local writes.
- `future/not shipped`: documented as roadmap, not a current interface.
