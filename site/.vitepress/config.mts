import { defineConfig, type HeadConfig } from "vitepress";
import { writeFileSync } from "node:fs";
import { join } from "node:path";

function trailingSlash(value: string): string {
  return value.endsWith("/") ? value : `${value}/`;
}

const siteUrl = trailingSlash(process.env.NIGHTWARD_SITE_URL ?? "https://nightward.aethereal.dev/");
const base = process.env.NIGHTWARD_SITE_BASE ?? new URL(siteUrl).pathname;
const siteTitle = "Nightward";
const siteDescription =
  "Find AI-tool risks before you sync: scan agent configs, MCP servers, and dotfiles for secrets, broad local access, and machine-only state.";
const socialImage = new URL("og-image.png", siteUrl).href;
const umamiScriptUrl = process.env.NIGHTWARD_UMAMI_SCRIPT_URL;
const umamiWebsiteId = process.env.NIGHTWARD_UMAMI_WEBSITE_ID;
const umamiDomains = process.env.NIGHTWARD_UMAMI_DOMAINS ?? "nightward.aethereal.dev";

const pageDescriptions: Record<string, string> = {
  "": "Nightward audits AI-agent configs, MCP servers, and dotfiles sync risk locally, with redacted reports and an OpenTUI review flow.",
  "guide/install": "Install Nightward with the npm launcher, signed GitHub Release binaries, or a local Rust source build.",
  "guide/tui": "Explore Nightward's OpenTUI dashboard, findings, analysis, fix-plan, inventory, backup, and help screens from scrubbed fixture media.",
  "guide/privacy-model": "Understand Nightward's local-first privacy model, write paths, redaction rules, optional providers, and website analytics boundary.",
  "reference/cli": "Generated Nightward CLI reference for scan, analyze, provider, policy, report, TUI, and MCP server commands.",
  "integrations/raycast": "Use Nightward's read-only Raycast extension for menu-bar status, findings, analysis, provider doctor, and redacted exports.",
  "integrations/github-action": "Run Nightward in GitHub Actions for workspace scans, policy checks, SARIF upload, and release-gated CI review.",
  "security/release-verification": "Verify Nightward signed releases, checksums, npm launcher behavior, and installed binaries before trusting a release.",
};

const analyticsHead: HeadConfig[] = umamiScriptUrl && umamiWebsiteId
  ? [
      [
        "script",
        {
          defer: "",
          src: umamiScriptUrl,
          "data-website-id": umamiWebsiteId,
          "data-domains": umamiDomains,
          "data-do-not-track": "true",
          "data-exclude-search": "true",
          "data-exclude-hash": "true",
        },
      ],
    ]
  : [];

function pageRoute(page: string): string {
  return page.replace(/(^|\/)index\.md$/, "$1").replace(/\.md$/, "").replace(/\/$/, "");
}

function pageUrl(page: string): string {
  return new URL(pageRoute(page), siteUrl).href;
}

function descriptionFor(page: string, fallback?: string): string {
  return pageDescriptions[pageRoute(page)] ?? fallback ?? siteDescription;
}

function titleFor(pageTitle?: string): string {
  return pageTitle && pageTitle !== siteTitle ? `${pageTitle} | ${siteTitle}` : siteTitle;
}

export default defineConfig({
  title: siteTitle,
  description: siteDescription,
  base,
  cleanUrls: true,
  lastUpdated: true,
  metaChunk: true,
  sitemap: {
    hostname: siteUrl,
  },
  head: [
    ["link", { rel: "icon", type: "image/svg+xml", href: `${base}favicon.svg` }],
    ["meta", { name: "theme-color", content: "#0f172a" }],
    ...analyticsHead,
  ],
  transformPageData(pageData) {
    pageData.description = descriptionFor(pageData.relativePath, pageData.description);
  },
  transformHead({ page, pageData, description }) {
    const url = pageUrl(page);
    const pageDescription = descriptionFor(page, description);
    const pageTitle = titleFor(pageData.title);
    return [
      ["link", { rel: "canonical", href: url }],
      ["meta", { name: "description", content: pageDescription }],
      ["meta", { property: "og:type", content: "website" }],
      ["meta", { property: "og:site_name", content: siteTitle }],
      ["meta", { property: "og:title", content: pageTitle }],
      ["meta", { property: "og:description", content: pageDescription }],
      ["meta", { property: "og:url", content: url }],
      ["meta", { property: "og:image", content: socialImage }],
      ["meta", { property: "og:image:alt", content: "Nightward release page preview with install command and local-first security posture." }],
      ["meta", { name: "twitter:card", content: "summary_large_image" }],
      ["meta", { name: "twitter:title", content: pageTitle }],
      ["meta", { name: "twitter:description", content: pageDescription }],
      ["meta", { name: "twitter:image", content: socialImage }],
    ];
  },
  buildEnd(siteConfig) {
    writeFileSync(
      join(siteConfig.outDir, "robots.txt"),
      `User-agent: *\nAllow: /\n\nSitemap: ${new URL("sitemap.xml", siteUrl).href}\n`,
    );
  },
  themeConfig: {
    logo: "/logo.svg",
    search: {
      provider: "local",
    },
    socialLinks: [
      { icon: "github", link: "https://github.com/JSONbored/nightward" },
    ],
    editLink: {
      pattern: "https://github.com/JSONbored/nightward/edit/main/site/:path",
      text: "Edit this page on GitHub",
    },
    nav: [
      { text: "Start", link: "/guide/what-is-nightward" },
      { text: "Use Nightward", link: "/guide/getting-started" },
      { text: "Integrations", link: "/integrations/mcp-server" },
      { text: "Trust & Security", link: "/security/threat-model" },
      { text: "Reference", link: "/reference/cli" },
      { text: "Contribute", link: "/contribute/adapters-and-rules" },
      { text: "GitHub", link: "https://github.com/JSONbored/nightward" },
    ],
    sidebar: [
      {
        text: "Start",
        collapsed: false,
        items: [
          { text: "What is Nightward?", link: "/guide/what-is-nightward" },
          { text: "Getting started", link: "/guide/getting-started" },
          { text: "Install", link: "/guide/install" },
          {
            text: "Quick paths",
            collapsed: false,
            items: [
              { text: "Before syncing dotfiles", link: "/start/before-syncing-dotfiles" },
              { text: "Audit an MCP workstation", link: "/start/audit-mcp-workstation" },
              { text: "Run in CI", link: "/start/run-in-ci" },
            ],
          },
        ],
      },
      {
        text: "Use Nightward",
        collapsed: false,
        items: [
          { text: "TUI", link: "/guide/tui" },
          { text: "CLI", link: "/guide/cli" },
          { text: "MCP security", link: "/guide/mcp-security" },
          { text: "Remediation", link: "/guide/remediation" },
          { text: "Provider execution", link: "/use/provider-execution" },
          { text: "Report history", link: "/use/report-history" },
          { text: "Policy and SARIF", link: "/guide/policy-and-sarif" },
          { text: "Privacy model", link: "/guide/privacy-model" },
          { text: "Troubleshooting", link: "/use/troubleshooting" },
        ],
      },
      {
        text: "Integrations",
        collapsed: false,
        items: [
          { text: "MCP server", link: "/integrations/mcp-server" },
          { text: "Raycast", link: "/integrations/raycast" },
          { text: "GitHub Action", link: "/integrations/github-action" },
          { text: "Trunk", link: "/integrations/trunk" },
        ],
      },
      {
        text: "Trust & Security",
        collapsed: false,
        items: [
          { text: "Threat model", link: "/security/threat-model" },
          { text: "Release verification", link: "/security/release-verification" },
        ],
      },
      {
        text: "Reference",
        collapsed: true,
        items: [
          {
            text: "Commands and config",
            items: [
              { text: "CLI", link: "/reference/cli" },
              { text: "Config", link: "/reference/config" },
              { text: "JSON output", link: "/reference/json-output" },
            ],
          },
          {
            text: "Rules and coverage",
            items: [
              { text: "Rules", link: "/reference/rules" },
              { text: "Providers", link: "/reference/providers" },
              { text: "Support matrix", link: "/reference/support-matrix" },
              { text: "Output surfaces", link: "/reference/output-surfaces" },
            ],
          },
          { text: "Distribution", link: "/reference/distribution" },
        ],
      },
      {
        text: "Contribute",
        collapsed: true,
        items: [
          { text: "Adapters and rules", link: "/contribute/adapters-and-rules" },
          { text: "Docs maintenance", link: "/contribute/docs-maintenance" },
        ],
      },
      { text: "Roadmap", link: "/roadmap" },
    ],
    footer: {
      message: "Local-first. No telemetry. No default network calls. No live config mutation.",
      copyright: "Released under the MIT License.",
    },
  },
});
