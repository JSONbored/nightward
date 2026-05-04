#!/usr/bin/env node
import { execFileSync } from "node:child_process";
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
import { dirname, join, resolve } from "node:path";
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
const toolPath = `${process.env.HOME}/.cargo/bin:${process.env.HOME}/go/bin:/opt/homebrew/bin:${process.env.PATH || ""}`;

const views = [
  ["overview", "overview", "Overview"],
  ["findings", "findings", "Findings"],
  ["analysis", "analysis", "Analysis"],
  ["fix-plan", "fix-plan", "Fix Plan"],
  ["inventory", "inventory", "Inventory"],
  ["backup", "backup", "Backup"],
  ["help", "help", "Help"],
];

function run(command, args, options = {}) {
  execFileSync(command, args, {
    cwd: repoRoot,
    env: { ...process.env, PATH: toolPath, ...options.env },
    stdio: options.stdio ?? "inherit",
  });
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
Type "NIGHTWARD_TUI_VIEW=${view} target/debug/nw tui --input site/public/demo/nightward-sample-scan.json"
Enter
Show
Sleep 3500ms
Type "q"
Sleep 200ms
`,
  );
  return tape;
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
  const stamps = Number.isFinite(duration) && duration > 0
    ? [0.35, 0.45, 0.55, 0.65, 0.75].map((pct) => (duration * pct).toFixed(2))
    : ["1.20", "1.60", "2.00", "2.40", "2.80"];
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
  const largest = candidates.sort((a, b) => statSync(b).size - statSync(a).size)[0];
  copyFileSync(largest, outputPng);
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

  mkdirSync(outputDir, { recursive: true });
  run("cargo", ["build", "-p", "nightward-cli", "--bin", "nw"]);

  for (const [slug, view, label] of views) {
    const gif = join(tempDir, `${slug}.gif`);
    const png = join(outputDir, `${slug}.png`);
    const tape = writeStillTape(view, gif);
    console.log(`capturing ${label}`);
    run("vhs", [tape]);
    extractBestPng(gif, png);
    assertScrubbed(`${label} PNG`, png);
  }

  copyFileSync(join(outputDir, "overview.png"), legacyPng);
  copyFileSync(join(outputDir, "overview.png"), posterOutput);

  console.log("capturing walkthrough GIF");
  run("vhs", [tapePath]);
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
