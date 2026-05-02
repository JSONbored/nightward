# Trunk

Nightward includes a [Trunk](https://trunk.io/) plugin definition so teams can run policy and SARIF checks through Trunk Code Quality.

```sh
trunk plugins add --id nightward https://github.com/JSONbored/nightward v0.1.4
trunk check enable nightward-policy
```

Pin the plugin source to a release tag or SHA. Trunk’s plugin docs call out that remote plugin refs should not use branch names because branch refs are unstable.

## Linters

| Linter | Purpose | Network |
| --- | --- | --- |
| `nightward-policy` | Runs workspace policy checks and emits SARIF. | No default network calls |
| `nightward-analyze` | Runs policy checks with offline Nightward analysis signals included. | No default network calls |

Both linters are read-only. They inspect the workspace and write Trunk/SARIF output only through the normal Trunk execution path.

## Publishing Path

There is no separate “marketplace publish” button for a custom Trunk plugin. The public discovery path is:

1. Keep the plugin definition in Nightward and tag releases.
2. Document `trunk plugins add --id nightward https://github.com/JSONbored/nightward <tag>`.
3. If broad demand appears, propose inclusion or an example in [`trunk-io/plugins`](https://github.com/trunk-io/plugins) or Trunk docs.

Nightward should keep the repo-owned plugin first because it lets users pin directly to Nightward release tags and review exactly what the plugin runs.
