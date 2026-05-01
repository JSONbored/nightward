# Before Syncing Dotfiles

Use this path before copying AI-tool config into a private dotfiles repo.

## Run

```sh
nw doctor --json
nw scan --json --output nightward-scan.json
nw fix plan --all --json
nw plan backup --target ~/dotfiles
nw report html --input nightward-scan.json --output nightward-report.html
```

## Review

1. Start with `secret-auth` and `machine-local` inventory items. These should usually stay out of dotfiles.
2. Review every MCP finding before copying config. Package executors, shell wrappers, broad filesystem roots, local endpoints, and inline credentials deserve a human decision.
3. Use fix plans as review material. Nightward does not apply live config mutations in v1.
4. Keep generated reports private unless they were produced from fixtures or scrubbed manually.

## Next Scan

For scheduled scans, compare the newest report with the prior one:

```sh
nw report history
nw report diff --from previous.json --to current.json
nw report html --input current.json --previous previous.json --output current.html
```
