#!/usr/bin/env node
import { execFileSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const check = process.argv.includes("--check");
const outDir = check ? mkdtempSync(join(tmpdir(), "nightward-docs-")) : "site/reference";

function runNightward(args) {
  return execFileSync("go", ["run", "./cmd/nw", ...args], {
    encoding: "utf8",
    env: { ...process.env, NIGHTWARD_HOME: join(tmpdir(), "nightward-docs-home") },
    stdio: ["ignore", "pipe", "pipe"],
  }).trimEnd();
}

function parseJSON(args) {
  return JSON.parse(runNightward(args));
}

function write(name, content) {
  writeFileSync(join(outDir, name), `${content.trimEnd()}\n`);
}

const help = runNightward(["--help"]);
const providers = parseJSON(["providers", "list", "--json"]);
const rules = parseJSON(["rules", "list", "--json"]);
const policyExplain = runNightward(["policy", "explain"]);
const defaultPolicy = runNightward(["policy", "init"]);

const providerLinks = new Map([
  ["nightward", { home: "https://github.com/JSONbored/nightward" }],
  [
    "gitleaks",
    {
      home: "https://github.com/gitleaks/gitleaks",
      install: "https://github.com/gitleaks/gitleaks#installing",
    },
  ],
  [
    "trufflehog",
    {
      home: "https://github.com/trufflesecurity/trufflehog",
      install: "https://github.com/trufflesecurity/trufflehog#installation",
    },
  ],
  [
    "semgrep",
    {
      home: "https://semgrep.dev/",
      install: "https://semgrep.dev/docs/getting-started/",
    },
  ],
  [
    "trivy",
    {
      home: "https://trivy.dev/",
      install: "https://trivy.dev/latest/getting-started/installation/",
    },
  ],
  [
    "osv-scanner",
    {
      home: "https://google.github.io/osv-scanner/",
      install: "https://google.github.io/osv-scanner/installation/",
    },
  ],
  [
    "socket",
    {
      home: "https://socket.dev/",
      install: "https://docs.socket.dev/docs/socket-cli",
    },
  ],
]);

function providerName(provider) {
  const link = providerLinks.get(provider.name);
  return link?.home ? `[${provider.name}](${link.home})` : provider.name;
}

function providerInstallDocs(provider) {
  const link = providerLinks.get(provider.name);
  return link?.install ? `[docs](${link.install})` : "built-in";
}

write(
  "cli.md",
  `# CLI Reference

This page is generated from \`go run ./cmd/nw --help\`.

\`\`\`text
${help}
\`\`\`
`,
);

write(
  "providers.md",
  `# Provider Reference

This page is generated from \`nw providers list --json\`.

Nightward never installs providers. Local providers run only when selected with \`--with\`. Online-capable providers also require \`--online\` or \`allow_online_providers: true\` in policy config.

| Provider | Mode | Command | Default | Install | Privacy | Capabilities |
| --- | --- | --- | --- | --- | --- | --- |
${providers
  .map((provider) =>
    [
      providerName(provider),
      provider.online ? "online-capable" : "local/offline",
      provider.command ? `\`${provider.command}\`` : "built-in",
      provider.default ? "yes" : "no",
      providerInstallDocs(provider),
      provider.privacy,
      provider.capabilities,
    ].join(" | "),
  )
  .map((row) => `| ${row} |`)
  .join("\n")}

## Online-Capable Providers

- \`trivy\`: explicit filesystem scan with JSON output. Vulnerability database activity can contact upstream services, so Nightward requires \`--online\`.
- \`osv-scanner\`: explicit source scan against vulnerability data. Nightward requires \`--online\`.
- \`socket\`: creates a remote Socket scan artifact and uploads dependency manifest metadata. Nightward does not fetch remote Socket reports in v1.
`,
);

write(
  "rules.md",
  `# Rule Reference

This page is generated from \`nw rules list --json\`.

| Rule | Severity | Category | Fix | Summary |
| --- | --- | --- | --- | --- |
${rules
  .map((rule) =>
    [
      `\`${rule.id}\``,
      rule.default_severity,
      rule.category,
      rule.fix_kind ? `\`${rule.fix_kind}\`` : "manual",
      rule.title,
    ].join(" | "),
  )
  .map((row) => `| ${row} |`)
  .join("\n")}
`,
);

write(
  "config.md",
  `# Config Reference

This page is generated from \`nw policy explain\` and \`nw policy init\`.

## Policy Behavior

\`\`\`text
${policyExplain}
\`\`\`

## Default Config

\`\`\`yaml
${defaultPolicy}
\`\`\`
`,
);

if (check) {
  const changed = [];
  for (const name of ["cli.md", "providers.md", "rules.md", "config.md"]) {
    const expected = readFileSync(join(outDir, name), "utf8");
    const actual = readFileSync(join("site/reference", name), "utf8");
    if (actual !== expected) {
      changed.push(name);
    }
  }
  rmSync(outDir, { recursive: true, force: true });
  if (changed.length > 0) {
    console.error(`Generated reference docs are stale: ${changed.join(", ")}`);
    console.error("Run: node scripts/generate-reference-docs.mjs");
    process.exit(1);
  }
}
