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
import { applyAction, listActions, normalizePreferences } from "./nightward";
import type { NightwardAction } from "./types";

export default function Command() {
  const runtime = normalizePreferences(getPreferenceValues());
  const { data, error, isLoading, revalidate } = usePromise(async () =>
    listActions(runtime),
  );

  if (error) {
    return (
      <List isLoading={false}>
        <List.EmptyView
          title="Nightward actions failed"
          description={error.message}
          icon={Icon.ExclamationMark}
        />
      </List>
    );
  }

  const actions = data ?? [];
  return (
    <List
      isLoading={isLoading}
      searchBarPlaceholder="Search Nightward actions..."
      isShowingDetail
    >
      {actions.length === 0 && !isLoading ? (
        <List.EmptyView
          title="No actions returned"
          description="Nightward did not expose any local action specs."
          icon={Icon.Circle}
        />
      ) : null}
      {["setup", "schedule", "backup", "providers"].map((category) => (
        <List.Section key={category} title={sectionTitle(category)}>
          {actions
            .filter((action) => action.category === category)
            .map((action) => (
              <ActionItem
                key={action.id}
                action={action}
                onRefresh={revalidate}
                runtime={runtime}
              />
            ))}
        </List.Section>
      ))}
    </List>
  );
}

function ActionItem({
  action,
  onRefresh,
  runtime,
}: {
  action: NightwardAction;
  onRefresh: () => void;
  runtime: ReturnType<typeof normalizePreferences>;
}) {
  return (
    <List.Item
      title={action.title}
      subtitle={action.available ? action.category : "blocked"}
      icon={{
        source: action.available ? Icon.Bolt : Icon.Lock,
        tintColor: actionColor(action),
      }}
      accessories={[
        {
          text: action.risk,
          icon: { source: Icon.Circle, tintColor: actionColor(action) },
        },
      ]}
      detail={
        <List.Item.Detail
          markdown={actionMarkdown(action)}
          metadata={
            <List.Item.Detail.Metadata>
              <List.Item.Detail.Metadata.TagList title="Status">
                <List.Item.Detail.Metadata.TagList.Item
                  text={action.available ? "available" : "blocked"}
                  color={action.available ? Color.Green : Color.Red}
                />
                <List.Item.Detail.Metadata.TagList.Item
                  text={action.risk}
                  color={actionColor(action)}
                />
              </List.Item.Detail.Metadata.TagList>
              <List.Item.Detail.Metadata.Separator />
              <List.Item.Detail.Metadata.Label
                title="Confirmation"
                text={
                  action.requires_confirmation ? "required" : "not required"
                }
              />
              <List.Item.Detail.Metadata.Label
                title="Online"
                text={action.requires_online ? "possible" : "no"}
              />
              <List.Item.Detail.Metadata.Label
                title="Reversible"
                text={action.reversible ? "yes" : "no"}
              />
              <List.Item.Detail.Metadata.Label
                title="Action ID"
                text={action.id}
              />
            </List.Item.Detail.Metadata>
          }
        />
      }
      actions={
        <ActionPanel>
          <ActionPanel.Section title="Apply">
            <Action
              title={action.available ? "Apply Action" : "Action Blocked"}
              icon={action.available ? Icon.Bolt : Icon.Lock}
              style={
                action.risk === "high"
                  ? Action.Style.Destructive
                  : Action.Style.Regular
              }
              onAction={() =>
                void applySelectedAction(runtime, action, onRefresh)
              }
            />
          </ActionPanel.Section>
          <ActionPanel.Section title="Copy">
            <Action.CopyToClipboard
              title="Copy Action ID"
              content={action.id}
            />
            {action.command.length > 0 ? (
              <Action.CopyToClipboard
                title="Copy Command"
                content={action.command.join(" ")}
              />
            ) : null}
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

async function applySelectedAction(
  runtime: ReturnType<typeof normalizePreferences>,
  action: NightwardAction,
  onRefresh: () => void,
) {
  if (!action.available) {
    await showToast({
      style: Toast.Style.Failure,
      title: "Action is blocked",
      message: action.blocked_reason,
    });
    return;
  }
  const confirmed = await confirmAlert({
    title: `Apply ${action.title}?`,
    message: [
      action.description,
      action.requires_online ? "This action may use the network." : "",
      action.writes.length > 0 ? `Writes: ${action.writes.join(", ")}` : "",
    ]
      .filter(Boolean)
      .join("\n\n"),
    primaryAction: {
      title: "Apply",
      style:
        action.risk === "high"
          ? Alert.ActionStyle.Destructive
          : Alert.ActionStyle.Default,
    },
  });
  if (!confirmed) return;
  const toast = await showToast({
    style: Toast.Style.Animated,
    title: `Applying ${action.title}`,
  });
  try {
    const result = await applyAction(runtime, action.id);
    toast.style = Toast.Style.Success;
    toast.title = "Action applied";
    toast.message = result.message;
    onRefresh();
  } catch (error) {
    toast.style = Toast.Style.Failure;
    toast.title = "Action failed";
    toast.message = error instanceof Error ? error.message : String(error);
  }
}

function actionMarkdown(action: NightwardAction): string {
  return [
    `# ${action.title}`,
    "",
    action.description,
    "",
    "## Responsibility",
    "Nightward is beta software. Apply actions only after reviewing writes, commands, and third-party behavior.",
    "",
    action.blocked_reason ? "## Blocked" : "",
    action.blocked_reason,
    action.command.length > 0 ? "## Command" : "",
    action.command.length > 0 ? `\`${action.command.join(" ")}\`` : "",
    action.writes.length > 0 ? "## Writes" : "",
    ...action.writes.map((write) => `- \`${write}\``),
  ]
    .filter(Boolean)
    .join("\n");
}

function sectionTitle(category: string): string {
  return category.charAt(0).toUpperCase() + category.slice(1);
}

function actionColor(action: NightwardAction): Color {
  if (!action.available) return Color.SecondaryText;
  if (action.risk === "high" || action.risk === "critical") return Color.Red;
  if (action.risk === "medium") return Color.Yellow;
  if (action.risk === "low") return Color.Blue;
  return Color.Green;
}
