<script lang="ts">
  import { defaultGoogleDriveFolderName, appStore } from "@/stores/app";
  import { openExternal } from "@/lib/wails";
  import type { AppStoreState } from "@/stores/app";
  import type { MediaType, OutputFormat } from "@/types/simkl";

  type SettingsSectionId = "destination" | "automation" | "simkl";
  type BackupAction = "export" | "import" | null;
  type SettingsSection = {
    id: SettingsSectionId;
    label: string;
    description: string;
    status: string;
  };
  type SummaryRow = {
    label: string;
    value: string;
    mono?: boolean;
  };

  const weekdayOptions = [
    { value: "mon", label: "Mon" },
    { value: "tue", label: "Tue" },
    { value: "wed", label: "Wed" },
    { value: "thu", label: "Thu" },
    { value: "fri", label: "Fri" },
    { value: "sat", label: "Sat" },
    { value: "sun", label: "Sun" },
  ] as const;

  const weekdayLabels: Record<string, string> = weekdayOptions.reduce(
    (accumulator, option) => {
      accumulator[option.value] = option.label;
      return accumulator;
    },
    {} as Record<string, string>,
  );

  const mediaOptions: Array<{ value: MediaType; label: string }> = [
    { value: "shows", label: "Series" },
    { value: "movies", label: "Movies" },
    { value: "anime", label: "Anime" },
  ];

  const mediaLabels: Record<MediaType, string> = {
    shows: "Series",
    movies: "Movies",
    anime: "Anime",
  };

  let activeSection: SettingsSectionId = "destination";
  let showClientSecret = false;
  let showGoogleDriveClientSecret = false;
  let showTelegramBotToken = false;
  let showGoogleDriveGuide = false;
  let oauthCode = "";
  let activeBackupAction: BackupAction = null;
  let exportPassword = "";
  let exportPasswordConfirm = "";
  let importPassword = "";
  let showExportPassword = false;
  let showImportPassword = false;

  $: state = $appStore;
  $: normalizedGoogleDriveFolderName =
    state.googleDriveFolderName.trim() || defaultGoogleDriveFolderName;
  $: backupDestinationLabel =
    state.backupStorage === "gdrive"
      ? `Google Drive / ${normalizedGoogleDriveFolderName}`
      : state.backupStorage === "telegram"
        ? `Telegram chat ${state.telegramChatId.trim() || "not configured"}`
        : state.exportDirectory.trim() || "Choose a folder";
  $: simklSecretLabel = state.hasClientSecret
    ? "Secret stored"
    : "Secret missing";
  $: googleDriveSecretLabel = state.hasGoogleDriveClientSecret
    ? "Secret stored"
    : "Secret missing";
  $: simklSettingsDirty =
    state.clientId.trim() !== state.savedClientId ||
    Boolean(state.clientSecret.trim());
  $: localDestinationDirty =
    state.backupStorage !== state.savedBackupStorage ||
    state.exportDirectory.trim() !== state.savedExportDirectory.trim();
  $: googleDriveSettingsDirty =
    state.backupStorage !== state.savedBackupStorage ||
    state.googleDriveClientId.trim() !== state.savedGoogleDriveClientId ||
    Boolean(state.googleDriveClientSecret.trim()) ||
    normalizedGoogleDriveFolderName !== state.savedGoogleDriveFolderName;
  $: telegramSettingsDirty =
    state.backupStorage !== state.savedBackupStorage ||
    Boolean(state.telegramBotToken.trim()) ||
    state.telegramChatId.trim() !== state.savedTelegramChatId ||
    state.telegramThreadId.trim() !== state.savedTelegramThreadId ||
    state.telegramCaption.trim() !== state.savedTelegramCaption;
  $: hasTypedClientSecret = Boolean(state.clientSecret.trim());
  $: showClientSecretToggle = hasTypedClientSecret || state.hasClientSecret;
  $: clientSecretInputType =
    showClientSecret && hasTypedClientSecret ? "text" : "password";
  $: hasTypedGoogleDriveClientSecret = Boolean(
    state.googleDriveClientSecret.trim(),
  );
  $: showGoogleDriveClientSecretToggle =
    hasTypedGoogleDriveClientSecret || state.hasGoogleDriveClientSecret;
  $: googleDriveClientSecretInputType =
    showGoogleDriveClientSecret && hasTypedGoogleDriveClientSecret
      ? "text"
      : "password";
  $: hasTypedTelegramBotToken = Boolean(state.telegramBotToken.trim());
  $: telegramBotTokenInputType = showTelegramBotToken && hasTypedTelegramBotToken
    ? "text"
    : "password";
  $: hasGoogleDriveCredentials =
    Boolean(state.googleDriveClientId.trim()) &&
    (state.hasGoogleDriveClientSecret ||
      Boolean(state.googleDriveClientSecret.trim()));
  $: googleDriveSecretHelp = state.hasGoogleDriveClientSecret
    ? `${googleDriveSecretLabel}. Leave blank to keep the saved secret. Only newly typed text can be shown here.`
    : "Paste the Desktop OAuth client secret to enable browser login. Only newly typed text can be shown here.";
  $: canStartDeviceLogin = Boolean(state.clientId.trim()) && !state.isSaving;
  $: showLoginActions = !state.isAuthorized || simklSettingsDirty;
  $: showConnectDriveAction =
    hasGoogleDriveCredentials &&
    (!state.hasGoogleDriveToken || googleDriveSettingsDirty) &&
    !state.pendingGoogleDriveAuth;
  $: canConnectDrive = showConnectDriveAction && !state.isSaving;
  $: showDisconnectDriveAction = state.hasGoogleDriveToken;
  $: showOpenDriveAction = Boolean(
    state.googleDriveFolderUrl || state.pendingGoogleDriveAuth,
  );
  $: driveStatus =
    state.backupStorage !== "gdrive"
      ? "Not selected"
      : state.hasGoogleDriveToken
        ? "Connected"
        : "Not connected";
  $: telegramStatus =
    state.backupStorage !== "telegram"
      ? "Not selected"
      : state.hasTelegramBotToken && state.telegramChatId.trim()
        ? "Configured"
        : "Incomplete";
  $: scheduleDaysLabel =
    state.scheduleFrequency !== "weekly"
      ? "Every day"
      : state.scheduleDays.length
        ? state.scheduleDays.map((day) => weekdayLabels[day] ?? day).join(", ")
        : "No days selected";
  $: scheduleContentLabel = state.scheduleContent.length
    ? state.scheduleContent
        .map((item) => mediaLabels[item] ?? item)
        .join(", ")
    : "No media selected";
  $: latestMessage =
    state.authMessage ||
    state.scheduleMessage ||
    "Shared settings here drive the GUI, CLI, and recurring backup flow.";
  $: sections = [
    {
      id: "destination",
      label: "Destination",
      description: "Backup location, Drive credentials, Telegram bot settings, and browser approval.",
      status:
        state.backupStorage === "gdrive"
          ? state.pendingGoogleDriveAuth
            ? "Approval pending"
            : state.hasGoogleDriveToken
              ? "Drive connected"
              : "Drive setup required"
          : state.backupStorage === "telegram"
            ? state.hasTelegramBotToken && state.telegramChatId.trim()
              ? "Telegram ready"
              : "Telegram setup required"
            : state.exportDirectory.trim()
              ? "Local folder ready"
              : "Folder required",
    },
    {
      id: "automation",
      label: "Automation",
      description: "Recurring backup schedule, content scope, and output profile.",
      status: !state.scheduleSupported
        ? "Task Scheduler unavailable"
        : state.scheduleEnabled
          ? `${state.scheduleFrequency} at ${state.scheduleTime}`
          : "Recurring backup disabled",
    },
    {
      id: "simkl",
      label: "Simkl",
      description: "Simkl app credentials, tokens, and login flows.",
      status:
        state.pendingAuth || state.oauthPending
          ? "Approval pending"
          : state.isAuthorized
            ? "Connected"
            : "Not connected",
    },
  ] satisfies SettingsSection[];
  $: activeSectionItem =
    sections.find((section) => section.id === activeSection) ?? sections[0];
  $: activeSummaryRows =
    activeSection === "automation"
      ? [
          {
            label: "Schedule",
            value: state.scheduleEnabled
              ? `${state.scheduleFrequency} ${state.scheduleTime}`
              : "Disabled",
          },
          { label: "Days", value: scheduleDaysLabel },
          {
            label: "Output",
            value: `${state.scheduleOutputFormat.toUpperCase()} / ${state.scheduleFieldMode}`,
            mono: true,
          },
          { label: "Content", value: scheduleContentLabel },
        ]
      : activeSection === "simkl"
        ? [
            {
              label: "Status",
              value: state.isAuthorized ? "Connected" : "Not connected",
            },
            { label: "Secret", value: simklSecretLabel },
            {
              label: "Flow",
              value: state.pendingAuth
                ? "Device approval in progress"
                : state.oauthPending
                  ? "OAuth code exchange pending"
                  : "Ready to connect",
            },
            {
              label: "Client",
              value: state.clientId.trim() ? "Configured" : "Missing",
            },
          ]
        : [
            {
              label: "Mode",
              value:
                state.backupStorage === "gdrive"
                  ? "Google Drive"
                  : state.backupStorage === "telegram"
                    ? "Telegram"
                    : "Local folder",
            },
            {
              label: "Destination",
              value: backupDestinationLabel,
              mono: true,
            },
            {
              label: "Drive",
              value:
                state.backupStorage === "gdrive"
                  ? state.hasGoogleDriveToken
                    ? "Connected"
                    : "Connect in browser"
                  : "Inactive",
            },
            {
              label: "Remote",
              value:
                state.backupStorage === "gdrive"
                  ? normalizedGoogleDriveFolderName
                  : state.backupStorage === "telegram"
                    ? telegramStatus
                    : "Inactive",
            },
          ] satisfies SummaryRow[];
  $: currentSetupRows = [
    { label: "Config file path", value: state.configPath || "Not saved yet" },
    {
      label: "Backup storage",
      value:
        state.backupStorage === "gdrive"
          ? "Google Drive"
          : state.backupStorage === "telegram"
            ? "Telegram"
            : "Local folder",
    },
    { label: "Destination", value: backupDestinationLabel },
    { label: "Simkl", value: state.isAuthorized ? "Connected" : "Not connected" },
    { label: "Google Drive", value: driveStatus },
    { label: "Telegram", value: telegramStatus },
  ];
  $: exportInputType = showExportPassword ? "text" : "password";
  $: importInputType = showImportPassword ? "text" : "password";
  $: exportPasswordMismatch =
    Boolean(exportPassword.trim()) &&
    Boolean(exportPasswordConfirm.trim()) &&
    exportPassword !== exportPasswordConfirm;
  $: canExportBackup =
    Boolean(exportPassword.trim()) &&
    Boolean(exportPasswordConfirm.trim()) &&
    !exportPasswordMismatch &&
    !state.isSaving &&
    !state.isSettingsBackupBusy;
  $: canImportBackup =
    Boolean(importPassword.trim()) &&
    !state.isSaving &&
    !state.isSettingsBackupBusy;

  function patchState(patch: Partial<AppStoreState>) {
    appStore.patchState(patch);
  }

  function textValue(event: Event) {
    return (event.currentTarget as HTMLInputElement).value;
  }

  function selectValue(event: Event) {
    return (event.currentTarget as HTMLSelectElement).value;
  }

  function checkedValue(event: Event) {
    return (event.currentTarget as HTMLInputElement).checked;
  }

  function openHelpUrl(url: string) {
    openExternal(url);
  }

  function toggleScheduleDay(day: string) {
    const next = new Set(state.scheduleDays);
    if (next.has(day)) {
      next.delete(day);
    } else {
      next.add(day);
    }

    patchState({
      scheduleDays: weekdayOptions
        .filter((option) => next.has(option.value))
        .map((option) => option.value),
    });
  }

  function toggleScheduleContent(value: MediaType) {
    const next = new Set(state.scheduleContent);
    if (next.has(value)) {
      next.delete(value);
    } else {
      next.add(value);
    }

    patchState({
      scheduleContent: mediaOptions
        .filter((option) => next.has(option.value))
        .map((option) => option.value),
    });
  }

  function formatScheduleTimestamp(value: string) {
    const trimmed = value.trim();
    if (!trimmed) {
      return "Not scheduled yet";
    }

    const parsed = new Date(trimmed);
    if (Number.isNaN(parsed.getTime())) {
      return trimmed;
    }

    return new Intl.DateTimeFormat(undefined, {
      dateStyle: "medium",
      timeStyle: "short",
    }).format(parsed);
  }

  function openBackupAction(action: BackupAction) {
    activeBackupAction = activeBackupAction === action ? null : action;

    if (activeBackupAction !== "export") {
      exportPassword = "";
      exportPasswordConfirm = "";
      showExportPassword = false;
    }

    if (activeBackupAction !== "import") {
      importPassword = "";
      showImportPassword = false;
    }
  }

  async function handleExportBackup() {
    if (!canExportBackup) {
      return;
    }

    await appStore.exportSettingsBackup(exportPassword);
    exportPassword = "";
    exportPasswordConfirm = "";
    showExportPassword = false;
    activeBackupAction = null;
  }

  async function handleImportBackup() {
    if (!canImportBackup) {
      return;
    }

    await appStore.importSettingsBackup(importPassword);
    importPassword = "";
    showImportPassword = false;
    activeBackupAction = null;
  }

  function handleExchangeOAuthCode() {
    const trimmed = oauthCode.trim();
    if (!trimmed) {
      return;
    }

    void appStore.submitOAuthCode(trimmed);
    oauthCode = "";
  }
</script>

<div class="pane-workbench settings-workbench">
  <aside class="settings-rail">
    <div class="settings-rail__header">
      <h2 class="settings-rail__title">Sections</h2>
      <p class="settings-rail__copy">
        Edit storage, automation, and Simkl access without turning the page
        into one long form.
      </p>
    </div>

    <div class="settings-rail__nav" aria-label="Settings sections" role="tablist">
      {#each sections as section (section.id)}
        <button
          id={`settings-tab-${section.id}`}
          aria-controls={`settings-panel-${section.id}`}
          aria-selected={activeSection === section.id}
          class="settings-rail__item"
          class:settings-rail__item--active={activeSection === section.id}
          role="tab"
          type="button"
          onclick={() => {
            activeSection = section.id;
          }}
        >
          <span class="settings-rail__item-label">{section.label}</span>
          <span class="settings-rail__item-copy">{section.description}</span>
          <span class="settings-rail__item-status">{section.status}</span>
        </button>
      {/each}
    </div>

    <div class="settings-rail__footer">
      <div class="settings-note">
        <div class="settings-note__label">Latest message</div>
        <p class="settings-note__text">{latestMessage}</p>
      </div>
    </div>
  </aside>

  <div
    id={`settings-panel-${activeSectionItem.id}`}
    aria-labelledby={`settings-tab-${activeSectionItem.id}`}
    class="settings-main-pane"
    role="tabpanel"
  >
    <div class="settings-main">
      <header class="settings-main__header">
        <div class="settings-main__heading">
          <h2 class="settings-main__title">{activeSectionItem.label}</h2>
          <p class="settings-main__copy">{activeSectionItem.description}</p>
        </div>
        <div class="settings-main__status">{activeSectionItem.status}</div>
      </header>

      {#if activeSection === "destination"}
        <div class="settings-stack">
          <section class="settings-block">
            <div class="settings-block__header">
              <div>
                <h3 class="settings-block__title">Storage mode</h3>
                <p class="settings-block__copy">
                  Choose whether backups are written to a local folder or sent
                  to Google Drive or Telegram.
                </p>
              </div>
            </div>

            <div class="storage-switch" role="radiogroup" aria-label="Backup storage">
              <button
                aria-checked={state.backupStorage === "local"}
                class="storage-switch__button"
                class:storage-switch__button--active={state.backupStorage === "local"}
                role="radio"
                type="button"
                onclick={() => {
                  patchState({ backupStorage: "local" });
                }}
              >
                Local folder
              </button>
              <button
                aria-checked={state.backupStorage === "gdrive"}
                class="storage-switch__button"
                class:storage-switch__button--active={state.backupStorage === "gdrive"}
                role="radio"
                type="button"
                onclick={() => {
                  patchState({ backupStorage: "gdrive" });
                }}
              >
                Google Drive
              </button>
              <button
                aria-checked={state.backupStorage === "telegram"}
                class="storage-switch__button"
                class:storage-switch__button--active={state.backupStorage === "telegram"}
                role="radio"
                type="button"
                onclick={() => {
                  patchState({ backupStorage: "telegram" });
                }}
              >
                Telegram
              </button>
            </div>
          </section>

          {#if state.backupStorage === "local"}
            <section class="settings-block">
              <div class="settings-block__header">
                <div>
                  <h3 class="settings-block__title">Local folder</h3>
                  <p class="settings-block__copy">
                    This folder is shared by the quick export, advanced export,
                    and recurring backup flow.
                  </p>
                </div>
              </div>

              <label class="field">
                <span>Export directory</span>
                <div class="input-with-action">
                  <input
                    placeholder="Choose a folder to store the backup"
                    value={state.exportDirectory}
                    oninput={(event) => {
                      patchState({ exportDirectory: textValue(event) });
                    }}
                  />
                  <button
                    class="button"
                    type="button"
                    onclick={() => void appStore.pickExportDirectory()}
                  >
                    Browse
                  </button>
                </div>
              </label>

              <div class="pane-toolbar">
                {#if localDestinationDirty}
                  <button
                    class="button button--primary"
                    disabled={state.isSaving}
                    type="button"
                    onclick={() => void appStore.saveSettings()}
                  >
                    Save destination
                  </button>
                {/if}
              </div>

              <dl class="settings-data-list">
                <div class="settings-data-list__row">
                  <dt>Mode</dt>
                  <dd>Local folder</dd>
                </div>
                <div class="settings-data-list__row">
                  <dt>Target</dt>
                  <dd class="settings-mono">{backupDestinationLabel}</dd>
                </div>
              </dl>
            </section>
          {:else if state.backupStorage === "gdrive"}
            <section class="settings-block">
              <div class="settings-block__header">
                <div>
                  <h3 class="settings-block__title">Google Drive credentials</h3>
                  <p class="settings-block__copy">
                    Paste the Desktop OAuth client for this app, then finish the
                    approval flow in the browser.
                  </p>
                </div>
              </div>

              <div class="pane-form-grid pane-form-grid--two">
                <label class="field">
                  <span>Google client ID</span>
                  <input
                    placeholder="Desktop OAuth client ID"
                    value={state.googleDriveClientId}
                    oninput={(event) => {
                      patchState({ googleDriveClientId: textValue(event) });
                    }}
                  />
                  <small>
                    Create a Desktop app OAuth client in Google Cloud Console.
                  </small>
                  <div class="credential-links">
                    <button
                      class="credential-link"
                      type="button"
                      onclick={() =>
                        openHelpUrl("https://console.cloud.google.com/apis/credentials")}
                    >
                      Google Cloud credentials
                    </button>
                    <button
                      class="credential-link"
                      type="button"
                      onclick={() =>
                        openHelpUrl(
                          "https://developers.google.com/identity/protocols/oauth2/native-app",
                        )}
                    >
                      Desktop OAuth guide
                    </button>
                  </div>
                </label>

                <label class="field">
                  <span>Google client secret</span>
                  <div class="input-with-action">
                    <input
                      placeholder={state.hasGoogleDriveClientSecret
                        ? "Secret already stored"
                        : "Desktop OAuth client secret"}
                      type={googleDriveClientSecretInputType}
                      value={state.googleDriveClientSecret}
                      oninput={(event) => {
                        patchState({ googleDriveClientSecret: textValue(event) });
                      }}
                    />
                    {#if showGoogleDriveClientSecretToggle}
                      <button
                        aria-label={hasTypedGoogleDriveClientSecret
                          ? showGoogleDriveClientSecret
                            ? "Hide Google client secret"
                            : "Show Google client secret"
                          : "Stored secret cannot be revealed"}
                        class="button secret-toggle"
                        disabled={!hasTypedGoogleDriveClientSecret}
                        title={hasTypedGoogleDriveClientSecret
                          ? showGoogleDriveClientSecret
                            ? "Hide Google client secret"
                            : "Show Google client secret"
                          : "Stored secret cannot be revealed"}
                        type="button"
                        onclick={() => {
                          showGoogleDriveClientSecret = !showGoogleDriveClientSecret;
                        }}
                      >
                        <svg
                          aria-hidden="true"
                          class="secret-toggle__icon"
                          viewBox="0 0 16 16"
                        >
                          <path
                            d="M1.25 8C2.36 5.88 4.79 4 8 4C11.21 4 13.64 5.88 14.75 8C13.64 10.12 11.21 12 8 12C4.79 12 2.36 10.12 1.25 8Z"
                          />
                          <circle cx="8" cy="8" r="2.15" />
                          {#if !showGoogleDriveClientSecret}
                            <path d="M2.2 13.2L13.8 2.8" />
                          {/if}
                        </svg>
                      </button>
                    {/if}
                  </div>
                  <small>{googleDriveSecretHelp}</small>
                </label>
              </div>
            </section>

            <section class="settings-block">
              <div class="settings-block__header">
                <div>
                  <h3 class="settings-block__title">Backup folder</h3>
                  <p class="settings-block__copy">
                    The app reuses this Drive root and uploads each run into a
                    dated child folder.
                  </p>
                </div>
              </div>

              <label class="field">
                <span>Drive folder name</span>
                <input
                  placeholder={defaultGoogleDriveFolderName}
                  value={state.googleDriveFolderName}
                  oninput={(event) => {
                    patchState({ googleDriveFolderName: textValue(event) });
                  }}
                />
              </label>

              <section class="setup-guide">
                <button
                  aria-expanded={showGoogleDriveGuide}
                  class="setup-guide__toggle"
                  type="button"
                  onclick={() => {
                    showGoogleDriveGuide = !showGoogleDriveGuide;
                  }}
                >
                  <span>Google setup guide</span>
                  <span class="setup-guide__toggle-meta">
                    {showGoogleDriveGuide ? "Hide" : "Show"}
                  </span>
                </button>

                {#if showGoogleDriveGuide}
                  <div
                    aria-labelledby="google-drive-guide-title"
                    class="setup-guide__panel"
                  >
                    <div class="setup-guide__header">
                      <h4 class="setup-guide__title" id="google-drive-guide-title">
                        Google setup guide
                      </h4>
                      <p class="setup-guide__description">
                        Use this when Google shows the wrong app name, blocks
                        access, or asks for public URLs.
                      </p>
                    </div>

                    <div class="credential-links credential-links--guide">
                      <button
                        class="credential-link"
                        type="button"
                        onclick={() =>
                          openHelpUrl("https://console.cloud.google.com/auth/branding")}
                      >
                        Branding page
                      </button>
                      <button
                        class="credential-link"
                        type="button"
                        onclick={() =>
                          openHelpUrl("https://console.cloud.google.com/auth/audience")}
                      >
                        Audience page
                      </button>
                      <button
                        class="credential-link"
                        type="button"
                        onclick={() =>
                          openHelpUrl(
                            "https://developers.google.com/workspace/guides/configure-oauth-consent",
                          )}
                      >
                        Consent screen docs
                      </button>
                    </div>

                    <ol class="setup-guide__steps">
                      <li>
                        Create a <strong>Desktop app</strong> OAuth client in
                        Google Cloud Credentials, then paste its client ID and
                        secret here.
                      </li>
                      <li>
                        Google uses <strong>Branding &gt; App name</strong>, not
                        the Desktop client label.
                      </li>
                      <li>
                        For personal use, keep the project in
                        <strong>Audience &gt; Testing</strong> and add your own
                        account to <strong>Test users</strong>.
                      </li>
                      <li>
                        A callback like <code>http://127.0.0.1:20028</code> is
                        expected for a desktop app.
                      </li>
                      <li>
                        Switch to <strong>Production</strong> only when you also
                        have a public home page and privacy policy.
                      </li>
                    </ol>
                  </div>
                {/if}
              </section>

              <dl class="settings-data-list">
                <div class="settings-data-list__row">
                  <dt>Connection</dt>
                  <dd>{state.hasGoogleDriveToken ? "Connected" : "Not connected"}</dd>
                </div>
                <div class="settings-data-list__row">
                  <dt>Target</dt>
                  <dd class="settings-mono">{backupDestinationLabel}</dd>
                </div>
              </dl>

              <div class="pane-toolbar">
                {#if googleDriveSettingsDirty}
                  <button
                    class="button button--primary"
                    disabled={state.isSaving}
                    type="button"
                    onclick={() => void appStore.saveSettings()}
                  >
                    Save storage settings
                  </button>
                {/if}
                {#if showConnectDriveAction}
                  <button
                    class="button"
                    disabled={!canConnectDrive}
                    type="button"
                    onclick={() => void appStore.beginGoogleDriveLogin()}
                  >
                    Connect in browser
                  </button>
                {/if}
                {#if showDisconnectDriveAction}
                  <button
                    class="button button--danger"
                    disabled={!state.hasGoogleDriveToken}
                    type="button"
                    onclick={() => void appStore.disconnectGoogleDrive()}
                  >
                    Disconnect
                  </button>
                {/if}
                {#if showOpenDriveAction}
                  <button
                    class="button"
                    disabled={!state.googleDriveFolderUrl && !state.pendingGoogleDriveAuth}
                    type="button"
                    onclick={() => appStore.openPendingGoogleDriveUrl()}
                  >
                    {state.pendingGoogleDriveAuth
                      ? "Open approval page"
                      : "Open Drive folder"}
                  </button>
                {/if}
              </div>

              <div class="settings-note settings-note--inline">
                <div class="settings-note__text">
                  Connecting in the browser finishes the approval flow and
                  creates the Drive folder immediately.
                </div>
              </div>
            </section>
          {:else if state.backupStorage === "telegram"}
            <section class="settings-block">
              <div class="settings-block__header">
                <div>
                  <h3 class="settings-block__title">Telegram bot destination</h3>
                  <p class="settings-block__copy">
                    Send generated backup files to a Telegram chat, group, channel, or forum topic through a bot.
                  </p>
                </div>
              </div>

              <div class="pane-form-grid pane-form-grid--two">
                <label class="field">
                  <span>Bot token</span>
                  <div class="input-with-action">
                    <input
                      placeholder={state.hasTelegramBotToken
                        ? "Token already stored"
                        : "123456:ABC-DEF..."}
                      type={telegramBotTokenInputType}
                      value={state.telegramBotToken}
                      oninput={(event) => {
                        patchState({ telegramBotToken: textValue(event) });
                      }}
                    />
                    <button
                      class="button button--ghost"
                      disabled={!hasTypedTelegramBotToken}
                      type="button"
                      onclick={() => {
                        showTelegramBotToken = !showTelegramBotToken;
                      }}
                    >
                      {showTelegramBotToken ? "Hide" : "Show"}
                    </button>
                  </div>
                  <small>Leave blank to keep the saved token.</small>
                </label>

                <label class="field">
                  <span>Chat ID</span>
                  <input
                    placeholder="-1001234567890"
                    value={state.telegramChatId}
                    oninput={(event) => {
                      patchState({ telegramChatId: textValue(event) });
                    }}
                  />
                  <small>Add the bot to the target chat or channel first.</small>
                </label>

                <label class="field">
                  <span>Thread ID</span>
                  <input
                    placeholder="Optional forum topic ID"
                    value={state.telegramThreadId}
                    oninput={(event) => {
                      patchState({ telegramThreadId: textValue(event) });
                    }}
                  />
                </label>

                <label class="field">
                  <span>Caption</span>
                  <input
                    placeholder="SimklExpoGter backup"
                    value={state.telegramCaption}
                    oninput={(event) => {
                      patchState({ telegramCaption: textValue(event) });
                    }}
                  />
                </label>
              </div>

              <dl class="settings-data-list">
                <div class="settings-data-list__row">
                  <dt>Status</dt>
                  <dd>{telegramStatus}</dd>
                </div>
                <div class="settings-data-list__row">
                  <dt>Target</dt>
                  <dd class="settings-mono">{backupDestinationLabel}</dd>
                </div>
              </dl>

              <div class="pane-toolbar">
                {#if telegramSettingsDirty}
                  <button
                    class="button button--primary"
                    disabled={state.isSaving}
                    type="button"
                    onclick={() => void appStore.saveSettings()}
                  >
                    Save Telegram settings
                  </button>
                {/if}
              </div>
            </section>
          {/if}
        </div>
      {:else if activeSection === "automation"}
        <div class="settings-stack">
          <section class="settings-block">
            <div class="settings-block__header settings-block__header--split">
              <div>
                <h3 class="settings-block__title">Schedule</h3>
                <p class="settings-block__copy">
                  Run the export pipeline on a daily or weekly schedule through
                  Windows Task Scheduler.
                </p>
              </div>
              <span class="field-help">
                <button
                  aria-label="Explain recurring backup scheduling"
                  class="field-help__button"
                  type="button"
                >
                  i
                </button>
                <span class="field-help__popover" role="tooltip">
                  <strong>Daily</strong> runs once every day at the configured
                  time.
                  <br />
                  <strong>Weekly</strong> runs on the selected weekdays at the
                  configured time.
                </span>
              </span>
            </div>

            <label class="option-check">
              <input
                checked={state.scheduleEnabled}
                disabled={!state.scheduleSupported || state.isSaving}
                type="checkbox"
                onchange={(event) => {
                  patchState({ scheduleEnabled: checkedValue(event) });
                }}
              />
              <span>Enable recurring background backup</span>
            </label>

            <div class="pane-form-grid pane-form-grid--two">
              <label class="field">
                <span>Frequency</span>
                <select
                  disabled={!state.scheduleSupported || state.isSaving}
                  value={state.scheduleFrequency}
                  onchange={(event) => {
                    patchState({
                      scheduleFrequency: selectValue(event) as "daily" | "weekly",
                    });
                  }}
                >
                  <option value="daily">daily</option>
                  <option value="weekly">weekly</option>
                </select>
              </label>

              <label class="field">
                <span>Run time</span>
                <input
                  disabled={!state.scheduleSupported || state.isSaving}
                  maxlength="5"
                  placeholder="02:00"
                  type="time"
                  value={state.scheduleTime}
                  oninput={(event) => {
                    patchState({ scheduleTime: textValue(event) });
                  }}
                />
              </label>
            </div>

            {#if state.scheduleFrequency === "weekly"}
              <fieldset class="fieldset">
                <legend class="fieldset__title">Weekdays</legend>
                <div class="options-grid">
                  {#each weekdayOptions as day (day.value)}
                    <label class="option-check">
                      <input
                        checked={state.scheduleDays.includes(day.value)}
                        disabled={!state.scheduleSupported || state.isSaving}
                        type="checkbox"
                        onchange={() => {
                          toggleScheduleDay(day.value);
                        }}
                      />
                      <span>{day.label}</span>
                    </label>
                  {/each}
                </div>
              </fieldset>
            {/if}
          </section>

          <section class="settings-block">
            <div class="settings-block__header">
              <div>
                <h3 class="settings-block__title">Backup profile</h3>
                <p class="settings-block__copy">
                  Choose output format, field density, content scope, and
                  activity-aware behavior.
                </p>
              </div>
            </div>

            <div class="pane-form-grid pane-form-grid--two">
              <label class="field">
                <span>Output format</span>
                <select
                  disabled={!state.scheduleSupported || state.isSaving}
                  value={state.scheduleOutputFormat}
                  onchange={(event) => {
                    patchState({
                      scheduleOutputFormat: selectValue(event) as OutputFormat,
                    });
                  }}
                >
                  <option value="csv">csv</option>
                  <option value="json">json</option>
                  <option value="both">both</option>
                </select>
              </label>

              <label class="field">
                <span>Field mode</span>
                <select
                  disabled={!state.scheduleSupported || state.isSaving}
                  value={state.scheduleFieldMode}
                  onchange={(event) => {
                    patchState({
                      scheduleFieldMode: selectValue(event) as "compact" | "all",
                    });
                  }}
                >
                  <option value="all">all</option>
                  <option value="compact">compact</option>
                </select>
              </label>
            </div>

            <fieldset class="fieldset">
              <legend class="fieldset__title">Media content</legend>
              <div class="options-grid">
                {#each mediaOptions as option (option.value)}
                  <label class="option-check">
                    <input
                      checked={state.scheduleContent.includes(option.value)}
                      disabled={!state.scheduleSupported || state.isSaving}
                      type="checkbox"
                      onchange={() => {
                        toggleScheduleContent(option.value);
                      }}
                    />
                    <span>{option.label}</span>
                  </label>
                {/each}
              </div>
            </fieldset>

            <label class="option-check">
              <input
                checked={state.scheduleUseActivityCheck}
                disabled={!state.scheduleSupported || state.isSaving}
                type="checkbox"
                onchange={(event) => {
                  patchState({ scheduleUseActivityCheck: checkedValue(event) });
                }}
              />
              <span>Use activity check before the scheduled export</span>
            </label>

            <label class="option-check">
              <input
                checked={state.scheduleRunIfBackupIsStale}
                disabled={!state.scheduleSupported || state.isSaving}
                type="checkbox"
                onchange={(event) => {
                  patchState({ scheduleRunIfBackupIsStale: checkedValue(event) });
                }}
              />
              <span>Skip scheduled backup while the last successful backup is still fresh</span>
            </label>

            <label class="field">
              <span>Maximum backup age</span>
              <input
                disabled={!state.scheduleSupported || state.isSaving || !state.scheduleRunIfBackupIsStale}
                placeholder="24h, 3d, 1w"
                value={state.scheduleMaxBackupAge}
                oninput={(event) => {
                  patchState({ scheduleMaxBackupAge: textValue(event) });
                }}
              />
              <small class="field__hint">
                Scheduled runs start only when the last successful backup is older than this value. Examples: 12h, 24h, 3d, 1w.
              </small>
            </label>

            <div class="pane-toolbar">
              <button
                class="button button--primary"
                disabled={state.isSaving}
                type="button"
                onclick={() => void appStore.saveSchedule()}
              >
                Save recurring backup
              </button>
            </div>
          </section>

          <section class="settings-block">
            <div class="settings-block__header">
              <div>
                <h3 class="settings-block__title">Scheduler state</h3>
                <p class="settings-block__copy">
                  Review the registered task and the last known run details.
                </p>
              </div>
            </div>

            <dl class="settings-data-list">
              <div class="settings-data-list__row">
                <dt>Task</dt>
                <dd>{state.scheduleTaskName || "SimklExpoGterRecurringBackup"}</dd>
              </div>
              <div class="settings-data-list__row">
                <dt>Status</dt>
                <dd>
                  {state.scheduleStatus ||
                    (state.scheduleEnabled
                      ? "Recurring backup task is enabled."
                      : "Recurring backups are disabled.")}
                </dd>
              </div>
              <div class="settings-data-list__row">
                <dt>Destination</dt>
                <dd class="settings-mono">
                  {state.scheduleOutputDirectoryPreview || "Not set"}
                </dd>
              </div>
              <div class="settings-data-list__row">
                <dt>Next run</dt>
                <dd>{formatScheduleTimestamp(state.scheduleNextRunAt)}</dd>
              </div>
              <div class="settings-data-list__row">
                <dt>Last run</dt>
                <dd>{formatScheduleTimestamp(state.scheduleLastRunAt)}</dd>
              </div>
              <div class="settings-data-list__row">
                <dt>Last successful backup</dt>
                <dd>
                  {formatScheduleTimestamp(state.scheduleLastSuccessfulBackupAt)}
                  {#if state.scheduleLastSuccessfulBackupKind}
                    ({state.scheduleLastSuccessfulBackupKind})
                  {/if}
                </dd>
              </div>
              <div class="settings-data-list__row">
                <dt>Stale threshold</dt>
                <dd>
                  {state.scheduleRunIfBackupIsStale
                    ? `${state.scheduleMaxBackupAge || "24h"} (${state.scheduleBackupFresh ? "fresh" : "stale or unknown"})`
                    : "disabled"}
                </dd>
              </div>
              {#if state.scheduleLastResult}
                <div class="settings-data-list__row">
                  <dt>Last result</dt>
                  <dd>{state.scheduleLastResult}</dd>
                </div>
              {/if}
            </dl>

            <div
              class="settings-note settings-note--inline"
              class:settings-note--warning={state.scheduleEnabled && !state.scheduleInstalled}
            >
              <div class="settings-note__text">
                {state.scheduleMessage ||
                  "Recurring backups use the saved destination and the saved Simkl token."}
              </div>
            </div>
          </section>
        </div>
      {:else}
        <div class="settings-stack">
          <section class="settings-block">
            <div class="settings-block__header">
              <div>
                <h3 class="settings-block__title">Credentials</h3>
                <p class="settings-block__copy">
                  Save the Simkl app credentials used by the desktop client.
                </p>
              </div>
            </div>

            <div class="pane-form-grid pane-form-grid--two">
              <label class="field">
                <span>Client ID</span>
                <input
                  placeholder="Paste your Simkl client ID"
                  value={state.clientId}
                  oninput={(event) => {
                    patchState({ clientId: textValue(event) });
                  }}
                />
                <small>Get both values from your Simkl app registration.</small>
                <div class="credential-links">
                  <button
                    class="credential-link"
                    type="button"
                    onclick={() => openHelpUrl("https://simkl.com/settings/developer/")}
                  >
                    Developer settings
                  </button>
                  <button
                    class="credential-link"
                    type="button"
                    onclick={() =>
                      openHelpUrl(
                        "https://docs.simkl.org/how-to-use-simkl/for-developers/how-to-register-an-app",
                      )}
                  >
                    Registration guide
                  </button>
                  <button
                    class="credential-link"
                    type="button"
                    onclick={() => openHelpUrl("https://simkl.docs.apiary.io/")}
                  >
                    OAuth / API docs
                  </button>
                </div>
              </label>

              <label class="field">
                <span>Client secret</span>
                <div class="input-with-action">
                  <input
                    placeholder={state.hasClientSecret
                      ? "Secret already stored"
                      : "Paste your Simkl client secret"}
                    type={clientSecretInputType}
                    value={state.clientSecret}
                    oninput={(event) => {
                      patchState({ clientSecret: textValue(event) });
                    }}
                  />
                  {#if showClientSecretToggle}
                    <button
                      aria-label={hasTypedClientSecret
                        ? showClientSecret
                          ? "Hide client secret"
                          : "Show client secret"
                        : "Stored secret cannot be revealed"}
                      class="button secret-toggle"
                      disabled={!hasTypedClientSecret}
                      title={hasTypedClientSecret
                        ? showClientSecret
                          ? "Hide client secret"
                          : "Show client secret"
                        : "Stored secret cannot be revealed"}
                      type="button"
                      onclick={() => {
                        showClientSecret = !showClientSecret;
                      }}
                    >
                      <svg
                        aria-hidden="true"
                        class="secret-toggle__icon"
                        viewBox="0 0 16 16"
                      >
                        <path
                          d="M1.25 8C2.36 5.88 4.79 4 8 4C11.21 4 13.64 5.88 14.75 8C13.64 10.12 11.21 12 8 12C4.79 12 2.36 10.12 1.25 8Z"
                        />
                        <circle cx="8" cy="8" r="2.15" />
                        {#if !showClientSecret}
                          <path d="M2.2 13.2L13.8 2.8" />
                        {/if}
                      </svg>
                    </button>
                  {/if}
                </div>
                <small>
                  Leave blank to keep the saved secret. Only newly typed text
                  can be shown here.
                </small>
              </label>
            </div>

            <div class="pane-toolbar">
              {#if simklSettingsDirty}
                <button
                  class="button button--primary"
                  disabled={state.isSaving}
                  type="button"
                  onclick={() => void appStore.saveSettings()}
                >
                  Save Simkl settings
                </button>
              {/if}
            </div>
          </section>

          <section class="settings-block">
            <div class="settings-block__header">
              <div>
                <h3 class="settings-block__title">Authorization</h3>
                <p class="settings-block__copy">
                  Start Simkl PIN login, or clear the saved access token.
                </p>
              </div>
            </div>

            <dl class="settings-data-list">
              <div class="settings-data-list__row">
                <dt>Status</dt>
                <dd>{state.isAuthorized ? "Connected" : "Not connected"}</dd>
              </div>
              <div class="settings-data-list__row">
                <dt>Secret</dt>
                <dd>{simklSecretLabel}</dd>
              </div>
              <div class="settings-data-list__row">
                <dt>Flow</dt>
                <dd>
                  {state.pendingAuth
                    ? "PIN approval in progress"
                    : state.oauthPending
                      ? "Legacy OAuth code exchange pending"
                      : "Ready to connect"}
                </dd>
              </div>
            </dl>

            <div class="pane-toolbar">
              {#if showLoginActions}
                <button
                  class="button"
                  disabled={!canStartDeviceLogin}
                  type="button"
                  onclick={() => void appStore.beginDeviceLogin()}
                >
                  PIN login
                </button>
              {/if}
              <button
                class="button button--danger"
                disabled={!state.isAuthorized}
                type="button"
                onclick={() => void appStore.clearToken()}
              >
                Clear token
              </button>
            </div>
          </section>

          {#if state.pendingAuth}
            <section class="settings-block">
              <div class="settings-block__header">
                <div>
                  <h3 class="settings-block__title">PIN approval</h3>
                  <p class="settings-block__copy">
                    Open the Simkl PIN page in the browser and enter the temporary
                    code shown below.
                  </p>
                </div>
              </div>

              <dl class="settings-data-list">
                <div class="settings-data-list__row">
                  <dt>Code</dt>
                  <dd class="settings-mono">{state.pendingAuth.userCode}</dd>
                </div>
                <div class="settings-data-list__row">
                  <dt>URL</dt>
                  <dd class="settings-mono">{state.pendingAuth.verificationUrl}</dd>
                </div>
                <div class="settings-data-list__row">
                  <dt>Expires</dt>
                  <dd>{state.pendingAuth.expiresAt}</dd>
                </div>
              </dl>

              <div class="pane-toolbar">
                <button
                  class="button"
                  type="button"
                  onclick={() => appStore.openPendingAuthUrl()}
                >
                  Open Simkl page
                </button>
              </div>
            </section>
          {/if}

          {#if state.oauthPending}
            <section class="settings-block">
              <div class="settings-block__header">
                <div>
                  <h3 class="settings-block__title">Legacy OAuth code exchange</h3>
                  <p class="settings-block__copy">
                    Only use this if your Simkl developer app has a registered
                    redirect URI.
                  </p>
                </div>
              </div>

              <form
                class="settings-inline-form"
                onsubmit={(event) => {
                  event.preventDefault();
                  handleExchangeOAuthCode();
                }}
              >
                <label class="field">
                  <span>Authorization code</span>
                  <div class="input-with-action">
                    <input
                      class="code-input"
                      placeholder="Paste the returned code"
                      value={oauthCode}
                      oninput={(event) => {
                        oauthCode = textValue(event);
                      }}
                    />
                    <button
                      class="button button--primary"
                      disabled={!oauthCode.trim() || state.isSaving}
                      type="submit"
                    >
                      Exchange
                    </button>
                  </div>
                </label>
              </form>
            </section>
          {/if}
        </div>
      {/if}
    </div>
  </div>

  <aside class="settings-side-pane">
    <div class="settings-side">
      <section class="settings-side__block">
        <div class="settings-side__header">
          <h3 class="settings-side__title">{activeSectionItem.label}</h3>
          <p class="settings-side__copy">{activeSectionItem.description}</p>
        </div>

        <dl class="settings-summary-list">
          {#each activeSummaryRows as row (row.label)}
            <div class="settings-summary-list__row">
              <dt>{row.label}</dt>
              <dd class:settings-mono={row.mono}>{row.value}</dd>
            </div>
          {/each}
        </dl>
      </section>

      <section class="settings-side__block">
        <div class="settings-side__header">
          <h3 class="settings-side__title">Environment</h3>
          <p class="settings-side__copy">
            The GUI, CLI, and scheduler all read from the same saved config.
          </p>
        </div>

        <dl class="settings-summary-list settings-summary-list--stacked">
          {#each currentSetupRows as row (row.label)}
            <div class="settings-summary-list__row">
              <dt>{row.label}</dt>
              <dd class:settings-mono={row.label === "Config file path" || row.label === "Destination"}>
                {row.value}
              </dd>
            </div>
          {/each}
        </dl>
      </section>

      <section class="settings-side__block">
        <div class="settings-side__header">
          <h3 class="settings-side__title">Settings backup</h3>
          <p class="settings-side__copy">
            Export or restore the full saved config as one encrypted backup.
          </p>
        </div>

        <div class="settings-note settings-note--warning">
          <div class="settings-note__text">
            Includes client secrets, access tokens, refresh tokens, Google
            Drive connection data, schedule settings, and the activity
            snapshot.
          </div>
        </div>

        <div class="settings-vault-actions">
          <button
            class="button"
            type="button"
            onclick={() => {
              openBackupAction("export");
            }}
          >
            {activeBackupAction === "export" ? "Hide export" : "Export backup"}
          </button>
          <button
            class="button"
            type="button"
            onclick={() => {
              openBackupAction("import");
            }}
          >
            {activeBackupAction === "import" ? "Hide import" : "Import backup"}
          </button>
        </div>

        {#if activeBackupAction === "export"}
          <form
            class="settings-inline-form"
            onsubmit={(event) => {
              event.preventDefault();
              void handleExportBackup();
            }}
          >
            <label class="field">
              <span>Password</span>
              <div class="input-with-action">
                <input
                  placeholder="Enter a backup password"
                  type={exportInputType}
                  value={exportPassword}
                  oninput={(event) => {
                    exportPassword = textValue(event);
                  }}
                />
                <button
                  aria-label={showExportPassword
                    ? "Hide backup password"
                    : "Show backup password"}
                  class="button secret-toggle"
                  type="button"
                  onclick={() => {
                    showExportPassword = !showExportPassword;
                  }}
                >
                  <svg
                    aria-hidden="true"
                    class="secret-toggle__icon"
                    viewBox="0 0 16 16"
                  >
                    <path
                      d="M1.25 8C2.36 5.88 4.79 4 8 4C11.21 4 13.64 5.88 14.75 8C13.64 10.12 11.21 12 8 12C4.79 12 2.36 10.12 1.25 8Z"
                    />
                    <circle cx="8" cy="8" r="2.15" />
                    {#if !showExportPassword}
                      <path d="M2.2 13.2L13.8 2.8" />
                    {/if}
                  </svg>
                </button>
              </div>
            </label>

            <label class="field">
              <span>Confirm password</span>
              <input
                placeholder="Repeat the backup password"
                type={exportInputType}
                value={exportPasswordConfirm}
                oninput={(event) => {
                  exportPasswordConfirm = textValue(event);
                }}
              />
            </label>

            {#if exportPasswordMismatch}
              <div class="settings-note settings-note--warning">
                <div class="settings-note__text">The backup passwords do not match.</div>
              </div>
            {/if}

            <div class="pane-toolbar">
              <button
                class="button button--primary"
                disabled={!canExportBackup}
                type="submit"
              >
                {state.isSettingsBackupBusy ? "Exporting..." : "Choose backup file"}
              </button>
            </div>
          </form>
        {/if}

        {#if activeBackupAction === "import"}
          <form
            class="settings-inline-form"
            onsubmit={(event) => {
              event.preventDefault();
              void handleImportBackup();
            }}
          >
            <label class="field">
              <span>Password</span>
              <div class="input-with-action">
                <input
                  placeholder="Enter the backup password"
                  type={importInputType}
                  value={importPassword}
                  oninput={(event) => {
                    importPassword = textValue(event);
                  }}
                />
                <button
                  aria-label={showImportPassword
                    ? "Hide backup password"
                    : "Show backup password"}
                  class="button secret-toggle"
                  type="button"
                  onclick={() => {
                    showImportPassword = !showImportPassword;
                  }}
                >
                  <svg
                    aria-hidden="true"
                    class="secret-toggle__icon"
                    viewBox="0 0 16 16"
                  >
                    <path
                      d="M1.25 8C2.36 5.88 4.79 4 8 4C11.21 4 13.64 5.88 14.75 8C13.64 10.12 11.21 12 8 12C4.79 12 2.36 10.12 1.25 8Z"
                    />
                    <circle cx="8" cy="8" r="2.15" />
                    {#if !showImportPassword}
                      <path d="M2.2 13.2L13.8 2.8" />
                    {/if}
                  </svg>
                </button>
              </div>
              <small>
                Import replaces the saved config file, including all sensitive
                values.
              </small>
            </label>

            <div class="pane-toolbar">
              <button
                class="button button--primary"
                disabled={!canImportBackup}
                type="submit"
              >
                {state.isSettingsBackupBusy ? "Importing..." : "Choose backup file"}
              </button>
            </div>
          </form>
        {/if}
      </section>
    </div>
  </aside>
</div>

<style>
  .settings-workbench {
    grid-template-columns: 220px minmax(0, 1fr) 320px;
    height: 100%;
  }

  .settings-rail,
  .settings-main-pane,
  .settings-side-pane {
    min-height: 0;
  }

  .settings-rail {
    display: grid;
    grid-template-rows: auto minmax(0, 1fr) auto;
    overflow: hidden;
    border-right: 1px solid var(--border);
    background: var(--surface-2);
  }

  .settings-rail__header,
  .settings-rail__footer {
    padding: 18px 18px 16px;
  }

  .settings-rail__header {
    border-bottom: 1px solid var(--border);
  }

  .settings-rail__title,
  .settings-side__title,
  .settings-block__title,
  .setup-guide__title,
  .settings-main__title {
    margin: 0;
    font-size: 15px;
    font-weight: 600;
    line-height: 1.3;
    color: var(--text);
  }

  .settings-rail__copy,
  .settings-side__copy,
  .settings-block__copy,
  .settings-main__copy,
  .setup-guide__description {
    margin: 0;
    font-size: 12px;
    line-height: 1.55;
    color: var(--text-muted);
  }

  .settings-rail__nav {
    min-height: 0;
    display: grid;
    align-content: start;
    gap: 4px;
    padding: 10px 8px;
    overflow-y: auto;
  }

  .settings-rail__item {
    display: grid;
    gap: 4px;
    width: 100%;
    padding: 12px 12px 11px;
    border: 1px solid transparent;
    border-radius: 8px;
    background: transparent;
    text-align: left;
    transition:
      background-color 120ms ease,
      border-color 120ms ease,
      color 120ms ease;
  }

  .settings-rail__item:hover {
    background: rgba(255, 255, 255, 0.03);
    border-color: rgba(255, 255, 255, 0.04);
  }

  .settings-rail__item--active {
    border-color: var(--accent-border);
    background: rgba(59, 209, 181, 0.08);
  }

  .settings-rail__item-label {
    font-size: 13px;
    font-weight: 600;
    line-height: 1.3;
    color: var(--text);
  }

  .settings-rail__item-copy {
    font-size: 11px;
    line-height: 1.45;
    color: var(--text-muted);
  }

  .settings-rail__item-status {
    font-size: 11px;
    line-height: 1.35;
    color: var(--text-subtle);
    font-family: "IBM Plex Mono", "Cascadia Code", Consolas, monospace;
  }

  .settings-note {
    display: grid;
    gap: 6px;
    padding: 12px;
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 8px;
    background: rgba(255, 255, 255, 0.02);
  }

  .settings-note__label {
    font-size: 11px;
    font-weight: 600;
    line-height: 1.3;
    color: var(--text);
  }

  .settings-note__text {
    font-size: 12px;
    line-height: 1.55;
    color: var(--text-muted);
  }

  .settings-note--inline {
    padding: 10px 12px;
  }

  .settings-note--warning {
    border-color: rgba(191, 104, 104, 0.35);
    background: rgba(110, 46, 46, 0.12);
  }

  .settings-note--warning .settings-note__text {
    color: #f2c4c4;
  }

  .settings-main-pane,
  .settings-side-pane {
    overflow-y: auto;
  }

  .settings-side-pane {
    border-left: 1px solid var(--border);
    background: var(--surface-2);
  }

  .settings-main {
    max-width: 980px;
    margin: 0 auto;
    padding: 22px 24px 28px;
    display: grid;
    gap: 18px;
  }

  .settings-main__header {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 12px;
    align-items: start;
  }

  .settings-main__heading {
    display: grid;
    gap: 4px;
  }

  .settings-main__status {
    padding: 7px 10px;
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 8px;
    background: var(--surface-2);
    font-size: 11px;
    line-height: 1.35;
    color: var(--text);
    font-family: "IBM Plex Mono", "Cascadia Code", Consolas, monospace;
    white-space: nowrap;
  }

  .settings-stack,
  .settings-side {
    display: grid;
    gap: 16px;
    align-content: start;
  }

  .settings-side {
    min-height: 100%;
    padding: 18px;
  }

  .settings-block,
  .settings-side__block {
    display: grid;
    gap: 14px;
    padding: 18px;
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 8px;
    background: var(--surface-2);
  }

  .settings-block__header,
  .settings-side__header {
    display: grid;
    gap: 4px;
  }

  .settings-block__header--split {
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 12px;
    align-items: start;
  }

  .settings-main-pane .field > span,
  .settings-main-pane .fieldset__title,
  .settings-side-pane .field > span {
    font-size: 12px;
    font-weight: 600;
    line-height: 1.35;
    letter-spacing: 0;
    text-transform: none;
    color: var(--text-muted);
  }

  .settings-main-pane .field small,
  .settings-side-pane .field small {
    font-size: 12px;
    line-height: 1.5;
  }

  .settings-data-list,
  .settings-summary-list {
    display: grid;
    gap: 0;
    margin: 0;
  }

  .settings-data-list__row,
  .settings-summary-list__row {
    display: grid;
    gap: 4px 12px;
    min-width: 0;
    padding: 10px 0;
    border-top: 1px solid rgba(255, 255, 255, 0.05);
  }

  .settings-data-list__row:first-child,
  .settings-summary-list__row:first-child {
    padding-top: 0;
    border-top: 0;
  }

  .settings-data-list__row {
    grid-template-columns: 130px minmax(0, 1fr);
  }

  .settings-summary-list__row {
    grid-template-columns: minmax(0, 1fr);
  }

  .settings-data-list dt,
  .settings-summary-list dt {
    font-size: 11px;
    font-weight: 600;
    line-height: 1.35;
    color: var(--text-muted);
  }

  .settings-data-list dd,
  .settings-summary-list dd {
    margin: 0;
    font-size: 12px;
    line-height: 1.55;
    color: var(--text);
    overflow-wrap: anywhere;
  }

  .settings-mono {
    font-family: "IBM Plex Mono", "Cascadia Code", Consolas, monospace;
  }

  .settings-vault-actions {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }

  .settings-inline-form {
    display: grid;
    gap: 12px;
    padding-top: 2px;
  }

  .storage-switch {
    display: inline-grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    border: 1px solid rgba(255, 255, 255, 0.07);
    border-radius: 8px;
    overflow: hidden;
    background: #11161c;
  }

  .storage-switch__button {
    min-height: 38px;
    padding: 0 14px;
    border: 0;
    border-right: 1px solid rgba(255, 255, 255, 0.06);
    background: transparent;
    color: var(--text-muted);
    text-align: left;
    transition:
      background-color 120ms ease,
      color 120ms ease;
  }

  .storage-switch__button:last-child {
    border-right: 0;
  }

  .storage-switch__button:hover {
    background: rgba(255, 255, 255, 0.03);
    color: var(--text);
  }

  .storage-switch__button--active {
    background: rgba(59, 209, 181, 0.1);
    color: var(--text);
  }

  .setup-guide {
    display: grid;
    gap: 10px;
  }

  .setup-guide__toggle {
    min-height: 34px;
    padding: 0 12px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 8px;
    background: #11161c;
    color: var(--text);
    text-align: left;
  }

  .setup-guide__toggle:hover {
    background: var(--surface-3);
  }

  .setup-guide__toggle-meta {
    color: var(--text-muted);
    font-size: 11px;
    line-height: 1.35;
  }

  .setup-guide__panel {
    display: grid;
    gap: 12px;
    padding: 14px;
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 8px;
    background: rgba(255, 255, 255, 0.02);
  }

  .setup-guide__steps {
    margin: 0;
    padding-left: 18px;
    display: grid;
    gap: 8px;
    color: var(--text-muted);
    font-size: 12px;
    line-height: 1.55;
  }

  .setup-guide__steps strong,
  .setup-guide__steps code {
    color: var(--text);
  }

  @media (max-width: 1480px) {
    .settings-workbench {
      grid-template-columns: 208px minmax(0, 1fr) 304px;
    }
  }

  @media (max-width: 1320px) {
    .settings-workbench {
      grid-template-columns: 208px minmax(0, 1fr);
    }

    .settings-side-pane {
      grid-column: 1 / -1;
      border-left: 0;
      border-top: 1px solid var(--border);
      max-height: 360px;
    }
  }

  @media (max-width: 940px) {
    .settings-workbench {
      grid-template-columns: minmax(0, 1fr);
      grid-template-rows: auto auto auto;
      height: auto;
    }

    .settings-rail {
      border-right: 0;
      border-bottom: 1px solid var(--border);
    }

    .settings-rail__nav {
      grid-template-columns: repeat(3, minmax(180px, 1fr));
      overflow-x: auto;
      overflow-y: hidden;
    }

    .settings-main-pane,
    .settings-side-pane {
      border-left: 0;
      border-top: 1px solid var(--border);
      overflow: visible;
      max-height: none;
    }

    .settings-main__header,
    .settings-block__header--split,
    .settings-vault-actions,
    .pane-form-grid--two,
    .settings-data-list__row {
      grid-template-columns: minmax(0, 1fr);
    }
  }
</style>
