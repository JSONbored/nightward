export const onlineProviderNames = new Set(["trivy", "osv-scanner", "socket"]);

export type ProviderInstallInfo = {
  command?: string;
  url: string;
  note: string;
};

const providerInstallInfo: Record<string, ProviderInstallInfo> = {
  gitleaks: {
    command: "brew install gitleaks",
    url: "https://github.com/gitleaks/gitleaks#installing",
    note: "Local secret scanner. Homebrew is the lowest-friction macOS path.",
  },
  trufflehog: {
    command: "brew install trufflehog",
    url: "https://github.com/trufflesecurity/trufflehog#installation",
    note: "Local secret scanner. Nightward runs it with verification disabled by default.",
  },
  semgrep: {
    command: "brew install semgrep",
    url: "https://semgrep.dev/docs/getting-started/",
    note: "Local static analyzer. Nightward only runs Semgrep with a repo-local config.",
  },
  trivy: {
    command: "brew install trivy",
    url: "https://trivy.dev/latest/getting-started/installation/",
    note: "Online-capable scanner. Nightward requires Allow Online Providers before use.",
  },
  "osv-scanner": {
    command: "brew install osv-scanner",
    url: "https://google.github.io/osv-scanner/installation/",
    note: "Online-capable vulnerability scanner. Nightward requires Allow Online Providers before use.",
  },
  socket: {
    command: "npm install -g socket",
    url: "https://docs.socket.dev/docs/socket-cli",
    note: "Remote scan creation provider. Nightward requires Allow Online Providers before use.",
  },
};

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

export function installInfoForProvider(
  provider: string,
): ProviderInstallInfo | undefined {
  return providerInstallInfo[normalizeProviderName(provider)];
}
