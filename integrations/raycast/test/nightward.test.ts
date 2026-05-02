import assert from "node:assert/strict";
import test from "node:test";
import {
  analysisReport,
  explainFinding,
  exportFixPlanMarkdown,
  fixPlan,
  normalizePreferences,
  providersDoctor,
  reportsDir,
  runNightwardJSON,
  type RuntimeOptions,
} from "../src/nightward";
import {
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

  const result = await runNightwardJSON<{ ok: boolean }>(["doctor", "--json"], options);
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

  const result = await runNightwardJSON<{ ok: boolean }>(["scan", "--json"], options);
  assert.deepEqual(result, { ok: true });
  assert.deepEqual(calls, ["nw", "nightward"]);
});

test("reports directory follows the optional home override", () => {
  assert.equal(reportsDir("/tmp/example"), "/tmp/example/.local/state/nightward/reports");
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
