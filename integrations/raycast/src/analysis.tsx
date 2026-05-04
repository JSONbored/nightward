import {
  Action,
  ActionPanel,
  Color,
  Detail,
  Icon,
  LaunchType,
  List,
  getPreferenceValues,
  launchCommand,
  showToast,
  Toast,
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
import {
  clearSelectedProviders,
  readSelectedProviders,
} from "./provider-settings";
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
        markdown={[
          "# Nightward Analysis",
          "",
          "Nightward could not run the selected analysis providers.",
          "",
          "Open Provider Doctor to install missing tools, or clear the Raycast provider selection to return to built-in offline analysis.",
          "",
          "## Error",
          `\`${error.message}\``,
        ].join("\n")}
        actions={<AnalysisErrorActions onRefresh={revalidate} />}
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
      searchBarPlaceholder="Search signals..."
      isShowingDetail
    >
      <List.Section title="Summary">
        <List.Item
          title="Analysis"
          subtitle={`${data.report.summary.total_signals} signals`}
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
      {signalSections(data.report.signals).length === 0 ? (
        <List.EmptyView
          title="No analysis signals"
          description="Nightward did not emit any analysis signals for this scan."
          icon={Icon.CheckCircle}
        />
      ) : null}
      {signalSections(data.report.signals).map(([severity, signals]) => (
        <List.Section
          key={severity}
          title={`${severity.charAt(0).toUpperCase() + severity.slice(1)} Signals`}
          subtitle={String(signals.length)}
        >
          {signals.map((signal) => (
            <SignalItem
              key={signal.id}
              signal={signal}
              onRefresh={revalidate}
            />
          ))}
        </List.Section>
      ))}
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
      detail={
        <List.Item.Detail
          markdown={signalMarkdown(signal)}
          metadata={
            <List.Item.Detail.Metadata>
              <List.Item.Detail.Metadata.TagList title="Severity">
                <List.Item.Detail.Metadata.TagList.Item
                  text={signal.severity}
                  color={severityColor(signal.severity)}
                />
              </List.Item.Detail.Metadata.TagList>
              <List.Item.Detail.Metadata.Separator />
              <List.Item.Detail.Metadata.Label
                title="Provider"
                text={signal.provider}
              />
              <List.Item.Detail.Metadata.Label
                title="Category"
                text={signal.category}
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
        blockedProviders.length > 0
          ? [
              "",
              "## Online Providers Blocked",
              "Enable online providers in extension preferences before running selected online-capable providers.",
            ].join("\n")
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
          <List.Item.Detail.Metadata.Separator />
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

function signalSections(
  signals: AnalysisSignal[],
): Array<[AnalysisSignal["severity"], AnalysisSignal[]]> {
  return (["critical", "high", "medium", "low", "info"] as const)
    .map((severity): [AnalysisSignal["severity"], AnalysisSignal[]] => [
      severity,
      sortedSignals(signals).filter((signal) => signal.severity === severity),
    ])
    .filter(([, sectionSignals]) => sectionSignals.length > 0);
}

function AnalysisActions({ onRefresh }: { onRefresh: () => void }) {
  return (
    <ActionPanel>
      <Action title="Refresh" icon={Icon.ArrowClockwise} onAction={onRefresh} />
      <Action.OpenInBrowser title="Open Privacy Model" url={docsUrl} />
    </ActionPanel>
  );
}

function AnalysisErrorActions({ onRefresh }: { onRefresh: () => void }) {
  return (
    <ActionPanel>
      <ActionPanel.Section title="Recover">
        <Action
          title="Open Provider Doctor"
          icon={Icon.Heartbeat}
          onAction={() => void openProviderDoctor()}
        />
        <Action
          title="Clear Selected Providers"
          icon={Icon.Trash}
          onAction={() => void clearProviders(onRefresh)}
        />
      </ActionPanel.Section>
      <ActionPanel.Section title="Refresh">
        <Action
          title="Refresh"
          icon={Icon.ArrowClockwise}
          onAction={onRefresh}
        />
        <Action.OpenInBrowser title="Open Privacy Model" url={docsUrl} />
      </ActionPanel.Section>
    </ActionPanel>
  );
}

async function openProviderDoctor() {
  try {
    await launchCommand({
      name: "provider-doctor",
      type: LaunchType.UserInitiated,
    });
  } catch (error) {
    await showToast({
      style: Toast.Style.Failure,
      title: "Could not open Provider Doctor",
      message: error instanceof Error ? error.message : "Unknown Raycast error",
    });
  }
}

async function clearProviders(onRefresh: () => void) {
  await clearSelectedProviders();
  await showToast({
    style: Toast.Style.Success,
    title: "Provider selection cleared",
    message: "Nightward Analysis will use built-in offline analysis.",
  });
  onRefresh();
}
