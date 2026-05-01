import { defineConfig } from "vitepress";

const base = process.env.NIGHTWARD_SITE_BASE ?? "/nightward/";
const siteUrl = "https://jsonbored.github.io/nightward/";
const siteTitle = "Nightward";
const siteDescription =
  "Find AI-tool risks before you sync: scan agent configs, MCP servers, and dotfiles for secrets, broad local access, and machine-only state.";
const socialImage = `${siteUrl}og-image.png`;

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
    ["link", { rel: "canonical", href: siteUrl }],
    ["meta", { name: "theme-color", content: "#0f172a" }],
    ["meta", { name: "description", content: siteDescription }],
    ["meta", { property: "og:type", content: "website" }],
    ["meta", { property: "og:site_name", content: siteTitle }],
    ["meta", { property: "og:title", content: siteTitle }],
    ["meta", { property: "og:description", content: siteDescription }],
    ["meta", { property: "og:url", content: siteUrl }],
    ["meta", { property: "og:image", content: socialImage }],
    ["meta", { property: "og:image:alt", content: "Nightward release page preview with install command and local-first security posture." }],
    ["meta", { name: "twitter:card", content: "summary_large_image" }],
    ["meta", { name: "twitter:title", content: siteTitle }],
    ["meta", { name: "twitter:description", content: siteDescription }],
    ["meta", { name: "twitter:image", content: socialImage }],
  ],
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
      { text: "Start", link: "/start/before-syncing-dotfiles" },
      { text: "Use Nightward", link: "/guide/getting-started" },
      { text: "Integrations", link: "/integrations/github-action" },
      { text: "Trust & Security", link: "/security/threat-model" },
      { text: "Reference", link: "/reference/cli" },
      { text: "Contribute", link: "/contribute/adapters-and-rules" },
      { text: "GitHub", link: "https://github.com/JSONbored/nightward" },
    ],
    sidebar: [
      {
        text: "Start",
        items: [
          { text: "What is Nightward?", link: "/guide/what-is-nightward" },
          { text: "Before syncing dotfiles", link: "/start/before-syncing-dotfiles" },
          { text: "Audit an MCP workstation", link: "/start/audit-mcp-workstation" },
          { text: "Run in CI", link: "/start/run-in-ci" },
          { text: "Getting started", link: "/guide/getting-started" },
          { text: "Install", link: "/guide/install" },
        ],
      },
      {
        text: "Use Nightward",
        items: [
          { text: "Privacy model", link: "/guide/privacy-model" },
          { text: "TUI", link: "/guide/tui" },
          { text: "CLI", link: "/guide/cli" },
          { text: "MCP security", link: "/guide/mcp-security" },
          { text: "Remediation", link: "/guide/remediation" },
          { text: "Policy and SARIF", link: "/guide/policy-and-sarif" },
          { text: "Provider execution", link: "/use/provider-execution" },
          { text: "Report history", link: "/use/report-history" },
          { text: "Troubleshooting", link: "/use/troubleshooting" },
        ],
      },
      {
        text: "Integrations",
        items: [
          { text: "GitHub Action", link: "/integrations/github-action" },
          { text: "Trunk", link: "/integrations/trunk" },
          { text: "Raycast", link: "/integrations/raycast" },
        ],
      },
      {
        text: "Reference",
        items: [
          { text: "CLI", link: "/reference/cli" },
          { text: "Config", link: "/reference/config" },
          { text: "Providers", link: "/reference/providers" },
          { text: "JSON output", link: "/reference/json-output" },
          { text: "Rules", link: "/reference/rules" },
          { text: "Support matrix", link: "/reference/support-matrix" },
          { text: "Output surfaces", link: "/reference/output-surfaces" },
          { text: "Distribution", link: "/reference/distribution" },
        ],
      },
      {
        text: "Trust & Security",
        items: [
          { text: "Threat model", link: "/security/threat-model" },
          { text: "Release verification", link: "/security/release-verification" },
        ],
      },
      {
        text: "Contribute",
        items: [
          { text: "Adapters and rules", link: "/contribute/adapters-and-rules" },
        ],
      },
      { text: "Roadmap", items: [{ text: "Roadmap", link: "/roadmap" }] },
    ],
    footer: {
      message: "Local-first. No telemetry. No default network calls. No live config mutation.",
      copyright: "Released under the MIT License.",
    },
  },
});
