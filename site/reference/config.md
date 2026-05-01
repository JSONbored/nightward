# Config Reference

Nightward policy config is `.nightward.yml`.

Generate the default config:

```sh
nw policy init --dry-run
```

Explain fields:

```sh
nw policy explain
```

## Controls

- Severity threshold.
- Ignored finding IDs with required reasons.
- Ignored rules with required reasons.
- Trusted commands and packages.
- Portable path allowlists.
- Machine-local deny/skip paths.
- SARIF category/name overrides.
- Analysis provider controls.

Unknown keys should fail clearly. Ignore entries require non-empty reasons.
