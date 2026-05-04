# Config Reference

This page is generated from `nw policy explain` and `nw policy init`.

## Policy Behavior

```text
{
  "severity_threshold": "high",
  "ignore_findings": [],
  "ignore_rules": [],
  "include_analysis": false,
  "analysis_threshold": "high",
  "analysis_providers": [],
  "allow_online_providers": false
}
```

## Default Config

```yaml
severity_threshold: high
ignore_findings: []
ignore_rules: []
include_analysis: false
analysis_threshold: high
analysis_providers: []
allow_online_providers: false
```

Finding ignores use `id` plus a non-empty `reason`. Rule ignores should use `rule` plus a non-empty `reason`; legacy `id` remains accepted as an alias for rule ignores.
