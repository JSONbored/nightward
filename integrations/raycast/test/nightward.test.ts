import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { test } from "vitest";
import {
  analysisReport,
  explainFinding,
  explainSignal,
  exportFixPlanMarkdown,
  fixPlan,
  applyAction,
  latestReportPair,
  listActions,
  NightwardCommandError,
  normalizePreferences,
  previewAction,
  providersDoctor,
  reportDiff,
  reportHistory,
  reportsDir,
  runNightwardJSON,
  type RuntimeOptions,
} from "../src/nightward";
import {
  installInfoForProvider,
  normalizeProviderSelection,
  selectedAnalysisProviders,
  selectedOnlineProviders,
} from "../src/provider-options";

test("normalizes empty preferences to nw without a home override", () => {
  const options = normalizePreferences({});
  assert.equal(options.executable, "nw");
  assert.equal(options.homeOverride, undefined);
  assert.equal(options.allowOnlineProviders, false);
  assert.equal(options.timeoutMs, 20000);
});

test("passes NIGHTWARD_HOME and parses JSON output", async () => {
  let observedHome = "";
  const options: RuntimeOptions = {
    executable: "nightward",
    homeOverride: "/tmp/nightward-home",
    allowOnlineProviders: false,
    timeoutMs: 1000,
    execFileImpl: (_file, _args, options, callback) => {
      observedHome = options.env?.NIGHTWARD_HOME ?? "";
      callback(null, '{"ok":true}', "");
    },
  };

  const result = await runNightwardJSON<{ ok: boolean }>(
    ["doctor", "--json"],
    options,
  );
  assert.deepEqual(result, { ok: true });
  assert.equal(observedHome, "/tmp/nightward-home");
});

test("extends Raycast PATH with common local CLI locations", async () => {
  const originalPath = process.env.PATH;
  let observedPath = "";
  process.env.PATH = "/usr/bin:/bin";
  try {
    const options: RuntimeOptions = {
      executable: "/tmp/nightward-raycast/nw",
      allowOnlineProviders: false,
      timeoutMs: 1000,
      execFileImpl: (_file, _args, options, callback) => {
        observedPath = options.env?.PATH ?? "";
        callback(null, '{"ok":true}', "");
      },
    };

    const result = await runNightwardJSON<{ ok: boolean }>(
      ["providers", "doctor", "--json"],
      options,
    );

    assert.deepEqual(result, { ok: true });
    assert.ok(observedPath.split(":").includes("/tmp/nightward-raycast"));
    assert.ok(observedPath.split(":").includes("/opt/homebrew/bin"));
    assert.ok(observedPath.split(":").includes("/usr/local/bin"));
  } finally {
    process.env.PATH = originalPath;
  }
});

test("passes json flag before finding id for explain finding", async () => {
  let observedArgs: string[] = [];
  const options: RuntimeOptions = {
    executable: "nightward",
    allowOnlineProviders: false,
    timeoutMs: 1000,
    execFileImpl: (_file, args, _options, callback) => {
      observedArgs = args;
      callback(
        null,
        JSON.stringify({
          id: "finding-id",
          tool: "Codex",
          path: "/tmp/config.toml",
          severity: "medium",
          rule: "mcp_secret_env",
          message: "test",
          evidence: "env_key=API_TOKEN",
          recommended_action: "test",
        }),
        "",
      );
    },
  };

  const result = await explainFinding("finding-id", options);

  assert.equal(result.id, "finding-id");
  assert.deepEqual(observedArgs, [
    "findings",
    "explain",
    "--json",
    "finding-id",
  ]);
});

test("falls back from nw to nightward when nw is missing", async () => {
  const calls: string[] = [];
  const options: RuntimeOptions = {
    executable: "nw",
    allowOnlineProviders: false,
    timeoutMs: 1000,
    execFileImpl: (file, _args, _options, callback) => {
      calls.push(file);
      if (file === "nw") {
        const error = new Error("spawn nw ENOENT") as NodeJS.ErrnoException;
        error.code = "ENOENT";
        callback(error, "", "");
        return;
      }
      callback(null, '{"ok":true}', "");
    },
  };

  const result = await runNightwardJSON<{ ok: boolean }>(
    ["scan", "--json"],
    options,
  );
  assert.deepEqual(result, { ok: true });
  assert.deepEqual(calls, ["nw", "nightward"]);
});

test("reports directory follows the optional home override", () => {
  assert.equal(
    reportsDir("/tmp/example"),
    "/tmp/example/.local/state/nightward/reports",
  );
});

test("normalizes and gates provider selections", () => {
  const selected = normalizeProviderSelection([
    "gitleaks",
    " TRIVY ",
    "gitleaks",
    "socket",
  ]);

  assert.deepEqual(selected, ["gitleaks", "trivy", "socket"]);
  assert.deepEqual(selectedAnalysisProviders(selected, false), ["gitleaks"]);
  assert.deepEqual(selectedAnalysisProviders(selected, true), selected);
  assert.deepEqual(selectedOnlineProviders(selected), ["trivy", "socket"]);
});

test("provider install metadata exposes safe user-run commands", () => {
  assert.equal(
    installInfoForProvider("gitleaks")?.command,
    "brew install gitleaks",
  );
  assert.equal(
    installInfoForProvider("SOCKET")?.command,
    "npm install -g socket",
  );
  assert.equal(installInfoForProvider("unknown"), undefined);
});

test("provider doctor routes installs through the shared action registry", async () => {
  const source = await readFile(
    new URL("../src/provider-doctor.tsx", import.meta.url),
    "utf8",
  );

  assert.doesNotMatch(source, /node:child_process|\/bin\/zsh|execFile\(/);
  assert.match(source, /provider\.install\.\$\{provider\}/);
  assert.match(source, /previewAction/);
  assert.match(source, /applyAction/);
});

test("analysis command passes selected local providers without online gate", async () => {
  let observedArgs: string[] = [];
  const options: RuntimeOptions = {
    executable: "nightward",
    allowOnlineProviders: false,
    timeoutMs: 1000,
    execFileImpl: (_file, args, _options, callback) => {
      observedArgs = args;
      callback(null, baseAnalysisJSON(), "");
    },
  };

  await analysisReport(options, ["gitleaks", "trivy"]);

  assert.deepEqual(observedArgs, [
    "analyze",
    "--all",
    "--with",
    "gitleaks",
    "--json",
  ]);
});

test("analysis command passes online gate only when preference allows it", async () => {
  let observedArgs: string[] = [];
  const options: RuntimeOptions = {
    executable: "nightward",
    allowOnlineProviders: true,
    timeoutMs: 1000,
    execFileImpl: (_file, args, _options, callback) => {
      observedArgs = args;
      callback(null, baseAnalysisJSON(), "");
    },
  };

  await analysisReport(options, ["osv-scanner", "socket"]);

  assert.deepEqual(observedArgs, [
    "analyze",
    "--all",
    "--with",
    "osv-scanner,socket",
    "--online",
    "--json",
  ]);
});

test("provider doctor reflects selected providers and online preference", async () => {
  let observedArgs: string[] = [];
  const options: RuntimeOptions = {
    executable: "nightward",
    allowOnlineProviders: true,
    timeoutMs: 1000,
    execFileImpl: (_file, args, _options, callback) => {
      observedArgs = args;
      callback(null, "[]", "");
    },
  };

  await providersDoctor(options, ["socket"]);

  assert.deepEqual(observedArgs, [
    "providers",
    "doctor",
    "--with",
    "socket",
    "--online",
    "--json",
  ]);
});

test("action helpers call the shared CLI action surface", async () => {
  const observed: string[][] = [];
  const options: RuntimeOptions = {
    executable: "nightward",
    allowOnlineProviders: false,
    timeoutMs: 1000,
    execFileImpl: (_file, args, _options, callback) => {
      observed.push(args);
      if (args.includes("list")) {
        callback(null, "[]", "");
        return;
      }
      if (args.includes("preview")) {
        callback(
          null,
          JSON.stringify({
            action: { id: "backup.snapshot" },
            steps: [],
            warnings: [],
          }),
          "",
        );
        return;
      }
      callback(
        null,
        JSON.stringify({
          action_id: "backup.snapshot",
          status: "applied",
          message: "ok",
          writes: [],
        }),
        "",
      );
    },
  };

  await listActions(options);
  await previewAction(options, "backup.snapshot");
  await applyAction(options, "backup.snapshot");

  assert.deepEqual(observed[0], ["actions", "list", "--json"]);
  assert.deepEqual(observed[1], [
    "actions",
    "preview",
    "backup.snapshot",
    "--json",
  ]);
  assert.deepEqual(observed[2], [
    "actions",
    "apply",
    "backup.snapshot",
    "--confirm",
    "--json",
  ]);
});

test("explain signal passes finding id before flags", async () => {
  let observedArgs: string[] = [];
  const options: RuntimeOptions = {
    executable: "nightward",
    allowOnlineProviders: false,
    timeoutMs: 1000,
    execFileImpl: (_file, args, _options, callback) => {
      observedArgs = args;
      callback(null, baseAnalysisJSON(), "");
    },
  };

  await explainSignal("finding-123", options);

  assert.deepEqual(observedArgs, [
    "analyze",
    "finding",
    "finding-123",
    "--json",
  ]);
});

test("fix plan helpers pass scoped selectors", async () => {
  const observed: string[][] = [];
  const options: RuntimeOptions = {
    executable: "nightward",
    allowOnlineProviders: false,
    timeoutMs: 1000,
    execFileImpl: (_file, args, _options, callback) => {
      observed.push(args);
      if (args.includes("export")) {
        callback(null, "# Nightward Fix Plan\n", "");
        return;
      }
      callback(
        null,
        JSON.stringify({
          generated_at: "2026-05-01T00:00:00Z",
          summary: { total: 0, safe: 0, review: 0, blocked: 0 },
          fixes: [],
        }),
        "",
      );
    },
  };

  await fixPlan(options, { rule: "mcp_unpinned_package" });
  await exportFixPlanMarkdown(options, { findingId: "finding-123" });

  assert.deepEqual(observed[0], [
    "fix",
    "plan",
    "--rule",
    "mcp_unpinned_package",
    "--json",
  ]);
  assert.deepEqual(observed[1], [
    "fix",
    "export",
    "--finding",
    "finding-123",
    "--format",
    "markdown",
  ]);
});

test("report diff helper calls the CLI compare path", async () => {
  let observedArgs: string[] = [];
  const options: RuntimeOptions = {
    executable: "nightward",
    allowOnlineProviders: false,
    timeoutMs: 1000,
    execFileImpl: (_file, args, _options, callback) => {
      observedArgs = args;
      callback(
        null,
        JSON.stringify({
          generated_at: "2026-05-01T00:00:00Z",
          base: "/tmp/old.json",
          head: "/tmp/new.json",
          summary: {
            added: 1,
            removed: 0,
            changed: 0,
            max_added_severity: "high",
          },
          added: [],
          removed: [],
          changed: [],
        }),
        "",
      );
    },
  };

  const diff = await reportDiff(options, "/tmp/old.json", "/tmp/new.json");

  assert.equal(diff.summary.added, 1);
  assert.deepEqual(observedArgs, [
    "report",
    "diff",
    "--from",
    "/tmp/old.json",
    "--to",
    "/tmp/new.json",
  ]);
});

test("report history helper loads read-only history and selects latest pair", async () => {
  let observedArgs: string[] = [];
  const options: RuntimeOptions = {
    executable: "nightward",
    allowOnlineProviders: false,
    timeoutMs: 1000,
    execFileImpl: (_file, args, _options, callback) => {
      observedArgs = args;
      callback(
        null,
        JSON.stringify([
          {
            path: "/tmp/current.json",
            report_name: "current.json",
            mod_time: "2026-05-06T00:00:00Z",
            findings: 2,
            size_bytes: 100,
          },
          {
            path: "/tmp/previous.json",
            report_name: "previous.json",
            mod_time: "2026-05-05T00:00:00Z",
            findings: 1,
            size_bytes: 100,
          },
        ]),
        "",
      );
    },
  };

  const history = await reportHistory(options);
  const pair = latestReportPair(history);

  assert.deepEqual(observedArgs, ["report", "history", "--json"]);
  assert.equal(pair.base.path, "/tmp/previous.json");
  assert.equal(pair.head.path, "/tmp/current.json");
});

test("latest report pair errors when history cannot be compared", () => {
  assert.throws(
    () => latestReportPair([]),
    (error) =>
      error instanceof NightwardCommandError &&
      /At least two saved Nightward reports/.test(error.message),
  );
});

test("report diff helper surfaces redacted CLI failures", async () => {
  const options: RuntimeOptions = {
    executable: "nightward",
    allowOnlineProviders: false,
    timeoutMs: 1000,
    execFileImpl: (_file, _args, _options, callback) => {
      const error = new Error("exit 1") as NodeJS.ErrnoException;
      callback(error, "", "parse failed: API_TOKEN=secret-fixture-value\nmore");
    },
  };

  await assert.rejects(
    () => reportDiff(options, "/tmp/old.json", "/tmp/new.json"),
    (error) =>
      error instanceof NightwardCommandError &&
      /report diff/.test(error.command) &&
      /API_TOKEN=\[redacted\]/.test(error.message),
  );
});

function baseAnalysisJSON(): string {
  return JSON.stringify({
    generated_at: "2026-05-01T00:00:00Z",
    mode: "home",
    summary: {
      total_subjects: 0,
      total_signals: 0,
      signals_by_severity: {},
      signals_by_category: {},
      signals_by_provider: {},
      highest_severity: "info",
      provider_warnings: 0,
      no_known_risk_signals: true,
    },
    providers: [],
    subjects: [],
    signals: [],
  });
}
