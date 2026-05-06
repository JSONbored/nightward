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
  basename,
  dashboardMarkdown,
  findingFixLabel,
  findingMarkdown,
  findingTitle,
  fixPlanTotal,
  maxSeverity,
  reportDiffMarkdown,
  reportDiffSubtitle,
  severityColor,
  sortedFindings,
} from "./format";
import {
  analysisReport,
  doctor,
  fixPlan,
  normalizePreferences,
  reportDiff,
  reportsDir,
  scan,
  type RuntimeOptions,
} from "./nightward";
import type {
  AnalysisReport,
  DoctorReport,
  FixPlan,
  ReportRecord,
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
          title="Findings"
          subtitle={`${titleCase(maxSeverity(data.report.findings))} risk`}
          icon={{
            source: Icon.Shield,
            tintColor: severityColor(maxSeverity(data.report.findings)),
          }}
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
          subtitle={`${fixPlanTotal(data.fixPlan)} items`}
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
              lastReport={data.doctor.schedule.last_report}
            />
          }
        />
        <List.Item
          title="Analysis"
          subtitle="Offline context"
          icon={{
            source: Icon.MagnifyingGlass,
            tintColor: severityColor(
              data.analysis.summary.highest_severity || "info",
            ),
          }}
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
          subtitle={data.doctor.schedule.installed ? "On" : "Off"}
          icon={{
            source: Icon.Clock,
            tintColor: data.doctor.schedule.installed
              ? Color.Green
              : Color.Yellow,
          }}
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

      {(data.doctor.schedule.history?.length ?? 0) >= 2 ? (
        <List.Section title="Report History">
          <List.Item
            title="Compare Latest Report"
            subtitle={`${data.doctor.schedule.history?.[1]?.findings ?? 0} -> ${data.doctor.schedule.history?.[0]?.findings ?? 0} findings`}
            icon={{ source: Icon.BarChart, tintColor: Color.Blue }}
            detail={
              <List.Item.Detail
                markdown={[
                  "# Report Compare",
                  "",
                  `Base: \`${basename(data.doctor.schedule.history?.[1]?.path ?? "")}\``,
                  `Head: \`${basename(data.doctor.schedule.history?.[0]?.path ?? "")}\``,
                  "",
                  "Open the compare view to load the full diff from Nightward.",
                ].join("\n")}
              />
            }
            actions={
              <ActionPanel>
                <Action.Push
                  title="Open Compare"
                  icon={Icon.BarChart}
                  target={
                    <ReportCompareDetail
                      runtime={runtime}
                      base={data.doctor.schedule.history![1]}
                      head={data.doctor.schedule.history![0]}
                    />
                  }
                />
                <Action.ShowInFinder
                  title="Show Latest Report"
                  path={data.doctor.schedule.history![0].path}
                />
                <Action.ShowInFinder
                  title="Show Previous Report"
                  path={data.doctor.schedule.history![1].path}
                />
              </ActionPanel>
            }
          />
        </List.Section>
      ) : null}

      <List.Section title="Findings To Review">
        {sortedFindings(data.report.findings)
          .slice(0, 8)
          .map((finding) => (
            <List.Item
              key={finding.id}
              title={findingTitle(finding)}
              icon={{
                source: Icon.Warning,
                tintColor: severityColor(finding.severity),
              }}
              detail={<FindingPreviewDetail finding={finding} />}
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
      metadata={
        <List.Item.Detail.Metadata>
          <List.Item.Detail.Metadata.TagList title="Posture">
            <List.Item.Detail.Metadata.TagList.Item
              text={maxSeverity(data.report.findings)}
              color={severityColor(maxSeverity(data.report.findings))}
            />
          </List.Item.Detail.Metadata.TagList>
          <List.Item.Detail.Metadata.Separator />
          <List.Item.Detail.Metadata.Label
            title="Findings"
            text={String(data.report.summary.total_findings)}
          />
          <List.Item.Detail.Metadata.Label
            title="Items"
            text={String(data.report.summary.total_items)}
          />
          <List.Item.Detail.Metadata.Label
            title="Analysis Signals"
            text={String(data.analysis.summary.total_signals)}
          />
          <List.Item.Detail.Metadata.Label
            title="Fix Plan Items"
            text={String(fixPlanTotal(data.fixPlan))}
          />
          <List.Item.Detail.Metadata.Separator />
          <List.Item.Detail.Metadata.Label
            title="Home"
            text={data.report.home}
          />
          <List.Item.Detail.Metadata.Label
            title="Generated"
            text={data.report.generated_at}
          />
        </List.Item.Detail.Metadata>
      }
    />
  );
}

function AnalysisDetail({ analysis }: { analysis: AnalysisReport }) {
  return (
    <List.Item.Detail
      markdown={[
        "# Analysis",
        "",
        "Offline analysis adds context to Nightward findings. It does not prove that an MCP server, package, or endpoint is safe.",
      ].join("\n")}
      metadata={
        <List.Item.Detail.Metadata>
          <List.Item.Detail.Metadata.TagList title="Mode">
            <List.Item.Detail.Metadata.TagList.Item
              text={analysis.mode}
              color={Color.Blue}
            />
          </List.Item.Detail.Metadata.TagList>
          <List.Item.Detail.Metadata.Separator />
          <List.Item.Detail.Metadata.Label
            title="Signals"
            text={String(analysis.summary.total_signals)}
          />
          <List.Item.Detail.Metadata.Label
            title="Subjects"
            text={String(analysis.summary.total_subjects)}
          />
          <List.Item.Detail.Metadata.Label
            title="Providers"
            text={`${analysis.providers.length} checked`}
          />
          <List.Item.Detail.Metadata.Label
            title="Provider Warnings"
            text={String(analysis.summary.provider_warnings)}
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
          <List.Item.Detail.Metadata.TagList title="Adapters">
            <List.Item.Detail.Metadata.TagList.Item
              text={`${found.length} found`}
              color={found.length > 0 ? Color.Green : Color.SecondaryText}
            />
          </List.Item.Detail.Metadata.TagList>
          <List.Item.Detail.Metadata.Separator />
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
    doctor.schedule.installed
      ? "Scheduled scans are installed for this machine."
      : "Scheduled scans are off for this machine.",
  ];
  if (doctor.schedule.history && doctor.schedule.history.length > 0) {
    lines.push("", "## Report History");
    for (const record of doctor.schedule.history.slice(0, 5)) {
      lines.push(
        `- \`${record.findings}\` findings${record.highest_severity ? `, highest \`${record.highest_severity}\`` : ""} - ${record.report_name} - \`${record.mod_time}\``,
      );
    }
  }
  return (
    <List.Item.Detail
      markdown={lines.join("\n")}
      metadata={
        <List.Item.Detail.Metadata>
          <List.Item.Detail.Metadata.TagList title="Schedule">
            <List.Item.Detail.Metadata.TagList.Item
              text={doctor.schedule.installed ? "on" : "off"}
              color={doctor.schedule.installed ? Color.Green : Color.Yellow}
            />
          </List.Item.Detail.Metadata.TagList>
          <List.Item.Detail.Metadata.Separator />
          <List.Item.Detail.Metadata.Label
            title="Platform"
            text={doctor.schedule.platform}
          />
          <List.Item.Detail.Metadata.Label
            title="Reports"
            text={String(doctor.schedule.history?.length ?? 0)}
          />
          {doctor.schedule.last_run ? (
            <List.Item.Detail.Metadata.Label
              title="Last Run"
              text={doctor.schedule.last_run}
            />
          ) : null}
          {doctor.schedule.last_findings !== undefined ? (
            <List.Item.Detail.Metadata.Label
              title="Last Findings"
              text={String(doctor.schedule.last_findings)}
            />
          ) : null}
          <List.Item.Detail.Metadata.Label
            title="Report Dir"
            text={doctor.schedule.report_dir}
          />
          <List.Item.Detail.Metadata.Label
            title="Log Dir"
            text={doctor.schedule.log_dir}
          />
        </List.Item.Detail.Metadata>
      }
    />
  );
}

function ReportCompareDetail({
  runtime,
  base,
  head,
}: {
  runtime: RuntimeOptions;
  base: ReportRecord;
  head: ReportRecord;
}) {
  const { data, error, isLoading, revalidate } = usePromise(() =>
    reportDiff(runtime, base.path, head.path),
  );
  if (error) {
    return (
      <Detail
        markdown={`# Report Compare\n\n${error.message}`}
        actions={
          <ActionPanel>
            <Action
              title="Refresh"
              icon={Icon.ArrowClockwise}
              onAction={revalidate}
            />
            <Action.ShowInFinder title="Show Latest Report" path={head.path} />
            <Action.ShowInFinder
              title="Show Previous Report"
              path={base.path}
            />
          </ActionPanel>
        }
      />
    );
  }
  if (!data) {
    return <Detail isLoading={isLoading} markdown="# Report Compare" />;
  }
  const markdown = reportDiffMarkdown(data);
  return (
    <Detail
      isLoading={isLoading}
      markdown={markdown}
      metadata={
        <Detail.Metadata>
          <Detail.Metadata.TagList title="Change">
            <Detail.Metadata.TagList.Item
              text={reportDiffSubtitle(data)}
              color={severityColor(data.summary.max_added_severity)}
            />
          </Detail.Metadata.TagList>
          <Detail.Metadata.Separator />
          <Detail.Metadata.Label
            title="Added"
            text={String(data.summary.added)}
          />
          <Detail.Metadata.Label
            title="Removed"
            text={String(data.summary.removed)}
          />
          <Detail.Metadata.Label
            title="Changed"
            text={String(data.summary.changed)}
          />
          <Detail.Metadata.Label
            title="Max Added"
            text={data.summary.max_added_severity}
          />
          <Detail.Metadata.Separator />
          <Detail.Metadata.Label title="Base" text={basename(base.path)} />
          <Detail.Metadata.Label title="Head" text={basename(head.path)} />
        </Detail.Metadata>
      }
      actions={
        <ActionPanel>
          <Action.CopyToClipboard
            title="Copy Compare Markdown"
            content={markdown}
          />
          <Action
            title="Refresh"
            icon={Icon.ArrowClockwise}
            onAction={revalidate}
          />
          <Action.ShowInFinder title="Show Latest Report" path={head.path} />
          <Action.ShowInFinder title="Show Previous Report" path={base.path} />
        </ActionPanel>
      }
    />
  );
}

function FixPlanDetail({ plan }: { plan: FixPlan }) {
  const lines = [
    "# Fix Plan",
    "",
    "Nightward fix plans are review material. Config changes stay outside Raycast fix exports.",
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
  return (
    <List.Item.Detail
      markdown={lines.join("\n")}
      metadata={
        <List.Item.Detail.Metadata>
          <List.Item.Detail.Metadata.Label
            title="Total"
            text={String(fixPlanTotal(plan))}
          />
          <List.Item.Detail.Metadata.Label
            title="Safe"
            text={{ value: String(plan.summary.safe), color: Color.Green }}
          />
          <List.Item.Detail.Metadata.Label
            title="Review"
            text={{ value: String(plan.summary.review), color: Color.Yellow }}
          />
          <List.Item.Detail.Metadata.Label
            title="Blocked"
            text={{ value: String(plan.summary.blocked), color: Color.Red }}
          />
        </List.Item.Detail.Metadata>
      }
    />
  );
}

function FindingPreviewDetail({
  finding,
}: {
  finding: ScanReport["findings"][number];
}) {
  return (
    <List.Item.Detail
      markdown={findingMarkdown(finding)}
      metadata={
        <List.Item.Detail.Metadata>
          <List.Item.Detail.Metadata.TagList title="Severity">
            <List.Item.Detail.Metadata.TagList.Item
              text={finding.severity}
              color={severityColor(finding.severity)}
            />
          </List.Item.Detail.Metadata.TagList>
          <List.Item.Detail.Metadata.Separator />
          <List.Item.Detail.Metadata.Label title="Rule" text={finding.rule} />
          <List.Item.Detail.Metadata.Label title="Tool" text={finding.tool} />
          {finding.server ? (
            <List.Item.Detail.Metadata.Label
              title="Server"
              text={finding.server}
            />
          ) : null}
          <List.Item.Detail.Metadata.Label title="Path" text={finding.path} />
          <List.Item.Detail.Metadata.Label
            title="Fix"
            text={findingFixLabel(finding)}
          />
        </List.Item.Detail.Metadata>
      }
    />
  );
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

function adapterSubtitle(report: ScanReport): string {
  const found = report.adapters.filter((adapter) => adapter.available).length;
  return `${found}/${report.adapters.length} found`;
}

function titleCase(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1);
}
