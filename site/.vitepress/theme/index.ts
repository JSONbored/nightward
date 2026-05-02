import DefaultTheme from "vitepress/theme";
import { h } from "vue";
import McpClientTabs from "./components/McpClientTabs.vue";
import ReleasePill from "./components/ReleasePill.vue";
import "./custom.css";

export default {
  extends: DefaultTheme,
  enhanceApp({ app }) {
    app.component("McpClientTabs", McpClientTabs);
  },
  Layout: () => {
    return h(DefaultTheme.Layout, null, {
      "nav-bar-content-after": () => h(ReleasePill),
      "layout-bottom": () => h(ReleasePill, { placement: "footer" }),
    });
  },
};
