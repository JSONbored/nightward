import {
  Action,
  ActionPanel,
  Detail,
  Icon,
  getPreferenceValues,
} from "@raycast/api";
import { usePromise } from "@raycast/utils";
import { signalMarkdown } from "./format";
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
      actions={
        <ActionPanel>
          {signal ? (
            <Action.CopyToClipboard
              title="Copy Signal ID"
              content={signal.id}
            />
          ) : null}
          <Action
            title="Refresh"
            icon={Icon.ArrowClockwise}
            onAction={revalidate}
          />
        </ActionPanel>
      }
    />
  );
}
