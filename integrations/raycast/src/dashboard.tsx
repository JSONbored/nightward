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
  adapterSummary,
  classificationColor,
  dashboardMarkdown,
  findingFixLabel,
  findingMarkdown,
  findingSubtitle,
  findingTitle,
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
  Classification,
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
      searchBarPlaceholder="Search Nightward status..."
      isShowingDetail
    >
      <List.Section title="Status">
        <List.Item
          title="Scan Summary"
          subtitle={`${data.report.summary.total_items} items - ${data.report.summary.total_findings} findings`}
          icon={{
            source: Icon.Shield,
            tintColor: severityColor(maxSeverity(data.report.findings)),
          }}
          accessories={[{ text: maxSeverity(data.report.findings) }]}
          detail={<DashboardDetail data={data} />}
          actions={
            <DashboardActions
              onRefresh={revalidate}
              reportDir={data.doctor.schedule.report_dir}
            />
          }
        />
        <List.Item
          title="Scheduled Scan"
          subtitle={
            data.doctor.schedule.installed ? "installed" : "not installed"
          }
          icon={{
            source: Icon.Clock,
            tintColor: data.doctor.schedule.installed
              ? Color.Green
              : Color.Yellow,
          }}
          accessories={[
            {
              text:
                data.doctor.schedule.last_findings !== undefined
                  ? `${data.doctor.schedule.last_findings} last findings`
                  : "no report yet",
            },
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
            />
          }
        />
        <List.Item
          title="Fix Plan"
          subtitle={`${data.fixPlan.summary.safe} safe - ${data.fixPlan.summary.review} review - ${data.fixPlan.summary.blocked} blocked`}
          icon={{
            source: Icon.List,
            tintColor:
              data.fixPlan.summary.review > 0 ? Color.Yellow : Color.Green,
          }}
          detail={<FixPlanDetail plan={data.fixPlan} />}
          actions={
            <DashboardActions
              onRefresh={revalidate}
              reportDir={data.doctor.schedule.report_dir}
            />
          }
        />
        <List.Item
          title="Analysis"
          subtitle={`${data.analysis.summary.total_signals} signals - ${data.analysis.summary.provider_warnings} provider warnings`}
          icon={{
            source: Icon.MagnifyingGlass,
            tintColor: severityColor(
              data.analysis.summary.highest_severity || "info",
            ),
          }}
          accessories={[
            { text: data.analysis.summary.highest_severity || "info" },
          ]}
          detail={
            <List.Item.Detail
              markdown={`# Analysis\n\nSignals: \`${data.analysis.summary.total_signals}\`\n\nProvider warnings: \`${data.analysis.summary.provider_warnings}\`\n\nOffline analysis does not claim a server or package is safe.`}
            />
          }
          actions={
            <DashboardActions
              onRefresh={revalidate}
              reportDir={data.doctor.schedule.report_dir}
            />
          }
        />
      </List.Section>

      <List.Section title="Classifications">
        {classificationRows(data.report).map(([classification, count]) => (
          <List.Item
            key={classification}
            title={classification}
            subtitle={`${count} item${count === 1 ? "" : "s"}`}
            icon={{
              source: Icon.Dot,
              tintColor: classificationColor(classification),
            }}
            detail={
              <List.Item.Detail
                markdown={`# ${classification}\n\n${count} discovered item${count === 1 ? "" : "s"}.`}
              />
            }
            actions={
              <DashboardActions
                onRefresh={revalidate}
                reportDir={data.doctor.schedule.report_dir}
              />
            }
          />
        ))}
      </List.Section>

      <List.Section title="Top Findings">
        {sortedFindings(data.report.findings)
          .slice(0, 8)
          .map((finding) => (
            <List.Item
              key={finding.id}
              title={findingTitle(finding)}
              subtitle={findingSubtitle(finding)}
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
      markdown={dashboardMarkdown(data.report, data.doctor)}
      metadata={
        <List.Item.Detail.Metadata>
          <List.Item.Detail.Metadata.Label
            title="Adapters"
            text={adapterSummary(data.report.adapters)}
          />
          <List.Item.Detail.Metadata.Label
            title="Fix Plan"
            text={`${data.fixPlan.summary.total} total`}
          />
          <List.Item.Detail.Metadata.Label
            title="Analysis"
            text={`${data.analysis.summary.total_signals} signals`}
          />
          <List.Item.Detail.Metadata.Label
            title="Report Directory"
            text={data.doctor.schedule.report_dir}
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
        `- \`${record.findings}\` findings - ${record.report_name} - \`${record.mod_time}\``,
      );
    }
  }
  return <List.Item.Detail markdown={lines.join("\n")} />;
}

function FixPlanDetail({ plan }: { plan: FixPlan }) {
  const lines = [
    "# Fix Plan",
    "",
    `Total: \`${plan.summary.total}\``,
    `Safe: \`${plan.summary.safe}\``,
    `Review: \`${plan.summary.review}\``,
    `Blocked: \`${plan.summary.blocked}\``,
    "",
    "Nightward exports fix plans only. It does not mutate local configs from Raycast.",
  ];
  return <List.Item.Detail markdown={lines.join("\n")} />;
}

function DashboardActions({
  onRefresh,
  reportDir,
}: {
  onRefresh: () => void;
  reportDir: string;
}) {
  return (
    <ActionPanel>
      <Action title="Refresh" icon={Icon.ArrowClockwise} onAction={onRefresh} />
      <Action.ShowInFinder title="Open Report Folder" path={reportDir} />
      <Action.OpenInBrowser title="Open Nightward Docs" url={docsUrl} />
      <Action.CopyToClipboard title="Copy Reports Path" content={reportDir} />
    </ActionPanel>
  );
}

function classificationRows(
  report: ScanReport,
): Array<[Classification, number]> {
  const order: Classification[] = [
    "portable",
    "machine-local",
    "secret-auth",
    "runtime-cache",
    "app-owned",
    "unknown",
  ];
  return order
    .map((classification): [Classification, number] => [
      classification,
      report.summary.items_by_classification[classification] ?? 0,
    ])
    .filter(([, count]) => count > 0);
}
