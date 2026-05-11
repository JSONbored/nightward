import {
  Action,
  ActionPanel,
  Alert,
  Color,
  Icon,
  List,
  Toast,
  confirmAlert,
  getPreferenceValues,
  showToast,
} from "@raycast/api";
import { usePromise } from "@raycast/utils";
import {
  approveApproval,
  denyApproval,
  listApprovals,
  normalizePreferences,
} from "./nightward";
import type { NightwardApproval } from "./types";

export default function Command() {
  const runtime = normalizePreferences(getPreferenceValues());
  const { data, error, isLoading, revalidate } = usePromise(async () =>
    listApprovals(runtime),
  );

  if (error) {
    return (
      <List isLoading={false}>
        <List.EmptyView
          title="Nightward approvals failed"
          description={error.message}
          icon={Icon.ExclamationMark}
        />
      </List>
    );
  }

  const approvals = data?.approvals ?? [];
  return (
    <List
      isLoading={isLoading}
      searchBarPlaceholder="Search MCP approvals..."
      isShowingDetail
    >
      {approvals.length === 0 && !isLoading ? (
        <List.EmptyView
          title="No MCP approvals"
          description="Nightward has no pending or recent MCP action approval requests."
          icon={Icon.CheckCircle}
        />
      ) : null}
      {[
        "pending",
        "approved",
        "applied",
        "denied",
        "expired",
        "failed",
        "invalidated",
      ].map((status) => (
        <List.Section key={status} title={sectionTitle(status)}>
          {approvals
            .filter((approval) => approval.status === status)
            .map((approval) => (
              <ApprovalItem
                key={approval.approval_id}
                approval={approval}
                onRefresh={revalidate}
                runtime={runtime}
              />
            ))}
        </List.Section>
      ))}
    </List>
  );
}

function ApprovalItem({
  approval,
  onRefresh,
  runtime,
}: {
  approval: NightwardApproval;
  onRefresh: () => void;
  runtime: ReturnType<typeof normalizePreferences>;
}) {
  const action = approval.preview.action;
  return (
    <List.Item
      title={action.title}
      subtitle={approval.action_id}
      icon={{
        source: approval.status === "pending" ? Icon.Clock : Icon.CheckCircle,
        tintColor: statusColor(approval.status),
      }}
      accessories={[
        {
          text: approval.status,
          icon: {
            source: Icon.Circle,
            tintColor: statusColor(approval.status),
          },
        },
      ]}
      detail={
        <List.Item.Detail
          markdown={approvalMarkdown(approval)}
          metadata={
            <List.Item.Detail.Metadata>
              <List.Item.Detail.Metadata.TagList title="Status">
                <List.Item.Detail.Metadata.TagList.Item
                  text={approval.status}
                  color={statusColor(approval.status)}
                />
                <List.Item.Detail.Metadata.TagList.Item
                  text={action.risk}
                  color={riskColor(action.risk)}
                />
              </List.Item.Detail.Metadata.TagList>
              <List.Item.Detail.Metadata.Separator />
              <List.Item.Detail.Metadata.Label
                title="Requested By"
                text={approval.requested_by}
              />
              <List.Item.Detail.Metadata.Label
                title="Requested"
                text={approval.requested_at}
              />
              <List.Item.Detail.Metadata.Label
                title="Expires"
                text={approval.expires_at}
              />
              <List.Item.Detail.Metadata.Label
                title="Approval ID"
                text={approval.approval_id}
              />
            </List.Item.Detail.Metadata>
          }
        />
      }
      actions={
        <ActionPanel>
          <ActionPanel.Section title="Review">
            {approval.status === "pending" && (
              <>
                <Action
                  title="Approve for MCP Apply"
                  icon={Icon.CheckCircle}
                  onAction={() =>
                    void approveSelectedApproval(runtime, approval, onRefresh)
                  }
                />
                <Action
                  title="Deny Request"
                  icon={Icon.XMarkCircle}
                  style={Action.Style.Destructive}
                  onAction={() =>
                    void denySelectedApproval(runtime, approval, onRefresh)
                  }
                />
              </>
            )}
          </ActionPanel.Section>
          <ActionPanel.Section title="Copy">
            <Action.CopyToClipboard
              title="Copy Approval ID"
              content={approval.approval_id}
            />
            <Action.CopyToClipboard
              title="Copy Action ID"
              content={approval.action_id}
            />
            <Action.CopyToClipboard
              title="Copy Writes"
              content={action.writes.join("\n")}
            />
          </ActionPanel.Section>
          <ActionPanel.Section title="Refresh">
            <Action
              title="Refresh"
              icon={Icon.ArrowClockwise}
              onAction={onRefresh}
            />
          </ActionPanel.Section>
        </ActionPanel>
      }
    />
  );
}

async function approveSelectedApproval(
  runtime: ReturnType<typeof normalizePreferences>,
  approval: NightwardApproval,
  onRefresh: () => void,
) {
  if (approval.status !== "pending") {
    await showToast({
      style: Toast.Style.Failure,
      title: "Approval is not pending",
      message: approval.status,
    });
    return;
  }
  const action = approval.preview.action;
  const confirmed = await confirmAlert({
    title: `Approve ${action.title}?`,
    message: [
      "This lets the MCP client apply this exact one-time ticket. It does not approve any other action.",
      action.description,
      action.requires_online ? "This action may use the network." : "",
      action.command.length > 0 ? `Command: ${action.command.join(" ")}` : "",
      action.writes.length > 0 ? `Writes: ${action.writes.join(", ")}` : "",
    ]
      .filter(Boolean)
      .join("\n\n"),
    primaryAction: {
      title: "Approve",
      style:
        action.risk === "high" || action.risk === "critical"
          ? Alert.ActionStyle.Destructive
          : Alert.ActionStyle.Default,
    },
  });
  if (!confirmed) return;
  const applied = await mutateApproval("Approving MCP action", () =>
    approveApproval(runtime, approval.approval_id),
  );
  if (applied) onRefresh();
}

async function denySelectedApproval(
  runtime: ReturnType<typeof normalizePreferences>,
  approval: NightwardApproval,
  onRefresh: () => void,
) {
  if (approval.status !== "pending") {
    await showToast({
      style: Toast.Style.Failure,
      title: "Approval is not pending",
      message: approval.status,
    });
    return;
  }
  const confirmed = await confirmAlert({
    title: `Deny ${approval.preview.action.title}?`,
    message: "The MCP client will not be able to apply this ticket.",
    primaryAction: {
      title: "Deny",
      style: Alert.ActionStyle.Destructive,
    },
  });
  if (!confirmed) return;
  const denied = await mutateApproval("Denying MCP action", () =>
    denyApproval(runtime, approval.approval_id),
  );
  if (denied) onRefresh();
}

async function mutateApproval(
  title: string,
  apply: () => Promise<NightwardApproval>,
): Promise<boolean> {
  const toast = await showToast({ style: Toast.Style.Animated, title });
  try {
    const result = await apply();
    toast.style = Toast.Style.Success;
    toast.title = "Approval updated";
    toast.message = `${result.action_id}: ${result.status}`;
    return true;
  } catch (error) {
    toast.style = Toast.Style.Failure;
    toast.title = "Approval failed";
    toast.message = error instanceof Error ? error.message : String(error);
    return false;
  }
}

function approvalMarkdown(approval: NightwardApproval): string {
  const action = approval.preview.action;
  return [
    `# ${action.title}`,
    "",
    `Status: \`${approval.status}\``,
    `Approval: \`${approval.approval_id}\``,
    `Requested by: \`${approval.requested_by}\``,
    `Expires: \`${approval.expires_at}\``,
    "",
    "Approving this request lets the MCP client apply this exact ticket once. It does not approve disclosure acceptance, hidden config edits, or any other action.",
    "",
    action.description,
    "",
    action.command.length > 0 ? "## Command" : "",
    action.command.length > 0 ? `\`${action.command.join(" ")}\`` : "",
    action.writes.length > 0 ? "## Writes" : "",
    ...action.writes.map((write) => `- \`${write}\``),
    approval.decision_reason ? "## Decision" : "",
    approval.decision_reason ?? "",
  ]
    .filter((line) => line !== undefined && line !== null)
    .join("\n");
}

function sectionTitle(status: string): string {
  return status.replace(/^\w/, (letter) => letter.toUpperCase());
}

function statusColor(status: string): Color {
  switch (status) {
    case "pending":
      return Color.Yellow;
    case "approved":
      return Color.Green;
    case "applied":
      return Color.Blue;
    case "denied":
    case "failed":
    case "invalidated":
      return Color.Red;
    case "expired":
      return Color.SecondaryText;
    default:
      return Color.SecondaryText;
  }
}

function riskColor(risk: string): Color {
  switch (risk) {
    case "high":
    case "critical":
      return Color.Red;
    case "medium":
      return Color.Orange;
    case "low":
      return Color.Blue;
    case "none":
    case "safe":
      return Color.Green;
    default:
      return Color.SecondaryText;
  }
}
