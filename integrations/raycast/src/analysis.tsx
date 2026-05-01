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
  signalSubtitle,
  signalTitle,
  sortedSignals,
} from "./format";
import { analysisReport, normalizePreferences } from "./nightward";
import {
  selectedAnalysisProviders,
  selectedOnlineProviders,
} from "./provider-options";
import { readSelectedProviders } from "./provider-settings";
import type { AnalysisReport, AnalysisSignal } from "./types";

const docsUrl =
  "https://github.com/JSONbored/nightward/blob/main/docs/privacy-model.md";

export default function Command() {
  const runtime = normalizePreferences(getPreferenceValues());
  const { data, error, isLoading, revalidate } = usePromise(async () => {
    const selectedProviders = await readSelectedProviders();
    const activeProviders = selectedAnalysisProviders(
      selectedProviders,
      runtime.allowOnlineProviders,
    );
    const blockedProviders = runtime.allowOnlineProviders
      ? []
      : selectedOnlineProviders(selectedProviders);
    const report = await analysisReport(runtime, selectedProviders);
    return { report, selectedProviders, activeProviders, blockedProviders };
  });

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
          subtitle={`${data.report.summary.total_signals} signals - ${data.report.summary.provider_warnings} provider warnings`}
          icon={{
            source: Icon.MagnifyingGlass,
            tintColor: severityColor(
              data.report.summary.highest_severity || "info",
            ),
          }}
          detail={
            <AnalysisDetail
              report={data.report}
              activeProviders={data.activeProviders}
              blockedProviders={data.blockedProviders}
            />
          }
          actions={<AnalysisActions onRefresh={revalidate} />}
        />
      </List.Section>
      <List.Section title="Signals">
        {sortedSignals(data.report.signals).map((signal) => (
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
      title={signalTitle(signal)}
      subtitle={signalSubtitle(signal)}
      icon={{ source: Icon.Warning, tintColor: severityColor(signal.severity) }}
      accessories={[
        {
          tag: {
            value: signal.severity,
            color: severityColor(signal.severity),
          },
        },
        { text: signal.confidence },
      ]}
      detail={
        <List.Item.Detail
          markdown={signalMarkdown(signal)}
          metadata={
            <List.Item.Detail.Metadata>
              <List.Item.Detail.Metadata.Label
                title="Provider"
                text={signal.provider}
              />
              <List.Item.Detail.Metadata.Label
                title="Category"
                text={signal.category}
              />
              <List.Item.Detail.Metadata.Label
                title="Severity"
                text={signal.severity}
              />
              <List.Item.Detail.Metadata.Label
                title="Confidence"
                text={signal.confidence}
              />
              {signal.path ? (
                <List.Item.Detail.Metadata.Label
                  title="Path"
                  text={signal.path}
                />
              ) : null}
            </List.Item.Detail.Metadata>
          }
        />
      }
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

function AnalysisDetail({
  report,
  activeProviders,
  blockedProviders,
}: {
  report: AnalysisReport;
  activeProviders: string[];
  blockedProviders: string[];
}) {
  return (
    <List.Item.Detail
      markdown={[
        analysisMarkdown(report),
        "",
        "## Raycast Provider Selection",
        activeProviders.length > 0
          ? `Active providers: \`${activeProviders.join(", ")}\``
          : "Active providers: built-in offline analysis only.",
        blockedProviders.length > 0
          ? `Blocked online-capable providers: \`${blockedProviders.join(", ")}\``
          : "",
      ]
        .filter(Boolean)
        .join("\n")}
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
          <List.Item.Detail.Metadata.Label
            title="Selected Providers"
            text={
              activeProviders.length > 0
                ? activeProviders.join(", ")
                : "built-in"
            }
          />
          {blockedProviders.length > 0 ? (
            <List.Item.Detail.Metadata.Label
              title="Blocked Online Providers"
              text={blockedProviders.join(", ")}
            />
          ) : null}
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
