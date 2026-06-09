package appsvc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"SimklExpoGter/internal/config"
	"SimklExpoGter/internal/exporter"
	"SimklExpoGter/internal/gdrive"
	"SimklExpoGter/internal/scheduler"
	"SimklExpoGter/internal/simkl"
)

func TestBuildFetchPlanReturnsSingleRequestForFullSnapshot(t *testing.T) {
	plan := buildFetchPlan(exporter.Request{
		Extended: "full",
	})

	if len(plan) != 1 {
		t.Fatalf("expected a single full-snapshot request, got %d", len(plan))
	}
	if plan[0].Type != "" || plan[0].Status != "" {
		t.Fatalf("expected the full snapshot request to omit type and status, got %+v", plan[0])
	}
}

func TestBuildFetchPlanExpandsStatusesAcrossTypes(t *testing.T) {
	plan := buildFetchPlan(exporter.Request{
		Types:    []string{"movies", "anime"},
		Statuses: []string{"completed", "dropped"},
	})

	if len(plan) != 4 {
		t.Fatalf("expected 4 requests, got %d", len(plan))
	}

	expected := map[string]bool{
		"movies:completed": true,
		"movies:dropped":   true,
		"anime:completed":  true,
		"anime:dropped":    true,
	}

	for _, item := range plan {
		key := item.Type + ":" + item.Status
		if !expected[key] {
			t.Fatalf("unexpected request generated: %+v", item)
		}
		delete(expected, key)
	}

	if len(expected) != 0 {
		t.Fatalf("missing requests: %+v", expected)
	}
}

func TestNormalizeRequestUsesDefaultOutputDirectory(t *testing.T) {
	request, err := normalizeRequest(exporter.Request{}, config.Settings{}, true)
	if err != nil {
		t.Fatalf("normalizeRequest returned error: %v", err)
	}

	if request.ExportDirectory != DefaultExportDirectory() {
		t.Fatalf("expected default export directory %q, got %q", DefaultExportDirectory(), request.ExportDirectory)
	}
}

func TestNormalizeRequestSkipsDefaultLocalDirectoryForGoogleDrive(t *testing.T) {
	request, err := normalizeRequest(exporter.Request{}, config.Settings{}, false)
	if err != nil {
		t.Fatalf("normalizeRequest returned error: %v", err)
	}

	if request.ExportDirectory != "" {
		t.Fatalf("expected Google Drive export to keep the staging directory empty before temp setup, got %q", request.ExportDirectory)
	}
}

func TestRunExportRequiresClientID(t *testing.T) {
	service := newTestService(t, config.Settings{})

	_, err := service.RunExport(exporter.Request{})
	if !IsPrerequisiteError(err) {
		t.Fatalf("expected prerequisite error, got %v", err)
	}
}

func TestRunExportRequiresAccessToken(t *testing.T) {
	service := newTestService(t, config.Settings{ClientID: "client-1"})

	_, err := service.RunExport(exporter.Request{})
	if !IsPrerequisiteError(err) {
		t.Fatalf("expected prerequisite error, got %v", err)
	}
}

func TestStandardAuthURLRequiresClientID(t *testing.T) {
	service := newTestService(t, config.Settings{})

	_, err := service.StandardAuthURL()
	if !IsPrerequisiteError(err) {
		t.Fatalf("expected prerequisite error, got %v", err)
	}
}

func TestStandardAuthURLIncludesClientID(t *testing.T) {
	service := newTestService(t, config.Settings{ClientID: "client-1"})

	authURL, err := service.StandardAuthURL()
	if err != nil {
		t.Fatalf("StandardAuthURL returned error: %v", err)
	}
	if want := "client_id=client-1"; !contains(authURL, want) {
		t.Fatalf("expected auth URL to contain %q, got %q", want, authURL)
	}
}

func TestExchangeOAuthCodeRequiresSecret(t *testing.T) {
	service := newTestService(t, config.Settings{ClientID: "client-1"})

	_, err := service.ExchangeOAuthCode("ABCD-1234")
	if !IsPrerequisiteError(err) {
		t.Fatalf("expected prerequisite error, got %v", err)
	}
}

func TestAuthSummaryAndClearAccessToken(t *testing.T) {
	service := newTestService(t, config.Settings{
		ClientID:     "client-1",
		ClientSecret: "secret-1",
		AccessToken:  "token-1",
	})

	summary, err := service.AuthSummary()
	if err != nil {
		t.Fatalf("AuthSummary returned error: %v", err)
	}
	if !summary.ReadyForHeadlessRun {
		t.Fatal("expected headless run to be ready")
	}

	if _, err := service.ClearAccessToken(); err != nil {
		t.Fatalf("ClearAccessToken returned error: %v", err)
	}

	summary, err = service.AuthSummary()
	if err != nil {
		t.Fatalf("AuthSummary returned error after clear: %v", err)
	}
	if summary.AccessTokenConfigured {
		t.Fatal("expected access token to be cleared")
	}
}

func TestExchangeGoogleDriveCodeProvisionsFolder(t *testing.T) {
	store := config.NewStoreAtPath(filepath.Join(t.TempDir(), "settings.json"))
	if err := store.Save(config.Settings{
		Backup: config.BackupSettings{
			StorageKind: config.BackupStorageGDrive,
			GoogleDrive: config.GoogleDriveSettings{
				ClientID:     "google-client",
				ClientSecret: "google-secret",
				FolderName:   "Simkl Backups",
			},
		},
	}); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	drive := &fakeDrive{
		exchangedToken: config.OAuthToken{
			AccessToken:  "fresh-access",
			RefreshToken: "fresh-refresh",
		},
		uploadResult: gdrive.UploadResult{
			Token: config.OAuthToken{
				AccessToken:  "refreshed-access",
				RefreshToken: "fresh-refresh",
			},
			FolderID:   "folder-123",
			FolderName: "Simkl Backups",
			FolderURL:  "https://drive.google.com/drive/folders/folder-123",
		},
	}
	service := NewWithDepsAndSchedulerAndDrive(
		store,
		simkl.NewClient(),
		exporter.NewService(),
		drive,
		nil,
		"",
	)

	settings, err := service.ExchangeGoogleDriveCode("auth-code", "http://127.0.0.1/callback", "verifier")
	if err != nil {
		t.Fatalf("ExchangeGoogleDriveCode returned error: %v", err)
	}

	if !drive.exchangeCalled {
		t.Fatal("expected Google Drive code exchange to be called")
	}
	if !drive.uploadCalled {
		t.Fatal("expected Google Drive folder provisioning to be called")
	}
	if len(drive.uploadPaths) != 0 {
		t.Fatalf("expected folder provisioning without file uploads, got paths %+v", drive.uploadPaths)
	}
	if got, want := settings.Backup.GoogleDrive.FolderID, "folder-123"; got != want {
		t.Fatalf("expected folder ID %q, got %q", want, got)
	}
	if got, want := settings.Backup.GoogleDrive.FolderURL, "https://drive.google.com/drive/folders/folder-123"; got != want {
		t.Fatalf("expected folder URL %q, got %q", want, got)
	}
	if got, want := settings.Backup.GoogleDrive.Token.AccessToken, "refreshed-access"; got != want {
		t.Fatalf("expected persisted access token %q, got %q", want, got)
	}
	if got, want := settings.Backup.GoogleDrive.Token.RefreshToken, "fresh-refresh"; got != want {
		t.Fatalf("expected persisted refresh token %q, got %q", want, got)
	}
}

func TestExportAndImportEncryptedSettingsRoundTrip(t *testing.T) {
	backupPath := filepath.Join(t.TempDir(), "settings-backup")
	service := newTestService(t, config.Settings{
		ClientID:        "simkl-client",
		ClientSecret:    "simkl-secret",
		AccessToken:     "simkl-token",
		ExportDirectory: filepath.Join(t.TempDir(), "exports"),
		LastActivities: map[string]any{
			"all": "2026-04-09T08:00:00Z",
		},
		Backup: config.BackupSettings{
			StorageKind: config.BackupStorageGDrive,
			GoogleDrive: config.GoogleDriveSettings{
				ClientID:     "drive-client",
				ClientSecret: "drive-secret",
				FolderID:     "folder-123",
				FolderName:   "Simkl Backups",
				FolderURL:    "https://drive.google.com/drive/folders/folder-123",
				Token: config.OAuthToken{
					AccessToken:  "drive-access",
					RefreshToken: "drive-refresh",
				},
			},
		},
		Schedule: config.ScheduleSettings{
			Enabled:          true,
			Frequency:        "weekly",
			Time:             "21:15",
			Days:             []string{"mon", "wed"},
			OutputFormat:     "both",
			FieldMode:        "all",
			Content:          []string{"shows", "movies"},
			UseActivityCheck: true,
		},
	})

	exportedPath, err := service.ExportEncryptedSettings(backupPath, "correct horse battery staple")
	if err != nil {
		t.Fatalf("ExportEncryptedSettings returned error: %v", err)
	}
	if want := backupPath + SettingsBackupExtension; exportedPath != want {
		t.Fatalf("expected exported path %q, got %q", want, exportedPath)
	}

	payload, err := os.ReadFile(exportedPath)
	if err != nil {
		t.Fatalf("failed to read exported backup: %v", err)
	}
	if strings.Contains(string(payload), "simkl-secret") || strings.Contains(string(payload), "drive-refresh") {
		t.Fatal("expected encrypted backup payload to avoid storing secrets in plaintext")
	}

	if _, err := service.SaveSettings(SaveSettingsInput{
		ClientID:           "overwritten-client",
		SetClientID:        true,
		ExportDirectory:    filepath.Join(t.TempDir(), "other"),
		SetExportDirectory: true,
	}); err != nil {
		t.Fatalf("SaveSettings returned error: %v", err)
	}

	if err := service.ImportEncryptedSettings(exportedPath, "correct horse battery staple"); err != nil {
		t.Fatalf("ImportEncryptedSettings returned error: %v", err)
	}

	settings, err := service.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings returned error: %v", err)
	}

	if got, want := settings.ClientID, "simkl-client"; got != want {
		t.Fatalf("expected client ID %q, got %q", want, got)
	}
	if got, want := settings.ClientSecret, "simkl-secret"; got != want {
		t.Fatalf("expected client secret %q, got %q", want, got)
	}
	if got, want := settings.AccessToken, "simkl-token"; got != want {
		t.Fatalf("expected access token %q, got %q", want, got)
	}
	if got, want := settings.Backup.GoogleDrive.ClientSecret, "drive-secret"; got != want {
		t.Fatalf("expected Google Drive client secret %q, got %q", want, got)
	}
	if got, want := settings.Backup.GoogleDrive.Token.RefreshToken, "drive-refresh"; got != want {
		t.Fatalf("expected Google Drive refresh token %q, got %q", want, got)
	}
	if got, want := settings.Backup.GoogleDrive.FolderID, "folder-123"; got != want {
		t.Fatalf("expected Google Drive folder ID %q, got %q", want, got)
	}
	if got, want := settings.Schedule.Time, "21:15"; got != want {
		t.Fatalf("expected schedule time %q, got %q", want, got)
	}
	if got, want := settings.LastActivities["all"], "2026-04-09T08:00:00Z"; got != want {
		t.Fatalf("expected activity snapshot %q, got %#v", want, got)
	}
}

func TestImportEncryptedSettingsRejectsWrongPassword(t *testing.T) {
	service := newTestService(t, config.Settings{
		ClientID:     "simkl-client",
		ClientSecret: "simkl-secret",
	})

	exportedPath, err := service.ExportEncryptedSettings(
		filepath.Join(t.TempDir(), "settings-backup"),
		"top secret",
	)
	if err != nil {
		t.Fatalf("ExportEncryptedSettings returned error: %v", err)
	}

	err = service.ImportEncryptedSettings(exportedPath, "wrong password")
	if err == nil {
		t.Fatal("expected ImportEncryptedSettings to fail with the wrong password")
	}
	if !strings.Contains(err.Error(), "failed to decrypt") {
		t.Fatalf("expected wrong-password error, got %v", err)
	}
}

func TestRunExportDoesNotPersistOutputOverrideAndUpdatesActivities(t *testing.T) {
	persistedDir := t.TempDir()
	overrideDir := t.TempDir()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/sync/activities":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"all":"2024-02-02T00:00:00Z"}`))
		case r.URL.Path == "/sync/all-items/":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"movies":[],"shows":[],"anime":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	store := config.NewStoreAtPath(filepath.Join(t.TempDir(), "settings.json"))
	settings := config.Settings{
		ClientID:        "client-1",
		ClientSecret:    "secret-1",
		AccessToken:     "token-1",
		ExportDirectory: persistedDir,
	}
	if err := store.Save(settings); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	service := NewWithDeps(store, simkl.NewClientWithBaseURL(server.URL, server.Client()), exporter.NewService())

	result, err := service.RunExport(exporter.Request{
		UseActivityCheck:    true,
		ExportDirectory:     overrideDir,
		IncludeEpisodeFiles: false,
		OutputFormat:        exporter.FormatCSV,
	})
	if err != nil {
		t.Fatalf("RunExport returned error: %v", err)
	}
	if result.OutputDirectory != overrideDir {
		t.Fatalf("expected output directory %q, got %q", overrideDir, result.OutputDirectory)
	}

	savedSettings, err := service.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings returned error: %v", err)
	}
	if savedSettings.ExportDirectory != persistedDir {
		t.Fatalf("expected persisted export directory %q, got %q", persistedDir, savedSettings.ExportDirectory)
	}
	if savedSettings.LastActivities["all"] != "2024-02-02T00:00:00Z" {
		t.Fatalf("expected last activities to be updated, got %+v", savedSettings.LastActivities)
	}
}

func TestRunExportUsesGoogleDriveRunSubfolder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sync/all-items/":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"movies":[],"shows":[],"anime":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	store := config.NewStoreAtPath(filepath.Join(t.TempDir(), "settings.json"))
	if err := store.Save(config.Settings{
		ClientID:    "client-1",
		AccessToken: "token-1",
		Backup: config.BackupSettings{
			StorageKind: config.BackupStorageGDrive,
			GoogleDrive: config.GoogleDriveSettings{
				ClientID:     "google-client",
				ClientSecret: "google-secret",
				Token: config.OAuthToken{
					RefreshToken: "refresh-token",
				},
				FolderID:   "root-folder",
				FolderName: "Simkl Backups",
				FolderURL:  "https://drive.google.com/drive/folders/root-folder",
			},
		},
	}); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	drive := &fakeDrive{
		uploadResult: gdrive.UploadResult{
			Token: config.OAuthToken{
				AccessToken:  "refreshed-access",
				RefreshToken: "refresh-token",
			},
			FolderID:         "root-folder",
			FolderName:       "Simkl Backups",
			FolderURL:        "https://drive.google.com/drive/folders/root-folder",
			UploadFolderID:   "run-folder",
			UploadFolderName: "backup-20260409-101112",
			UploadFolderURL:  "https://drive.google.com/drive/folders/run-folder",
			Files: []gdrive.UploadedFile{
				{Name: "simkl-full-backup-movies-items-20260409-101112.csv"},
				{Name: "simkl-full-backup-shows-items-20260409-101112.csv"},
				{Name: "simkl-full-backup-anime-items-20260409-101112.csv"},
			},
		},
	}

	service := NewWithDepsAndSchedulerAndDrive(
		store,
		simkl.NewClientWithBaseURL(server.URL, server.Client()),
		exporter.NewService(),
		drive,
		nil,
		"",
	)

	result, err := service.RunExport(exporter.Request{
		OutputFormat:   exporter.FormatCSV,
		FilenamePrefix: "simkl-full-backup",
	})
	if err != nil {
		t.Fatalf("RunExport returned error: %v", err)
	}

	if !drive.uploadCalled {
		t.Fatal("expected Google Drive upload to run")
	}
	if len(drive.uploadPaths) != 3 {
		t.Fatalf("expected 3 staged files to upload, got %d", len(drive.uploadPaths))
	}

	wantLabel := "Google Drive / Simkl Backups / backup-20260409-101112"
	if got := result.DestinationLabel; got != wantLabel {
		t.Fatalf("expected destination label %q, got %q", wantLabel, got)
	}
	if got, want := result.DestinationURL, "https://drive.google.com/drive/folders/run-folder"; got != want {
		t.Fatalf("expected destination URL %q, got %q", want, got)
	}

	for _, file := range result.Files {
		if !strings.HasPrefix(file.Path, wantLabel+" / ") {
			t.Fatalf("expected uploaded file path to use run subfolder, got %q", file.Path)
		}
	}

	settings, err := service.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings returned error: %v", err)
	}
	if got, want := settings.Backup.GoogleDrive.FolderURL, "https://drive.google.com/drive/folders/root-folder"; got != want {
		t.Fatalf("expected saved root folder URL %q, got %q", want, got)
	}
	if got, want := settings.Backup.GoogleDrive.FolderID, "root-folder"; got != want {
		t.Fatalf("expected saved root folder ID %q, got %q", want, got)
	}
}

func TestSaveScheduleRequiresAuthorization(t *testing.T) {
	service := newTestService(t, config.Settings{ClientID: "client-1"})

	_, _, err := service.SaveSchedule(ScheduleSettingsInput{
		Enabled:   true,
		Frequency: "daily",
		Time:      "03:15",
	})
	if !IsPrerequisiteError(err) {
		t.Fatalf("expected prerequisite error, got %v", err)
	}
}

func TestSaveSchedulePersistsSettingsAndSyncsScheduler(t *testing.T) {
	store := config.NewStoreAtPath(filepath.Join(t.TempDir(), "settings.json"))
	if err := store.Save(config.Settings{
		ClientID:    "client-1",
		AccessToken: "token-1",
	}); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	manager := &fakeScheduler{
		state: scheduler.State{
			Supported:  true,
			Installed:  true,
			TaskName:   "SimklExpoGterRecurringBackup",
			Status:     "Ready",
			NextRunAt:  "2026-04-08T02:30:00+03:00",
			LastResult: "0",
		},
	}
	service := NewWithDepsAndScheduler(
		store,
		simkl.NewClient(),
		exporter.NewService(),
		manager,
		`C:\Apps\SimklExpoGter.exe`,
	)

	settings, state, err := service.SaveSchedule(ScheduleSettingsInput{
		Enabled:          true,
		Frequency:        "weekly",
		Time:             "02:30",
		Days:             []string{"wed", "mon"},
		OutputFormat:     "json",
		FieldMode:        "compact",
		Content:          []string{"series", "anime"},
		UseActivityCheck: true,
	})
	if err != nil {
		t.Fatalf("SaveSchedule returned error: %v", err)
	}

	if !manager.syncCalled {
		t.Fatal("expected scheduler sync to be called")
	}
	if manager.lastConfig.ExecutablePath != `C:\Apps\SimklExpoGter.exe` {
		t.Fatalf("expected executable path to be preserved, got %q", manager.lastConfig.ExecutablePath)
	}
	if got, want := strings.Join(manager.lastConfig.Arguments, " "), "run --scheduled --format json --field-mode compact --content shows,anime --activity-check"; got != want {
		t.Fatalf("expected scheduler arguments %q, got %q", want, got)
	}
	if got, want := strings.Join(manager.lastConfig.Days, ","), "mon,wed"; got != want {
		t.Fatalf("expected normalized days %q, got %q", want, got)
	}
	if !settings.Schedule.Enabled {
		t.Fatal("expected schedule to be persisted as enabled")
	}
	if got, want := strings.Join(settings.Schedule.Content, ","), "shows,anime"; got != want {
		t.Fatalf("expected persisted content %q, got %q", want, got)
	}
	if got, want := settings.Schedule.OutputFormat, "json"; got != want {
		t.Fatalf("expected persisted output format %q, got %q", want, got)
	}
	if got, want := state.OutputFormat, "json"; got != want {
		t.Fatalf("expected schedule state output format %q, got %q", want, got)
	}
	if !state.Installed || state.Status != "Ready" {
		t.Fatalf("expected installed scheduler state, got %+v", state)
	}
}

func TestSaveScheduleDisableRemovesScheduledTask(t *testing.T) {
	store := config.NewStoreAtPath(filepath.Join(t.TempDir(), "settings.json"))
	if err := store.Save(config.Settings{
		ClientID:    "client-1",
		AccessToken: "token-1",
		Schedule: config.ScheduleSettings{
			Enabled:   true,
			Frequency: "daily",
			Time:      "02:00",
		},
	}); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	manager := &fakeScheduler{}
	service := NewWithDepsAndScheduler(
		store,
		simkl.NewClient(),
		exporter.NewService(),
		manager,
		`C:\Apps\SimklExpoGter.exe`,
	)

	settings, _, err := service.SaveSchedule(ScheduleSettingsInput{Enabled: false})
	if err != nil {
		t.Fatalf("SaveSchedule returned error: %v", err)
	}

	if !manager.removeCalled {
		t.Fatal("expected scheduled task removal to be called")
	}
	if settings.Schedule.Enabled {
		t.Fatal("expected persisted schedule to be disabled")
	}
}

func newTestService(t *testing.T, settings config.Settings) *Service {
	t.Helper()

	store := config.NewStoreAtPath(filepath.Join(t.TempDir(), "settings.json"))
	if err := store.Save(settings); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	return NewWithDeps(store, simkl.NewClient(), exporter.NewService())
}

func contains(value, fragment string) bool {
	return strings.Contains(value, fragment)
}

type fakeScheduler struct {
	state        scheduler.State
	lastConfig   scheduler.Config
	syncCalled   bool
	removeCalled bool
}

type fakeDrive struct {
	exchangedToken config.OAuthToken
	uploadResult   gdrive.UploadResult
	exchangeCalled bool
	uploadCalled   bool
	uploadPaths    []string
}

func (f *fakeDrive) AuthURL(clientID, clientSecret, redirectURI, state, verifier string) (string, error) {
	return "https://example.com/auth", nil
}

func (f *fakeDrive) ExchangeCode(context.Context, string, string, string, string, string) (config.OAuthToken, error) {
	f.exchangeCalled = true
	return f.exchangedToken, nil
}

func (f *fakeDrive) UploadFiles(_ context.Context, _ config.GoogleDriveSettings, localPaths []string) (gdrive.UploadResult, error) {
	f.uploadCalled = true
	f.uploadPaths = append([]string(nil), localPaths...)
	return f.uploadResult, nil
}

func (f *fakeScheduler) Supported() bool {
	return true
}

func (f *fakeScheduler) Sync(config scheduler.Config) (scheduler.State, error) {
	f.syncCalled = true
	f.lastConfig = config
	state := f.state
	state.Supported = true
	state.TaskName = config.TaskName
	if state.TaskName == "" {
		state.TaskName = config.TaskName
	}
	return state, nil
}

func (f *fakeScheduler) Remove(string) error {
	f.removeCalled = true
	return nil
}

func (f *fakeScheduler) Query(taskName string) (scheduler.State, error) {
	state := f.state
	state.Supported = true
	if state.TaskName == "" {
		state.TaskName = taskName
	}
	return state, nil
}
