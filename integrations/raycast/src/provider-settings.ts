import { LocalStorage } from "@raycast/api";
import {
  normalizeProviderSelection,
  parseProviderSelection,
  serializeProviderSelection,
} from "./provider-options";

const selectedProvidersKey = "nightward.selectedProviders.v1";

export async function readSelectedProviders(): Promise<string[]> {
  const value = await LocalStorage.getItem<string>(selectedProvidersKey);
  return parseProviderSelection(value);
}

export async function setProviderSelected(
  provider: string,
  selected: boolean,
): Promise<string[]> {
  const current = await readSelectedProviders();
  const next = selected
    ? normalizeProviderSelection([...current, provider])
    : current.filter((name) => name !== provider.trim().toLowerCase());
  await LocalStorage.setItem(
    selectedProvidersKey,
    serializeProviderSelection(next),
  );
  return next;
}

export async function clearSelectedProviders(): Promise<void> {
  await LocalStorage.removeItem(selectedProvidersKey);
}
