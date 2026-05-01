import {
  Clipboard,
  Color,
  Icon,
  LaunchType,
  MenuBarExtra,
  getPreferenceValues,
  launchCommand,
  open,
  showHUD,
} from "@raycast/api";
import { usePromise } from "@raycast/utils";
import { menuBarStatus, menuBarStatusMarkdown, severityColor } from "./format";
import {
  analysisReport,
  doctor,
  normalizePreferences,
  reportsDir,
  scan,
} from "./nightward";
import type { MenuBarStatus } from "./format";
import type { RiskLevel } from "./types";

const docsUrl = "https://github.com/JSONbored/nightward#readme";

export default function Command() {
  const runtime = normalizePreferences(getPreferenceValues());
  const { data, error, isLoading, revalidate } = usePromise(async () => {
    const [report, doctorReport, analysis] = await Promise.all([
      scan(runtime),
      doctor(runtime),
      analysisReport(runtime),
    ]);
    return {
      status: menuBarStatus(report, doctorReport, analysis),
      reportDir: doctorReport.schedule.report_dir,
    };
  });

  if (error) {
    return (
      <MenuBarExtra
        title="NW !"
        icon={{ source: Icon.Warning, tintColor: Color.Red }}
        tooltip={`Nightward failed: ${error.message}`}
        isLoading={isLoading}
      >
        <MenuBarExtra.Section title="Nightward">
          <MenuBarExtra.Item
            title="Nightward Failed"
            subtitle={error.message}
          />
          <MenuBarExtra.Item
            title="Refresh"
            icon={Icon.ArrowClockwise}
            onAction={revalidate}
          />
          <MenuBarExtra.Item
            title="Open Dashboard"
            icon={Icon.Window}
            onAction={() => void openCommand("dashboard")}
          />
          <MenuBarExtra.Item
            title="Open Reports Folder"
            icon={Icon.Folder}
            onAction={() => void open(reportsDir(runtime.homeOverride))}
          />
        </MenuBarExtra.Section>
      </MenuBarExtra>
    );
  }

  const status = data?.status;
  return (
    <MenuBarExtra
      title={status?.title ?? "NW"}
      icon={{
        source: statusIcon(status?.risk ?? "info"),
        tintColor: severityColor(status?.risk ?? "info"),
      }}
      tooltip={status?.tooltip ?? "Loading Nightward status"}
      isLoading={isLoading}
    >
      {status ? (
        <StatusItems
          status={status}
          reportDir={data.reportDir}
          onRefresh={revalidate}
        />
      ) : (
        <MenuBarExtra.Item title="Loading Nightward status..." />
      )}
    </MenuBarExtra>
  );
}

function StatusItems({
  status,
  reportDir,
  onRefresh,
}: {
  status: MenuBarStatus;
  reportDir: string;
  onRefresh: () => void;
}) {
  return (
    <>
      <MenuBarExtra.Section title="Status">
        <MenuBarExtra.Item
          title={`${status.findings} findings`}
          subtitle={`critical ${status.critical} - high ${status.high} - medium ${status.medium}`}
          icon={{ source: Icon.Warning, tintColor: severityColor(status.risk) }}
        />
        <MenuBarExtra.Item
          title={`${status.signals} analysis signals`}
          subtitle={`${status.providerWarnings} provider warnings`}
          icon={Icon.MagnifyingGlass}
        />
        <MenuBarExtra.Item
          title={
            status.scheduled ? "Scheduled scan installed" : "No scheduled scan"
          }
          subtitle={
            status.lastFindings !== undefined
              ? `${status.lastFindings} findings in last report`
              : "no scheduled report yet"
          }
          icon={{
            source: Icon.Clock,
            tintColor: status.scheduled ? Color.Green : Color.Yellow,
          }}
        />
      </MenuBarExtra.Section>

      <MenuBarExtra.Section title="Open">
        <MenuBarExtra.Item
          title="Dashboard"
          icon={Icon.Window}
          onAction={() => void openCommand("dashboard")}
        />
        <MenuBarExtra.Item
          title="Findings"
          icon={Icon.List}
          onAction={() => void openCommand("findings")}
        />
        <MenuBarExtra.Item
          title="Provider Doctor"
          icon={Icon.Heartbeat}
          onAction={() => void openCommand("provider-doctor")}
        />
        <MenuBarExtra.Item
          title="Reports Folder"
          subtitle={reportDir}
          icon={Icon.Folder}
          onAction={() => void open(reportDir)}
        />
      </MenuBarExtra.Section>

      <MenuBarExtra.Section title="Actions">
        <MenuBarExtra.Item
          title="Refresh"
          icon={Icon.ArrowClockwise}
          onAction={onRefresh}
        />
        <MenuBarExtra.Item
          title="Copy Status Summary"
          icon={Icon.Clipboard}
          onAction={() => void copyStatus(status)}
        />
        <MenuBarExtra.Item
          title="Open Nightward Docs"
          icon={Icon.Book}
          onAction={() => void open(docsUrl)}
        />
      </MenuBarExtra.Section>
    </>
  );
}

function statusIcon(risk: RiskLevel): Icon {
  if (risk === "critical" || risk === "high") return Icon.ExclamationMark;
  if (risk === "medium") return Icon.Warning;
  return Icon.Shield;
}

async function openCommand(name: string) {
  await launchCommand({ name, type: LaunchType.UserInitiated });
}

async function copyStatus(status: MenuBarStatus) {
  await Clipboard.copy(menuBarStatusMarkdown(status));
  await showHUD("Copied Nightward status");
}
