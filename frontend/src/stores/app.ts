import { get, writable, type Writable } from "svelte/store";
import type {
  AppState,
  BackupStorageKind,
  DeviceAuthSession,
  DeviceAuthStatus,
  GoogleDriveAuthSession,
  GoogleDriveAuthStatus,
  OutputFormat,
  ScheduleState,
} from "@/types/simkl";
import {
  checkDeviceAuth,
  checkGoogleDriveAuth,
  disconnectGoogleDrive as disconnectGoogleDriveBinding,
  exchangeOAuthCode,
  exportSettingsBackup as exportSettingsBackupBinding,
  getAppState,
  getStandardAuthURL,
  importSettingsBackup as importSettingsBackupBinding,
  logout,
  openExternal,
  saveSchedule as saveScheduleBinding,
  saveSettings as saveSettingsBinding,
  selectDirectory,
  startDeviceAuth,
  startGoogleDriveAuth,
} from "@/lib/wails";

export const defaultGoogleDriveFolderName = "SimklExpoGter Backups";

export interface AppStoreState {
  isLoading: boolean;
  isSaving: boolean;
  isSettingsBackupBusy: boolean;
  isAuthorized: boolean;
  hasClientSecret: boolean;
  savedBackupStorage: BackupStorageKind;
  exportDirectory: string;
  savedExportDirectory: string;
  configPath: string;
  savedClientId: string;
  clientId: string;
  clientSecret: string;
  backupStorage: BackupStorageKind;
  savedGoogleDriveClientId: string;
  googleDriveClientId: string;
  googleDriveClientSecret: string;
  hasGoogleDriveClientSecret: boolean;
  hasGoogleDriveToken: boolean;
  savedGoogleDriveFolderName: string;
  googleDriveFolderName: string;
  googleDriveFolderUrl: string;
  pendingAuth: DeviceAuthSession | null;
  pendingGoogleDriveAuth: GoogleDriveAuthSession | null;
  oauthPending: boolean;
  authMessage: string;
  exportProgress: string;
  scheduleSupported: boolean;
  scheduleEnabled: boolean;
  savedScheduleEnabled: boolean;
  scheduleInstalled: boolean;
  scheduleFrequency: "daily" | "weekly";
  savedScheduleFrequency: "daily" | "weekly";
  scheduleTime: string;
  savedScheduleTime: string;
  scheduleDays: string[];
  savedScheduleDays: string[];
  scheduleOutputFormat: OutputFormat;
  savedScheduleOutputFormat: OutputFormat;
  scheduleFieldMode: "compact" | "all";
  savedScheduleFieldMode: "compact" | "all";
  scheduleContent: Array<"shows" | "movies" | "anime">;
  savedScheduleContent: Array<"shows" | "movies" | "anime">;
  scheduleUseActivityCheck: boolean;
  savedScheduleUseActivityCheck: boolean;
  scheduleMaxBackupAge: string;
  savedScheduleMaxBackupAge: string;
  scheduleRunIfBackupIsStale: boolean;
  savedScheduleRunIfBackupIsStale: boolean;
  scheduleLastSuccessfulBackupAt: string;
  scheduleLastSuccessfulBackupKind: string;
  scheduleBackupFresh: boolean;
  scheduleBackupAgeSeconds: number;
  scheduleTaskName: string;
  scheduleStatus: string;
  scheduleNextRunAt: string;
  scheduleLastRunAt: string;
  scheduleLastResult: string;
  scheduleMessage: string;
  scheduleUsesSavedOutput: boolean;
  scheduleOutputDirectoryPreview: string;
  backupDestinationLabel: string;
}

const initialState: AppStoreState = {
  isLoading: false,
  isSaving: false,
  isSettingsBackupBusy: false,
  isAuthorized: false,
  hasClientSecret: false,
  savedBackupStorage: "local",
  exportDirectory: "",
  savedExportDirectory: "",
  configPath: "",
  savedClientId: "",
  clientId: "",
  clientSecret: "",
  backupStorage: "local",
  savedGoogleDriveClientId: "",
  googleDriveClientId: "",
  googleDriveClientSecret: "",
  hasGoogleDriveClientSecret: false,
  hasGoogleDriveToken: false,
  savedGoogleDriveFolderName: defaultGoogleDriveFolderName,
  googleDriveFolderName: defaultGoogleDriveFolderName,
  googleDriveFolderUrl: "",
  pendingAuth: null,
  pendingGoogleDriveAuth: null,
  oauthPending: false,
  authMessage: "",
  exportProgress: "Idle",
  scheduleSupported: false,
  scheduleEnabled: false,
  savedScheduleEnabled: false,
  scheduleInstalled: false,
  scheduleFrequency: "daily",
  savedScheduleFrequency: "daily",
  scheduleTime: "02:00",
  savedScheduleTime: "02:00",
  scheduleDays: ["mon"],
  savedScheduleDays: ["mon"],
  scheduleOutputFormat: "csv",
  savedScheduleOutputFormat: "csv",
  scheduleFieldMode: "all",
  savedScheduleFieldMode: "all",
  scheduleContent: ["shows", "movies", "anime"],
  savedScheduleContent: ["shows", "movies", "anime"],
  scheduleUseActivityCheck: false,
  savedScheduleUseActivityCheck: false,
  scheduleMaxBackupAge: "24h",
  savedScheduleMaxBackupAge: "24h",
  scheduleRunIfBackupIsStale: true,
  savedScheduleRunIfBackupIsStale: true,
  scheduleLastSuccessfulBackupAt: "",
  scheduleLastSuccessfulBackupKind: "",
  scheduleBackupFresh: false,
  scheduleBackupAgeSeconds: 0,
  scheduleTaskName: "",
  scheduleStatus: "",
  scheduleNextRunAt: "",
  scheduleLastRunAt: "",
  scheduleLastResult: "",
  scheduleMessage: "",
  scheduleUsesSavedOutput: true,
  scheduleOutputDirectoryPreview: "",
  backupDestinationLabel: "Choose a destination",
};

function resolveBackupDestinationLabel(
  state: Pick<
    AppStoreState,
    "backupStorage" | "exportDirectory" | "googleDriveFolderName" | "googleDriveFolderUrl"
  >,
) {
  return state.backupStorage === "gdrive"
    ? `Google Drive / ${
        state.googleDriveFolderName || defaultGoogleDriveFolderName
      }`
    : state.exportDirectory.trim() || "Choose a folder";
}

let devicePollTimer: ReturnType<typeof setTimeout> | null = null;
let devicePollGeneration = 0;
let googleDrivePollTimer: ReturnType<typeof setTimeout> | null = null;
let googleDrivePollGeneration = 0;

function defaultScheduleState(): ScheduleState {
  return {
    supported: false,
    enabled: false,
    installed: false,
    frequency: "daily",
    time: "02:00",
    days: ["mon"],
    outputFormat: "csv",
    fieldMode: "all",
    content: ["shows", "movies", "anime"],
    useActivityCheck: false,
    maxBackupAge: "24h",
    runIfBackupIsStale: true,
    lastSuccessfulBackupAt: "",
    lastSuccessfulBackupKind: "",
    backupFresh: false,
    backupAgeSeconds: 0,
    taskName: "",
    status: "",
    nextRunAt: "",
    lastRunAt: "",
    lastResult: "",
    message: "",
    usesSavedOutput: true,
    outputDirectoryPreview: "",
  };
}

function extractErrorMessage(error: unknown, fallback: string) {
  if (error instanceof Error && error.message.trim()) {
    return error.message.trim();
  }

  return fallback;
}

function mapAppState(state: AppState): Partial<AppStoreState> {
  return {
    savedBackupStorage: state.settings.backupStorage ?? "local",
    savedExportDirectory: state.settings.exportDirectory ?? "",
    savedClientId: (state.settings.clientId ?? "").trim(),
    clientId: state.settings.clientId ?? "",
    clientSecret: "",
    hasClientSecret: Boolean(state.settings.hasClientSecret),
    exportDirectory:
      state.settings.exportDirectory ||
      state.settings.suggestedExportDirectory ||
      "",
    configPath: state.settings.configPath ?? "",
    isAuthorized: Boolean(state.settings.hasAccessToken),
    backupStorage: state.settings.backupStorage ?? "local",
    savedGoogleDriveClientId: (state.settings.googleDriveClientId ?? "").trim(),
    googleDriveClientId: state.settings.googleDriveClientId ?? "",
    googleDriveClientSecret: "",
    hasGoogleDriveClientSecret: Boolean(
      state.settings.hasGoogleDriveClientSecret,
    ),
    hasGoogleDriveToken: Boolean(state.settings.hasGoogleDriveToken),
    savedGoogleDriveFolderName:
      (state.settings.googleDriveFolderName || defaultGoogleDriveFolderName)
        .trim() || defaultGoogleDriveFolderName,
    googleDriveFolderName:
      (state.settings.googleDriveFolderName || defaultGoogleDriveFolderName)
        .trim() || defaultGoogleDriveFolderName,
    googleDriveFolderUrl: state.settings.googleDriveFolderUrl || "",
    backupDestinationLabel: resolveBackupDestinationLabel({
      backupStorage: state.settings.backupStorage ?? "local",
      exportDirectory:
        state.settings.exportDirectory || state.settings.suggestedExportDirectory || "",
      googleDriveFolderName:
        (state.settings.googleDriveFolderName || defaultGoogleDriveFolderName)
          .trim() || defaultGoogleDriveFolderName,
      googleDriveFolderUrl: state.settings.googleDriveFolderUrl || "",
    }),
    pendingAuth: state.pendingAuth ?? null,
    pendingGoogleDriveAuth: state.pendingGoogleDriveAuth ?? null,
  };
}

function mapScheduleState(state?: ScheduleState): Partial<AppStoreState> {
  const next = state ?? defaultScheduleState();
  return {
    scheduleSupported: Boolean(next.supported),
    scheduleEnabled: Boolean(next.enabled),
    savedScheduleEnabled: Boolean(next.enabled),
    scheduleInstalled: Boolean(next.installed),
    scheduleFrequency: next.frequency ?? "daily",
    savedScheduleFrequency: next.frequency ?? "daily",
    scheduleTime: next.time || "02:00",
    savedScheduleTime: next.time || "02:00",
    scheduleDays: [...(next.days ?? ["mon"])],
    savedScheduleDays: [...(next.days ?? ["mon"])],
    scheduleOutputFormat: next.outputFormat ?? "csv",
    savedScheduleOutputFormat: next.outputFormat ?? "csv",
    scheduleFieldMode: next.fieldMode ?? "all",
    savedScheduleFieldMode: next.fieldMode ?? "all",
    scheduleContent: [...(next.content ?? ["shows", "movies", "anime"])],
    savedScheduleContent: [...(next.content ?? ["shows", "movies", "anime"])],
    scheduleUseActivityCheck: Boolean(next.useActivityCheck),
    savedScheduleUseActivityCheck: Boolean(next.useActivityCheck),
    scheduleMaxBackupAge: next.maxBackupAge || "24h",
    savedScheduleMaxBackupAge: next.maxBackupAge || "24h",
    scheduleRunIfBackupIsStale: next.runIfBackupIsStale !== false,
    savedScheduleRunIfBackupIsStale: next.runIfBackupIsStale !== false,
    scheduleLastSuccessfulBackupAt: next.lastSuccessfulBackupAt ?? "",
    scheduleLastSuccessfulBackupKind: next.lastSuccessfulBackupKind ?? "",
    scheduleBackupFresh: Boolean(next.backupFresh),
    scheduleBackupAgeSeconds: next.backupAgeSeconds ?? 0,
    scheduleTaskName: next.taskName ?? "",
    scheduleStatus: next.status ?? "",
    scheduleNextRunAt: next.nextRunAt ?? "",
    scheduleLastRunAt: next.lastRunAt ?? "",
    scheduleLastResult: next.lastResult ?? "",
    scheduleMessage: next.message ?? "",
    scheduleUsesSavedOutput: next.usesSavedOutput !== false,
    scheduleOutputDirectoryPreview: next.outputDirectoryPreview ?? "",
  };
}

function createAppStore() {
  const state = writable<AppStoreState>(initialState);

  function patchState(patch: Partial<AppStoreState>) {
    state.update((current) => {
      const next = { ...current, ...patch };
      return {
        ...next,
        backupDestinationLabel: resolveBackupDestinationLabel(next),
      };
    });
  }

  function stopDevicePolling() {
    devicePollGeneration += 1;
    if (devicePollTimer !== null) {
      clearTimeout(devicePollTimer);
      devicePollTimer = null;
    }
  }

  function stopGoogleDrivePolling() {
    googleDrivePollGeneration += 1;
    if (googleDrivePollTimer !== null) {
      clearTimeout(googleDrivePollTimer);
      googleDrivePollTimer = null;
    }
  }

  function applyAppState(appState: AppState) {
    patchState({
      ...mapAppState(appState),
      ...mapScheduleState(appState.schedule),
    });
  }

  function applyDeviceAuthStatus(status: DeviceAuthStatus) {
    if (status.session) {
      patchState({ pendingAuth: status.session });
    }

    patchState({ authMessage: status.message });

    if (status.state === "authorized") {
      patchState({
        isAuthorized: true,
        pendingAuth: null,
        oauthPending: false,
        exportProgress: "Ready for export",
      });
      stopDevicePolling();
    }

    if (status.state === "expired" || status.state === "error") {
      patchState({ pendingAuth: null, exportProgress: "Device login expired" });
      stopDevicePolling();
    }
  }

  function applyGoogleDriveAuthStatus(status: GoogleDriveAuthStatus) {
    if (status.session) {
      patchState({ pendingGoogleDriveAuth: status.session });
    }

    patchState({ authMessage: status.message });

    if (status.state === "authorized") {
      patchState({ pendingGoogleDriveAuth: null });
      stopGoogleDrivePolling();
    }

    if (status.state === "expired" || status.state === "error") {
      patchState({ pendingGoogleDriveAuth: null });
      stopGoogleDrivePolling();
    }
  }

  function startDevicePolling(intervalSeconds?: number) {
    stopDevicePolling();
    const generation = devicePollGeneration;
    const initialDelaySeconds = Math.max(
      intervalSeconds ?? get(state).pendingAuth?.intervalSeconds ?? 5,
      3,
    );

    function scheduleNextPoll(delaySeconds: number) {
      if (generation !== devicePollGeneration) return;

      const timeout = Math.max(delaySeconds, 3) * 1000;
      devicePollTimer = setTimeout(() => {
        void pollDeviceAuth();
      }, timeout);
    }

    async function pollDeviceAuth() {
      if (generation !== devicePollGeneration) return;

      try {
        const status = await checkDeviceAuth();
        if (generation !== devicePollGeneration) return;

        applyDeviceAuthStatus(status);
        if (status.state === "authorized") {
          await loadAppState();
          return;
        }

        if (status.state === "expired" || status.state === "error") {
          return;
        }

        const nextDelaySeconds =
          status.session?.intervalSeconds ??
          get(state).pendingAuth?.intervalSeconds ??
          initialDelaySeconds;
        scheduleNextPoll(nextDelaySeconds);
      } catch (error) {
        if (generation !== devicePollGeneration) return;

        patchState({
          authMessage: extractErrorMessage(
            error,
            "Failed to poll Simkl login status.",
          ),
        });
        const retryDelaySeconds =
          get(state).pendingAuth?.intervalSeconds ?? initialDelaySeconds;
        scheduleNextPoll(retryDelaySeconds);
      }
    }

    scheduleNextPoll(initialDelaySeconds);
  }

  function startGoogleDrivePolling() {
    stopGoogleDrivePolling();
    const generation = googleDrivePollGeneration;

    function scheduleNextPoll() {
      if (generation !== googleDrivePollGeneration) return;

      googleDrivePollTimer = setTimeout(() => {
        void pollGoogleDriveAuth();
      }, 2000);
    }

    async function pollGoogleDriveAuth() {
      if (generation !== googleDrivePollGeneration) return;

      try {
        const status = await checkGoogleDriveAuth();
        if (generation !== googleDrivePollGeneration) return;

        applyGoogleDriveAuthStatus(status);
        if (status.state === "authorized") {
          await loadAppState();
          patchState({
            authMessage: status.message || "Google Drive connected.",
          });
          return;
        }

        if (status.state === "expired" || status.state === "error") {
          return;
        }

        scheduleNextPoll();
      } catch (error) {
        if (generation !== googleDrivePollGeneration) return;

        patchState({
          authMessage: extractErrorMessage(
            error,
            "Failed to poll Google Drive authorization status.",
          ),
        });
        scheduleNextPoll();
      }
    }

    scheduleNextPoll();
  }

  async function loadAppState() {
    patchState({ isLoading: true });
    try {
      applyAppState(await getAppState());
      if (get(state).pendingAuth && !get(state).isAuthorized) {
        patchState({ exportProgress: "Waiting for Simkl approval" });
        startDevicePolling(get(state).pendingAuth?.intervalSeconds);
      } else {
        stopDevicePolling();
        patchState({
          exportProgress: get(state).isAuthorized
            ? "Ready for export"
            : "Connect Simkl in Settings",
        });
      }

      if (get(state).pendingGoogleDriveAuth) {
        startGoogleDrivePolling();
      } else {
        stopGoogleDrivePolling();
      }
    } catch (error) {
      patchState({
        authMessage: extractErrorMessage(error, "Failed to load app state."),
        exportProgress: "Unable to load app state",
      });
    } finally {
      patchState({ isLoading: false });
    }
  }

  async function saveSettings() {
    patchState({ isSaving: true });
    try {
      const current = get(state);
      const next = await saveSettingsBinding({
        clientId: current.clientId.trim(),
        clientSecret: current.clientSecret.trim(),
        exportDirectory: current.exportDirectory.trim(),
        backupStorage: current.backupStorage,
        googleDriveClientId: current.googleDriveClientId.trim(),
        googleDriveClientSecret: current.googleDriveClientSecret.trim(),
        googleDriveFolderName: current.googleDriveFolderName.trim(),
      });
      applyAppState(next);
      patchState({ authMessage: "Settings saved." });
    } catch (error) {
      patchState({
        authMessage: extractErrorMessage(error, "Failed to save settings."),
      });
      throw error;
    } finally {
      patchState({ isSaving: false });
    }
  }

  async function saveSchedule() {
    patchState({ isSaving: true });
    try {
      const current = get(state);
      const next = await saveScheduleBinding({
        enabled: current.scheduleEnabled,
        frequency: current.scheduleFrequency,
        time: current.scheduleTime.trim(),
        days: [...current.scheduleDays],
        outputFormat: current.scheduleOutputFormat,
        fieldMode: current.scheduleFieldMode,
        content: [...current.scheduleContent],
        useActivityCheck: current.scheduleUseActivityCheck,
        maxBackupAge: current.scheduleMaxBackupAge.trim() || "24h",
        runIfBackupIsStale: current.scheduleRunIfBackupIsStale,
      });
      applyAppState(next);
      patchState({ authMessage: "Recurring backup settings saved." });
    } catch (error) {
      patchState({
        authMessage: extractErrorMessage(
          error,
          "Failed to save recurring backup settings.",
        ),
      });
      throw error;
    } finally {
      patchState({ isSaving: false });
    }
  }

  async function pickExportDirectory() {
    try {
      const selected = await selectDirectory();
      if (!selected) return;

      patchState({ exportDirectory: selected });
      await saveSettings();
    } catch (error) {
      patchState({
        authMessage: extractErrorMessage(
          error,
          "Failed to choose export directory.",
        ),
      });
    }
  }

  async function beginDeviceLogin() {
    patchState({ authMessage: "", oauthPending: false });

    try {
      await saveSettings();
      const session = await startDeviceAuth();
      patchState({
        pendingAuth: session,
        isAuthorized: false,
        exportProgress: "Waiting for device approval",
        authMessage: `Enter code ${session.userCode} at Simkl to continue.`,
      });
      startDevicePolling(session.intervalSeconds);
    } catch (error) {
      patchState({
        authMessage: extractErrorMessage(
          error,
          "Failed to start Simkl PIN login.",
        ),
      });
    }
  }

  async function beginOAuthLogin() {
    patchState({ authMessage: "", pendingAuth: null });
    stopDevicePolling();

    try {
      await saveSettings();
      const authUrl = await getStandardAuthURL();
      openExternal(authUrl);
      patchState({
        oauthPending: true,
        exportProgress: "Waiting for legacy OAuth code",
        authMessage:
          "Legacy OAuth only: use this only when your Simkl app has a registered redirect URI.",
      });
    } catch (error) {
      patchState({
        authMessage: extractErrorMessage(
          error,
          "Failed to get the Simkl OAuth URL. Prefer PIN login for desktop/CLI.",
        ),
      });
    }
  }

  async function beginGoogleDriveLogin() {
    patchState({ authMessage: "", pendingGoogleDriveAuth: null });
    stopGoogleDrivePolling();

    try {
      await saveSettings();
      const session = await startGoogleDriveAuth();
      patchState({
        pendingGoogleDriveAuth: session,
        authMessage:
          "Finish the Google Drive approval flow in your browser.",
      });
      openExternal(session.authUrl);
      startGoogleDrivePolling();
    } catch (error) {
      patchState({
        authMessage: extractErrorMessage(
          error,
          "Failed to start Google Drive authorization.",
        ),
      });
    }
  }

  async function submitOAuthCode(code: string) {
    const trimmed = code.trim();
    if (!trimmed) return;

    patchState({ authMessage: "" });

    try {
      await saveSettings();
      const next = await exchangeOAuthCode(trimmed);
      applyAppState(next);
      patchState({
        oauthPending: false,
        exportProgress: "OAuth authorization complete",
        authMessage: "Simkl access token saved.",
      });
    } catch (error) {
      patchState({
        authMessage: extractErrorMessage(
          error,
          "Failed to exchange the authorization code.",
        ),
      });
    }
  }

  async function clearToken() {
    try {
      const next = await logout();
      applyAppState(next);
      patchState({
        pendingAuth: null,
        oauthPending: false,
        exportProgress: "Authorization cleared",
        authMessage: "Access token removed.",
      });
      stopDevicePolling();
    } catch (error) {
      patchState({
        authMessage: extractErrorMessage(
          error,
          "Failed to clear the saved token.",
        ),
      });
    }
  }

  async function disconnectGoogleDrive() {
    try {
      const next = await disconnectGoogleDriveBinding();
      applyAppState(next);
      patchState({ pendingGoogleDriveAuth: null, authMessage: "Google Drive connection removed." });
      stopGoogleDrivePolling();
    } catch (error) {
      patchState({
        authMessage: extractErrorMessage(
          error,
          "Failed to disconnect Google Drive.",
        ),
      });
    }
  }

  async function exportSettingsBackup(password: string) {
    patchState({ isSettingsBackupBusy: true });
    try {
      await saveSettings();
      const exportedPath = await exportSettingsBackupBinding(password.trim());
      if (!exportedPath) {
        return "";
      }

      patchState({
        authMessage: `Encrypted settings backup exported to ${exportedPath}.`,
      });
      return exportedPath;
    } catch (error) {
      patchState({
        authMessage: extractErrorMessage(
          error,
          "Failed to export the encrypted settings backup.",
        ),
      });
      throw error;
    } finally {
      patchState({ isSettingsBackupBusy: false });
    }
  }

  async function importSettingsBackup(password: string) {
    patchState({ isSettingsBackupBusy: true });
    try {
      const importedPath = await importSettingsBackupBinding(password.trim());
      if (!importedPath) {
        return "";
      }

      await loadAppState();
      patchState({
        authMessage: `Encrypted settings backup imported from ${importedPath}.`,
      });
      return importedPath;
    } catch (error) {
      patchState({
        authMessage: extractErrorMessage(
          error,
          "Failed to import the encrypted settings backup.",
        ),
      });
      throw error;
    } finally {
      patchState({ isSettingsBackupBusy: false });
    }
  }

  function openPendingAuthUrl() {
    const current = get(state);
    const target = current.pendingAuth?.pinUrl ?? current.pendingAuth?.verificationUrl;
    if (!target) return;

    openExternal(target);
  }

  function openPendingGoogleDriveUrl() {
    const current = get(state);
    const target =
      current.pendingGoogleDriveAuth?.authUrl ?? current.googleDriveFolderUrl;
    if (!target) return;

    openExternal(target);
  }

  return {
    subscribe: state.subscribe,
    patchState,
    loadAppState,
    saveSettings,
    saveSchedule,
    pickExportDirectory,
    beginDeviceLogin,
    beginOAuthLogin,
    beginGoogleDriveLogin,
    submitOAuthCode,
    clearToken,
    disconnectGoogleDrive,
    exportSettingsBackup,
    importSettingsBackup,
    openPendingAuthUrl,
    openPendingGoogleDriveUrl,
  };
}

export const appStore = createAppStore();
