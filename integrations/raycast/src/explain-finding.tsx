import {
  Action,
  ActionPanel,
  Detail,
  Icon,
  getPreferenceValues,
} from "@raycast/api";
import { usePromise } from "@raycast/utils";
import { findingMarkdown } from "./format";
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
            <Detail.Metadata.Label title="Severity" text={data.severity} />
            <Detail.Metadata.Label title="Rule" text={data.rule} />
            <Detail.Metadata.Label title="Path" text={data.path} />
          </Detail.Metadata>
        ) : undefined
      }
      actions={
        data ? (
          <ActionPanel>
            <Action.CopyToClipboard title="Copy Finding ID" content={data.id} />
            <Action.CopyToClipboard
              title="Copy Recommended Action"
              content={data.fix_steps?.[0] ?? data.recommended_action}
            />
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
          </ActionPanel>
        ) : undefined
      }
    />
  );
}
