import {
  Clipboard,
  Toast,
  getPreferenceValues,
  showHUD,
  showToast,
} from "@raycast/api";
import { exportFixPlanMarkdown, normalizePreferences } from "./nightward";

export default async function Command() {
  const runtime = normalizePreferences(getPreferenceValues());
  try {
    const markdown = await exportFixPlanMarkdown(runtime);
    await Clipboard.copy(markdown);
    await showHUD("Copied redacted Nightward fix plan");
  } catch (error) {
    await showToast({
      style: Toast.Style.Failure,
      title: "Could not export fix plan",
      message:
        error instanceof Error ? error.message : "Unknown Nightward error",
    });
  }
}
