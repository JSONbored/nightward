# Config Reference

This page is generated from `nw policy explain` and `nw policy init`.

## Policy Behavior

```text
Nightward policy config is optional and read-only.

Supported file: .nightward.yml

Fields:
  severity_threshold: info|low|medium|high|critical
  ignore_findings: [{id, reason}]
  ignore_rules: [{rule, reason}]
  trusted_commands: command names to suppress command-trust policy noise when evidence matches
  trusted_packages: package names to suppress unpinned-package policy noise when evidence matches
  portable_allow_paths: reviewed portable path prefixes for future adapter policy
  machine_local_deny_paths: path prefixes that should remain local-only
  include_analysis: include offline analysis signals in policy decisions
  analysis_threshold: optional signal threshold when include_analysis is true
  analysis_providers: optional provider names for future explicit provider analysis
  allow_online_providers: allow selected network-capable providers when analysis_providers requests them
  sarif.tool_name: SARIF tool display name
  sarif.category: SARIF automation category
  sarif.information_uri: SARIF tool information URI
  sarif.semantic_version: SARIF semantic version

Ignore entries must include a reason. Nightward never expands or prints secret values from policy config.
```

## Default Config

```yaml
severity_threshold: high
ignore_findings: []
ignore_rules: []
trusted_commands: []
trusted_packages: []
portable_allow_paths: []
machine_local_deny_paths: []
include_analysis: false
analysis_threshold: high
analysis_providers: []
allow_online_providers: false
sarif:
    tool_name: Nightward
    category: nightward
    information_uri: https://github.com/JSONbored/nightward
    semantic_version: 0.1.4
```
