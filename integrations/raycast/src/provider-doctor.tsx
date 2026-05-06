import {
  Action,
  ActionPanel,
  Alert,
  Clipboard,
  Color,
  Icon,
  List,
  confirmAlert,
  getPreferenceValues,
  openExtensionPreferences,
  open,
  showHUD,
  showToast,
  Toast,
} from "@raycast/api";
import { usePromise } from "@raycast/utils";
import { useEffect, useMemo, useState } from "react";
import {
  applyAction,
  normalizePreferences,
  previewAction,
  providersDoctor,
  type RuntimeOptions,
} from "./nightward";
import { installInfoForProvider, isOnlineProvider } from "./provider-options";
import {
  clearSelectedProviders,
  readSelectedProviders,
  setProviderSelected,
} from "./provider-settings";
import type { ProviderStatus } from "./types";

export default function Command() {
  const runtime = normalizePreferences(getPreferenceValues());
  const [selectedProviders, setSelectedProviders] = useState<string[]>([]);
  const selectedKey = selectedProviders.join(",");
  const selectedSet = useMemo(
    () => new Set(selectedProviders),
    [selectedProviders],
  );
  const { data, error, isLoading, revalidate } = usePromise(
    async (providerKey: string) => {
      const providers = providerKey ? providerKey.split(",") : [];
      return providersDoctor(runtime, providers);
    },
    [selectedKey],
  );

  useEffect(() => {
    void readSelectedProviders().then(setSelectedProviders);
  }, []);

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
      {(data ?? []).length === 0 && !isLoading ? (
        <List.EmptyView
          title="No providers returned"
          description="Nightward provider doctor did not return provider status."
          icon={Icon.ExclamationMark}
        />
      ) : null}
      <List.Section title="Local Providers">
        {(data ?? [])
          .filter((provider) => !provider.online)
          .map((provider) => (
            <ProviderItem
              key={provider.name}
              provider={provider}
              runtime={runtime}
              selected={selectedSet.has(provider.name)}
              onlineAllowed={runtime.allowOnlineProviders}
              onRefresh={revalidate}
              onSelectedChange={setSelectedProviders}
            />
          ))}
      </List.Section>
      <List.Section title="Online-Capable Providers">
        {(data ?? [])
          .filter((provider) => provider.online)
          .map((provider) => (
            <ProviderItem
              key={provider.name}
              provider={provider}
              runtime={runtime}
              selected={selectedSet.has(provider.name)}
              onlineAllowed={runtime.allowOnlineProviders}
              onRefresh={revalidate}
              onSelectedChange={setSelectedProviders}
            />
          ))}
      </List.Section>
    </List>
  );
}

function ProviderItem({
  provider,
  runtime,
  selected,
  onlineAllowed,
  onRefresh,
  onSelectedChange,
}: {
  provider: ProviderStatus;
  runtime: RuntimeOptions;
  selected: boolean;
  onlineAllowed: boolean;
  onRefresh: () => void;
  onSelectedChange: (providers: string[]) => void;
}) {
  const blockedByPreference =
    selected && isOnlineProvider(provider.name) && !onlineAllowed;
  const installInfo = installInfoForProvider(provider.name);
  const installCommand = installInfo?.command;
  const selectionLabel = provider.default
    ? "built-in"
    : blockedByPreference
      ? "online blocked"
      : selected && !provider.available
        ? "missing selected"
        : selected
          ? "selected"
          : provider.status;
  return (
    <List.Item
      title={provider.name}
      subtitle={providerSubtitle(provider, selected, onlineAllowed)}
      icon={{
        source: providerIcon(provider),
        tintColor: providerColor(provider),
      }}
      detail={
        <List.Item.Detail
          markdown={providerMarkdown(provider, selected, onlineAllowed)}
          metadata={
            <List.Item.Detail.Metadata>
              <List.Item.Detail.Metadata.TagList title="Status">
                <List.Item.Detail.Metadata.TagList.Item
                  text={selectionLabel}
                  color={
                    provider.status === "skipped"
                      ? Color.SecondaryText
                      : !provider.available
                        ? Color.Red
                        : blockedByPreference
                          ? Color.Yellow
                          : selected || provider.default
                            ? Color.Blue
                            : providerColor(provider)
                  }
                />
              </List.Item.Detail.Metadata.TagList>
              <List.Item.Detail.Metadata.Separator />
              <List.Item.Detail.Metadata.Label
                title="Available"
                text={provider.available ? "yes" : "no"}
              />
              <List.Item.Detail.Metadata.Label
                title="Selected"
                text={provider.default ? "built-in" : selected ? "yes" : "no"}
              />
              <List.Item.Detail.Metadata.Label
                title="Online Gate"
                text={
                  provider.online
                    ? onlineAllowed
                      ? "allowed"
                      : "blocked"
                    : "not needed"
                }
              />
              <List.Item.Detail.Metadata.Label
                title="Execution"
                text={provider.online ? "online-capable" : "local"}
              />
              {provider.command ? (
                <List.Item.Detail.Metadata.Label
                  title="Command"
                  text={provider.command}
                />
              ) : null}
            </List.Item.Detail.Metadata>
          }
        />
      }
      actions={
        <ActionPanel>
          <ActionPanel.Section title="Provider">
            {provider.default ? null : (
              <Action
                title={
                  selected
                    ? "Disable for Raycast Analysis"
                    : "Enable for Raycast Analysis"
                }
                icon={selected ? Icon.XMarkCircle : Icon.PlusCircle}
                onAction={() =>
                  void toggleProvider(
                    provider.name,
                    !selected,
                    onSelectedChange,
                  )
                }
              />
            )}
            {provider.online && !onlineAllowed ? (
              <Action
                title="Allow Online Providers in Preferences"
                icon={Icon.Gear}
                onAction={() => void openExtensionPreferences()}
              />
            ) : null}
            <Action
              title="Clear Selected Providers"
              icon={Icon.Trash}
              onAction={() => void clearProviders(onSelectedChange)}
            />
          </ActionPanel.Section>

          {installInfo ? (
            <ActionPanel.Section title="Install">
              {installCommand ? (
                <>
                  {!provider.available ? (
                    <Action
                      title="Install Provider"
                      icon={Icon.Download}
                      onAction={() =>
                        void installProvider(runtime, provider.name, onRefresh)
                      }
                    />
                  ) : null}
                  <Action
                    title="Copy Install Command"
                    icon={Icon.Terminal}
                    onAction={() =>
                      void copyInstallCommand(provider.name, installCommand)
                    }
                  />
                </>
              ) : null}
              <Action
                title="Open Install Docs"
                icon={Icon.Book}
                onAction={() => void open(installInfo.url)}
              />
            </ActionPanel.Section>
          ) : null}

          <ActionPanel.Section title="Copy">
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

function providerSubtitle(
  provider: ProviderStatus,
  selected: boolean,
  onlineAllowed: boolean,
): string {
  if (provider.default) {
    return "built-in";
  }
  if (selected && provider.online && !onlineAllowed) {
    return "online blocked";
  }
  if (selected && !provider.available) {
    return "missing selected";
  }
  if (selected) return "selected";
  if (provider.status === "skipped") return "skipped";
  if (provider.available) return provider.status;
  return "missing";
}

function providerMarkdown(
  provider: ProviderStatus,
  selected: boolean,
  onlineAllowed: boolean,
): string {
  const onlineBlocked = selected && provider.online && !onlineAllowed;
  const installInfo = installInfoForProvider(provider.name);
  return [
    `# ${provider.name}`,
    "",
    "## Privacy",
    provider.privacy,
    "",
    "## Capability",
    provider.capabilities,
    onlineBlocked
      ? [
          "",
          "## Online Gate",
          "This provider is selected, but Raycast will not run it until **Allow Online Providers** is enabled in extension preferences.",
        ].join("\n")
      : "",
    installInfo ? "" : "",
    installInfo ? "## Install" : "",
    installInfo?.command ? `Command: \`${installInfo.command}\`` : "",
    installInfo?.note ?? "",
    provider.detail ? "" : "",
    provider.detail ? "## Detail" : "",
    provider.detail ? provider.detail : "",
  ]
    .filter(Boolean)
    .join("\n");
}

async function copyInstallCommand(provider: string, command: string) {
  await Clipboard.copy(command);
  await showHUD(`Copied ${provider} install command`);
}

async function installProvider(
  runtime: RuntimeOptions,
  provider: string,
  onRefresh: () => void,
) {
  const actionId = `provider.install.${provider}`;
  let preview;
  try {
    preview = await previewAction(runtime, actionId);
  } catch (error) {
    await showToast({
      style: Toast.Style.Failure,
      title: `Could not preview ${provider} install`,
      message: error instanceof Error ? error.message.slice(0, 180) : undefined,
    });
    return;
  }

  const action = preview.action;
  if (!action.available) {
    await showToast({
      style: Toast.Style.Failure,
      title: `${provider} install is blocked`,
      message: action.blocked_reason || "Nightward action is not available.",
    });
    return;
  }

  const command = action.command.join(" ");
  const confirmed = await confirmAlert({
    title: `Apply ${action.title}?`,
    message: [
      action.description,
      command ? `Command: ${command}` : "",
      ...preview.warnings,
      "Nightward will apply this through the shared action registry and audit the result.",
    ]
      .filter(Boolean)
      .join("\n\n"),
    primaryAction: {
      title: "Install",
      style: Alert.ActionStyle.Default,
    },
  });
  if (!confirmed) return;

  const toast = await showToast({
    style: Toast.Style.Animated,
    title: `Installing ${provider}`,
    message: command,
  });
  try {
    const result = await applyAction(runtime, actionId);
    toast.style = Toast.Style.Success;
    toast.title = `${provider} installed`;
    toast.message =
      result.message.slice(0, 180) || "Provider is ready to refresh.";
    onRefresh();
  } catch (error) {
    toast.style = Toast.Style.Failure;
    toast.title = `Could not install ${provider}`;
    toast.message =
      error instanceof Error ? error.message.slice(0, 180) : "Action failed.";
  }
}

async function toggleProvider(
  provider: string,
  selected: boolean,
  onSelectedChange: (providers: string[]) => void,
) {
  const providers = await setProviderSelected(provider, selected);
  onSelectedChange(providers);
  await showToast({
    style: Toast.Style.Success,
    title: selected ? "Provider enabled" : "Provider disabled",
    message: `${provider} ${selected ? "will be used" : "will not be used"} in Raycast Analysis.`,
  });
}

async function clearProviders(onSelectedChange: (providers: string[]) => void) {
  await clearSelectedProviders();
  onSelectedChange([]);
  await showToast({
    style: Toast.Style.Success,
    title: "Provider selection cleared",
    message: "Raycast Analysis will use built-in offline analysis only.",
  });
}

function providerIcon(provider: ProviderStatus): Icon {
  if (provider.status === "ready") return Icon.CheckCircle;
  if (provider.status === "blocked") return Icon.Lock;
  if (provider.status === "skipped") return Icon.Circle;
  if (provider.available) return Icon.Circle;
  return Icon.XMarkCircle;
}

function providerColor(provider: ProviderStatus): Color {
  if (provider.status === "ready") return Color.Green;
  if (provider.status === "blocked") return Color.Yellow;
  if (provider.status === "skipped") return Color.SecondaryText;
  if (provider.available) return Color.Blue;
  return Color.Red;
}
