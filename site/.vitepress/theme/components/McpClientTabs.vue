<script setup lang="ts">
import { computed, ref } from "vue";
import { siClaude, siCursor, siGoogle, siWindsurf } from "simple-icons";

type Client = {
  id: string;
  name: string;
  icon?: { path: string; hex: string; title: string };
  badge?: string;
  file: string;
  note: string;
  language: string;
  config: string;
};

const clients: Client[] = [
  {
    id: "claude",
    name: "Claude",
    icon: siClaude,
    file: "Claude Code: claude mcp add, or Claude Desktop: claude_desktop_config.json",
    note: "Use the CLI form for Claude Code. Use the JSON block in Claude Desktop's MCP config.",
    language: "sh",
    config:
      "claude mcp add --transport stdio --scope user nightward -- nw mcp serve\n\n# Claude Desktop JSON\n{\n  \"mcpServers\": {\n    \"nightward\": {\n      \"type\": \"stdio\",\n      \"command\": \"nw\",\n      \"args\": [\"mcp\", \"serve\"],\n      \"env\": {}\n    }\n  }\n}",
  },
  {
    id: "cursor",
    name: "Cursor",
    icon: siCursor,
    file: "~/.cursor/mcp.json",
    note: "Use global config for workstation-wide access, or .cursor/mcp.json inside one project.",
    language: "json",
    config:
      "{\n  \"mcpServers\": {\n    \"nightward\": {\n      \"command\": \"nw\",\n      \"args\": [\"mcp\", \"serve\"]\n    }\n  }\n}",
  },
  {
    id: "codex",
    name: "Codex",
    badge: "CX",
    file: "~/.codex/config.toml",
    note: "Codex reads MCP servers from the shared CLI/IDE config.",
    language: "toml",
    config:
      "[mcp_servers.nightward]\ncommand = \"nw\"\nargs = [\"mcp\", \"serve\"]",
  },
  {
    id: "antigravity",
    name: "Antigravity",
    icon: siGoogle,
    file: "Manage MCP Servers -> View raw config, usually ~/.gemini/antigravity/mcp_config.json",
    note: "Current public examples use the mcpServers JSON shape. If your Antigravity build opens a VS Code-style file, use the servers shape shown below the tabs.",
    language: "json",
    config:
      "{\n  \"mcpServers\": {\n    \"nightward\": {\n      \"command\": \"nw\",\n      \"args\": [\"mcp\", \"serve\"]\n    }\n  }\n}",
  },
  {
    id: "windsurf",
    name: "Windsurf",
    icon: siWindsurf,
    file: "~/.codeium/windsurf/mcp_config.json",
    note: "Windsurf Cascade reads mcp_config.json and lets you enable tools per MCP server.",
    language: "json",
    config:
      "{\n  \"mcpServers\": {\n    \"nightward\": {\n      \"command\": \"nw\",\n      \"args\": [\"mcp\", \"serve\"]\n    }\n  }\n}",
  },
];

const activeId = ref(clients[0].id);
const active = computed(
  () => clients.find((client) => client.id === activeId.value) ?? clients[0],
);
</script>

<template>
  <div class="nw-client-tabs">
    <div class="nw-client-tabs__buttons" role="tablist" aria-label="MCP client configuration examples">
      <button
        v-for="client in clients"
        :key="client.id"
        type="button"
        class="nw-client-tabs__button"
        :class="{ 'nw-client-tabs__button--active': client.id === active.id }"
        role="tab"
        :aria-selected="client.id === active.id"
        @click="activeId = client.id"
      >
        <svg
          v-if="client.icon"
          class="nw-client-tabs__icon"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <path :fill="`#${client.icon.hex}`" :d="client.icon.path" />
        </svg>
        <span v-else class="nw-client-tabs__badge" aria-hidden="true">{{ client.badge }}</span>
        <span>{{ client.name }}</span>
      </button>
    </div>

    <div class="nw-client-tabs__panel" role="tabpanel">
      <p class="nw-client-tabs__file"><strong>Config:</strong> <code>{{ active.file }}</code></p>
      <p>{{ active.note }}</p>
      <pre><code>{{ active.config }}</code></pre>
    </div>
  </div>
</template>
