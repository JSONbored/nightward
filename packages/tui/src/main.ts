import { spawnSync } from "node:child_process";
import { readFileSync } from "node:fs";
import {
  Box,
  Text,
  createCliRenderer,
  instantiate,
  type CliRenderer,
  type Renderable
} from "@opentui/core";
import {
  filteredFindings,
  highestSeverity,
  nextSeverity,
  normalizeBundle,
  redact,
  selectedFinding,
  severityCounts,
  severityOrder,
  tabs,
  totalFindings,
  truncate,
  type NightwardBackupEntry,
  type NightwardBundle,
  type NightwardFinding,
  type NightwardFix,
  type NightwardItem,
  type NightwardReport,
  type NightwardSignal,
  type TuiState
} from "./model";

const colors = {
  bg: "#070A12",
  panel: "#0E1424",
  panel2: "#111A2E",
  line: "#26324A",
  text: "#E8EEF8",
  muted: "#7D8799",
  critical: "#FF4D6D",
  high: "#FF8A3D",
  medium: "#FFD166",
  low: "#67E8F9",
  info: "#A3E635",
  cyan: "#2DD4BF",
  blue: "#60A5FA",
  purple: "#A78BFA",
  green: "#34D399"
};

async function main() {
  const bundle = loadBundle(process.argv.slice(2));
  const renderer = await createCliRenderer({
    screenMode: "alternate-screen",
    consoleMode: "disabled",
    backgroundColor: colors.bg,
    useMouse: true,
    enableMouseMovement: true,
    exitOnCtrlC: true,
    targetFps: 30,
    maxFps: 60
  });

  const state: TuiState = {
    tab: 0,
    cursor: 0,
    severity: "",
    search: "",
    searchMode: false,
    status: "read-only"
  };

  let current: Renderable | undefined;
  const rerender = () => {
    if (current) {
      renderer.root.remove("app");
      current.destroyRecursively();
    }
    current = instantiate(renderer, appView(bundle, state, renderer));
    renderer.root.add(current);
    renderer.requestRender();
  };

  renderer.keyInput.on("keypress", (key) => {
    const name = key.name || key.sequence;
    if (state.searchMode) {
      if (name === "return" || name === "enter") {
        state.searchMode = false;
        state.cursor = 0;
        state.status = state.search ? `search: ${state.search}` : "search cleared";
      } else if (name === "escape") {
        state.searchMode = false;
        state.status = "search cancelled";
      } else if (name === "backspace") {
        state.search = state.search.slice(0, -1);
        state.cursor = 0;
      } else if (key.sequence && key.sequence.length === 1 && !key.ctrl && !key.meta) {
        state.search += key.sequence;
        state.cursor = 0;
      }
      rerender();
      return;
    }

    if (key.ctrl && name === "c") {
      renderer.destroy();
      process.exit(0);
    }
    switch (name) {
      case "q":
      case "escape":
        renderer.destroy();
        process.exit(0);
      case "tab":
      case "right":
      case "l":
        state.tab = (state.tab + 1) % tabs.length;
        state.cursor = 0;
        state.status = `opened ${tabs[state.tab]}`;
        break;
      case "left":
      case "h":
        state.tab = (state.tab + tabs.length - 1) % tabs.length;
        state.cursor = 0;
        state.status = `opened ${tabs[state.tab]}`;
        break;
      case "down":
      case "j":
        state.cursor += 1;
        state.status = "moved selection";
        break;
      case "up":
      case "k":
        state.cursor = Math.max(0, state.cursor - 1);
        state.status = "moved selection";
        break;
      case "/":
        state.searchMode = true;
        state.status = "type to search";
        break;
      case "s":
        state.severity = nextSeverity(state.severity);
        state.cursor = 0;
        state.status = state.severity ? `severity: ${state.severity}` : "severity: all";
        break;
      case "x":
        state.severity = "";
        state.search = "";
        state.cursor = 0;
        state.status = "filters cleared";
        break;
      case "1":
      case "2":
      case "3":
      case "4":
      case "5":
      case "6":
      case "7":
        state.tab = Number(name) - 1;
        state.cursor = 0;
        state.status = `opened ${tabs[state.tab]}`;
        break;
    }
    rerender();
  });

  rerender();
  renderer.start();
}

function loadBundle(args: string[]): NightwardBundle {
  const inputIndex = args.indexOf("--input");
  if (inputIndex >= 0 && args[inputIndex + 1]) {
    return normalizeBundle(JSON.parse(readFileSync(args[inputIndex + 1], "utf8")) as NightwardReport | NightwardBundle);
  }

  const bin = process.env.NIGHTWARD_BIN || "nightward";
  const result = spawnSync(bin, ["scan", "--json"], { encoding: "utf8" });
  if (result.status === 0 && result.stdout.trim()) {
    return normalizeBundle(JSON.parse(result.stdout) as NightwardReport);
  }
  throw new Error(`could not load Nightward report. Pass --input scan.json or set NIGHTWARD_BIN. ${result.stderr || ""}`.trim());
}

function appView(bundle: NightwardBundle, state: TuiState, renderer: CliRenderer) {
  const width = Math.max(96, renderer.terminalWidth || 120);
  const height = Math.max(28, renderer.terminalHeight || 36);
  return Box(
    {
      id: "app",
      width,
      height,
      flexDirection: "row",
      backgroundColor: colors.bg
    },
    sidebar(bundle, state, height),
    mainPanel(bundle, state, width - 30, height)
  );
}

function sidebar(bundle: NightwardBundle, state: TuiState, height: number) {
  const report = bundle.scan;
  const counts = severityCounts(report);
  const risk = highestSeverity(report);
  return Box(
    {
      width: 30,
      height,
      flexDirection: "column",
      paddingX: 1,
      paddingY: 1,
      backgroundColor: colors.panel,
      border: ["right"],
      borderColor: colors.line
    },
    Text({ height: 1, content: "NIGHTWARD", fg: colors.cyan, attributes: 1 }),
    Text({ height: 1, content: "AI config risk console", fg: colors.muted }),
    spacer(),
    badge(`risk ${risk}`, colorForSeverity(risk), 24),
    Text({ height: 1, content: `${totalFindings(report)} findings`, fg: colors.text }),
    Text({ height: 1, content: `${counts.critical} critical  ${counts.high} high`, fg: colors.muted }),
    spacer(),
    ...tabs.map((tab, index) => navItem(index, tab, state.tab === index)),
    spacer(),
    Text({ height: 1, content: "filters", fg: colors.muted, attributes: 1 }),
    Text({ height: 1, content: `severity  ${state.severity || "all"}`, fg: state.severity ? colorForSeverity(state.severity) : colors.text }),
    Text({ height: 1, content: `search    ${state.search || "none"}`, fg: state.search ? colors.cyan : colors.text }),
    spacer(),
    Text({ height: 1, content: "keys", fg: colors.muted, attributes: 1 }),
    Text({ height: 1, content: "tab/1-7 navigate", fg: colors.muted }),
    Text({ height: 1, content: "/ search  s severity", fg: colors.muted }),
    Text({ height: 1, content: "q quit", fg: colors.muted })
  );
}

function mainPanel(bundle: NightwardBundle, state: TuiState, width: number, height: number) {
  return Box(
    {
      width,
      height,
      flexDirection: "column",
      paddingX: 2,
      paddingY: 1,
      backgroundColor: colors.bg
    },
    header(bundle.scan, state),
    content(bundle, state, width, Math.max(10, height - 8)),
    footer(state)
  );
}

function header(report: NightwardReport, state: TuiState) {
  const generated = report.generated_at ? new Date(report.generated_at).toLocaleString() : "unknown";
  return Box(
    {
      height: 6,
      flexDirection: "column",
      border: ["bottom"],
      borderColor: colors.line
    },
    Text({ height: 1, content: "Review what AI tools can read, run, and accidentally sync.", fg: colors.text, attributes: 1 }),
    Text({ height: 1, content: `generated ${generated}`, fg: colors.muted }),
    Box(
      { height: 3, flexDirection: "row", columnGap: 1, paddingY: 1 },
      statCard("findings", String(totalFindings(report)), colorForSeverity(highestSeverity(report))),
      statCard("items", String(report.summary?.total_items ?? report.items?.length ?? 0), colors.blue),
      statCard("mode", report.scan_mode || (report.workspace ? "workspace" : "home"), colors.purple),
      statCard("active", tabs[state.tab], colors.cyan)
    )
  );
}

function content(bundle: NightwardBundle, state: TuiState, width: number, height: number) {
  const report = bundle.scan;
  switch (state.tab) {
    case 1:
      return findingsView(report, state, width, height);
    case 2:
      return analysisView(bundle, state, height);
    case 3:
      return fixPlanView(bundle, state, width, height);
    case 4:
      return inventoryView(report, state, width, height);
    case 5:
      return backupView(bundle, state, height);
    case 6:
      return helpView(height);
    default:
      return overviewView(report, height);
  }
}

function overviewView(report: NightwardReport, height: number) {
  const counts = severityCounts(report);
  return Box(
    { height, flexDirection: "row", columnGap: 2, paddingY: 1 },
    Box(
      { width: "50%", height: "100%", flexDirection: "column", border: true, borderStyle: "rounded", borderColor: colorForSeverity(highestSeverity(report)), padding: 1, backgroundColor: colors.panel },
      Text({ height: 1, content: "risk posture", fg: colors.text, attributes: 1 }),
      spacer(),
      ...severityOrder.map((severity) => Text({ height: 1, content: `${severity.padEnd(9)} ${bar(counts[severity], Math.max(1, totalFindings(report)), 20)} ${counts[severity]}`, fg: colorForSeverity(severity) })),
      spacer(),
      Text({ height: 1, content: "next action", fg: colors.cyan, attributes: 1 }),
      Text({ height: 2, content: nextAction(report), fg: colors.text, wrapMode: "word" })
    ),
    Box(
      { flexGrow: 1, height: "100%", flexDirection: "column", border: true, borderStyle: "rounded", borderColor: colors.blue, padding: 1, backgroundColor: colors.panel2 },
      Text({ height: 1, content: "recent findings", fg: colors.text, attributes: 1 }),
      spacer(),
      ...filteredFindings(report, { severity: "", search: "" }).slice(0, 9).map((finding) => findingRow(finding, false, 64))
    )
  );
}

function findingsView(report: NightwardReport, state: TuiState, _width: number, height: number) {
  const findings = filteredFindings(report, state);
  const selected = selectedFinding(report, state);
  return Box(
    { height, flexDirection: "row", columnGap: 2, paddingY: 1 },
    Box(
      { width: "48%", height: "100%", flexDirection: "column", border: true, borderStyle: "rounded", borderColor: colors.line, padding: 1, backgroundColor: colors.panel },
      Text({ height: 1, content: `${findings.length} matching findings`, fg: colors.text, attributes: 1 }),
      Text({ height: 1, content: `severity=${state.severity || "all"} search=${state.search || "none"}`, fg: colors.muted }),
      spacer(),
      ...findings.slice(0, 12).map((finding, index) => findingRow(finding, index === state.cursor, 72))
    ),
    detailPanel(selected)
  );
}

function analysisView(bundle: NightwardBundle, state: TuiState, height: number) {
  const sourceSignals = bundle.analysis?.signals?.length ? bundle.analysis.signals : fallbackSignals(bundle.scan);
  const signals = sourceSignals.filter((signal) => {
    if (state.severity && (signal.severity || "").toLowerCase() !== state.severity) {
      return false;
    }
    const query = state.search.trim().toLowerCase();
    if (!query) return true;
    return [signal.provider, signal.rule, signal.category, signal.message, signal.evidence]
      .filter(Boolean)
      .join("\n")
      .toLowerCase()
      .includes(query);
  });
  const selected = signals[Math.min(state.cursor, Math.max(0, signals.length - 1))];
  return Box(
    { height, flexDirection: "row", columnGap: 2, paddingY: 1 },
    Box(
      { width: "48%", height: "100%", flexDirection: "column", border: true, borderStyle: "rounded", borderColor: colors.purple, padding: 1, backgroundColor: colors.panel },
      Text({ height: 1, content: `${signals.length} normalized signals`, fg: colors.text, attributes: 1 }),
      Text({ height: 1, content: `${bundle.analysis?.summary?.provider_warnings || 0} provider warnings`, fg: colors.muted }),
      spacer(),
      ...signals.slice(0, 12).map((signal, index) => signalRow(signal, index === state.cursor))
    ),
    signalDetailPanel(selected)
  );
}

function fixPlanView(bundle: NightwardBundle, state: TuiState, _width: number, height: number) {
  const fixes = bundle.fix_plan?.fixes || fallbackFixes(bundle.scan);
  const filtered = fixes.filter((fix) => {
    if (state.severity && (fix.severity || "").toLowerCase() !== state.severity) {
      return false;
    }
    const query = state.search.trim().toLowerCase();
    if (!query) return true;
    return [fix.finding_id, fix.rule, fix.status, fix.summary, ...(fix.steps || [])]
      .filter(Boolean)
      .join("\n")
      .toLowerCase()
      .includes(query);
  });
  const selected = filtered[Math.min(state.cursor, Math.max(0, filtered.length - 1))];
  return Box(
    { height, flexDirection: "row", columnGap: 2, paddingY: 1 },
    Box(
      { width: "45%", height: "100%", flexDirection: "column", border: true, borderStyle: "rounded", borderColor: colors.green, padding: 1, backgroundColor: colors.panel },
      Text({ height: 1, content: "plan-only remediation", fg: colors.text, attributes: 1 }),
      Text({ height: 1, content: "No config is mutated from this interface.", fg: colors.muted }),
      spacer(),
      ...filtered.slice(0, 12).map((fix, index) => fixRow(fix, index === state.cursor))
    ),
    fixDetailPanel(selected)
  );
}

function inventoryView(report: NightwardReport, state: TuiState, _width: number, height: number) {
  const items = report.items || [];
  const selected = items[Math.min(state.cursor, Math.max(0, items.length - 1))];
  return Box(
    { height, flexDirection: "row", columnGap: 2, paddingY: 1 },
    Box(
      { width: "55%", height: "100%", flexDirection: "column", border: true, borderStyle: "rounded", borderColor: colors.blue, padding: 1, backgroundColor: colors.panel },
      Text({ height: 1, content: `${items.length} discovered config paths`, fg: colors.text, attributes: 1 }),
      spacer(),
      ...items.slice(0, 14).map((item, index) => inventoryRow(item, index === state.cursor))
    ),
    Box(
      { flexGrow: 1, height: "100%", flexDirection: "column", border: true, borderStyle: "rounded", borderColor: colors.line, padding: 1, backgroundColor: colors.panel2 },
      Text({ height: 1, content: "path detail", fg: colors.text, attributes: 1 }),
      spacer(),
      Text({ height: 1, content: selected?.tool || "none", fg: colors.cyan }),
      Text({ height: 1, content: selected?.classification || "", fg: colors.muted }),
      Text({ height: 2, content: selected?.path || "", fg: colors.text, wrapMode: "word" }),
      Text({ height: 3, content: selected?.reason || "", fg: colors.muted, wrapMode: "word" })
    )
  );
}

function backupView(bundle: NightwardBundle, state: TuiState, height: number) {
  const entries = bundle.backup_plan?.entries?.length ? bundle.backup_plan.entries : fallbackBackupEntries(bundle.scan);
  const selected = entries[Math.min(state.cursor, Math.max(0, entries.length - 1))];
  const summary = bundle.backup_plan?.summary || backupSummary(entries);
  return Box(
    { height, flexDirection: "row", columnGap: 2, paddingY: 1 },
    Box(
      { width: "50%", height: "100%", flexDirection: "column", border: true, borderStyle: "rounded", borderColor: colors.medium, padding: 1, backgroundColor: colors.panel },
      Text({ height: 1, content: "backup preview", fg: colors.text, attributes: 1 }),
      Text({ height: 1, content: `${summary?.included || 0} include  ${summary?.review || 0} review  ${summary?.excluded || 0} exclude`, fg: colors.muted }),
      spacer(),
      ...entries.slice(0, 13).map((entry, index) => backupRow(entry, index === state.cursor))
    ),
    backupDetailPanel(selected, bundle.backup_plan?.target_root)
  );
}

function helpView(height: number) {
  return Box(
    { height, flexDirection: "column", paddingY: 1 },
    Box(
      { width: "100%", height: 12, border: true, borderStyle: "rounded", borderColor: colors.purple, padding: 1, backgroundColor: colors.panel },
      Text({ height: 1, content: "keyboard", fg: colors.text, attributes: 1 }),
      spacer(),
      Text({ height: 1, content: "tab/1-7         switch section", fg: colors.text }),
      Text({ height: 1, content: "right/l          next section", fg: colors.text }),
      Text({ height: 1, content: "left/h           previous section", fg: colors.text }),
      Text({ height: 1, content: "up/down/j/k      move selection", fg: colors.text }),
      Text({ height: 1, content: "s                cycle severity filter", fg: colors.text }),
      Text({ height: 1, content: "/                search findings", fg: colors.text }),
      Text({ height: 1, content: "x                clear filters", fg: colors.text }),
      Text({ height: 1, content: "q/esc            quit", fg: colors.text })
    )
  );
}

function detailPanel(finding: NightwardFinding | undefined) {
  if (!finding) {
    return emptyPanel("finding detail", "No matching finding selected.");
  }
  return Box(
    { flexGrow: 1, height: "100%", flexDirection: "column", border: true, borderStyle: "rounded", borderColor: colorForSeverity(finding.severity), padding: 1, backgroundColor: colors.panel2 },
    Text({ height: 1, content: finding.rule || "finding", fg: colors.text, attributes: 1 }),
    Text({ height: 1, content: `${finding.tool || "unknown"} / ${finding.severity || "info"} / ${finding.server || "no server"}`, fg: colors.muted }),
    spacer(),
    labelBlock("message", finding.message),
    labelBlock("evidence", finding.evidence),
    labelBlock("impact", finding.impact),
    labelBlock("recommendation", finding.recommendation || finding.fix_summary)
  );
}

function signalDetailPanel(signal: NightwardSignal | undefined) {
  if (!signal) {
    return emptyPanel("analysis detail", "No matching signal selected.");
  }
  return Box(
    { flexGrow: 1, height: "100%", flexDirection: "column", border: true, borderStyle: "rounded", borderColor: colorForSeverity(signal.severity), padding: 1, backgroundColor: colors.panel2 },
    Text({ height: 1, content: signal.rule || "signal", fg: colors.text, attributes: 1 }),
    Text({ height: 1, content: `${signal.provider || "nightward"} / ${signal.category || "unknown"} / ${signal.severity || "info"}`, fg: colors.muted }),
    spacer(),
    labelBlock("message", signal.message),
    labelBlock("evidence", signal.evidence),
    labelBlock("recommendation", signal.recommended_action),
    labelBlock("why", signal.why_this_matters)
  );
}

function fixDetailPanel(fix: NightwardFix | undefined) {
  if (!fix) {
    return emptyPanel("fix plan", "No plan-only remediation is available for this filter.");
  }
  return Box(
    { flexGrow: 1, height: "100%", flexDirection: "column", border: true, borderStyle: "rounded", borderColor: colors.green, padding: 1, backgroundColor: colors.panel2 },
    Text({ height: 1, content: fix.fix_kind || fix.status || "review", fg: colors.green, attributes: 1 }),
    Text({ height: 1, content: `${fix.rule || "rule"} / ${fix.severity || "info"}`, fg: colors.muted }),
    Text({ height: 2, content: redact(fix.summary), fg: colors.text, wrapMode: "word" }),
    spacer(),
    Text({ height: 1, content: "steps", fg: colors.cyan, attributes: 1 }),
    ...(fix.steps || ["Review this finding and decide whether to trust, pin, narrow, or externalize it."]).slice(0, 8).map((step, index) =>
      Text({ height: 2, content: `${index + 1}. ${redact(step)}`, fg: colors.text, wrapMode: "word" })
    )
  );
}

function backupDetailPanel(entry: NightwardBackupEntry | undefined, targetRoot?: string) {
  if (!entry) {
    return emptyPanel("backup detail", "No backup entry selected.");
  }
  return Box(
    { flexGrow: 1, height: "100%", flexDirection: "column", border: true, borderStyle: "rounded", borderColor: actionColor(entry.action), padding: 1, backgroundColor: colors.panel2 },
    Text({ height: 1, content: entry.action || "review", fg: actionColor(entry.action), attributes: 1 }),
    Text({ height: 1, content: `${entry.tool || "tool"} / ${entry.classification || "unknown"} / ${entry.risk || "info"}`, fg: colors.muted }),
    spacer(),
    labelBlock("source", entry.source),
    labelBlock("target", entry.target || targetRoot),
    labelBlock("reason", entry.reason),
    labelBlock("recommendation", entry.recommended_action)
  );
}

function emptyPanel(title: string, message: string) {
  return Box(
    { flexGrow: 1, height: "100%", flexDirection: "column", border: true, borderStyle: "rounded", borderColor: colors.line, padding: 1, backgroundColor: colors.panel2 },
    Text({ height: 1, content: title, fg: colors.text, attributes: 1 }),
    spacer(),
    Text({ height: 2, content: message, fg: colors.muted, wrapMode: "word" })
  );
}

function footer(state: TuiState) {
  const prompt = state.searchMode ? `search: ${state.search}` : "tab sections  / search  s severity  x clear  q quit";
  return Box(
    { height: 2, border: ["top"], borderColor: colors.line, paddingY: 0 },
    Text({ height: 1, content: `${prompt}    ${state.status}`, fg: state.searchMode ? colors.cyan : colors.muted })
  );
}

function navItem(index: number, label: string, active: boolean) {
  const accent = active ? colors.cyan : colors.line;
  return Box(
    { height: 2, flexDirection: "row", border: ["left"], borderColor: accent, paddingLeft: 1, backgroundColor: active ? colors.panel2 : colors.panel },
    Text({ height: 1, content: `${index + 1} ${label}`, fg: active ? colors.text : colors.muted, attributes: active ? 1 : 0 })
  );
}

function statCard(label: string, value: string, color: string) {
  return Box(
    { width: 17, height: 3, flexDirection: "column", border: true, borderStyle: "rounded", borderColor: color, paddingX: 1, backgroundColor: colors.panel },
    Text({ height: 1, content: label, fg: colors.muted }),
    Text({ height: 1, content: truncate(value, 13), fg: color, attributes: 1 })
  );
}

function findingRow(finding: NightwardFinding, selected: boolean, width: number) {
  const severity = (finding.severity || "info").toLowerCase();
  return Box(
    { height: 3, flexDirection: "column", border: ["left"], borderColor: colorForSeverity(severity), paddingLeft: 1, backgroundColor: selected ? "#1A2440" : colors.panel },
    Text({ height: 1, content: `${severity.toUpperCase().padEnd(8)} ${truncate(finding.rule, 24)}`, fg: selected ? colors.text : colorForSeverity(severity), attributes: selected ? 1 : 0 }),
    Text({ height: 1, content: truncate(finding.message, width), fg: selected ? colors.text : colors.muted })
  );
}

function signalRow(signal: NightwardSignal, selected: boolean) {
  const severity = (signal.severity || "info").toLowerCase();
  return Box(
    { height: 3, flexDirection: "column", border: ["left"], borderColor: colorForSeverity(severity), paddingLeft: 1, backgroundColor: selected ? "#1A2440" : colors.panel },
    Text({ height: 1, content: `${severity.toUpperCase().padEnd(8)} ${truncate(signal.rule, 26)}`, fg: selected ? colors.text : colorForSeverity(severity), attributes: selected ? 1 : 0 }),
    Text({ height: 1, content: truncate(`${signal.provider || "nightward"}: ${signal.message || ""}`, 72), fg: selected ? colors.text : colors.muted })
  );
}

function fixRow(fix: NightwardFix, selected: boolean) {
  return Box(
    { height: 3, flexDirection: "column", border: ["left"], borderColor: statusColor(fix.status), paddingLeft: 1, backgroundColor: selected ? "#16301F" : colors.panel },
    Text({ height: 1, content: `${(fix.status || "review").toUpperCase().padEnd(8)} ${truncate(fix.rule, 28)}`, fg: selected ? colors.text : statusColor(fix.status), attributes: selected ? 1 : 0 }),
    Text({ height: 1, content: truncate(fix.summary || fix.finding_id, 72), fg: selected ? colors.text : colors.muted })
  );
}

function inventoryRow(item: NightwardItem, selected: boolean) {
  return Box(
    { height: 2, flexDirection: "column", border: ["left"], borderColor: selected ? colors.blue : colors.line, paddingLeft: 1, backgroundColor: selected ? "#17213A" : colors.panel },
    Text({ height: 1, content: `${truncate(item.tool, 12).padEnd(12)} ${truncate(item.classification, 14).padEnd(14)} ${truncate(item.path, 54)}`, fg: selected ? colors.text : colors.muted })
  );
}

function backupRow(entry: NightwardBackupEntry, selected: boolean) {
  return Box(
    { height: 3, flexDirection: "column", border: ["left"], borderColor: actionColor(entry.action), paddingLeft: 1, backgroundColor: selected ? "#302A16" : colors.panel },
    Text({ height: 1, content: `${(entry.action || "review").toUpperCase().padEnd(8)} ${truncate(entry.tool, 18)}`, fg: selected ? colors.text : actionColor(entry.action), attributes: selected ? 1 : 0 }),
    Text({ height: 1, content: truncate(entry.source, 72), fg: selected ? colors.text : colors.muted })
  );
}

function labelBlock(label: string, value?: string) {
  return Box(
    { height: value ? 5 : 0, flexDirection: "column" },
    Text({ height: 1, content: value ? label : "", fg: colors.cyan, attributes: 1 }),
    Text({ height: 3, content: redact(value), fg: colors.text, wrapMode: "word" })
  );
}

function badge(value: string, color: string, width: number) {
  return Box(
    { width, height: 3, border: true, borderStyle: "rounded", borderColor: color, paddingX: 1, backgroundColor: colors.panel2 },
    Text({ height: 1, content: value.toUpperCase(), fg: color, attributes: 1 })
  );
}

function spacer() {
  return Text({ height: 1, content: "" });
}

function bar(value: number, total: number, width: number): string {
  const filled = Math.max(0, Math.min(width, Math.round((value / total) * width)));
  return "#".repeat(filled).padEnd(width, "-");
}

function colorForSeverity(severity: Risk | undefined): string {
  switch ((severity || "info").toLowerCase()) {
    case "critical":
      return colors.critical;
    case "high":
      return colors.high;
    case "medium":
      return colors.medium;
    case "low":
      return colors.low;
    default:
      return colors.info;
  }
}

function statusColor(status: string | undefined): string {
  switch ((status || "").toLowerCase()) {
    case "safe":
      return colors.green;
    case "blocked":
      return colors.critical;
    default:
      return colors.medium;
  }
}

function actionColor(action: string | undefined): string {
  switch ((action || "").toLowerCase()) {
    case "include":
      return colors.green;
    case "exclude":
      return colors.critical;
    default:
      return colors.medium;
  }
}

function fallbackFixes(report: NightwardReport): NightwardFix[] {
  return filteredFindings(report, { severity: "", search: "" })
    .filter((finding) => finding.fix_available || finding.recommendation || finding.recommended_action || finding.fix_summary)
    .map((finding) => ({
      finding_id: finding.id,
      rule: finding.rule,
      severity: finding.severity,
      status: finding.fix_available ? "review" : "blocked",
      summary: finding.fix_summary || finding.recommendation || finding.recommended_action || finding.message,
      steps: finding.fix_steps,
      fix_kind: finding.fix_kind
    }));
}

function fallbackSignals(report: NightwardReport): NightwardSignal[] {
  return filteredFindings(report, { severity: "", search: "" }).map((finding) => ({
    id: finding.id,
    provider: "nightward",
    rule: finding.rule,
    category: categoryForRule(finding.rule),
    severity: finding.severity,
    confidence: "medium",
    message: finding.message,
    evidence: finding.evidence,
    recommended_action: finding.recommended_action || finding.recommendation,
    why_this_matters: finding.impact
  }));
}

function fallbackBackupEntries(report: NightwardReport): NightwardBackupEntry[] {
  return (report.items || []).map((item) => {
    const action = backupAction(item.classification);
    return {
      source: item.path,
      target: `${report.home || "~"}/dotfiles/config/${(item.tool || "tool").toLowerCase()}`,
      tool: item.tool,
      classification: item.classification,
      risk: item.risk,
      action,
      reason: item.reason,
      recommended_action: action === "include" ? "Copy after review." : "Keep out of portable dotfiles unless there is a documented reason."
    };
  });
}

function backupSummary(entries: NightwardBackupEntry[]) {
  return {
    included: entries.filter((entry) => entry.action === "include").length,
    review: entries.filter((entry) => entry.action === "review").length,
    excluded: entries.filter((entry) => entry.action === "exclude").length
  };
}

function backupAction(classification: string | undefined): string {
  switch (classification) {
    case "portable":
      return "include";
    case "secret-auth":
    case "runtime-cache":
    case "app-owned":
      return "exclude";
    default:
      return "review";
  }
}

function categoryForRule(rule: string | undefined): string {
  if (!rule) return "unknown";
  if (rule.includes("secret") || rule.includes("token")) return "secrets-exposure";
  if (rule.includes("package") || rule.includes("shell")) return "execution-risk";
  if (rule.includes("endpoint") || rule.includes("header")) return "network-exposure";
  if (rule.includes("filesystem") || rule.includes("path")) return "filesystem-scope";
  return "unknown";
}

function nextAction(report: NightwardReport): string {
  const counts = severityCounts(report);
  if (counts.critical > 0) {
    return "Externalize secret-like MCP values before syncing dotfiles or sharing configs.";
  }
  if (counts.high > 0) {
    return "Pin package executors and review remote MCP wrappers before publishing.";
  }
  if (totalFindings(report) > 0) {
    return "Review remaining medium/info findings and export a plan-only fix packet.";
  }
  return "No findings in this report. Keep scheduled scans enabled before future syncs.";
}

main().catch((error) => {
  console.error(error instanceof Error ? error.message : String(error));
  process.exit(1);
});
