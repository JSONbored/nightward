#!/usr/bin/env node
import { execFileSync, spawnSync } from "node:child_process";
import {
  copyFileSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  statSync,
  writeFileSync,
} from "node:fs";
import { tmpdir, userInfo } from "node:os";
import { dirname, isAbsolute, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const fixtureScan = join(repoRoot, "site", "public", "demo", "nightward-sample-scan.json");
const outputDir = join(repoRoot, "site", "public", "demo", "tui");
const legacyPng = join(repoRoot, "site", "public", "demo", "nightward-opentui.png");
const legacyGif = join(repoRoot, "site", "public", "demo", "nightward-opentui.gif");
const videoOutput = join(outputDir, "nightward-opentui.webm");
const posterOutput = join(outputDir, "poster.png");
const tapePath = join(repoRoot, "docs", "demo", "nightward-tui.tape");
const tempDir = mkdtempSync(join(tmpdir(), "nightward-tui-media-"));
let mediaHome = join(tempDir, "home");
const toolPath = `${process.env.HOME}/.cargo/bin:${process.env.HOME}/go/bin:/opt/homebrew/bin:${process.env.PATH || ""}`;

const views = [
  ["overview", "overview", "Overview"],
  ["findings", "findings", "Findings"],
  ["analysis", "analysis", "Analysis"],
  ["fix-plan", "fix-plan", "Fix Plan"],
  ["inventory", "inventory", "Inventory"],
  ["backup", "backup", "Backup"],
  ["actions", "actions", "Actions"],
  ["mcp-approvals", "mcp-approvals", "MCP Approvals"],
  ["help", "help", "Help"],
];

function run(command, args, options = {}) {
  execFileSync(command, args, {
    cwd: repoRoot,
    env: { ...process.env, PATH: toolPath, ...options.env },
    stdio: options.stdio ?? "inherit",
  });
}

function runChecked(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: repoRoot,
    env: { ...process.env, PATH: toolPath, ...options.env },
    encoding: "utf8",
    stdio: "pipe",
  });
  if (result.error || result.status !== 0) {
    const output = [result.stdout, result.stderr].filter(Boolean).join("\n").trim();
    if (output) {
      console.error(output);
    }
    throw result.error ?? new Error(`${command} ${args.join(" ")} failed with ${result.status}`);
  }
  return result;
}

function requireTool(command, args = ["--version"]) {
  try {
    execFileSync(command, args, {
      cwd: repoRoot,
      env: { ...process.env, PATH: toolPath },
      stdio: "ignore",
    });
  } catch {
    throw new Error(`${command} is required for TUI media generation`);
  }
}

function shellQuote(value) {
  return `'${String(value).replaceAll("'", "'\\''")}'`;
}

function nightwardEnvPrefix() {
  return `NIGHTWARD_HOME=${shellQuote(mediaHome)}`;
}

function writeStillTape(view, outputGif) {
  const tape = join(tempDir, `${view}.tape`);
  writeFileSync(
    tape,
    `Output "${outputGif}"

Set Shell "zsh"
Set FontSize 16
Set Width 1320
Set Height 760
Set Padding 18
Set BorderRadius 10
Set WindowBar Colorful
Set TypingSpeed 1ms
Set PlaybackSpeed 1.0
Set Framerate 12
Set Theme "TokyoNight"

Hide
Type "stty rows 36 cols 120"
Enter
Type "${nightwardEnvPrefix()} NIGHTWARD_TUI_CAPTURE=1 NIGHTWARD_TUI_CAPTURE_HOLD_MS=5000 NIGHTWARD_TUI_VIEW=${view} target/debug/nw tui --input site/public/demo/nightward-sample-scan.json"
Enter
Show
Sleep 5600ms
`,
  );
  return tape;
}

function writeWalkthroughTape() {
  const tape = join(tempDir, "nightward-tui-walkthrough.tape");
  const original = readFileSync(tapePath, "utf8");
  const command =
    'Type "target/debug/nw tui --input site/public/demo/nightward-sample-scan.json"';
  if (!original.includes(command)) {
    throw new Error(`walkthrough tape missing expected command pattern: ${command}`);
  }
  const text = original.replace(
    command,
    `Type "${nightwardEnvPrefix()} target/debug/nw tui --input site/public/demo/nightward-sample-scan.json"`,
  );
  writeFileSync(tape, text);
  return tape;
}

function resetMediaHomeState() {
  const resolvedMediaHome = resolve(mediaHome);
  const resolvedTempDir = resolve(tempDir);
  const tempRelative = relative(resolvedTempDir, resolvedMediaHome);
  const isTempChild =
    (tempRelative.length === 0 ||
      (!tempRelative.startsWith("..") && !isAbsolute(tempRelative)));
  const allowed =
    resolvedMediaHome === resolve("/tmp/nightward-fixture-home") || isTempChild;
  if (!allowed) {
    throw new Error(`refusing to reset unexpected media home: ${mediaHome}`);
  }
  for (const rel of [
    [".config", "nightward"],
    [".local", "state", "nightward"],
    [".cache", "nightward"],
  ]) {
    rmSync(join(resolvedMediaHome, ...rel), { recursive: true, force: true });
  }
}

function extractBestPng(inputGif, outputPng) {
  const durationRaw = execFileSync("ffprobe", [
    "-v",
    "error",
    "-show_entries",
    "format=duration",
    "-of",
    "default=noprint_wrappers=1:nokey=1",
    inputGif,
  ]).toString("utf8");
  const duration = Number.parseFloat(durationRaw);
  const stamps =
    Number.isFinite(duration) && duration > 0
      ? [0.42, 0.52, 0.62, 0.72].map((pct) => (duration * pct).toFixed(2))
      : ["2.10", "2.40", "2.70", "3.00"];
  const candidates = [];
  for (const stamp of stamps) {
    const candidate = join(tempDir, `${Date.now()}-${stamp}.png`);
    try {
      run("ffmpeg", [
        "-hide_banner",
        "-loglevel",
        "error",
        "-y",
        "-i",
        inputGif,
        "-ss",
        stamp,
        "-frames:v",
        "1",
        "-update",
        "1",
        candidate,
      ]);
      if (existsSync(candidate) && statSync(candidate).size > 0) {
        candidates.push(candidate);
      }
    } catch {
      // Keep trying later frames. VHS output timing can vary slightly by host.
    }
  }
  if (candidates.length === 0) {
    throw new Error(`failed to extract a still frame from ${inputGif}`);
  }
  copyFileSync(candidates[0], outputPng);
}

function assertScrubbed(label, path) {
  const text = readFileSync(path).toString("utf8");
  const forbidden = [
    "/Users/",
    userInfo().username,
    "super-secret-value",
    "ANTHROPIC_API_KEY",
    "OPENAI_API_KEY",
  ].filter(Boolean);
  const leaked = forbidden.filter((needle) => text.includes(needle));
  if (leaked.length > 0) {
    throw new Error(`${label} contains unsanitized fixture material: ${leaked.join(", ")}`);
  }
}

try {
  requireTool("cargo");
  requireTool("vhs");
  requireTool("ffmpeg", ["-version"]);
  requireTool("ffprobe", ["-version"]);
  if (!existsSync(fixtureScan)) {
    throw new Error("missing scrubbed sample scan; run `make demo-assets` first");
  }
  mediaHome = JSON.parse(readFileSync(fixtureScan, "utf8")).home || mediaHome;

  mkdirSync(outputDir, { recursive: true });
  resetMediaHomeState();
  mkdirSync(mediaHome, { recursive: true });
  run("cargo", ["build", "-p", "nightward-cli", "--bin", "nw"]);
  runChecked("target/debug/nw", ["disclosure", "accept", "--json"], {
    env: { NIGHTWARD_HOME: mediaHome },
  });
  runChecked(
    "target/debug/nw",
    ["approvals", "request", "backup.snapshot", "--client", "demo-mcp", "--json"],
    {
      env: { NIGHTWARD_HOME: mediaHome },
    },
  );

  for (const [slug, view, label] of views) {
    const gif = join(tempDir, `${slug}.gif`);
    const png = join(outputDir, `${slug}.png`);
    const tape = writeStillTape(view, gif);
    console.log(`capturing ${label}`);
    run("vhs", [tape], { env: { NIGHTWARD_HOME: mediaHome } });
    extractBestPng(gif, png);
    assertScrubbed(`${label} PNG`, png);
  }

  copyFileSync(join(outputDir, "overview.png"), legacyPng);
  copyFileSync(join(outputDir, "overview.png"), posterOutput);

  console.log("capturing walkthrough GIF");
  run("vhs", [writeWalkthroughTape()], { env: { NIGHTWARD_HOME: mediaHome } });
  assertScrubbed("walkthrough GIF", legacyGif);
  run("ffmpeg", [
    "-hide_banner",
    "-loglevel",
    "error",
    "-y",
    "-i",
    legacyGif,
    "-an",
    "-vf",
    "format=yuva420p",
    "-c:v",
    "libvpx-vp9",
    "-b:v",
    "0",
    "-crf",
    "38",
    videoOutput,
  ]);
  assertScrubbed("walkthrough WebM", videoOutput);

  console.log(`wrote ${outputDir}`);
} finally {
  rmSync(tempDir, { recursive: true, force: true });
}
