import { defineConfig } from "vitepress";

const base = process.env.NIGHTWARD_SITE_BASE ?? "/nightward/";

export default defineConfig({
  title: "Nightward",
  description: "Local-first security posture for AI agent state, MCP config, and dotfiles backup safety.",
  base,
  cleanUrls: true,
  lastUpdated: true,
  metaChunk: true,
  sitemap: {
    hostname: "https://jsonbored.github.io/nightward/",
  },
  head: [
    ["meta", { name: "theme-color", content: "#0f172a" }],
    ["meta", { property: "og:type", content: "website" }],
    ["meta", { property: "og:title", content: "Nightward" }],
    ["meta", { property: "og:description", content: "Audit AI agent state before it leaks into dotfiles." }],
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
      { text: "Guide", link: "/guide/what-is-nightward" },
      { text: "Install", link: "/guide/install" },
      { text: "Integrations", link: "/integrations/github-action" },
      { text: "Security", link: "/security/threat-model" },
      { text: "GitHub", link: "https://github.com/JSONbored/nightward" },
    ],
    sidebar: [
      {
        text: "Guide",
        items: [
          { text: "What is Nightward?", link: "/guide/what-is-nightward" },
          { text: "Getting started", link: "/guide/getting-started" },
          { text: "Install", link: "/guide/install" },
          { text: "Privacy model", link: "/guide/privacy-model" },
          { text: "TUI", link: "/guide/tui" },
          { text: "CLI", link: "/guide/cli" },
          { text: "MCP security", link: "/guide/mcp-security" },
          { text: "Remediation", link: "/guide/remediation" },
          { text: "Policy and SARIF", link: "/guide/policy-and-sarif" },
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
          { text: "Config", link: "/reference/config" },
          { text: "JSON output", link: "/reference/json-output" },
          { text: "Rules", link: "/reference/rules" },
        ],
      },
      {
        text: "Security",
        items: [
          { text: "Threat model", link: "/security/threat-model" },
          { text: "Release verification", link: "/security/release-verification" },
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
