export const onlineProviderNames = new Set(["trivy", "osv-scanner", "socket"]);

export function normalizeProviderName(name: string): string {
  return name.trim().toLowerCase();
}

export function normalizeProviderSelection(
  providers: readonly string[],
): string[] {
  const seen = new Set<string>();
  const selected: string[] = [];
  for (const provider of providers) {
    const normalized = normalizeProviderName(provider);
    if (!normalized || seen.has(normalized)) continue;
    seen.add(normalized);
    selected.push(normalized);
  }
  return selected;
}

export function parseProviderSelection(value: string | undefined): string[] {
  if (!value) return [];
  return normalizeProviderSelection(value.split(","));
}

export function serializeProviderSelection(
  providers: readonly string[],
): string {
  return normalizeProviderSelection(providers).join(",");
}

export function isOnlineProvider(provider: string): boolean {
  return onlineProviderNames.has(normalizeProviderName(provider));
}

export function selectedAnalysisProviders(
  selectedProviders: readonly string[],
  allowOnlineProviders: boolean,
): string[] {
  const selected = normalizeProviderSelection(selectedProviders);
  if (allowOnlineProviders) return selected;
  return selected.filter((provider) => !isOnlineProvider(provider));
}

export function selectedOnlineProviders(
  selectedProviders: readonly string[],
): string[] {
  return normalizeProviderSelection(selectedProviders).filter(isOnlineProvider);
}
