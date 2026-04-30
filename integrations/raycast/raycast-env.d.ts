/// <reference types="@raycast/api">

/* 🚧 🚧 🚧
 * This file is auto-generated from the extension's manifest.
 * Do not modify manually. Instead, update the `package.json` file.
 * 🚧 🚧 🚧 */

/* eslint-disable @typescript-eslint/ban-types */

type ExtensionPreferences = {
  /** Nightward Command - Path or command name used to run Nightward locally. */
  "nightwardPath": string,
  /** Home Override - Optional HOME-equivalent directory passed as NIGHTWARD_HOME for testing. */
  "homeOverride"?: string
}

/** Preferences accessible in all the extension's commands */
declare type Preferences = ExtensionPreferences

declare namespace Preferences {
  /** Preferences accessible in the `dashboard` command */
  export type Dashboard = ExtensionPreferences & {}
  /** Preferences accessible in the `findings` command */
  export type Findings = ExtensionPreferences & {}
  /** Preferences accessible in the `explain-finding` command */
  export type ExplainFinding = ExtensionPreferences & {}
  /** Preferences accessible in the `export-fix-plan` command */
  export type ExportFixPlan = ExtensionPreferences & {}
  /** Preferences accessible in the `open-report-folder` command */
  export type OpenReportFolder = ExtensionPreferences & {}
}

declare namespace Arguments {
  /** Arguments passed to the `dashboard` command */
  export type Dashboard = {}
  /** Arguments passed to the `findings` command */
  export type Findings = {}
  /** Arguments passed to the `explain-finding` command */
  export type ExplainFinding = {
  /** Finding ID */
  "findingId": string
}
  /** Arguments passed to the `export-fix-plan` command */
  export type ExportFixPlan = {}
  /** Arguments passed to the `open-report-folder` command */
  export type OpenReportFolder = {}
}

