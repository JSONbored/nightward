import {
  Action,
  ActionPanel,
  Color,
  Icon,
  List,
  getPreferenceValues,
} from "@raycast/api";
import { usePromise } from "@raycast/utils";
import { normalizePreferences, providersDoctor } from "./nightward";
import type { ProviderStatus } from "./types";

export default function Command() {
  const runtime = normalizePreferences(getPreferenceValues());
  const { data, error, isLoading, revalidate } = usePromise(() =>
    providersDoctor(runtime),
  );

  if (error) {
    return (
      <List isLoading={false}>
        <List.EmptyView
          title="Nightward provider doctor failed"
          description={error.message}
        />
      </List>
    );
  }

  return (
    <List
      isLoading={isLoading}
      searchBarPlaceholder="Search Nightward providers..."
      isShowingDetail
    >
      {(data ?? []).map((provider) => (
        <ProviderItem
          key={provider.name}
          provider={provider}
          onRefresh={revalidate}
        />
      ))}
    </List>
  );
}

function ProviderItem({
  provider,
  onRefresh,
}: {
  provider: ProviderStatus;
  onRefresh: () => void;
}) {
  return (
    <List.Item
      title={provider.name}
      subtitle={`${provider.status} - ${provider.capabilities}`}
      icon={{
        source: providerIcon(provider),
        tintColor: providerColor(provider),
      }}
      accessories={[
        { text: provider.enabled ? "enabled" : "disabled" },
        { text: provider.online ? "online-capable" : "offline" },
      ]}
      detail={
        <List.Item.Detail
          markdown={[
            `# ${provider.name}`,
            "",
            `Status: \`${provider.status}\``,
            `Available: \`${String(provider.available)}\``,
            `Enabled: \`${String(provider.enabled)}\``,
            `Privacy: ${provider.privacy}`,
            "",
            provider.detail ? `Detail: \`${provider.detail}\`` : "",
          ]
            .filter(Boolean)
            .join("\n")}
        />
      }
      actions={
        <ActionPanel>
          <Action.CopyToClipboard
            title="Copy Provider Name"
            content={provider.name}
          />
          {provider.command ? (
            <Action.CopyToClipboard
              title="Copy Command Name"
              content={provider.command}
            />
          ) : null}
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

function providerIcon(provider: ProviderStatus): Icon {
  if (provider.status === "ready") return Icon.CheckCircle;
  if (provider.status === "blocked") return Icon.Lock;
  if (provider.available) return Icon.Circle;
  return Icon.XMarkCircle;
}

function providerColor(provider: ProviderStatus): Color {
  if (provider.status === "ready") return Color.Green;
  if (provider.status === "blocked") return Color.Yellow;
  if (provider.available) return Color.Blue;
  return Color.Red;
}
