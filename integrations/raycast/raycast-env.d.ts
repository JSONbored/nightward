/// <reference types="@raycast/api">

/* 🚧 🚧 🚧
 * This file is auto-generated from the extension's manifest.
 * Do not modify manually. Instead, update the `package.json` file.
 * 🚧 🚧 🚧 */

/* eslint-disable @typescript-eslint/ban-types */

type ExtensionPreferences = {
  /** Nightward Command - Path or command name used to run Nightward locally. */
  "nightwardPath": string,
  /** Home Override - Optional HOME-equivalent path passed as NIGHTWARD_HOME for testing fixture homes. */
  "homeOverride"?: string,
  /** Allow Online Providers - Allow selected online-capable providers in Raycast Analysis. Socket creates a remote scan artifact. */
  "allowOnlineProviders": boolean
}

/** Preferences accessible in all the extension's commands */
declare type Preferences = ExtensionPreferences

declare namespace Preferences {
  /** Preferences accessible in the `dashboard` command */
  export type Dashboard = ExtensionPreferences & {}
  /** Preferences accessible in the `status-menu` command */
  export type StatusMenu = ExtensionPreferences & {}
  /** Preferences accessible in the `findings` command */
  export type Findings = ExtensionPreferences & {}
  /** Preferences accessible in the `analysis` command */
  export type Analysis = ExtensionPreferences & {}
  /** Preferences accessible in the `provider-doctor` command */
  export type ProviderDoctor = ExtensionPreferences & {}
  /** Preferences accessible in the `actions` command */
  export type Actions = ExtensionPreferences & {}
  /** Preferences accessible in the `explain-finding` command */
  export type ExplainFinding = ExtensionPreferences & {}
  /** Preferences accessible in the `explain-signal` command */
  export type ExplainSignal = ExtensionPreferences & {}
  /** Preferences accessible in the `export-fix-plan` command */
  export type ExportFixPlan = ExtensionPreferences & {}
  /** Preferences accessible in the `export-analysis` command */
  export type ExportAnalysis = ExtensionPreferences & {}
  /** Preferences accessible in the `open-report-folder` command */
  export type OpenReportFolder = ExtensionPreferences & {}
}

declare namespace Arguments {
  /** Arguments passed to the `dashboard` command */
  export type Dashboard = {}
  /** Arguments passed to the `status-menu` command */
  export type StatusMenu = {}
  /** Arguments passed to the `findings` command */
  export type Findings = {}
  /** Arguments passed to the `analysis` command */
  export type Analysis = {}
  /** Arguments passed to the `provider-doctor` command */
  export type ProviderDoctor = {}
  /** Arguments passed to the `actions` command */
  export type Actions = {}
  /** Arguments passed to the `explain-finding` command */
  export type ExplainFinding = {
  /** Finding ID */
  "findingId": string
}
  /** Arguments passed to the `explain-signal` command */
  export type ExplainSignal = {
  /** Finding ID */
  "findingId": string
}
  /** Arguments passed to the `export-fix-plan` command */
  export type ExportFixPlan = {}
  /** Arguments passed to the `export-analysis` command */
  export type ExportAnalysis = {}
  /** Arguments passed to the `open-report-folder` command */
  export type OpenReportFolder = {}
}

