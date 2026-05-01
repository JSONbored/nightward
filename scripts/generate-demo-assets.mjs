#!/usr/bin/env node
import { execFileSync, spawn } from "node:child_process";
import { chmodSync, existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, statSync, writeFileSync } from "node:fs";
import { tmpdir, userInfo } from "node:os";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const fixtureHome = join(repoRoot, "testdata", "homes", "policy");
const publicHome = "/tmp/nightward-fixture-home";
const publicHost = "nightward-fixture";
const generatedAt = "2026-05-01T00:00:00Z";
const outputDir = join(repoRoot, "site", "public", "demo");
const scanOutput = join(outputDir, "nightward-sample-scan.json");
const htmlOutput = join(outputDir, "nightward-sample-report.html");
const screenshotOutput = join(outputDir, "nightward-sample-report.png");
const tempDir = mkdtempSync(join(tmpdir(), "nightward-demo-"));
const rawScan = join(tempDir, "raw-scan.json");
let sourceHost = "";

function run(command, args, options = {}) {
  execFileSync(command, args, {
    cwd: repoRoot,
    env: { ...process.env, ...options.env },
    stdio: options.stdio ?? "pipe",
  });
}

function screenshotExists() {
  return existsSync(screenshotOutput) && statSync(screenshotOutput).size > 0;
}

function screenshotSize() {
  return existsSync(screenshotOutput) ? statSync(screenshotOutput).size : 0;
}

function captureScreenshot(chrome, args) {
  return new Promise((resolvePromise, rejectPromise) => {
    const child = spawn(chrome, args, {
      cwd: repoRoot,
      env: process.env,
      detached: true,
      stdio: ["ignore", "pipe", "pipe"],
    });
    let stderr = "";
    let completedAfterScreenshot = false;
    let timedOutWithScreenshot = false;
    let settled = false;
    let timeout;
    let poll;
    let lastScreenshotSize = 0;
    let stableScreenshotChecks = 0;

    const killGroup = (signal) => {
      if (!child.pid) {
        return;
      }
      try {
        process.kill(-child.pid, signal);
      } catch {
        try {
          child.kill(signal);
        } catch {
          // The process may already be gone.
        }
      }
    };

    const finish = (callback) => {
      if (settled) {
        return;
      }
      settled = true;
      clearTimeout(timeout);
      clearInterval(poll);
      callback();
    };

    poll = setInterval(() => {
      const size = screenshotSize();
      if (size > 0 && size === lastScreenshotSize) {
        stableScreenshotChecks += 1;
      } else {
        stableScreenshotChecks = 0;
        lastScreenshotSize = size;
      }
      if (stableScreenshotChecks >= 2) {
        completedAfterScreenshot = true;
        clearInterval(poll);
        killGroup("SIGTERM");
        setTimeout(() => killGroup("SIGKILL"), 1_500).unref();
      }
    }, 250);

    timeout = setTimeout(() => {
      timedOutWithScreenshot = screenshotExists();
      killGroup("SIGTERM");
      setTimeout(() => killGroup("SIGKILL"), 1_500).unref();
    }, 20_000);

    child.stderr.on("data", (chunk) => {
      stderr += chunk.toString();
    });
    child.on("error", (error) => {
      finish(() => rejectPromise(error));
    });
    child.on("close", (code, signal) => {
      finish(() => {
        if (code === 0 || ((completedAfterScreenshot || timedOutWithScreenshot) && screenshotExists())) {
          if (timedOutWithScreenshot) {
            console.warn("Chrome timed out after writing the demo screenshot; keeping the completed PNG.");
          }
          resolvePromise();
          return;
        }
        rejectPromise(new Error(`failed to capture demo screenshot: ${stderr || signal || code}`));
      });
    });
  });
}

function scrubValue(value, rawHost) {
  if (typeof value === "string") {
    let scrubbed = value.split(fixtureHome).join(publicHome);
    if (rawHost) {
      scrubbed = scrubbed.split(rawHost).join(publicHost);
    }
    return scrubbed;
  }
  if (Array.isArray(value)) {
    return value.map((entry) => scrubValue(entry, rawHost));
  }
  if (value && typeof value === "object") {
    return Object.fromEntries(Object.entries(value).map(([key, entry]) => [key, scrubValue(entry, rawHost)]));
  }
  return value;
}

function deterministicReport(rawReport) {
  const rawHost = rawReport.hostname || "";
  const report = scrubValue(rawReport, rawHost);
  report.generated_at = generatedAt;
  report.hostname = publicHost;
  report.home = publicHome;
  for (const item of report.items || []) {
    if (item.mod_time) {
      item.mod_time = generatedAt;
    }
  }
  return report;
}

function findChrome() {
  if (process.env.NIGHTWARD_CHROME && existsSync(process.env.NIGHTWARD_CHROME)) {
    return process.env.NIGHTWARD_CHROME;
  }
  const candidates = [
    "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
    "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
    "/Applications/Chromium.app/Contents/MacOS/Chromium",
    "/usr/bin/google-chrome",
    "/usr/bin/chromium",
    "/usr/bin/chromium-browser",
  ];
  return candidates.find((candidate) => existsSync(candidate)) || "";
}

function assertScrubbed(label, bytes) {
  const text = bytes.toString("utf8");
  const forbidden = [
    fixtureHome,
    "/Users/",
    userInfo().username,
    sourceHost,
    "super-secret-value",
  ].filter(Boolean);
  const leaked = forbidden.filter((needle) => text.includes(needle));
  if (leaked.length > 0) {
    throw new Error(`${label} contains unsanitized demo material: ${leaked.join(", ")}`);
  }
}

try {
  mkdirSync(outputDir, { recursive: true });
  run("go", ["run", "./cmd/nw", "scan", "--json", "--output", rawScan], {
    env: { NIGHTWARD_HOME: fixtureHome },
  });

  const rawReport = JSON.parse(readFileSync(rawScan, "utf8"));
  sourceHost = rawReport.hostname || "";
  const report = deterministicReport(rawReport);
  writeFileSync(scanOutput, `${JSON.stringify(report, null, 2)}\n`, { mode: 0o644 });
  assertScrubbed("sample scan", readFileSync(scanOutput));

  run("go", ["run", "./cmd/nw", "report", "html", "--input", scanOutput, "--output", htmlOutput], {
    env: { NIGHTWARD_HOME: fixtureHome },
  });
  const html = readFileSync(htmlOutput, "utf8").replace(/[ \t]+$/gm, "");
  writeFileSync(htmlOutput, html.endsWith("\n") ? html : `${html}\n`, { mode: 0o644 });
  chmodSync(htmlOutput, 0o644);
  assertScrubbed("sample HTML report", readFileSync(htmlOutput));

  const chrome = findChrome();
  if (!chrome) {
    throw new Error("Chrome/Chromium was not found. Set NIGHTWARD_CHROME to generate the demo screenshot.");
  }
  const chromeProfile = join(tempDir, "chrome-profile");
  rmSync(screenshotOutput, { force: true });
  await captureScreenshot(chrome, [
    "--headless=new",
    "--disable-background-networking",
    "--disable-gpu",
    "--hide-scrollbars",
    "--no-first-run",
    "--no-default-browser-check",
    "--no-sandbox",
    `--user-data-dir=${chromeProfile}`,
    "--window-size=1440,1100",
    `--screenshot=${screenshotOutput}`,
    pathToFileURL(htmlOutput).href,
  ]);
  chmodSync(screenshotOutput, 0o644);

  console.log(`Generated ${relative(repoRoot, scanOutput)}`);
  console.log(`Generated ${relative(repoRoot, htmlOutput)}`);
  console.log(`Generated ${relative(repoRoot, screenshotOutput)}`);
} finally {
  rmSync(tempDir, { recursive: true, force: true });
}
