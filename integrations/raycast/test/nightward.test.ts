import assert from "node:assert/strict";
import test from "node:test";
import { normalizePreferences, reportsDir, runNightwardJSON, type RuntimeOptions } from "../src/nightward";

test("normalizes empty preferences to nw without a home override", () => {
  const options = normalizePreferences({});
  assert.equal(options.executable, "nw");
  assert.equal(options.homeOverride, undefined);
  assert.equal(options.timeoutMs, 20000);
});

test("passes NIGHTWARD_HOME and parses JSON output", async () => {
  let observedHome = "";
  const options: RuntimeOptions = {
    executable: "nightward",
    homeOverride: "/tmp/nightward-home",
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

test("falls back from nw to nightward when nw is missing", async () => {
  const calls: string[] = [];
  const options: RuntimeOptions = {
    executable: "nw",
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
