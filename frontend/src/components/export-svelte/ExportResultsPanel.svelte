<svelte:options runes={true} />

<script lang="ts">
  import type { ExportResult } from "@/types/simkl";

  let {
    title = "Result inspector",
    description = "Generated files from the most recent run.",
    result = null,
  }: {
    title?: string;
    description?: string;
    result?: ExportResult | null;
  } = $props();

  function fileLabel(path: string, storageKind?: string) {
    const segments = path.split(/[/\\]/);
    return segments[segments.length - 1] || storageKind || path;
  }
</script>

<section class="pane-section export-results">
  <div class="pane-heading">
    <h3 class="pane-title">{title}</h3>
    <p class="pane-description">{description}</p>
  </div>

  {#if result}
    <div class="pane-flow">
      <div class="results-metrics">
        <div class="results-metric">
          <span>Total items</span>
          <strong>{result.itemCounts.all ?? 0}</strong>
        </div>
        <div class="results-metric">
          <span>Exported at</span>
          <strong>{result.exportedAt}</strong>
        </div>
        <div class="results-metric">
          <span>Activities checked</span>
          <strong>{result.activitiesChecked ? "Yes" : "No"}</strong>
        </div>
        <div class="results-metric">
          <span>Destination</span>
          <strong>{result.destinationLabel || result.outputDirectory}</strong>
        </div>
      </div>

      <div class="table-wrap">
        <table class="app-table">
          <thead>
            <tr>
              <th scope="col">File</th>
              <th scope="col">Type</th>
              <th scope="col">Rows</th>
              <th scope="col">Path</th>
            </tr>
          </thead>
          <tbody>
            {#each result.files as file, index (`${file.path}-${index}`)}
              <tr>
                <td>{fileLabel(file.path, file.storageKind)}</td>
                <td>{file.format} / {file.mediaType} / {file.kind}</td>
                <td>{file.rows}</td>
                <td class="table-path">{file.path}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  {:else}
    <div class="empty-panel">Run an export to populate the file list here.</div>
  {/if}
</section>

<style>
  .export-results {
    min-height: 0;
  }

  .results-metrics {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0;
    border-top: 1px solid var(--border);
  }

  .results-metric {
    min-width: 0;
    display: grid;
    gap: 4px;
    padding: 10px 12px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  }

  .results-metric:nth-child(odd) {
    padding-left: 0;
  }

  .results-metric:nth-child(even) {
    border-left: 1px solid var(--border);
  }

  .results-metric span {
    font-size: 10px;
    font-weight: 600;
    line-height: 1.2;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--text-subtle);
  }

  .results-metric strong {
    min-width: 0;
    font-size: 12px;
    line-height: 1.4;
    color: var(--text);
    font-family: "IBM Plex Mono", "Cascadia Code", Consolas, monospace;
    overflow-wrap: anywhere;
  }

  @media (max-width: 900px) {
    .results-metrics {
      grid-template-columns: minmax(0, 1fr);
    }

    .results-metric {
      padding-left: 0;
      padding-right: 0;
    }

    .results-metric:nth-child(even) {
      border-left: 0;
    }
  }
</style>
