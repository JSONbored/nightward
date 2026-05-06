import { Action, ActionPanel, Detail, Icon } from "@raycast/api";
import { usePromise } from "@raycast/utils";
import {
  basename,
  reportDiffMarkdown,
  reportDiffSubtitle,
  severityColor,
} from "./format";
import { reportDiff, type RuntimeOptions } from "./nightward";
import type { ReportRecord } from "./types";

export function ReportCompareDetail({
  runtime,
  base,
  head,
}: {
  runtime: RuntimeOptions;
  base: ReportRecord;
  head: ReportRecord;
}) {
  const { data, error, isLoading, revalidate } = usePromise(() =>
    reportDiff(runtime, base.path, head.path),
  );
  if (error) {
    return (
      <Detail
        markdown={`# Report Compare\n\n${error.message}`}
        actions={
          <ReportCompareActions
            base={base}
            head={head}
            onRefresh={revalidate}
          />
        }
      />
    );
  }
  if (!data) {
    return <Detail isLoading={isLoading} markdown="# Report Compare" />;
  }
  const markdown = reportDiffMarkdown(data);
  return (
    <Detail
      isLoading={isLoading}
      markdown={markdown}
      metadata={
        <Detail.Metadata>
          <Detail.Metadata.TagList title="Change">
            <Detail.Metadata.TagList.Item
              text={reportDiffSubtitle(data)}
              color={severityColor(data.summary.max_added_severity)}
            />
          </Detail.Metadata.TagList>
          <Detail.Metadata.Separator />
          <Detail.Metadata.Label
            title="Added"
            text={String(data.summary.added)}
          />
          <Detail.Metadata.Label
            title="Removed"
            text={String(data.summary.removed)}
          />
          <Detail.Metadata.Label
            title="Changed"
            text={String(data.summary.changed)}
          />
          <Detail.Metadata.Label
            title="Max Added"
            text={data.summary.max_added_severity}
          />
          <Detail.Metadata.Separator />
          <Detail.Metadata.Label title="Base" text={basename(base.path)} />
          <Detail.Metadata.Label title="Head" text={basename(head.path)} />
        </Detail.Metadata>
      }
      actions={
        <ReportCompareActions
          base={base}
          head={head}
          markdown={markdown}
          onRefresh={revalidate}
        />
      }
    />
  );
}

function ReportCompareActions({
  base,
  head,
  markdown,
  onRefresh,
}: {
  base: ReportRecord;
  head: ReportRecord;
  markdown?: string;
  onRefresh: () => void;
}) {
  return (
    <ActionPanel>
      {markdown ? (
        <Action.CopyToClipboard
          title="Copy Compare Markdown"
          content={markdown}
        />
      ) : null}
      <Action title="Refresh" icon={Icon.ArrowClockwise} onAction={onRefresh} />
      <Action.ShowInFinder title="Show Latest Report" path={head.path} />
      <Action.ShowInFinder title="Show Previous Report" path={base.path} />
    </ActionPanel>
  );
}
