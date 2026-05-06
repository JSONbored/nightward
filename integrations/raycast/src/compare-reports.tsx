import {
  Action,
  ActionPanel,
  Detail,
  Icon,
  getPreferenceValues,
} from "@raycast/api";
import { usePromise } from "@raycast/utils";
import {
  latestReportPair,
  normalizePreferences,
  reportHistory,
  reportsDir,
  reportsDirExists,
} from "./nightward";
import { ReportCompareDetail } from "./report-compare";

export default function Command() {
  const runtime = normalizePreferences(getPreferenceValues());
  const { data, error, isLoading, revalidate } = usePromise(async () => {
    const history = await reportHistory(runtime);
    return latestReportPair(history);
  });

  if (error) {
    const reportDir = reportsDir(runtime.homeOverride);
    const canOpenReports = reportsDirExists(runtime.homeOverride);
    return (
      <Detail
        markdown={`# Compare Nightward Reports\n\n${error.message}`}
        actions={
          <ActionPanel>
            <Action
              title="Refresh"
              icon={Icon.ArrowClockwise}
              onAction={revalidate}
            />
            {canOpenReports ? (
              <Action.ShowInFinder
                title="Open Report Folder"
                path={reportDir}
              />
            ) : null}
            <Action.CopyToClipboard
              title="Copy Reports Path"
              content={reportDir}
            />
          </ActionPanel>
        }
      />
    );
  }

  if (!data) {
    return (
      <Detail isLoading={isLoading} markdown="# Compare Nightward Reports" />
    );
  }

  return (
    <ReportCompareDetail runtime={runtime} base={data.base} head={data.head} />
  );
}
