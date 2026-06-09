<script lang="ts">
  import { appStore } from "@/stores/app";
  import { runExport } from "@/lib/wails";
  import type { ExportRequest, Grouping, OutputFormat } from "@/types/simkl";

  let selectedFormat: OutputFormat = "csv";
  let selectedGrouping: Grouping = "separate-files";
  let isExporting = false;
  let statusMessage = "No backup has been started yet.";

  $: state = $appStore;
  $: formatLabel = selectedFormat === "csv" ? "CSV" : "JSON";
  $: groupingLabel =
    selectedGrouping === "single-file"
      ? "Combine into single file"
      : "Separate files";
  $: canExport = Boolean(
    state.isAuthorized &&
      !isExporting &&
      (state.backupStorage === "gdrive"
        ? state.hasGoogleDriveToken
        : state.exportDirectory.trim()),
  );

  function buildRequest(): ExportRequest {
    return {
      types: ["shows", "movies", "anime"],
      statuses: ["watching", "plantowatch", "hold", "completed", "dropped"],
      dateFrom: "",
      extended: "full",
      episodeWatchedAt: true,
      includeMemos: true,
      includeNextWatchInfo: true,
      outputFormat: selectedFormat,
      fieldMode: "all",
      grouping: selectedGrouping,
      includeEpisodeFiles: true,
      useActivityCheck: false,
      exportDirectory:
        state.backupStorage === "gdrive" ? "" : state.exportDirectory.trim(),
      filenamePrefix: "simkl-full-backup",
    };
  }

  async function handleBrowseDirectory() {
    await appStore.pickExportDirectory();
  }

  async function handleExportAll() {
    if (!state.isAuthorized) {
      statusMessage = "Authorize in Settings before running the backup.";
      return;
    }

    if (state.backupStorage === "gdrive" && !state.hasGoogleDriveToken) {
      statusMessage = "Connect Google Drive in Settings before running the backup.";
      return;
    }

    if (state.backupStorage !== "gdrive" && !state.exportDirectory.trim()) {
      statusMessage = "Choose an export directory first.";
      return;
    }

    isExporting = true;
    appStore.patchState({ exportProgress: "Quick full backup in progress" });
    statusMessage = "Preparing the full backup...";

    try {
      await appStore.saveSettings();
      const result = await runExport(buildRequest());
      statusMessage = `Exported ${result.itemCounts.all ?? 0} items to ${result.destinationLabel || result.outputDirectory}.`;
      appStore.patchState({
        exportProgress: `Exported ${result.itemCounts.all ?? 0} items`,
      });
    } catch (error) {
      statusMessage =
        error instanceof Error && error.message.trim()
          ? error.message.trim()
          : "The export failed.";
      appStore.patchState({ exportProgress: "Quick export failed" });
    } finally {
      isExporting = false;
    }
  }
</script>

<div class="pane-workbench pane-workbench--easy">
  <section class="pane-column">
    <section class="pane-section">
      <div class="pane-heading">
        <h3 class="pane-title">Quick full backup</h3>
        <p class="pane-description">
          Run the default export preset across all media types with one action.
        </p>
      </div>

      <label class="field">
        <span>{state.backupStorage === "gdrive" ? "Backup destination" : "Export directory"}</span>
        {#if state.backupStorage !== "gdrive"}
          <div class="input-with-action">
            <input
              placeholder="Choose a folder to store the backup"
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
          {state.backupStorage === "gdrive"
            ? "The export is staged locally, then uploaded into a dated subfolder inside the saved Google Drive folder."
            : "This directory is shared with the Settings and Advanced export tabs."}
        </small>
      </label>
    </section>

    <section class="pane-section">
      <div class="pane-heading">
        <h3 class="pane-title">Output defaults</h3>
        <p class="pane-description">
          Set the file format and grouping for the full snapshot export.
        </p>
      </div>

      <div class="pane-form-grid pane-form-grid--two">
        <label class="field">
          <span>File format</span>
          <select bind:value={selectedFormat}>
            <option value="csv">csv</option>
            <option value="json">json</option>
          </select>
        </label>

        <label class="field">
          <span>Grouping</span>
          <select bind:value={selectedGrouping}>
            <option value="separate-files">separate-files</option>
            <option value="single-file">single-file</option>
          </select>
        </label>
      </div>
    </section>

    <section class="pane-section">
      <div class="pane-toolbar">
        <button
          class="button button--primary"
          disabled={!canExport}
          type="button"
          onclick={handleExportAll}
        >
          {isExporting ? "Exporting..." : "Export all data"}
        </button>
      </div>

      {#if state.backupStorage !== "gdrive" && !state.exportDirectory.trim()}
        <div class="notice">Choose an export directory before starting the backup.</div>
      {/if}

      <div class="notice">{statusMessage}</div>
    </section>
  </section>

  <section class="pane-column">
    <section class="pane-section">
      <div class="pane-heading">
        <h3 class="pane-title">Preset summary</h3>
        <p class="pane-description">
          The quick export always uses the full library snapshot.
        </p>
      </div>

      <dl class="detail-list">
        <div class="detail-list__row">
          <dt>Destination</dt>
          <dd>{state.backupDestinationLabel}</dd>
        </div>
        <div class="detail-list__row">
          <dt>Format</dt>
          <dd>{formatLabel}</dd>
        </div>
        <div class="detail-list__row">
          <dt>Grouping</dt>
          <dd>{groupingLabel}</dd>
        </div>
        <div class="detail-list__row">
          <dt>Media</dt>
          <dd>Shows, Movies, Anime</dd>
        </div>
        <div class="detail-list__row">
          <dt>Fields</dt>
          <dd>
            All fields, memos, next watch info, watched-at timestamps, episode
            files
          </dd>
        </div>
      </dl>
    </section>
  </section>
</div>
