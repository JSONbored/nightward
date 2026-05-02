import { describe, expect, test } from "bun:test";
import {
  filteredFindings,
  highestSeverity,
  nextSeverity,
  normalizeBundle,
  redact,
  selectedFinding,
  severityCounts,
  totalFindings,
  truncate,
  type NightwardReport
} from "./model";

const report: NightwardReport = {
  summary: {
    total_findings: 3,
    findings_by_severity: {
      critical: 1,
      high: 1,
      medium: 1
    }
  },
  findings: [
    {
      id: "critical-secret",
      severity: "critical",
      tool: "Codex",
      rule: "mcp_secret_env",
      message: "API_TOKEN=super-secret-value appears in config"
    },
    {
      id: "high-package",
      severity: "high",
      tool: "Cursor",
      rule: "mcp_unpinned_package",
      message: "unpinned package"
    },
    {
      id: "medium-local",
      severity: "medium",
      tool: "Cursor",
      rule: "mcp_local_endpoint",
      message: "localhost endpoint"
    }
  ]
};

describe("OpenTUI report model", () => {
  test("summarizes severity and finding counts", () => {
    expect(totalFindings(report)).toBe(3);
    expect(severityCounts(report)).toMatchObject({ critical: 1, high: 1, medium: 1, low: 0, info: 0 });
    expect(highestSeverity(report)).toBe("critical");
  });

  test("filters findings by severity and search", () => {
    expect(filteredFindings(report, { severity: "high", search: "" }).map((finding) => finding.id)).toEqual(["high-package"]);
    expect(filteredFindings(report, { severity: "", search: "localhost" }).map((finding) => finding.id)).toEqual(["medium-local"]);
  });

  test("selects safely and redacts secret-looking text", () => {
    expect(selectedFinding(report, { severity: "", search: "", cursor: 99 })?.id).toBe("medium-local");
    expect(redact("API_TOKEN=super-secret-value")).toBe("API_TOKEN=[redacted]");
    expect(truncate("API_TOKEN=super-secret-value and a long suffix", 24)).toBe("API_TOKEN=[redacted] ...");
  });

  test("cycles severity filters", () => {
    expect(nextSeverity("")).toBe("critical");
    expect(nextSeverity("critical")).toBe("high");
    expect(nextSeverity("info")).toBe("");
  });

  test("normalizes scan-only and bundled inputs", () => {
    expect(normalizeBundle(report).scan.summary?.total_findings).toBe(3);
    expect(normalizeBundle({ scan: report, fix_plan: { summary: { total: 1 } } }).fix_plan?.summary?.total).toBe(1);
  });
});
