import { Toast, getPreferenceValues, open, showToast } from "@raycast/api";
import {
  normalizePreferences,
  reportsDir,
  reportsDirExists,
} from "./nightward";

export default async function Command() {
  const runtime = normalizePreferences(getPreferenceValues());
  const dir = reportsDir(runtime.homeOverride);
  if (!reportsDirExists(runtime.homeOverride)) {
    await showToast({
      style: Toast.Style.Failure,
      title: "Report folder does not exist yet",
      message: dir,
    });
    return;
  }

  await open(dir);
}
