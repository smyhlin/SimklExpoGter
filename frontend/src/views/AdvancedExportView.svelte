<script lang="ts">
  import { appStore } from "@/stores/app";
  import { runExport } from "@/lib/wails";
  import type {
    ExportRequest,
    ExportResult,
    ExtendedMode,
    FieldMode,
    Grouping,
    MediaType,
    OutputFormat,
    PendingExportSummary,
    Status,
  } from "@/types/simkl";
  import ExportResultsPanel from "@/components/export-svelte/ExportResultsPanel.svelte";
  import ExportSummaryPanel from "@/components/export-svelte/ExportSummaryPanel.svelte";

  type AdvancedRequest = Omit<ExportRequest, "exportDirectory">;

  let isExporting = false;
  let result: ExportResult | null = null;
  let message = "No advanced export has been run yet.";
  let warnings: string[] = [];

  const mediaTypes: MediaType[] = ["shows", "movies", "anime"];
  const statuses: Status[] = [
    "watching",
    "plantowatch",
    "hold",
    "completed",
    "dropped",
  ];
  const extendedModes: ExtendedMode[] = [
    "full",
    "full_anime_seasons",
    "simkl_ids_only",
    "ids_only",
  ];
  const outputFormats: OutputFormat[] = ["csv", "json", "both"];
  const fieldModes: FieldMode[] = ["compact", "all"];
  const groupings: Grouping[] = ["single-file", "separate-files"];

  let request: AdvancedRequest = {
    types: [...mediaTypes],
    statuses: [...statuses],
    dateFrom: "",
    extended: "full",
    episodeWatchedAt: false,
    includeMemos: false,
    includeNextWatchInfo: false,
    outputFormat: "csv",
    fieldMode: "compact",
    grouping: "separate-files",
    includeEpisodeFiles: false,
    useActivityCheck: true,
    filenamePrefix: "simkl-export",
  };

  $: state = $appStore;
  $: usesRemoteStorage = state.backupStorage === "gdrive" || state.backupStorage === "telegram";
  $: summary = [
    { label: "Destination", value: state.backupDestinationLabel },
    {
      label: "Media",
      value: request.types.length
        ? request.types.map(readableMedia).join(", ")
        : "All media types",
    },
    {
      label: "Statuses",
      value: request.statuses.length
        ? request.statuses.map(readableStatus).join(", ")
        : "All statuses",
    },
    { label: "Date from", value: request.dateFrom || "Full snapshot" },
    { label: "Extended", value: request.extended },
    { label: "Output", value: `${request.outputFormat} / ${request.fieldMode}` },
    {
      label: "Grouping",
      value: request.grouping === "single-file" ? "One file" : "Separate files",
    },
    { label: "Activity-aware", value: request.useActivityCheck ? "Yes" : "No" },
  ] satisfies PendingExportSummary[];

  function readableMedia(value: MediaType) {
    return value === "shows"
      ? "TV Shows"
      : value === "movies"
        ? "Movies"
        : "Anime";
  }

  function readableStatus(value: Status) {
    return value === "plantowatch"
      ? "Plan to watch"
      : value.charAt(0).toUpperCase() + value.slice(1);
  }

  function toggleMediaType(type: MediaType) {
    request = {
      ...request,
      types: request.types.includes(type)
        ? request.types.filter((item) => item !== type)
        : [...request.types, type],
    };
  }

  function toggleStatus(status: Status) {
    request = {
      ...request,
      statuses: request.statuses.includes(status)
        ? request.statuses.filter((item) => item !== status)
        : [...request.statuses, status],
    };
  }

  function buildRequest(): ExportRequest {
    return {
      types: [...request.types],
      statuses: [...request.statuses],
      dateFrom: request.dateFrom,
      extended: request.extended,
      episodeWatchedAt: request.episodeWatchedAt,
      includeMemos: request.includeMemos,
      includeNextWatchInfo: request.includeNextWatchInfo,
      outputFormat: request.outputFormat,
      fieldMode: request.fieldMode,
      grouping: request.grouping,
      includeEpisodeFiles: request.includeEpisodeFiles,
      useActivityCheck: request.useActivityCheck,
      exportDirectory: usesRemoteStorage ? "" : state.exportDirectory.trim(),
      filenamePrefix: request.filenamePrefix,
    };
  }

  async function handleBrowseDirectory() {
    await appStore.pickExportDirectory();
  }

  async function handleAdvancedExport() {
    if (!state.isAuthorized) {
      message = "Authorize in Settings before running the advanced export.";
      return;
    }

    if (state.backupStorage === "gdrive" && !state.hasGoogleDriveToken) {
      message =
        "Connect Google Drive in Settings before running the advanced export.";
      return;
    }
    if (state.backupStorage === "telegram" && (!state.hasTelegramBotToken || !state.telegramChatId.trim())) {
      message = "Configure Telegram in Settings before running the advanced export.";
      return;
    }

    if (!usesRemoteStorage && !state.exportDirectory.trim()) {
      message = "Choose an export directory first.";
      return;
    }

    isExporting = true;
    appStore.patchState({ exportProgress: "Advanced export running" });
    warnings = [];
    message = "Preparing the export request...";

    try {
      await appStore.saveSettings();
      const response = await runExport(buildRequest());
      result = response;
      warnings = response.warnings ?? [];
      message = `Exported ${response.itemCounts.all ?? 0} items to ${response.destinationLabel || response.outputDirectory}.`;
      appStore.patchState({
        exportProgress: `Advanced export complete: ${response.itemCounts.all ?? 0} items`,
      });

      if (response.effectiveDateFrom) {
        request = { ...request, dateFrom: response.effectiveDateFrom };
      }
    } catch (error) {
      result = null;
      warnings = [];
      message =
        error instanceof Error && error.message.trim()
          ? error.message.trim()
          : "The export failed.";
      appStore.patchState({ exportProgress: "Advanced export failed" });
    } finally {
      isExporting = false;
    }
  }
</script>

<div class="pane-workbench pane-workbench--advanced">
  <section class="pane-column">
    <section class="pane-section">
      <div class="pane-heading">
        <h3 class="pane-title">Filtering</h3>
        <p class="pane-description">
          Select media, status, and date boundaries for the export query.
        </p>
      </div>

      <fieldset class="fieldset">
        <legend class="fieldset__title">Media types</legend>
        <div class="options-grid">
          {#each mediaTypes as type (type)}
            <label class="option-check">
              <input
                checked={request.types.includes(type)}
                type="checkbox"
                onchange={() => toggleMediaType(type)}
              />
              <span>{readableMedia(type)}</span>
            </label>
          {/each}
        </div>
        <small>Clear every option to include all media types.</small>
      </fieldset>

      <fieldset class="fieldset">
        <legend class="fieldset__title">Statuses</legend>
        <div class="options-grid">
          {#each statuses as status (status)}
            <label class="option-check">
              <input
                checked={request.statuses.includes(status)}
                type="checkbox"
                onchange={() => toggleStatus(status)}
              />
              <span>{readableStatus(status)}</span>
            </label>
          {/each}
        </div>
      </fieldset>

      <label class="field">
        <span>Date from</span>
        <input
          bind:value={request.dateFrom}
          placeholder="YYYY-MM-DDTHH:MM:SSZ"
        />
      </label>
    </section>
  </section>

  <section class="pane-column">
    <section class="pane-section">
      <div class="pane-heading">
        <h3 class="pane-title">Output configuration</h3>
        <p class="pane-description">
          Set file shape, format, grouping, and field density.
        </p>
      </div>

      <div class="pane-form-grid pane-form-grid--two">
        <label class="field">
          <span>Output format</span>
          <select bind:value={request.outputFormat}>
            {#each outputFormats as value (value)}
              <option value={value}>{value}</option>
            {/each}
          </select>
        </label>

        <label class="field">
          <span>Grouping</span>
          <select bind:value={request.grouping}>
            {#each groupings as value (value)}
              <option value={value}>{value}</option>
            {/each}
          </select>
        </label>

        <div class="field">
          <div class="field__meta">
            <label for="advanced-extended-mode">Extended mode</label>
            <span class="field-help">
              <button
                aria-label="Explain extended mode options"
                class="field-help__button"
                type="button"
              >
                i
              </button>
              <span class="field-help__popover" role="tooltip">
                <strong>full</strong> exports the standard full dataset.
                <br />
                <strong>full_anime_seasons</strong> expands anime output with
                season-level detail.
                <br />
                <strong>simkl_ids_only</strong> keeps only Simkl identifiers.
                <br />
                <strong>ids_only</strong> keeps the narrowest identifier-only
                payload.
              </span>
            </span>
          </div>
          <select id="advanced-extended-mode" bind:value={request.extended}>
            {#each extendedModes as value (value)}
              <option value={value}>{value}</option>
            {/each}
          </select>
        </div>

        <div class="field">
          <div class="field__meta">
            <label for="advanced-field-mode">Field mode</label>
            <span class="field-help">
              <button
                aria-label="Explain field mode options"
                class="field-help__button"
                type="button"
              >
                i
              </button>
              <span class="field-help__popover" role="tooltip">
                <strong>compact</strong> exports the core columns only.
                <br />
                <strong>all</strong> includes every available field and
                metadata column.
              </span>
            </span>
          </div>
          <select id="advanced-field-mode" bind:value={request.fieldMode}>
            {#each fieldModes as value (value)}
              <option value={value}>{value}</option>
            {/each}
          </select>
        </div>
      </div>
    </section>

    <section class="pane-section">
      <div class="pane-heading">
        <h3 class="pane-title">Destination and options</h3>
        <p class="pane-description">
          Choose the output path, filename prefix, and optional export payloads.
        </p>
      </div>

      <label class="field">
        <span>{usesRemoteStorage ? "Backup destination" : "Export directory"}</span>
        {#if !usesRemoteStorage}
          <div class="input-with-action">
            <input
              placeholder="Choose a folder to store the export"
              value={state.exportDirectory}
              oninput={(event) =>
                appStore.patchState({
                  exportDirectory: (event.currentTarget as HTMLInputElement).value,
                })}
            />
            <button class="button" type="button" onclick={handleBrowseDirectory}>
              Browse
            </button>
          </div>
        {:else}
          <div class="value-box value-box--path">{state.backupDestinationLabel}</div>
        {/if}
        <small>
          {usesRemoteStorage
            ? "This run uploads generated files to the configured remote backup destination."
            : "This path is shared with the quick backup tab and persisted in the config file."}
        </small>
      </label>

      <label class="field">
        <span>Filename prefix</span>
        <input bind:value={request.filenamePrefix} placeholder="simkl-export" />
      </label>

      <fieldset class="fieldset">
        <div class="fieldset__meta">
          <span class="fieldset__title">Execution</span>
          <span class="field-help">
            <button
              aria-label="Explain execution options"
              class="field-help__button"
              type="button"
            >
              i
            </button>
            <span class="field-help__popover" role="tooltip">
              <strong>Include episode files</strong> writes separate
              episode-level export files when that payload is available.
              <br />
              <strong>Use activity check before export</strong> checks recent
              Simkl activity first so unchanged libraries can avoid unnecessary
              full export work.
            </span>
          </span>
        </div>
        <div class="options-grid">
          <label class="option-check">
            <input bind:checked={request.includeEpisodeFiles} type="checkbox" />
            <span>Include episode files</span>
          </label>
          <label class="option-check">
            <input bind:checked={request.useActivityCheck} type="checkbox" />
            <span>Use activity check before export</span>
          </label>
        </div>
      </fieldset>

      <fieldset class="fieldset">
        <legend class="fieldset__title">Optional data</legend>
        <div class="options-grid">
          <label class="option-check">
            <input bind:checked={request.episodeWatchedAt} type="checkbox" />
            <span>Include episode watched timestamps</span>
          </label>
          <label class="option-check">
            <input bind:checked={request.includeMemos} type="checkbox" />
            <span>Include memos</span>
          </label>
          <label class="option-check">
            <input bind:checked={request.includeNextWatchInfo} type="checkbox" />
            <span>Include next-watch info</span>
          </label>
        </div>
      </fieldset>

      <div class="pane-toolbar">
        <button
          class="button button--primary"
          disabled={isExporting || !state.isAuthorized}
          type="button"
          onclick={handleAdvancedExport}
        >
          {isExporting ? "Exporting..." : "Run selective export"}
        </button>
      </div>
    </section>
  </section>

  <section class="pane-column">
    <ExportSummaryPanel
      {message}
      summary={summary}
      {warnings}
      title="Selected export shape"
      description="Review the request before or after you run the export."
    />
    <ExportResultsPanel
      {result}
      title="Result inspector"
      description="Generated files from the most recent run."
    />
  </section>
</div>
