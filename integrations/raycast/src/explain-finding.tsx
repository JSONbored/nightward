import {
  Action,
  ActionPanel,
  Color,
  Detail,
  Icon,
  getPreferenceValues,
} from "@raycast/api";
import { usePromise } from "@raycast/utils";
import {
  findingFixLabel,
  findingMarkdown,
  redactText,
  severityColor,
} from "./format";
import { explainFinding, normalizePreferences } from "./nightward";

type Arguments = {
  findingId: string;
};

export default function Command(props: { arguments: Arguments }) {
  const runtime = normalizePreferences(getPreferenceValues());
  const findingId = props.arguments.findingId.trim();
  const { data, error, isLoading, revalidate } = usePromise(
    (id: string) => explainFinding(id, runtime),
    [findingId],
  );

  if (error) {
    return <Detail markdown={`# Nightward Finding\n\n${error.message}`} />;
  }

  return (
    <Detail
      isLoading={isLoading}
      markdown={
        data ? findingMarkdown(data) : "# Nightward Finding\n\nLoading..."
      }
      metadata={
        data ? (
          <Detail.Metadata>
            <Detail.Metadata.Label title="ID" text={data.id} />
            <Detail.Metadata.Label title="Tool" text={data.tool} />
            <Detail.Metadata.TagList title="Severity">
              <Detail.Metadata.TagList.Item
                text={data.severity}
                color={severityColor(data.severity)}
              />
            </Detail.Metadata.TagList>
            <Detail.Metadata.TagList title="Fix">
              <Detail.Metadata.TagList.Item
                text={findingFixLabel(data)}
                color={data.requires_review ? Color.Yellow : Color.Green}
              />
            </Detail.Metadata.TagList>
            <Detail.Metadata.Separator />
            <Detail.Metadata.Label title="Rule" text={data.rule} />
            {data.server ? (
              <Detail.Metadata.Label title="Server" text={data.server} />
            ) : null}
            <Detail.Metadata.Label title="Path" text={data.path} />
            {data.confidence ? (
              <Detail.Metadata.Label
                title="Confidence"
                text={data.confidence}
              />
            ) : null}
            <Detail.Metadata.Label
              title="Requires Review"
              text={data.requires_review ? "yes" : "no"}
            />
          </Detail.Metadata>
        ) : undefined
      }
      actions={
        data ? (
          <ActionPanel>
            <ActionPanel.Section title="Copy">
              <Action.CopyToClipboard
                title="Copy Finding ID"
                content={data.id}
              />
              <Action.CopyToClipboard
                title="Copy Recommended Action"
                content={redactText(
                  data.fix_steps?.[0] ?? data.recommended_action,
                )}
              />
              <Action.CopyToClipboard title="Copy Path" content={data.path} />
            </ActionPanel.Section>
            <ActionPanel.Section title="Open">
              {data.docs_url ? (
                <Action.OpenInBrowser
                  title="Open Finding Docs"
                  url={data.docs_url}
                />
              ) : null}
              <Action
                title="Refresh"
                icon={Icon.ArrowClockwise}
                onAction={revalidate}
              />
            </ActionPanel.Section>
          </ActionPanel>
        ) : undefined
      }
    />
  );
}
