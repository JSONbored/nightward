#!/usr/bin/env node
import { createHash } from "node:crypto";
import { execFileSync, spawn } from "node:child_process";
import { chmodSync, existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, statSync, writeFileSync } from "node:fs";
import { tmpdir, userInfo } from "node:os";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const fixtureHome = join(repoRoot, "testdata", "homes", "policy");
const publicHome = "/tmp/nightward-fixture-home";
const publicHost = "nightward-fixture";
const generatedAt = "2026-04-30T18:00:00Z";
const outputDir = join(repoRoot, "site", "public", "demo");
const scanOutput = join(outputDir, "nightward-sample-scan.json");
const htmlOutput = join(outputDir, "nightward-sample-report.html");
const screenshotOutput = join(outputDir, "nightward-sample-report.png");
const ogImageOutput = join(repoRoot, "site", "public", "og-image.png");
const faviconPath = join(repoRoot, "site", "public", "favicon.svg");
const tempDir = mkdtempSync(join(tmpdir(), "nightward-demo-"));
const rawScan = join(tempDir, "raw-scan.json");
const toolPath = `${process.env.HOME}/.cargo/bin:/opt/homebrew/bin:${process.env.PATH || ""}`;
let sourceHost = "";

function run(command, args, options = {}) {
  execFileSync(command, args, {
    cwd: repoRoot,
    env: { ...process.env, PATH: toolPath, ...options.env },
    stdio: options.stdio ?? "pipe",
  });
}

function screenshotExists(path) {
  return existsSync(path) && statSync(path).size > 0;
}

function screenshotSize(path) {
  return existsSync(path) ? statSync(path).size : 0;
}

function captureScreenshot(chrome, args, outputPath) {
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
      const size = screenshotSize(outputPath);
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
      timedOutWithScreenshot = screenshotExists(outputPath);
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
        if (code === 0 || ((completedAfterScreenshot || timedOutWithScreenshot) && screenshotExists(outputPath))) {
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
    item.id = stableId(["item", item.tool || "", item.path || ""]);
    if (item.mod_time) {
      item.mod_time = generatedAt;
    }
  }
  for (const finding of report.findings || []) {
    finding.id = `${finding.rule}-${stableId([
      finding.rule || "",
      finding.tool || "",
      finding.path || "",
      finding.server || "",
      finding.evidence || "",
    ])}`;
  }
  return report;
}

function stableId(parts) {
  const hash = createHash("sha256");
  for (const part of parts) {
    hash.update(String(part));
    hash.update("\0");
  }
  return hash.digest("hex").slice(0, 12);
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

function writeOgPreviewHTML() {
  const logo = readFileSync(faviconPath, "utf8");
  const preview = join(tempDir, "og-preview.html");
  writeFileSync(
    preview,
    `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Nightward preview</title>
    <style>
      * { box-sizing: border-box; }
      body {
        margin: 0;
        width: 1200px;
        height: 630px;
        background: #071014;
        color: #f7fffd;
        font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      }
      main {
        display: grid;
        grid-template-columns: 1fr 340px;
        gap: 64px;
        align-items: center;
        width: 100%;
        height: 100%;
        padding: 56px 82px;
        background:
          linear-gradient(90deg, rgba(15, 118, 110, 0.26), transparent 58%),
          #071014;
      }
      .eyebrow {
        color: #5eead4;
        font-size: 24px;
        font-weight: 760;
        letter-spacing: 0;
        text-transform: uppercase;
      }
      h1 {
        margin: 18px 0 20px;
        max-width: 760px;
        font-size: 66px;
        line-height: 0.98;
        letter-spacing: 0;
      }
      p {
        margin: 0;
        max-width: 780px;
        color: #c7f9f4;
        font-size: 28px;
        line-height: 1.26;
      }
      .command {
        display: inline-block;
        margin-top: 32px;
        border: 1px solid rgba(94, 234, 212, 0.4);
        border-radius: 8px;
        padding: 18px 22px;
        background: rgba(255, 255, 255, 0.06);
        color: #ffffff;
        font-family: "SFMono-Regular", Consolas, "Liberation Mono", monospace;
        font-size: 28px;
      }
      .mark {
        display: grid;
        place-items: center;
        width: 300px;
        height: 300px;
        margin-left: auto;
        border: 1px solid rgba(94, 234, 212, 0.22);
        border-radius: 8px;
        background: rgba(255, 255, 255, 0.035);
      }
      .mark svg {
        width: 168px;
        height: 168px;
      }
    </style>
  </head>
  <body>
    <main>
      <section>
        <div class="eyebrow">Nightward</div>
        <h1>Find AI-tool risks before you sync.</h1>
        <p>Scan agent configs, MCP servers, and dotfiles for secrets, broad local access, and machine-only state.</p>
        <div class="command">npx @jsonbored/nightward</div>
      </section>
      <aside class="mark" aria-hidden="true">${logo}</aside>
    </main>
  </body>
</html>
`,
    { mode: 0o644 },
  );
  return preview;
}

try {
  mkdirSync(outputDir, { recursive: true });
  run("cargo", ["run", "--quiet", "--bin", "nw", "--", "scan", "--json", "--output", rawScan], {
    env: { NIGHTWARD_HOME: fixtureHome },
  });

  const rawReport = JSON.parse(readFileSync(rawScan, "utf8"));
  sourceHost = rawReport.hostname || "";
  const report = deterministicReport(rawReport);
  writeFileSync(scanOutput, `${JSON.stringify(report, null, 2)}\n`, { mode: 0o644 });
  assertScrubbed("sample scan", readFileSync(scanOutput));

  run("cargo", ["run", "--quiet", "--bin", "nw", "--", "report", "html", "--input", scanOutput, "--output", htmlOutput], {
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
  ], screenshotOutput);
  chmodSync(screenshotOutput, 0o644);
  assertScrubbed("sample report screenshot", readFileSync(screenshotOutput));

  const ogPreview = writeOgPreviewHTML();
  const ogChromeProfile = join(tempDir, "chrome-og-profile");
  rmSync(ogImageOutput, { force: true });
  await captureScreenshot(chrome, [
    "--headless=new",
    "--disable-background-networking",
    "--disable-gpu",
    "--hide-scrollbars",
    "--no-first-run",
    "--no-default-browser-check",
    "--no-sandbox",
    `--user-data-dir=${ogChromeProfile}`,
    "--window-size=1200,630",
    `--screenshot=${ogImageOutput}`,
    pathToFileURL(ogPreview).href,
  ], ogImageOutput);
  chmodSync(ogImageOutput, 0o644);
  assertScrubbed("Open Graph image", readFileSync(ogImageOutput));

  console.log(`Generated ${relative(repoRoot, scanOutput)}`);
  console.log(`Generated ${relative(repoRoot, htmlOutput)}`);
  console.log(`Generated ${relative(repoRoot, screenshotOutput)}`);
  console.log(`Generated ${relative(repoRoot, ogImageOutput)}`);
} finally {
  rmSync(tempDir, { recursive: true, force: true });
}
