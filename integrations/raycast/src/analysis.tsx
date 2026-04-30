import {
  Action,
  ActionPanel,
  Color,
  Detail,
  Icon,
  List,
  getPreferenceValues,
} from "@raycast/api";
import { usePromise } from "@raycast/utils";
import {
  analysisMarkdown,
  severityColor,
  signalMarkdown,
  sortedSignals,
} from "./format";
import { analysisReport, normalizePreferences } from "./nightward";
import type { AnalysisReport, AnalysisSignal } from "./types";

const docsUrl =
  "https://github.com/JSONbored/nightward/blob/main/docs/privacy-model.md";

export default function Command() {
  const runtime = normalizePreferences(getPreferenceValues());
  const { data, error, isLoading, revalidate } = usePromise(() =>
    analysisReport(runtime),
  );

  if (error) {
    return (
      <Detail
        markdown={`# Nightward Analysis\n\n${error.message}`}
        actions={<AnalysisActions onRefresh={revalidate} />}
      />
    );
  }

  if (!data) {
    return (
      <List
        isLoading={isLoading}
        searchBarPlaceholder="Loading Nightward analysis..."
      />
    );
  }

  return (
    <List
      isLoading={isLoading}
      searchBarPlaceholder="Search analysis signals..."
      isShowingDetail
    >
      <List.Section title="Summary">
        <List.Item
          title="Analysis Summary"
          subtitle={`${data.summary.total_signals} signals - ${data.summary.provider_warnings} provider warnings`}
          icon={{
            source: Icon.MagnifyingGlass,
            tintColor: severityColor(data.summary.highest_severity || "info"),
          }}
          detail={<AnalysisDetail report={data} />}
          actions={<AnalysisActions onRefresh={revalidate} />}
        />
      </List.Section>
      <List.Section title="Signals">
        {sortedSignals(data.signals).map((signal) => (
          <SignalItem key={signal.id} signal={signal} onRefresh={revalidate} />
        ))}
      </List.Section>
    </List>
  );
}

function SignalItem({
  signal,
  onRefresh,
}: {
  signal: AnalysisSignal;
  onRefresh: () => void;
}) {
  return (
    <List.Item
      title={signal.rule}
      subtitle={signal.message}
      icon={{ source: Icon.Warning, tintColor: severityColor(signal.severity) }}
      accessories={[
        { text: signal.severity },
        { text: signal.confidence },
        { text: signal.provider },
      ]}
      detail={<List.Item.Detail markdown={signalMarkdown(signal)} />}
      actions={
        <ActionPanel>
          <Action.CopyToClipboard title="Copy Signal ID" content={signal.id} />
          <Action.CopyToClipboard
            title="Copy Path"
            content={signal.path ?? ""}
            shortcut={{ modifiers: ["cmd"], key: "." }}
          />
          <Action.OpenInBrowser title="Open Privacy Model" url={docsUrl} />
          <Action
            title="Refresh"
            icon={Icon.ArrowClockwise}
            onAction={onRefresh}
          />
        </ActionPanel>
      }
    />
  );
}

function AnalysisDetail({ report }: { report: AnalysisReport }) {
  return (
    <List.Item.Detail
      markdown={analysisMarkdown(report)}
      metadata={
        <List.Item.Detail.Metadata>
          <List.Item.Detail.Metadata.Label
            title="Signals"
            text={String(report.summary.total_signals)}
          />
          <List.Item.Detail.Metadata.Label
            title="Subjects"
            text={String(report.summary.total_subjects)}
          />
          <List.Item.Detail.Metadata.Label
            title="Provider Warnings"
            text={String(report.summary.provider_warnings)}
          />
          <List.Item.Detail.Metadata.TagList title="Mode">
            <List.Item.Detail.Metadata.TagList.Item
              text={report.mode}
              color={Color.Blue}
            />
          </List.Item.Detail.Metadata.TagList>
        </List.Item.Detail.Metadata>
      }
    />
  );
}

function AnalysisActions({ onRefresh }: { onRefresh: () => void }) {
  return (
    <ActionPanel>
      <Action title="Refresh" icon={Icon.ArrowClockwise} onAction={onRefresh} />
      <Action.OpenInBrowser title="Open Privacy Model" url={docsUrl} />
    </ActionPanel>
  );
}
