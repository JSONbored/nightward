import {
  Clipboard,
  Color,
  Icon,
  LaunchType,
  MenuBarExtra,
  getPreferenceValues,
  launchCommand,
  open,
  showToast,
  showHUD,
  Toast,
} from "@raycast/api";
import { usePromise } from "@raycast/utils";
import { execFile } from "node:child_process";
import { existsSync } from "node:fs";
import { menuBarStatus, menuBarStatusMarkdown, severityColor } from "./format";
import {
  analysisReport,
  doctor,
  normalizePreferences,
  reportsDir,
  scan,
} from "./nightward";
import type { MenuBarStatus } from "./format";

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
      lastReport: doctorReport.schedule.last_report,
    };
  });

  if (error) {
    return (
      <MenuBarExtra
        title="!"
        icon={{ source: Icon.ExclamationMark, tintColor: Color.Red }}
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
            onAction={() =>
              void openLocalPath(
                reportsDir(runtime.homeOverride),
                "Reports folder is not available yet",
              )
            }
          />
        </MenuBarExtra.Section>
      </MenuBarExtra>
    );
  }

  const status = data?.status;
  return (
    <MenuBarExtra
      title={status?.title ?? ""}
      icon={{
        source: Icon.Shield,
        tintColor: severityColor(status?.risk ?? "info"),
      }}
      tooltip={status?.tooltip ?? "Loading Nightward status"}
      isLoading={isLoading}
    >
      {status ? (
        <StatusItems
          status={status}
          reportDir={data.reportDir}
          lastReport={data.lastReport}
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
  lastReport,
  onRefresh,
}: {
  status: MenuBarStatus;
  reportDir: string;
  lastReport?: string;
  onRefresh: () => void;
}) {
  const scheduleTitle = status.scheduled ? "Enabled" : "Off";

  return (
    <>
      <MenuBarExtra.Section title="Findings">
        {status.critical > 0 ? (
          <MenuBarExtra.Item
            title={`${status.critical} Critical`}
            icon={{ source: Icon.ExclamationMark, tintColor: Color.Red }}
          />
        ) : null}
        {status.high > 0 ? (
          <MenuBarExtra.Item
            title={`${status.high} High`}
            icon={{ source: Icon.Warning, tintColor: severityColor("high") }}
          />
        ) : null}
        {status.medium > 0 ? (
          <MenuBarExtra.Item
            title={`${status.medium} Medium`}
            icon={{ source: Icon.Circle, tintColor: Color.Yellow }}
          />
        ) : null}
        <MenuBarExtra.Item
          title={`${status.findings} Total`}
          icon={{ source: Icon.List, tintColor: severityColor(status.risk) }}
        />
      </MenuBarExtra.Section>

      <MenuBarExtra.Section title="Analysis">
        <MenuBarExtra.Item
          title={`${status.signals} Signals`}
          icon={Icon.MagnifyingGlass}
        />
        {status.providerWarnings > 0 ? (
          <MenuBarExtra.Item
            title={`${status.providerWarnings} Provider Warnings`}
            icon={{ source: Icon.ExclamationMark, tintColor: Color.Yellow }}
          />
        ) : null}
        {status.historyDelta ? (
          <MenuBarExtra.Item
            title={status.historyDelta}
            subtitle="Since previous scheduled scan"
            icon={Icon.ArrowClockwise}
          />
        ) : null}
      </MenuBarExtra.Section>

      <MenuBarExtra.Section title="Schedule">
        <MenuBarExtra.Item
          title={scheduleTitle}
          icon={{
            source: Icon.Clock,
            tintColor: status.scheduled ? Color.Green : Color.Yellow,
          }}
        />
        {status.lastFindings !== undefined ? (
          <MenuBarExtra.Item
            title={`${status.lastFindings} Findings in Latest Report`}
            icon={Icon.Document}
          />
        ) : (
          <MenuBarExtra.Item title="No Scheduled Report" icon={Icon.Document} />
        )}
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
          icon={Icon.Folder}
          onAction={() =>
            void openLocalPath(reportDir, "Reports folder is not available yet")
          }
        />
        {lastReport ? (
          <MenuBarExtra.Item
            title="Latest Report"
            icon={Icon.Document}
            onAction={() =>
              void openLocalPath(lastReport, "Latest report is not available")
            }
          />
        ) : null}
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
          onAction={() => void openDocs()}
        />
      </MenuBarExtra.Section>
    </>
  );
}

async function openCommand(name: string) {
  try {
    await launchCommand({ name, type: LaunchType.UserInitiated });
  } catch (error) {
    await showToast({
      style: Toast.Style.Failure,
      title: `Could not open ${name}`,
      message: error instanceof Error ? error.message : "Unknown Raycast error",
    });
  }
}

async function copyStatus(status: MenuBarStatus) {
  await Clipboard.copy(menuBarStatusMarkdown(status));
  await showHUD("Copied Nightward status");
}

async function openDocs() {
  try {
    await open(docsUrl);
  } catch (error) {
    await showToast({
      style: Toast.Style.Failure,
      title: "Could not open Nightward docs",
      message: error instanceof Error ? error.message : "Unknown Raycast error",
    });
  }
}

async function openLocalPath(path: string, missingTitle: string) {
  if (!existsSync(path)) {
    await showToast({
      style: Toast.Style.Failure,
      title: missingTitle,
      message: path,
    });
    return;
  }

  execFile("/usr/bin/open", [path], (error) => {
    if (error) {
      void showToast({
        style: Toast.Style.Failure,
        title: "Could not open path",
        message: error.message,
      });
    }
  });
}
