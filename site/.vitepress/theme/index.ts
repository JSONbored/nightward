import DefaultTheme from "vitepress/theme";
import { h } from "vue";
import ReleasePill from "./components/ReleasePill.vue";
import "./custom.css";

export default {
  extends: DefaultTheme,
  Layout: () => {
    return h(DefaultTheme.Layout, null, {
      "nav-bar-content-after": () => h(ReleasePill),
      "layout-bottom": () => h(ReleasePill, { placement: "footer" }),
    });
  },
};
