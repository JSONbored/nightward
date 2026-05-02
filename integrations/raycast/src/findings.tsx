import {
  Action,
  ActionPanel,
  Clipboard,
  Color,
  Detail,
  Icon,
  List,
  getPreferenceValues,
  showHUD,
  showToast,
  Toast,
} from "@raycast/api";
import { usePromise } from "@raycast/utils";
import { useState } from "react";
import {
  findingKeywords,
  findingFixLabel,
  findingMarkdown,
  findingSubtitle,
  findingTitle,
  policyIgnoreSnippet,
  redactText,
  severityColor,
  sortedFindings,
} from "./format";
import {
  exportFixPlanMarkdown,
  findings,
  normalizePreferences,
} from "./nightward";
import type { Finding, RiskLevel } from "./types";

const docsUrl =
  "https://github.com/JSONbored/nightward/blob/main/docs/remediation.md";

export default function Command() {
  const runtime = normalizePreferences(getPreferenceValues());
  const [severity, setSeverity] = useState<RiskLevel | "">("");
  const { data, error, isLoading, revalidate } = usePromise(() =>
    findings(runtime),
  );
  const allFindings = sortedFindings(data ?? []).filter(
    (finding) => !severity || finding.severity === severity,
  );

  if (error) {
    return <Detail markdown={`# Nightward Findings\n\n${error.message}`} />;
  }

  return (
    <List
      isLoading={isLoading}
      isShowingDetail
      searchBarPlaceholder="Search findings by rule, path, tool, server, or ID..."
      searchBarAccessory={
        <SeverityDropdown value={severity} onChange={setSeverity} />
      }
      filtering
    >
      <List.Section title="Findings" subtitle={`${allFindings.length}`}>
        {allFindings.map((finding) => (
          <FindingItem
            key={finding.id}
            finding={finding}
            onRefresh={revalidate}
          />
        ))}
      </List.Section>
    </List>
  );
}

function SeverityDropdown({
  value,
  onChange,
}: {
  value: RiskLevel | "";
  onChange: (value: RiskLevel | "") => void;
}) {
  return (
    <List.Dropdown
      tooltip="Filter by severity"
      value={value}
      onChange={(next) => onChange(next as RiskLevel | "")}
    >
      <List.Dropdown.Item title="All Severities" value="" />
      <List.Dropdown.Item title="Critical" value="critical" />
      <List.Dropdown.Item title="High" value="high" />
      <List.Dropdown.Item title="Medium" value="medium" />
      <List.Dropdown.Item title="Low" value="low" />
      <List.Dropdown.Item title="Info" value="info" />
    </List.Dropdown>
  );
}

function FindingItem({
  finding,
  onRefresh,
}: {
  finding: Finding;
  onRefresh: () => void;
}) {
  return (
    <List.Item
      title={findingTitle(finding)}
      subtitle={findingSubtitle(finding)}
      keywords={findingKeywords(finding)}
      icon={{
        source: Icon.Warning,
        tintColor: severityColor(finding.severity as RiskLevel),
      }}
      accessories={[
        {
          tag: {
            value: finding.severity,
            color: severityColor(finding.severity),
          },
        },
        finding.fix_available
          ? {
              tag: {
                value: findingFixLabel(finding),
                color: finding.requires_review ? Color.Yellow : Color.Green,
              },
            }
          : { text: findingFixLabel(finding) },
      ]}
      detail={<FindingDetail finding={finding} />}
      actions={<FindingActions finding={finding} onRefresh={onRefresh} />}
    />
  );
}

function FindingDetail({ finding }: { finding: Finding }) {
  return (
    <List.Item.Detail
      markdown={findingMarkdown(finding)}
      metadata={
        <List.Item.Detail.Metadata>
          <List.Item.Detail.Metadata.Label title="ID" text={finding.id} />
          <List.Item.Detail.Metadata.Label title="Tool" text={finding.tool} />
          <List.Item.Detail.Metadata.Label
            title="Severity"
            text={finding.severity}
          />
          <List.Item.Detail.Metadata.Label title="Rule" text={finding.rule} />
          {finding.server ? (
            <List.Item.Detail.Metadata.Label
              title="Server"
              text={finding.server}
            />
          ) : null}
          <List.Item.Detail.Metadata.Label title="Path" text={finding.path} />
          {finding.docs_url ? (
            <List.Item.Detail.Metadata.Link
              title="Docs"
              target={finding.docs_url}
              text="Open"
            />
          ) : null}
        </List.Item.Detail.Metadata>
      }
    />
  );
}

function FindingActions({
  finding,
  onRefresh,
}: {
  finding: Finding;
  onRefresh: () => void;
}) {
  const runtime = normalizePreferences(getPreferenceValues());
  const firstStep = finding.fix_steps?.[0] ?? finding.recommended_action;
  return (
    <ActionPanel>
      <ActionPanel.Section title="Review">
        <Action.CopyToClipboard title="Copy Finding ID" content={finding.id} />
        <Action.CopyToClipboard
          title="Copy Recommended Action"
          content={redactText(firstStep)}
        />
        <Action.CopyToClipboard title="Copy Path" content={finding.path} />
        <Action.CopyToClipboard
          title="Copy Reviewed Policy Ignore Snippet"
          icon={Icon.CheckCircle}
          content={policyIgnoreSnippet(finding)}
        />
      </ActionPanel.Section>
      <ActionPanel.Section title="Plan-Only Remediation">
        <Action
          title="Copy Fix Plan for Finding"
          icon={Icon.Document}
          onAction={async () =>
            copyFixPlan(
              runtime,
              { findingId: finding.id },
              "Copied finding fix plan",
            )
          }
        />
        <Action
          title="Copy Fix Plan for Rule"
          icon={Icon.List}
          onAction={async () =>
            copyFixPlan(runtime, { rule: finding.rule }, "Copied rule fix plan")
          }
        />
      </ActionPanel.Section>
      <ActionPanel.Section title="Open">
        <Action.CopyToClipboard
          title="Copy Redacted Evidence"
          content={redactText(finding.evidence ?? "")}
        />
        {finding.docs_url ? (
          <Action.OpenInBrowser
            title="Open Finding Docs"
            url={finding.docs_url}
          />
        ) : (
          <Action.OpenInBrowser title="Open Remediation Docs" url={docsUrl} />
        )}
        <Action
          title="Refresh"
          icon={Icon.ArrowClockwise}
          onAction={onRefresh}
        />
      </ActionPanel.Section>
    </ActionPanel>
  );
}

async function copyFixPlan(
  runtime: ReturnType<typeof normalizePreferences>,
  selector: { findingId?: string; rule?: string },
  message: string,
) {
  try {
    await Clipboard.copy(await exportFixPlanMarkdown(runtime, selector));
    await showHUD(message);
  } catch (error) {
    await showToast({
      style: Toast.Style.Failure,
      title: "Could not export fix plan",
      message:
        error instanceof Error ? error.message : "Unknown Nightward error",
    });
  }
}
