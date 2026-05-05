import { execFile } from "node:child_process";
import { existsSync } from "node:fs";
import { homedir } from "node:os";
import { delimiter, dirname, join } from "node:path";
import {
  selectedAnalysisProviders,
  selectedOnlineProviders,
} from "./provider-options";
import type {
  AnalysisReport,
  DoctorReport,
  DiffReport,
  Finding,
  FixPlan,
  ProviderStatus,
  ScanReport,
} from "./types";
import { redactText } from "./format";

type ExecFileOptions = {
  env?: NodeJS.ProcessEnv;
  maxBuffer?: number;
  timeout?: number;
};

type ExecFileCallback = (
  error: NodeJS.ErrnoException | null,
  stdout: string,
  stderr: string,
) => void;
type ExecFileLike = (
  file: string,
  args: string[],
  options: ExecFileOptions,
  callback: ExecFileCallback,
) => unknown;

export type NightwardPreferences = {
  nightwardPath?: string;
  homeOverride?: string;
  allowOnlineProviders?: boolean;
};

export type RuntimeOptions = {
  executable: string;
  homeOverride?: string;
  allowOnlineProviders: boolean;
  timeoutMs: number;
  execFileImpl?: ExecFileLike;
};

export class NightwardCommandError extends Error {
  readonly command: string;
  readonly stderr: string;

  constructor(command: string, stderr: string) {
    super(`${command} failed${stderr ? `: ${stderr}` : ""}`);
    this.name = "NightwardCommandError";
    this.command = command;
    this.stderr = stderr;
  }
}

export function normalizePreferences(
  preferences: NightwardPreferences = {},
): RuntimeOptions {
  return {
    executable: preferences.nightwardPath?.trim() || "nw",
    homeOverride: preferences.homeOverride?.trim() || undefined,
    allowOnlineProviders: preferences.allowOnlineProviders === true,
    timeoutMs: 20000,
  };
}

export async function scan(options: RuntimeOptions): Promise<ScanReport> {
  return runNightwardJSON<ScanReport>(["scan", "--json"], options);
}

export async function doctor(options: RuntimeOptions): Promise<DoctorReport> {
  return runNightwardJSON<DoctorReport>(["doctor", "--json"], options);
}

export async function findings(options: RuntimeOptions): Promise<Finding[]> {
  return runNightwardJSON<Finding[]>(["findings", "list", "--json"], options);
}

export async function explainFinding(
  id: string,
  options: RuntimeOptions,
): Promise<Finding> {
  return runNightwardJSON<Finding>(
    ["findings", "explain", "--json", id],
    options,
  );
}

export type FixPlanSelector = {
  findingId?: string;
  rule?: string;
};

export async function fixPlan(
  options: RuntimeOptions,
  selector: FixPlanSelector = {},
): Promise<FixPlan> {
  return runNightwardJSON<FixPlan>(
    ["fix", "plan", ...fixSelectorArgs(selector), "--json"],
    options,
  );
}

export async function analysisReport(
  options: RuntimeOptions,
  selectedProviders: string[] = [],
): Promise<AnalysisReport> {
  const args = ["analyze", "--all"];
  const providers = selectedAnalysisProviders(
    selectedProviders,
    options.allowOnlineProviders,
  );
  if (providers.length > 0) {
    args.push("--with", providers.join(","));
  }
  if (
    options.allowOnlineProviders &&
    selectedOnlineProviders(selectedProviders).length > 0
  ) {
    args.push("--online");
  }
  args.push("--json");
  return runNightwardJSON<AnalysisReport>(args, options);
}

export async function providersDoctor(
  options: RuntimeOptions,
  selectedProviders: string[] = [],
): Promise<ProviderStatus[]> {
  const args = ["providers", "doctor"];
  if (selectedProviders.length > 0) {
    args.push("--with", selectedProviders.join(","));
  }
  if (
    options.allowOnlineProviders &&
    selectedOnlineProviders(selectedProviders).length > 0
  ) {
    args.push("--online");
  }
  args.push("--json");
  return runNightwardJSON<ProviderStatus[]>(args, options);
}

export async function reportDiff(
  options: RuntimeOptions,
  base: string,
  head: string,
): Promise<DiffReport> {
  return runNightwardJSON<DiffReport>(
    ["report", "diff", "--from", base, "--to", head],
    options,
  );
}

export async function explainSignal(
  findingId: string,
  options: RuntimeOptions,
): Promise<AnalysisReport> {
  return runNightwardJSON<AnalysisReport>(
    ["analyze", "finding", findingId, "--json"],
    options,
  );
}

export async function exportAnalysisMarkdown(
  options: RuntimeOptions,
  selectedProviders: string[] = [],
): Promise<string> {
  const report = await analysisReport(options, selectedProviders);
  return [
    "# Nightward Analysis",
    "",
    `Generated: \`${report.generated_at}\``,
    `Mode: \`${report.mode}\``,
    `Signals: \`${report.summary.total_signals}\``,
    `Highest severity: \`${report.summary.highest_severity || "info"}\``,
    "",
    ...report.signals.map((signal) =>
      [
        `## ${signal.rule}`,
        "",
        `- Severity: \`${signal.severity}\``,
        `- Confidence: \`${signal.confidence}\``,
        `- Provider: \`${signal.provider}\``,
        signal.path ? `- Path: \`${signal.path}\`` : "",
        "",
        redactText(signal.message),
        "",
        `Recommended action: ${redactText(signal.recommended_action)}`,
      ]
        .filter(Boolean)
        .join("\n"),
    ),
  ].join("\n");
}

export async function exportFixPlanMarkdown(
  options: RuntimeOptions,
  selector: FixPlanSelector = {},
): Promise<string> {
  return runNightwardText(
    ["fix", "export", ...fixSelectorArgs(selector), "--format", "markdown"],
    options,
  );
}

export function reportsDir(homeOverride?: string): string {
  return join(
    homeOverride?.trim() || homedir(),
    ".local",
    "state",
    "nightward",
    "reports",
  );
}

export function reportsDirExists(homeOverride?: string): boolean {
  return existsSync(reportsDir(homeOverride));
}

export async function runNightwardJSON<T>(
  args: string[],
  options: RuntimeOptions,
): Promise<T> {
  const text = await runNightwardText(args, options);
  try {
    return JSON.parse(text) as T;
  } catch {
    throw new NightwardCommandError(
      commandLabel(options.executable, args),
      "Nightward returned malformed JSON",
    );
  }
}

export async function runNightwardText(
  args: string[],
  options: RuntimeOptions,
): Promise<string> {
  try {
    return await execNightward(options.executable, args, options);
  } catch (error) {
    if (options.executable === "nw" && isENOENT(error)) {
      return execNightward("nightward", args, options);
    }
    throw error;
  }
}

function execNightward(
  executable: string,
  args: string[],
  options: RuntimeOptions,
): Promise<string> {
  const env: NodeJS.ProcessEnv = {
    ...process.env,
    PATH: raycastCommandPath(executable),
  };
  if (options.homeOverride) {
    env.NIGHTWARD_HOME = options.homeOverride;
  }
  const execFileImpl = options.execFileImpl ?? (execFile as ExecFileLike);
  return new Promise((resolve, reject) => {
    execFileImpl(
      executable,
      args,
      {
        env,
        maxBuffer: 1024 * 1024 * 20,
        timeout: options.timeoutMs,
      },
      (error, stdout, stderr) => {
        if (error) {
          reject(
            new NightwardCommandError(
              commandLabel(executable, args),
              firstLine(redactText(stderr || error.message)),
            ),
          );
          return;
        }
        resolve(stdout);
      },
    );
  });
}

function commandLabel(executable: string, args: string[]): string {
  return [executable, ...args].join(" ");
}

function fixSelectorArgs(selector: FixPlanSelector): string[] {
  if (selector.findingId) return ["--finding", selector.findingId];
  if (selector.rule) return ["--rule", selector.rule];
  return ["--all"];
}

function raycastCommandPath(executable: string): string {
  const current = process.env.PATH?.split(delimiter) ?? [];
  const extra = [
    executable.includes("/") ? dirname(executable) : "",
    join(homedir(), ".local", "bin"),
    join(homedir(), ".cargo", "bin"),
    "/opt/homebrew/bin",
    "/usr/local/bin",
    "/usr/bin",
    "/bin",
    "/usr/sbin",
    "/sbin",
  ];
  return [...new Set([...current, ...extra].filter(Boolean))].join(delimiter);
}

function isENOENT(error: unknown): boolean {
  return (
    error instanceof NightwardCommandError &&
    /ENOENT|not found|no such file/i.test(error.stderr)
  );
}

function firstLine(value: string): string {
  return value.split(/\r?\n/).find(Boolean)?.slice(0, 300) ?? "";
}
