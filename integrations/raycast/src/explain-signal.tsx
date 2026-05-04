import {
  Action,
  ActionPanel,
  Detail,
  Icon,
  getPreferenceValues,
} from "@raycast/api";
import { usePromise } from "@raycast/utils";
import { severityColor, signalMarkdown } from "./format";
import { explainSignal, normalizePreferences } from "./nightward";

type Arguments = {
  findingId: string;
};

export default function Command(props: { arguments: Arguments }) {
  const runtime = normalizePreferences(getPreferenceValues());
  const findingId = props.arguments.findingId.trim();
  const { data, error, isLoading, revalidate } = usePromise(
    (id: string) => explainSignal(id, runtime),
    [findingId],
  );
  const signal = data?.signals[0];

  if (error) {
    return <Detail markdown={`# Nightward Signal\n\n${error.message}`} />;
  }
  return (
    <Detail
      isLoading={isLoading}
      markdown={
        signal
          ? signalMarkdown(signal)
          : "# Nightward Signal\n\nNo analysis signal matched that finding."
      }
      metadata={
        signal ? (
          <Detail.Metadata>
            <Detail.Metadata.Label title="ID" text={signal.id} />
            <Detail.Metadata.TagList title="Severity">
              <Detail.Metadata.TagList.Item
                text={signal.severity}
                color={severityColor(signal.severity)}
              />
            </Detail.Metadata.TagList>
            <Detail.Metadata.Separator />
            <Detail.Metadata.Label title="Provider" text={signal.provider} />
            <Detail.Metadata.Label title="Category" text={signal.category} />
            <Detail.Metadata.Label
              title="Confidence"
              text={signal.confidence}
            />
            <Detail.Metadata.Label title="Rule" text={signal.rule} />
            {signal.path ? (
              <Detail.Metadata.Label title="Path" text={signal.path} />
            ) : null}
          </Detail.Metadata>
        ) : undefined
      }
      actions={
        <ActionPanel>
          <ActionPanel.Section title="Copy">
            {signal ? (
              <>
                <Action.CopyToClipboard
                  title="Copy Signal ID"
                  content={signal.id}
                />
                {signal.path ? (
                  <Action.CopyToClipboard
                    title="Copy Path"
                    content={signal.path}
                  />
                ) : null}
              </>
            ) : null}
          </ActionPanel.Section>
          <ActionPanel.Section title="Refresh">
            <Action
              title="Refresh"
              icon={Icon.ArrowClockwise}
              onAction={revalidate}
            />
          </ActionPanel.Section>
        </ActionPanel>
      }
    />
  );
}
