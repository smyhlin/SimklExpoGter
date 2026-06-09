<svelte:options runes={true} />

<script lang="ts">
  import type { PendingExportSummary } from "@/types/simkl";

  let {
    title = "Selected export shape",
    description = "Review the request before or after you run the export.",
    summary = [],
    warnings = [],
    message = "",
  }: {
    title?: string;
    description?: string;
    summary?: PendingExportSummary[];
    warnings?: string[];
    message?: string;
  } = $props();
</script>

<section class="pane-section">
  <div class="pane-heading">
    <h3 class="pane-title">{title}</h3>
    <p class="pane-description">{description}</p>
  </div>

  <dl class="detail-list">
    {#each summary as entry (entry.label)}
      <div class="detail-list__row">
        <dt>{entry.label}</dt>
        <dd>{entry.value}</dd>
      </div>
    {/each}
  </dl>

  <div class="notice">
    {message || "No export has run yet."}
  </div>

  {#if warnings.length}
    <div class="pane-flow">
      {#each warnings as warning (warning)}
        <div class="notice notice--warning">{warning}</div>
      {/each}
    </div>
  {/if}
</section>
