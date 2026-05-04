import {
  Action,
  ActionPanel,
  Color,
  Detail,
  Icon,
  List,
  getPreferenceValues,
} from "@raycast/api";
import { usePromise } from "@raycast/utils";
import {
  dashboardMarkdown,
  findingFixLabel,
  findingMarkdown,
  findingTitle,
  fixPlanTotal,
  maxSeverity,
  severityColor,
  sortedFindings,
} from "./format";
import {
  analysisReport,
  doctor,
  fixPlan,
  normalizePreferences,
  reportsDir,
  scan,
} from "./nightward";
import type {
  AnalysisReport,
  DoctorReport,
  FixPlan,
  ScanReport,
} from "./types";

const docsUrl = "https://github.com/JSONbored/nightward#readme";

type DashboardData = {
  report: ScanReport;
  doctor: DoctorReport;
  fixPlan: FixPlan;
  analysis: AnalysisReport;
};

export default function Command() {
  const runtime = normalizePreferences(getPreferenceValues());
  const { data, error, isLoading, revalidate } = usePromise(async () => {
    const [report, doctorReport, plan, analysis] = await Promise.all([
      scan(runtime),
      doctor(runtime),
      fixPlan(runtime),
      analysisReport(runtime),
    ]);
    return { report, doctor: doctorReport, fixPlan: plan, analysis };
  });

  if (error) {
    return (
      <Detail
        markdown={`# Nightward Dashboard\n\n${error.message}`}
        actions={
          <DashboardActions
            onRefresh={revalidate}
            reportDir={reportsDir(runtime.homeOverride)}
            lastReport={undefined}
          />
        }
      />
    );
  }

  if (!data) {
    return (
      <List
        isLoading={isLoading}
        searchBarPlaceholder="Loading Nightward scan..."
      />
    );
  }

  return (
    <List
      isLoading={isLoading}
      searchBarPlaceholder="Search findings, actions, adapters, or report status..."
      isShowingDetail
    >
      <List.Section title="Review">
        <List.Item
          title={`${titleCase(maxSeverity(data.report.findings))} Risk`}
          subtitle={`${data.report.summary.total_findings} findings`}
          icon={{
            source: Icon.Shield,
            tintColor: severityColor(maxSeverity(data.report.findings)),
          }}
          accessories={severityAccessories(data.report)}
          detail={<DashboardDetail data={data} />}
          actions={
            <DashboardActions
              onRefresh={revalidate}
              reportDir={data.doctor.schedule.report_dir}
              lastReport={data.doctor.schedule.last_report}
            />
          }
        />
        <List.Item
          title="Fix Plan"
          subtitle={fixPlanSubtitle(data.fixPlan)}
          icon={{
            source: Icon.List,
            tintColor:
              data.fixPlan.summary.review > 0 ? Color.Yellow : Color.Green,
          }}
          accessories={[
            {
              text: `${fixPlanTotal(data.fixPlan)} total`,
            },
          ]}
          detail={<FixPlanDetail plan={data.fixPlan} />}
          actions={
            <DashboardActions
              onRefresh={revalidate}
              reportDir={data.doctor.schedule.report_dir}
              lastReport={data.doctor.schedule.last_report}
            />
          }
        />
        <List.Item
          title="Analysis"
          subtitle={`${data.analysis.summary.total_signals} local signals`}
          icon={{
            source: Icon.MagnifyingGlass,
            tintColor: severityColor(
              data.analysis.summary.highest_severity || "info",
            ),
          }}
          accessories={[
            {
              tag: {
                value: data.analysis.summary.highest_severity || "info",
                color: severityColor(
                  data.analysis.summary.highest_severity || "info",
                ),
              },
            },
          ]}
          detail={<AnalysisDetail analysis={data.analysis} />}
          actions={
            <DashboardActions
              onRefresh={revalidate}
              reportDir={data.doctor.schedule.report_dir}
              lastReport={data.doctor.schedule.last_report}
            />
          }
        />
      </List.Section>

      <List.Section title="Environment">
        <List.Item
          title="Schedule"
          subtitle={scheduleSubtitle(data.doctor)}
          icon={{
            source: Icon.Clock,
            tintColor: data.doctor.schedule.installed
              ? Color.Green
              : Color.Yellow,
          }}
          accessories={[
            ...(data.doctor.schedule.history &&
            data.doctor.schedule.history.length > 1
              ? [{ text: `${data.doctor.schedule.history.length} reports` }]
              : []),
          ]}
          detail={<ScheduleDetail doctor={data.doctor} />}
          actions={
            <DashboardActions
              onRefresh={revalidate}
              reportDir={data.doctor.schedule.report_dir}
              lastReport={data.doctor.schedule.last_report}
            />
          }
        />
        <List.Item
          title="Adapters"
          subtitle={adapterSubtitle(data.report)}
          icon={{
            source: Icon.HardDrive,
            tintColor: data.report.adapters.some((adapter) => adapter.available)
              ? Color.Green
              : Color.SecondaryText,
          }}
          detail={<AdaptersDetail data={data} />}
          actions={
            <DashboardActions
              onRefresh={revalidate}
              reportDir={data.doctor.schedule.report_dir}
              lastReport={data.doctor.schedule.last_report}
            />
          }
        />
      </List.Section>

      <List.Section title="Findings To Review">
        {sortedFindings(data.report.findings)
          .slice(0, 8)
          .map((finding) => (
            <List.Item
              key={finding.id}
              title={findingTitle(finding)}
              subtitle={
                finding.server ? `server ${finding.server}` : finding.tool
              }
              icon={{
                source: Icon.Warning,
                tintColor: severityColor(finding.severity),
              }}
              accessories={[
                {
                  tag: {
                    value: finding.severity,
                    color: severityColor(finding.severity),
                  },
                },
                { text: findingFixLabel(finding) },
              ]}
              detail={<List.Item.Detail markdown={findingMarkdown(finding)} />}
              actions={
                <ActionPanel>
                  <Action.CopyToClipboard
                    title="Copy Finding ID"
                    content={finding.id}
                  />
                  <Action.Push
                    title="Show Finding"
                    target={<Detail markdown={findingMarkdown(finding)} />}
                  />
                  <Action.OpenInBrowser
                    title="Open Nightward Docs"
                    url={docsUrl}
                  />
                  <Action
                    title="Refresh"
                    icon={Icon.ArrowClockwise}
                    onAction={revalidate}
                  />
                </ActionPanel>
              }
            />
          ))}
      </List.Section>
    </List>
  );
}

function DashboardDetail({ data }: { data: DashboardData }) {
  return (
    <List.Item.Detail
      markdown={dashboardMarkdown(
        data.report,
        data.doctor,
        data.fixPlan,
        data.analysis,
      )}
    />
  );
}

function AnalysisDetail({ analysis }: { analysis: AnalysisReport }) {
  return (
    <List.Item.Detail
      markdown={[
        "# Analysis",
        "",
        `Signals: \`${analysis.summary.total_signals}\``,
        `Highest severity: \`${analysis.summary.highest_severity || "info"}\``,
        `Provider warnings: \`${analysis.summary.provider_warnings}\``,
        "",
        "Offline analysis adds context to Nightward findings. It does not prove that an MCP server, package, or endpoint is safe.",
      ].join("\n")}
      metadata={
        <List.Item.Detail.Metadata>
          <List.Item.Detail.Metadata.Label title="Mode" text={analysis.mode} />
          <List.Item.Detail.Metadata.Label
            title="Subjects"
            text={String(analysis.summary.total_subjects)}
          />
          <List.Item.Detail.Metadata.Label
            title="Providers"
            text={`${analysis.providers.length} checked`}
          />
        </List.Item.Detail.Metadata>
      }
    />
  );
}

function AdaptersDetail({ data }: { data: DashboardData }) {
  const found = data.report.adapters.filter((adapter) => adapter.available);
  const missing = data.report.adapters.filter((adapter) => !adapter.available);
  return (
    <List.Item.Detail
      markdown={[
        "# Adapters",
        "",
        `${found.length}/${data.report.adapters.length} adapter surfaces were discovered in this scan.`,
        "",
        "## Found",
        ...(found.length > 0
          ? found.map((adapter) => `- ${adapter.name}`)
          : ["No adapter-specific config was found."]),
        "",
        "## Not Found",
        ...missing.slice(0, 8).map((adapter) => `- ${adapter.name}`),
      ].join("\n")}
      metadata={
        <List.Item.Detail.Metadata>
          <List.Item.Detail.Metadata.Label
            title="Found"
            text={String(found.length)}
          />
          <List.Item.Detail.Metadata.Label
            title="Checked"
            text={String(data.report.adapters.length)}
          />
        </List.Item.Detail.Metadata>
      }
    />
  );
}

function ScheduleDetail({ doctor }: { doctor: DoctorReport }) {
  const lines = [
    "# Scheduled Scan",
    "",
    `Installed: \`${doctor.schedule.installed ? "yes" : "no"}\``,
    `Platform: \`${doctor.schedule.platform}\``,
    `Report dir: \`${doctor.schedule.report_dir}\``,
    `Log dir: \`${doctor.schedule.log_dir}\``,
  ];
  if (doctor.schedule.last_run)
    lines.push(`Last run: \`${doctor.schedule.last_run}\``);
  if (doctor.schedule.last_report)
    lines.push(`Last report: \`${doctor.schedule.last_report}\``);
  if (doctor.schedule.last_findings !== undefined)
    lines.push(`Last findings: \`${doctor.schedule.last_findings}\``);
  if (doctor.schedule.history && doctor.schedule.history.length > 0) {
    lines.push("", "## Report History");
    for (const record of doctor.schedule.history.slice(0, 5)) {
      lines.push(
        `- \`${record.findings}\` findings${record.highest_severity ? `, highest \`${record.highest_severity}\`` : ""} - ${record.report_name} - \`${record.mod_time}\``,
      );
    }
  }
  return <List.Item.Detail markdown={lines.join("\n")} />;
}

function FixPlanDetail({ plan }: { plan: FixPlan }) {
  const lines = [
    "# Fix Plan",
    "",
    `Total: \`${fixPlanTotal(plan)}\``,
    `Safe: \`${plan.summary.safe}\``,
    `Review: \`${plan.summary.review}\``,
    `Blocked: \`${plan.summary.blocked}\``,
    "",
    "Nightward exports fix plans only. It does not mutate local configs from Raycast.",
  ];
  if (plan.groups && plan.groups.length > 0) {
    lines.push("", "## Grouped Review");
    for (const group of plan.groups.slice(0, 8)) {
      const title = group.label ?? group.title ?? group.key;
      const count = group.count ?? group.finding_count ?? 0;
      const summary =
        group.summary ?? group.actions?.[0] ?? "Review this group.";
      lines.push(`- \`${title}\` (${count}): ${summary}`);
    }
  }
  return <List.Item.Detail markdown={lines.join("\n")} />;
}

function DashboardActions({
  onRefresh,
  reportDir,
  lastReport,
}: {
  onRefresh: () => void;
  reportDir: string;
  lastReport?: string;
}) {
  return (
    <ActionPanel>
      <Action title="Refresh" icon={Icon.ArrowClockwise} onAction={onRefresh} />
      {lastReport ? (
        <Action.ShowInFinder title="Show Latest Report" path={lastReport} />
      ) : null}
      <Action.ShowInFinder title="Open Report Folder" path={reportDir} />
      <Action.OpenInBrowser title="Open Nightward Docs" url={docsUrl} />
      <Action.CopyToClipboard title="Copy Reports Path" content={reportDir} />
      {lastReport ? (
        <Action.CopyToClipboard
          title="Copy Latest Report Path"
          content={lastReport}
        />
      ) : null}
    </ActionPanel>
  );
}

function severityAccessories(report: ScanReport): Array<{ text: string }> {
  const critical = report.summary.findings_by_severity.critical ?? 0;
  const high = report.summary.findings_by_severity.high ?? 0;
  const medium = report.summary.findings_by_severity.medium ?? 0;
  const values = [
    critical > 0 ? { text: `${critical}C` } : undefined,
    high > 0 ? { text: `${high}H` } : undefined,
    medium > 0 ? { text: `${medium}M` } : undefined,
  ];
  return values.filter(Boolean) as Array<{ text: string }>;
}

function fixPlanSubtitle(plan: FixPlan): string {
  const parts = [];
  if (plan.summary.review > 0) parts.push(`${plan.summary.review} review`);
  if (plan.summary.safe > 0) parts.push(`${plan.summary.safe} safe`);
  if (plan.summary.blocked > 0) parts.push(`${plan.summary.blocked} blocked`);
  return parts.join(" - ");
}

function scheduleSubtitle(doctor: DoctorReport): string {
  if (!doctor.schedule.installed) return "off";
  if (doctor.schedule.last_findings !== undefined) {
    return `on - ${doctor.schedule.last_findings} latest`;
  }
  return "on - no report";
}

function adapterSubtitle(report: ScanReport): string {
  const found = report.adapters.filter((adapter) => adapter.available).length;
  return `${found}/${report.adapters.length} found`;
}

function titleCase(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1);
}
