<script setup lang="ts">
import { computed, onMounted, ref } from "vue";

const props = defineProps<{
  placement?: "nav" | "footer";
}>();

type GitHubRelease = {
  tag_name?: string;
  html_url?: string;
};

const tag = ref("Latest release");
const releaseUrl = ref("https://github.com/JSONbored/nightward/releases/latest");
const failed = ref(false);

const npmUrl = computed(() => {
  const version = tag.value.startsWith("v") ? tag.value.slice(1) : tag.value;
  if (!/^\d+\.\d+\.\d+/.test(version)) {
    return "https://www.npmjs.com/package/@jsonbored/nightward";
  }
  return `https://www.npmjs.com/package/@jsonbored/nightward/v/${version}`;
});

onMounted(async () => {
  try {
    const response = await fetch(
      "https://api.github.com/repos/JSONbored/nightward/releases/latest",
      { headers: { Accept: "application/vnd.github+json" } },
    );
    if (!response.ok) throw new Error(`GitHub returned ${response.status}`);
    const release = (await response.json()) as GitHubRelease;
    if (release.tag_name) tag.value = release.tag_name;
    if (release.html_url) releaseUrl.value = release.html_url;
  } catch {
    failed.value = true;
  }
});
</script>

<template>
  <div
    class="nw-release-pill"
    :class="{
      'nw-release-pill--footer': props.placement === 'footer',
      'nw-release-pill--stale': failed,
    }"
  >
    <a :href="releaseUrl" aria-label="Open the latest Nightward GitHub Release">
      <span>Release</span>
      <strong>{{ tag }}</strong>
    </a>
    <a :href="npmUrl" aria-label="Open the matching Nightward npm package version">
      npm
    </a>
  </div>
</template>

