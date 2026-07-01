<script lang="ts">
  import { onMount } from "svelte";
  import { appStore } from "@/stores/app";
  import EasyExportView from "@/views/EasyExportView.svelte";
  import AdvancedExportView from "@/views/AdvancedExportView.svelte";
  import SettingsView from "@/views/SettingsView.svelte";

  type TabId = "settings" | "easy" | "advanced";

  type TabItem = {
    id: TabId;
    label: string;
    description: string;
    status: string;
    component: any;
  };

  let activeTab: TabId = "settings";
  let ActiveComponent: any = SettingsView;

  const tabDefinitions: Array<
    Omit<TabItem, "status" | "component"> & { component: any }
  > = [
    {
      id: "settings",
      label: "Settings",
      description: "Storage, auth, and automation",
      component: SettingsView,
    },
    {
      id: "easy",
      label: "Easy Export",
      description: "One-click full backup",
      component: EasyExportView,
    },
    {
      id: "advanced",
      label: "Advanced Export",
      description: "Selective export and logs",
      component: AdvancedExportView,
    },
  ];

  $: state = $appStore;

  function buildTabStatus(tabId: TabId) {
    switch (tabId) {
      case "settings": {
        if (state.backupStorage === "gdrive") {
          if (state.pendingGoogleDriveAuth) return "Drive approval pending";
          return state.hasGoogleDriveToken
            ? "Google Drive ready"
            : "Drive setup required";
        }
        if (state.backupStorage === "telegram") {
          return state.hasTelegramBotToken && state.telegramChatId.trim()
            ? "Telegram ready"
            : "Telegram setup required";
        }

        return state.exportDirectory.trim() ? "Local folder ready" : "Folder required";
      }
      case "easy":
        return state.isAuthorized ? "Ready to export" : "Authorize first";
      case "advanced":
        return state.isAuthorized ? "Filtered export ready" : "Authorize first";
    }
  }

  $: tabs = tabDefinitions.map((tab) => ({
    ...tab,
    status: buildTabStatus(tab.id),
  })) as TabItem[];

  $: activeTabItem = tabs.find((tab) => tab.id === activeTab) ?? tabs[0];
  $: ActiveComponent = activeTabItem.component;
  $: workspaceSubtitle =
    activeTabItem?.description ??
    "Storage, auth, automation, and export operations";
  $: workspaceMessage =
    state.authMessage ||
    state.scheduleMessage ||
    "Shared settings here drive the GUI, CLI, and recurring backup flow.";

  $: connectionLabel = state.isAuthorized
    ? "Connected to Simkl"
    : "Not connected to Simkl";
  $: exportProgressLabel = state.exportProgress || "Idle";
  $: destinationLabel = state.backupDestinationLabel || "Choose a destination";

  onMount(() => {
    void appStore.loadAppState();
  });
</script>

<div class="desktop-shell">
  <aside class="desktop-sidebar">
    <div class="shell-brand">
      <div class="shell-brand__title">SimklExpoGter</div>
      <p class="shell-brand__copy">
        Storage, auth, automation, and export operations.
      </p>
    </div>

    <div class="shell-nav" aria-label="Workspace tabs" role="tablist">
      {#each tabs as tab (tab.id)}
        <button
          aria-selected={activeTab === tab.id}
          class:shell-nav__item--active={activeTab === tab.id}
          class="shell-nav__item"
          role="tab"
          type="button"
          onclick={() => {
            activeTab = tab.id;
          }}
        >
          <span class="shell-nav__title">{tab.label}</span>
          <p class="shell-nav__description">{tab.description}</p>
          <span class="shell-nav__status">{tab.status}</span>
        </button>
      {/each}
    </div>

    <div class="shell-sidebar__footer">
      <span class="shell-sidebar__footer-label">Latest message</span>
      <p class="shell-sidebar__footer-copy">{workspaceMessage}</p>
    </div>
  </aside>

  <main class="desktop-main">
    <header class="desktop-topbar">
      <h1 class="desktop-topbar__title">{activeTabItem.label}</h1>
      <p class="desktop-topbar__subtitle">{workspaceSubtitle}</p>
    </header>

    <section class="desktop-stage">
      <div class="desktop-stage__frame">
        {#key activeTab}
          <ActiveComponent />
        {/key}
      </div>
    </section>
  </main>

  <footer class="desktop-statusbar" aria-label="Application status">
    <div class="desktop-statusbar__item">
      <span class="desktop-statusbar__label">Connection:</span>
      <span
        class="desktop-statusbar__value"
        data-tone={state.isAuthorized ? "ready" : "muted"}
      >
        {connectionLabel}
      </span>
    </div>
    <div class="desktop-statusbar__item">
      <span class="desktop-statusbar__label">Export:</span>
      <span class="desktop-statusbar__value">{exportProgressLabel}</span>
    </div>
    <div class="desktop-statusbar__item">
      <span class="desktop-statusbar__label">Destination:</span>
      <span class="desktop-statusbar__value">{destinationLabel}</span>
    </div>
  </footer>
</div>
