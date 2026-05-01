import { Clipboard, Toast, getPreferenceValues, showToast } from "@raycast/api";
import { exportAnalysisMarkdown, normalizePreferences } from "./nightward";
import { readSelectedProviders } from "./provider-settings";

export default async function Command() {
  const runtime = normalizePreferences(getPreferenceValues());
  try {
    const selectedProviders = await readSelectedProviders();
    const markdown = await exportAnalysisMarkdown(runtime, selectedProviders);
    await Clipboard.copy(markdown);
    await showToast({
      style: Toast.Style.Success,
      title: "Copied Nightward analysis",
      message: "Redacted analysis report copied to clipboard.",
    });
  } catch (error) {
    await showToast({
      style: Toast.Style.Failure,
      title: "Analysis export failed",
      message: error instanceof Error ? error.message : "Unknown error",
    });
  }
}
