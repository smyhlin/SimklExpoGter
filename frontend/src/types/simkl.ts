export type MediaType = "shows" | "movies" | "anime";
export type Status =
  | "watching"
  | "plantowatch"
  | "hold"
  | "completed"
  | "dropped";
export type ExtendedMode =
  | "full"
  | "full_anime_seasons"
  | "simkl_ids_only"
  | "ids_only";
export type OutputFormat = "csv" | "json" | "both";
export type FieldMode = "compact" | "all";
export type Grouping = "single-file" | "separate-files";
export type ScheduleFrequency = "daily" | "weekly";
export type BackupStorageKind = "local" | "gdrive";

export interface AppSettings {
  clientId: string;
  hasClientSecret: boolean;
  exportDirectory: string;
  suggestedExportDirectory: string;
  hasAccessToken: boolean;
  configPath: string;
  backupStorage: BackupStorageKind;
  googleDriveClientId: string;
  hasGoogleDriveClientSecret: boolean;
  hasGoogleDriveToken: boolean;
  googleDriveFolderName: string;
  googleDriveFolderUrl: string;
}

export interface DeviceAuthSession {
  userCode: string;
  verificationUrl: string;
  pinUrl: string;
  intervalSeconds: number;
  expiresAt: string;
}

export interface GoogleDriveAuthSession {
  authUrl: string;
  expiresAt: string;
  redirectUri: string;
}

export interface AppState {
  settings: AppSettings;
  schedule: ScheduleState;
  lastActivities?: Record<string, unknown>;
  pendingAuth?: DeviceAuthSession;
  pendingGoogleDriveAuth?: GoogleDriveAuthSession;
}

export interface ScheduleState {
  supported: boolean;
  enabled: boolean;
  installed: boolean;
  frequency: ScheduleFrequency;
  time: string;
  days: string[];
  outputFormat: OutputFormat;
  fieldMode: FieldMode;
  content: MediaType[];
  useActivityCheck: boolean;
  maxBackupAge: string;
  runIfBackupIsStale: boolean;
  lastSuccessfulBackupAt?: string;
  lastSuccessfulBackupKind?: string;
  backupFresh: boolean;
  backupAgeSeconds?: number;
  taskName: string;
  status?: string;
  nextRunAt?: string;
  lastRunAt?: string;
  lastResult?: string;
  message?: string;
  usesSavedOutput: boolean;
  outputDirectoryPreview: string;
}

export interface DeviceAuthStatus {
  state: "pending" | "slow-down" | "authorized" | "expired" | "error";
  message: string;
  session?: DeviceAuthSession;
}

export interface GoogleDriveAuthStatus {
  state: "pending" | "authorized" | "expired" | "error";
  message: string;
  session?: GoogleDriveAuthSession;
}

export interface ExportRequest {
  types: MediaType[];
  statuses: Status[];
  dateFrom: string;
  extended: ExtendedMode;
  episodeWatchedAt: boolean;
  includeMemos: boolean;
  includeNextWatchInfo: boolean;
  outputFormat: OutputFormat;
  fieldMode: FieldMode;
  grouping: Grouping;
  includeEpisodeFiles: boolean;
  useActivityCheck: boolean;
  exportDirectory: string;
  filenamePrefix: string;
}

export interface ExportedFile {
  path: string;
  storageKind?: BackupStorageKind;
  format: OutputFormat | "csv" | "json";
  mediaType: MediaType | "all";
  kind: string;
  rows: number;
}

export interface ExportResult {
  exportedAt: string;
  outputDirectory: string;
  storageKind?: BackupStorageKind;
  destinationLabel?: string;
  destinationUrl?: string;
  files: ExportedFile[];
  itemCounts: Record<string, number>;
  warnings?: string[];
  activitiesChecked: boolean;
  effectiveDateFrom?: string;
}

export interface SaveSettingsInput {
  clientId: string;
  clientSecret: string;
  exportDirectory: string;
  backupStorage: BackupStorageKind;
  googleDriveClientId: string;
  googleDriveClientSecret: string;
  googleDriveFolderName: string;
}

export interface SaveScheduleInput {
  enabled: boolean;
  frequency: ScheduleFrequency;
  time: string;
  days: string[];
  outputFormat: OutputFormat;
  fieldMode: FieldMode;
  content: MediaType[];
  useActivityCheck: boolean;
  maxBackupAge: string;
  runIfBackupIsStale: boolean;
}

export interface PendingExportSummary {
  label: string;
  value: string;
}
