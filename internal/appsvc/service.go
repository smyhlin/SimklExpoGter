package appsvc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"SimklExpoGter/internal/config"
	"SimklExpoGter/internal/exporter"
	"SimklExpoGter/internal/gdrive"
	"SimklExpoGter/internal/scheduler"
	"SimklExpoGter/internal/simkl"
)

const (
	DefaultAppName        = "SimklExpoGter"
	OAuthOOBRedirectURI   = "urn:ietf:wg:oauth:2.0:oob"
	defaultFilenamePrefix = "simkl-export"
	defaultTaskName       = "SimklExpoGterRecurringBackup"
)

var typeAliases = map[string]string{
	"anime":  "anime",
	"movie":  "movies",
	"movies": "movies",
	"series": "shows",
	"show":   "shows",
	"shows":  "shows",
}

type Service struct {
	store          *config.Store
	client         *simkl.Client
	exporter       *exporter.Service
	drive          backupUploader
	scheduler      scheduler.Manager
	executablePath string
}

type backupUploader interface {
	AuthURL(clientID, clientSecret, redirectURI, state, verifier string) (string, error)
	ExchangeCode(context.Context, string, string, string, string, string) (config.OAuthToken, error)
	UploadFiles(context.Context, config.GoogleDriveSettings, []string) (gdrive.UploadResult, error)
}

type SaveSettingsInput struct {
	ClientID                   string
	ClientSecret               string
	ExportDirectory            string
	BackupStorage              string
	GoogleDriveClientID        string
	GoogleDriveClientSecret    string
	GoogleDriveFolderName      string
	SetClientID                bool
	SetClientSecret            bool
	SetExportDirectory         bool
	SetBackupStorage           bool
	SetGoogleDriveClientID     bool
	SetGoogleDriveClientSecret bool
	SetGoogleDriveFolderName   bool
}

type SaveSettingsResult struct {
	Settings        config.Settings
	ClientIDChanged bool
}

type ConfigSummary struct {
	ConfigPath                 string `json:"configPath"`
	ClientID                   string `json:"clientId"`
	HasClientSecret            bool   `json:"hasClientSecret"`
	HasAccessToken             bool   `json:"hasAccessToken"`
	ExportDirectory            string `json:"exportDirectory"`
	SuggestedExportDirectory   string `json:"suggestedExportDirectory"`
	BackupStorage              string `json:"backupStorage"`
	GoogleDriveClientID        string `json:"googleDriveClientId"`
	HasGoogleDriveClientSecret bool   `json:"hasGoogleDriveClientSecret"`
	HasGoogleDriveToken        bool   `json:"hasGoogleDriveToken"`
	GoogleDriveFolderName      string `json:"googleDriveFolderName"`
	GoogleDriveFolderURL       string `json:"googleDriveFolderUrl"`
	UpdatedAt                  string `json:"updatedAt,omitempty"`
}

type AuthSummary struct {
	ClientIDConfigured     bool `json:"clientIdConfigured"`
	ClientSecretConfigured bool `json:"clientSecretConfigured"`
	AccessTokenConfigured  bool `json:"accessTokenConfigured"`
	ReadyForHeadlessRun    bool `json:"readyForHeadlessRun"`
}

type ScheduleSettingsInput struct {
	Enabled            bool
	Frequency          string
	Time               string
	Days               []string
	OutputFormat       string
	FieldMode          string
	Content            []string
	UseActivityCheck   bool
	MaxBackupAge       string
	RunIfBackupIsStale bool
}

type ScheduleState struct {
	Supported                bool     `json:"supported"`
	Enabled                  bool     `json:"enabled"`
	Installed                bool     `json:"installed"`
	Frequency                string   `json:"frequency"`
	Time                     string   `json:"time"`
	Days                     []string `json:"days,omitempty"`
	OutputFormat             string   `json:"outputFormat"`
	FieldMode                string   `json:"fieldMode"`
	Content                  []string `json:"content"`
	UseActivityCheck         bool     `json:"useActivityCheck"`
	MaxBackupAge             string   `json:"maxBackupAge"`
	RunIfBackupIsStale       bool     `json:"runIfBackupIsStale"`
	LastSuccessfulBackupAt   string   `json:"lastSuccessfulBackupAt,omitempty"`
	LastSuccessfulBackupKind string   `json:"lastSuccessfulBackupKind,omitempty"`
	BackupFresh              bool     `json:"backupFresh"`
	BackupAgeSeconds         int64    `json:"backupAgeSeconds,omitempty"`
	TaskName                 string   `json:"taskName"`
	Status                   string   `json:"status,omitempty"`
	NextRunAt                string   `json:"nextRunAt,omitempty"`
	LastRunAt                string   `json:"lastRunAt,omitempty"`
	LastResult               string   `json:"lastResult,omitempty"`
	Message                  string   `json:"message,omitempty"`
	UsesSavedOutput          bool     `json:"usesSavedOutput"`
	OutputDirectoryPreview   string   `json:"outputDirectoryPreview"`
}

type ScheduledExportResult struct {
	Ran              bool            `json:"ran"`
	Skipped          bool            `json:"skipped"`
	Reason           string          `json:"reason,omitempty"`
	LastBackupAt     string          `json:"lastBackupAt,omitempty"`
	MaxBackupAge     string          `json:"maxBackupAge,omitempty"`
	BackupAgeSeconds int64           `json:"backupAgeSeconds,omitempty"`
	Result           exporter.Result `json:"result,omitempty"`
}

type prerequisiteError struct {
	message string
}

func (e *prerequisiteError) Error() string {
	return e.message
}

type usageError struct {
	message string
}

func (e *usageError) Error() string {
	return e.message
}

func IsPrerequisiteError(err error) bool {
	var target *prerequisiteError
	return errors.As(err, &target)
}

func IsUsageError(err error) bool {
	var target *usageError
	return errors.As(err, &target)
}

func New(appName string) (*Service, error) {
	store, err := config.NewStore(appName)
	if err != nil {
		return nil, err
	}

	executablePath, err := os.Executable()
	if err != nil {
		return nil, err
	}

	return NewWithDepsAndSchedulerAndDrive(
		store,
		simkl.NewClient(),
		exporter.NewService(),
		gdrive.NewService(),
		scheduler.NewManager(),
		executablePath,
	), nil
}

func NewWithDeps(store *config.Store, client *simkl.Client, exporterService *exporter.Service) *Service {
	return NewWithDepsAndSchedulerAndDrive(store, client, exporterService, nil, nil, "")
}

func NewWithDepsAndScheduler(store *config.Store, client *simkl.Client, exporterService *exporter.Service, schedulerManager scheduler.Manager, executablePath string) *Service {
	return NewWithDepsAndSchedulerAndDrive(store, client, exporterService, nil, schedulerManager, executablePath)
}

func NewWithDepsAndSchedulerAndDrive(
	store *config.Store,
	client *simkl.Client,
	exporterService *exporter.Service,
	driveService backupUploader,
	schedulerManager scheduler.Manager,
	executablePath string,
) *Service {
	if store == nil {
		store = config.NewStoreAtPath(filepath.Join(".", "settings.json"))
	}
	if client == nil {
		client = simkl.NewClient()
	}
	if exporterService == nil {
		exporterService = exporter.NewService()
	}
	if driveService == nil {
		driveService = gdrive.NewService()
	}
	if schedulerManager == nil {
		schedulerManager = scheduler.NewManager()
	}

	return &Service{
		store:          store,
		client:         client,
		exporter:       exporterService,
		drive:          driveService,
		scheduler:      schedulerManager,
		executablePath: executablePath,
	}
}

func (s *Service) Path() string {
	return s.store.Path()
}

func (s *Service) LoadSettings() (config.Settings, error) {
	return s.store.Load()
}

func (s *Service) SaveSettings(input SaveSettingsInput) (SaveSettingsResult, error) {
	settings, err := s.store.Load()
	if err != nil {
		return SaveSettingsResult{}, err
	}

	clientIDChanged := false

	if input.SetClientID {
		clientID := strings.TrimSpace(input.ClientID)
		clientIDChanged = clientID != settings.ClientID
		if clientIDChanged {
			settings.AccessToken = ""
		}
		settings.ClientID = clientID
	}

	if input.SetClientSecret {
		clientSecret := strings.TrimSpace(input.ClientSecret)
		if clientSecret != "" {
			settings.ClientSecret = clientSecret
		}
	}

	if input.SetExportDirectory {
		settings.ExportDirectory = strings.TrimSpace(input.ExportDirectory)
	}

	if input.SetBackupStorage {
		storageKind, err := normalizeBackupStorage(input.BackupStorage)
		if err != nil {
			return SaveSettingsResult{}, err
		}
		settings.Backup.StorageKind = storageKind
	}

	if input.SetGoogleDriveClientID {
		clientID := strings.TrimSpace(input.GoogleDriveClientID)
		if clientID != settings.Backup.GoogleDrive.ClientID {
			clearGoogleDriveConnection(&settings.Backup.GoogleDrive)
		}
		settings.Backup.GoogleDrive.ClientID = clientID
	}

	if input.SetGoogleDriveClientSecret {
		clientSecret := strings.TrimSpace(input.GoogleDriveClientSecret)
		if clientSecret != "" && clientSecret != settings.Backup.GoogleDrive.ClientSecret {
			clearGoogleDriveConnection(&settings.Backup.GoogleDrive)
			settings.Backup.GoogleDrive.ClientSecret = clientSecret
		}
	}

	if input.SetGoogleDriveFolderName {
		folderName := strings.TrimSpace(input.GoogleDriveFolderName)
		if folderName != settings.Backup.GoogleDrive.FolderName {
			settings.Backup.GoogleDrive.FolderID = ""
			settings.Backup.GoogleDrive.FolderURL = ""
		}
		settings.Backup.GoogleDrive.FolderName = folderName
	}

	if err := s.store.Save(settings); err != nil {
		return SaveSettingsResult{}, err
	}

	return SaveSettingsResult{
		Settings:        settings,
		ClientIDChanged: clientIDChanged,
	}, nil
}

func (s *Service) SaveAccessToken(token string) (config.Settings, error) {
	settings, err := s.store.Load()
	if err != nil {
		return config.Settings{}, err
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return config.Settings{}, errors.New("no access token returned by Simkl")
	}

	settings.AccessToken = token
	if err := s.store.Save(settings); err != nil {
		return config.Settings{}, err
	}

	return settings, nil
}

func (s *Service) ClearAccessToken() (config.Settings, error) {
	settings, err := s.store.Load()
	if err != nil {
		return config.Settings{}, err
	}

	settings.AccessToken = ""
	if err := s.store.Save(settings); err != nil {
		return config.Settings{}, err
	}

	return settings, nil
}

func (s *Service) SaveGoogleDriveToken(token config.OAuthToken) (config.Settings, error) {
	settings, err := s.store.Load()
	if err != nil {
		return config.Settings{}, err
	}

	if strings.TrimSpace(token.RefreshToken) == "" {
		token.RefreshToken = settings.Backup.GoogleDrive.Token.RefreshToken
	}
	settings.Backup.GoogleDrive.Token = token
	if err := s.store.Save(settings); err != nil {
		return config.Settings{}, err
	}

	return settings, nil
}

func (s *Service) ClearGoogleDriveToken() (config.Settings, error) {
	settings, err := s.store.Load()
	if err != nil {
		return config.Settings{}, err
	}

	clearGoogleDriveConnection(&settings.Backup.GoogleDrive)
	if err := s.store.Save(settings); err != nil {
		return config.Settings{}, err
	}

	return settings, nil
}

func (s *Service) RequestDeviceCode() (simkl.DeviceCodeResponse, error) {
	settings, err := s.store.Load()
	if err != nil {
		return simkl.DeviceCodeResponse{}, err
	}
	if strings.TrimSpace(settings.ClientID) == "" {
		return simkl.DeviceCodeResponse{}, newPrerequisiteError("enter your Simkl client ID first")
	}

	return s.client.RequestDeviceCode(context.Background(), settings.ClientID, "")
}

func (s *Service) PollDeviceCode(userCode string) (simkl.DeviceCodeStatusResponse, error) {
	settings, err := s.store.Load()
	if err != nil {
		return simkl.DeviceCodeStatusResponse{}, err
	}
	if strings.TrimSpace(settings.ClientID) == "" {
		return simkl.DeviceCodeStatusResponse{}, newPrerequisiteError("missing Simkl client ID")
	}

	return s.client.PollDeviceCode(context.Background(), settings.ClientID, userCode)
}

func (s *Service) OAuthRedirectURI() string {
	return OAuthOOBRedirectURI
}

func (s *Service) StandardAuthURL() (string, error) {
	settings, err := s.store.Load()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(settings.ClientID) == "" {
		return "", newPrerequisiteError("enter your Simkl client ID first")
	}

	v := url.Values{}
	v.Set("response_type", "code")
	v.Set("client_id", settings.ClientID)
	v.Set("redirect_uri", OAuthOOBRedirectURI)

	return "https://simkl.com/oauth/authorize?" + v.Encode(), nil
}

func (s *Service) ExchangeOAuthCode(code string) (config.Settings, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return config.Settings{}, newUsageError("paste the authorization code from the Simkl page first")
	}

	settings, err := s.store.Load()
	if err != nil {
		return config.Settings{}, err
	}
	if strings.TrimSpace(settings.ClientID) == "" {
		return config.Settings{}, newPrerequisiteError("enter your Simkl client ID first")
	}
	if strings.TrimSpace(settings.ClientSecret) == "" {
		return config.Settings{}, newPrerequisiteError("enter your Simkl client secret first")
	}

	resp, err := s.client.ExchangeCode(
		context.Background(),
		settings.ClientID,
		settings.ClientSecret,
		code,
		OAuthOOBRedirectURI,
	)
	if err != nil {
		return config.Settings{}, err
	}

	return s.SaveAccessToken(resp.AccessToken)
}

func (s *Service) GoogleDriveAuthURL(redirectURI, state, verifier string) (string, error) {
	settings, err := s.store.Load()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(settings.Backup.GoogleDrive.ClientID) == "" {
		return "", newPrerequisiteError("save your Google Drive client ID first")
	}
	if strings.TrimSpace(settings.Backup.GoogleDrive.ClientSecret) == "" {
		return "", newPrerequisiteError("save your Google Drive client secret first")
	}

	return s.drive.AuthURL(
		settings.Backup.GoogleDrive.ClientID,
		settings.Backup.GoogleDrive.ClientSecret,
		redirectURI,
		state,
		verifier,
	)
}

func (s *Service) ExchangeGoogleDriveCode(code, redirectURI, verifier string) (config.Settings, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return config.Settings{}, newUsageError("complete the Google Drive approval flow first")
	}

	settings, err := s.store.Load()
	if err != nil {
		return config.Settings{}, err
	}

	token, err := s.drive.ExchangeCode(
		context.Background(),
		settings.Backup.GoogleDrive.ClientID,
		settings.Backup.GoogleDrive.ClientSecret,
		code,
		redirectURI,
		verifier,
	)
	if err != nil {
		return config.Settings{}, err
	}

	settings.Backup.GoogleDrive.Token = mergeOAuthTokens(settings.Backup.GoogleDrive.Token, token)

	if s.drive != nil {
		uploadResult, err := s.drive.UploadFiles(context.Background(), settings.Backup.GoogleDrive, nil)
		if err != nil {
			return config.Settings{}, err
		}

		settings.Backup.GoogleDrive.Token = mergeOAuthTokens(settings.Backup.GoogleDrive.Token, uploadResult.Token)
		settings.Backup.GoogleDrive.FolderID = uploadResult.FolderID
		settings.Backup.GoogleDrive.FolderName = firstNonEmpty(uploadResult.FolderName, settings.Backup.GoogleDrive.FolderName, gdrive.DefaultFolderName)
		settings.Backup.GoogleDrive.FolderURL = uploadResult.FolderURL
	}

	if err := s.store.Save(settings); err != nil {
		return config.Settings{}, err
	}

	return settings, nil
}

func (s *Service) ConfigSummary() (ConfigSummary, error) {
	settings, err := s.store.Load()
	if err != nil {
		return ConfigSummary{}, err
	}

	updatedAt := ""
	if !settings.UpdatedAt.IsZero() {
		updatedAt = settings.UpdatedAt.Format(time.RFC3339)
	}

	return ConfigSummary{
		ConfigPath:                 s.store.Path(),
		ClientID:                   settings.ClientID,
		HasClientSecret:            strings.TrimSpace(settings.ClientSecret) != "",
		HasAccessToken:             strings.TrimSpace(settings.AccessToken) != "",
		ExportDirectory:            settings.ExportDirectory,
		SuggestedExportDirectory:   DefaultExportDirectory(),
		BackupStorage:              backupStorageOrDefault(settings.Backup.StorageKind),
		GoogleDriveClientID:        settings.Backup.GoogleDrive.ClientID,
		HasGoogleDriveClientSecret: strings.TrimSpace(settings.Backup.GoogleDrive.ClientSecret) != "",
		HasGoogleDriveToken:        hasGoogleDriveToken(settings.Backup.GoogleDrive),
		GoogleDriveFolderName:      googleDriveFolderName(settings.Backup.GoogleDrive),
		GoogleDriveFolderURL:       settings.Backup.GoogleDrive.FolderURL,
		UpdatedAt:                  updatedAt,
	}, nil
}

func (s *Service) AuthSummary() (AuthSummary, error) {
	settings, err := s.store.Load()
	if err != nil {
		return AuthSummary{}, err
	}

	summary := AuthSummary{
		ClientIDConfigured:     strings.TrimSpace(settings.ClientID) != "",
		ClientSecretConfigured: strings.TrimSpace(settings.ClientSecret) != "",
		AccessTokenConfigured:  strings.TrimSpace(settings.AccessToken) != "",
	}
	summary.ReadyForHeadlessRun = summary.ClientIDConfigured && summary.AccessTokenConfigured

	return summary, nil
}

func (s *Service) ScheduleState() (ScheduleState, error) {
	settings, err := s.store.Load()
	if err != nil {
		return ScheduleState{}, err
	}

	return s.buildScheduleState(settings), nil
}

func (s *Service) ScheduleStateForSettings(settings config.Settings) ScheduleState {
	return s.buildScheduleState(settings)
}

func (s *Service) SaveSchedule(input ScheduleSettingsInput) (config.Settings, ScheduleState, error) {
	settings, err := s.store.Load()
	if err != nil {
		return config.Settings{}, ScheduleState{}, err
	}

	normalized, err := normalizeScheduleSettings(input)
	if err != nil {
		return config.Settings{}, ScheduleState{}, err
	}

	if normalized.Enabled {
		if s.scheduler == nil || !s.scheduler.Supported() {
			return config.Settings{}, ScheduleState{}, errors.New("recurring scheduling is not supported on this platform")
		}
		if strings.TrimSpace(settings.ClientID) == "" {
			return config.Settings{}, ScheduleState{}, newPrerequisiteError("save your Simkl client ID before enabling recurring backups")
		}
		if strings.TrimSpace(settings.AccessToken) == "" {
			return config.Settings{}, ScheduleState{}, newPrerequisiteError("authorize the app with Simkl before enabling recurring backups")
		}
		if !isStableExecutablePath(s.executablePath) {
			return config.Settings{}, ScheduleState{}, errors.New("scheduled backups can only be configured from a built desktop executable")
		}
		if settings.Backup.StorageKind == config.BackupStorageGDrive && !googleDriveUploadReady(settings.Backup.GoogleDrive) {
			return config.Settings{}, ScheduleState{}, newPrerequisiteError("connect Google Drive before enabling recurring backups that upload to Google Drive")
		}

		if _, err := s.scheduler.Sync(s.buildSchedulerConfig(normalized)); err != nil {
			return config.Settings{}, ScheduleState{}, err
		}
	} else if s.scheduler != nil {
		if err := s.scheduler.Remove(defaultTaskName); err != nil {
			return config.Settings{}, ScheduleState{}, err
		}
	}

	settings.Schedule = normalized
	if err := s.store.Save(settings); err != nil {
		return config.Settings{}, ScheduleState{}, err
	}

	return settings, s.buildScheduleState(settings), nil
}

func (s *Service) RunScheduledExport(request exporter.Request, maxBackupAgeOverride string) (ScheduledExportResult, error) {
	settings, err := s.store.Load()
	if err != nil {
		return ScheduledExportResult{}, err
	}

	useStaleGuard := settings.Schedule.RunIfBackupIsStale || strings.TrimSpace(maxBackupAgeOverride) != ""
	maxBackupAge := strings.TrimSpace(maxBackupAgeOverride)
	if maxBackupAge == "" {
		maxBackupAge = scheduleMaxBackupAgeOrDefault(settings.Schedule.MaxBackupAge)
	}
	maxAge, ok := parseScheduleDuration(maxBackupAge)
	if !ok {
		return ScheduledExportResult{}, newUsageError("max backup age must be a duration such as 12h, 24h, 3d, or 1w")
	}

	now := time.Now().UTC()
	if useStaleGuard && !settings.LastSuccessfulBackupAt.IsZero() {
		age := now.Sub(settings.LastSuccessfulBackupAt)
		if age < 0 {
			age = 0
		}

		if age < maxAge {
			return ScheduledExportResult{
				Ran:              false,
				Skipped:          true,
				Reason:           fmt.Sprintf("last successful backup is fresh (%s old, threshold %s)", humanDuration(age), maxBackupAge),
				LastBackupAt:     settings.LastSuccessfulBackupAt.UTC().Format(time.RFC3339),
				MaxBackupAge:     maxBackupAge,
				BackupAgeSeconds: int64(age.Seconds()),
			}, nil
		}
	}

	result, err := s.RunExport(request)
	if err != nil {
		return ScheduledExportResult{}, err
	}

	settings, err = s.store.Load()
	if err != nil {
		return ScheduledExportResult{}, err
	}
	lastBackupAt := parseExportedAtOrNow(result.ExportedAt)
	settings.LastSuccessfulBackupAt = lastBackupAt
	settings.LastSuccessfulBackupKind = "scheduled"
	if err := s.store.Save(settings); err != nil {
		return ScheduledExportResult{}, err
	}

	return ScheduledExportResult{
		Ran:              true,
		Skipped:          false,
		LastBackupAt:     lastBackupAt.UTC().Format(time.RFC3339),
		MaxBackupAge:     maxBackupAge,
		BackupAgeSeconds: 0,
		Result:           result,
	}, nil
}

func (s *Service) RunExport(request exporter.Request) (exporter.Result, error) {
	settings, err := s.store.Load()
	if err != nil {
		return exporter.Result{}, err
	}
	if strings.TrimSpace(settings.ClientID) == "" {
		return exporter.Result{}, newPrerequisiteError("enter your Simkl client ID first")
	}
	if strings.TrimSpace(settings.AccessToken) == "" {
		return exporter.Result{}, newPrerequisiteError("authorize the app with Simkl first")
	}

	useGoogleDrive := shouldUseGoogleDriveOutput(settings, request.ExportDirectory)
	if useGoogleDrive && !googleDriveUploadReady(settings.Backup.GoogleDrive) {
		return exporter.Result{}, newPrerequisiteError("connect Google Drive before running backups that use Google Drive storage")
	}

	request, err = normalizeRequest(request, settings, !useGoogleDrive)
	if err != nil {
		return exporter.Result{}, err
	}

	if useGoogleDrive {
		request.ExportDirectory, err = os.MkdirTemp("", "simklexpogter-gdrive-*")
		if err != nil {
			return exporter.Result{}, err
		}
		defer os.RemoveAll(request.ExportDirectory)
	}

	var activities map[string]any
	var previousAll string
	var currentAll string

	if request.UseActivityCheck {
		activities, err = s.client.FetchActivities(context.Background(), settings.ClientID, settings.AccessToken)
		if err != nil {
			return exporter.Result{}, err
		}
		previousAll = extractActivityAll(settings.LastActivities)
		currentAll = extractActivityAll(activities)

		if outputDirIsEmpty(request.ExportDirectory) {
			previousAll = ""
			request.DateFrom = ""
		} else if request.DateFrom == "" {
			request.DateFrom = previousAll
		}
	}

	plan := buildFetchPlan(request)
	payload := map[string][]map[string]any{
		"movies": {},
		"shows":  {},
		"anime":  {},
	}
	seen := map[string]map[any]struct{}{
		"movies": {},
		"shows":  {},
		"anime":  {},
	}

	for _, itemRequest := range plan {
		response, err := s.client.FetchAllItems(context.Background(), settings.ClientID, settings.AccessToken, itemRequest)
		if err != nil {
			return exporter.Result{}, err
		}
		if err := mergePayload(payload, seen, response); err != nil {
			return exporter.Result{}, err
		}
	}

	result, err := s.exporter.WriteSnapshot(request, payload, activities)
	if err != nil {
		return exporter.Result{}, err
	}

	result.ActivitiesChecked = request.UseActivityCheck
	result.EffectiveDateFrom = request.DateFrom
	result.StorageKind = config.BackupStorageLocal
	result.DestinationLabel = request.ExportDirectory

	if request.UseActivityCheck && previousAll == "" && request.DateFrom == "" {
		result.Warnings = append(result.Warnings, "No saved /sync/activities snapshot was found, so this export ran as a full snapshot.")
	}
	if request.UseActivityCheck && previousAll != "" && currentAll != "" && previousAll == currentAll {
		result.Warnings = append(result.Warnings, "Simkl activity timestamps did not change since the last saved snapshot. Matching exports may be empty.")
	}

	settingsChanged := false
	if request.UseActivityCheck && activities != nil {
		settings.LastActivities = activities
		settingsChanged = true
	}

	if useGoogleDrive {
		if s.drive == nil {
			return exporter.Result{}, errors.New("google drive backup is not available")
		}

		uploadResult, err := s.drive.UploadFiles(context.Background(), settings.Backup.GoogleDrive, exportedFilePaths(result.Files))
		if err != nil {
			return exporter.Result{}, err
		}

		settings.Backup.GoogleDrive.Token = mergeOAuthTokens(settings.Backup.GoogleDrive.Token, uploadResult.Token)
		settings.Backup.GoogleDrive.FolderID = uploadResult.FolderID
		settings.Backup.GoogleDrive.FolderName = firstNonEmpty(uploadResult.FolderName, settings.Backup.GoogleDrive.FolderName, gdrive.DefaultFolderName)
		settings.Backup.GoogleDrive.FolderURL = uploadResult.FolderURL
		settingsChanged = true

		result = applyGoogleDriveUploadResult(result, uploadResult)
	}

	exportedAt := parseExportedAtOrNow(result.ExportedAt)
	settings.LastSuccessfulBackupAt = exportedAt
	settings.LastSuccessfulBackupKind = "manual"
	settingsChanged = true

	if settingsChanged {
		if err := s.store.Save(settings); err != nil {
			return exporter.Result{}, err
		}
	}

	return result, nil
}

func NormalizeCLITypes(values []string) ([]string, error) {
	return normalizeValues(values, simkl.AllowedTypes, typeAliases)
}

func NormalizeCLIStatuses(values []string) ([]string, error) {
	return normalizeValues(values, simkl.AllowedStatuses, nil)
}

func DefaultExportDirectory() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, "Downloads", DefaultAppName)
}

func newPrerequisiteError(message string) error {
	return &prerequisiteError{message: message}
}

func newUsageError(message string) error {
	return &usageError{message: message}
}

func normalizeRequest(request exporter.Request, settings config.Settings, useSavedLocalOutput bool) (exporter.Request, error) {
	var err error

	request.Types, err = normalizeValues(request.Types, simkl.AllowedTypes, typeAliases)
	if err != nil {
		return exporter.Request{}, err
	}
	request.Statuses, err = normalizeValues(request.Statuses, simkl.AllowedStatuses, nil)
	if err != nil {
		return exporter.Request{}, err
	}

	request.Extended, err = normalizeEnum(request.Extended, simkl.AllowedExtendedValues, "full")
	if err != nil {
		return exporter.Request{}, err
	}
	request.OutputFormat, err = normalizeEnum(request.OutputFormat, exporter.AllowedOutputFormats(), exporter.FormatCSV)
	if err != nil {
		return exporter.Request{}, err
	}
	request.FieldMode, err = normalizeEnum(request.FieldMode, exporter.AllowedFieldModes(), exporter.FieldModeAll)
	if err != nil {
		return exporter.Request{}, err
	}
	request.Grouping, err = normalizeEnum(request.Grouping, exporter.AllowedGroupings(), exporter.GroupingSeparate)
	if err != nil {
		return exporter.Request{}, err
	}

	request.DateFrom = strings.TrimSpace(request.DateFrom)
	request.ExportDirectory = strings.TrimSpace(request.ExportDirectory)
	request.FilenamePrefix = strings.TrimSpace(request.FilenamePrefix)
	if request.FilenamePrefix == "" {
		request.FilenamePrefix = defaultFilenamePrefix
	}

	if useSavedLocalOutput && request.ExportDirectory == "" {
		request.ExportDirectory = strings.TrimSpace(settings.ExportDirectory)
	}
	if useSavedLocalOutput && request.ExportDirectory == "" {
		request.ExportDirectory = DefaultExportDirectory()
	}

	return request, nil
}

func buildFetchPlan(request exporter.Request) []simkl.FetchRequest {
	base := simkl.FetchRequest{
		DateFrom:             request.DateFrom,
		Extended:             request.Extended,
		EpisodeWatchedAt:     request.EpisodeWatchedAt,
		IncludeMemos:         request.IncludeMemos,
		IncludeNextWatchInfo: request.IncludeNextWatchInfo,
	}

	if len(request.Statuses) == 0 {
		if len(request.Types) == 0 || len(request.Types) == len(simkl.AllowedTypes) {
			return []simkl.FetchRequest{base}
		}

		plan := make([]simkl.FetchRequest, 0, len(request.Types))
		for _, mediaType := range request.Types {
			item := base
			item.Type = mediaType
			plan = append(plan, item)
		}
		return plan
	}

	types := request.Types
	if len(types) == 0 {
		types = append([]string(nil), simkl.AllowedTypes...)
	}

	plan := make([]simkl.FetchRequest, 0, len(types)*len(request.Statuses))
	for _, mediaType := range types {
		for _, status := range request.Statuses {
			item := base
			item.Type = mediaType
			item.Status = status
			plan = append(plan, item)
		}
	}

	return plan
}

func mergePayload(target map[string][]map[string]any, seen map[string]map[any]struct{}, payload map[string]any) error {
	for _, mediaType := range simkl.AllowedTypes {
		rawItems, ok := payload[mediaType]
		if !ok || rawItems == nil {
			continue
		}

		list, ok := rawItems.([]any)
		if !ok {
			return fmt.Errorf("unexpected Simkl payload for %s", mediaType)
		}

		for _, rawItem := range list {
			item, ok := rawItem.(map[string]any)
			if !ok {
				return fmt.Errorf("unexpected Simkl item for %s", mediaType)
			}

			if id := getSimklID(item, mediaType); id != nil {
				if _, alreadySeen := seen[mediaType][id]; alreadySeen {
					continue
				}
				seen[mediaType][id] = struct{}{}
			}

			target[mediaType] = append(target[mediaType], item)
		}
	}

	return nil
}

func getSimklID(item map[string]any, mediaType string) any {
	key := "show"
	if mediaType == "movies" {
		key = "movie"
	}

	media, _ := item[key].(map[string]any)
	if media == nil {
		return nil
	}

	ids, _ := media["ids"].(map[string]any)
	if ids == nil {
		return nil
	}

	return ids["simkl"]
}

func normalizeValues(values []string, allowed []string, aliases map[string]string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	allowedSet := map[string]struct{}{}
	for _, item := range allowed {
		allowedSet[item] = struct{}{}
	}

	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.ToLower(strings.TrimSpace(raw))
		if value == "" {
			continue
		}
		if aliases != nil {
			if mapped, ok := aliases[value]; ok {
				value = mapped
			}
		}
		if _, ok := allowedSet[value]; !ok {
			return nil, newUsageError(fmt.Sprintf("unsupported value %q", raw))
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	return normalized, nil
}

func normalizeEnum(value string, allowed []string, fallback string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return fallback, nil
	}

	for _, item := range allowed {
		if normalized == item {
			return normalized, nil
		}
	}

	return "", newUsageError(fmt.Sprintf("unsupported value %q", value))
}

func extractActivityAll(activities map[string]any) string {
	if len(activities) == 0 {
		return ""
	}
	value, _ := activities["all"].(string)
	return strings.TrimSpace(value)
}

func outputDirIsEmpty(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return true
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			return false
		}
	}
	return true
}

func exportedFilePaths(files []exporter.ExportedFile) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		if strings.TrimSpace(file.Path) == "" {
			continue
		}
		paths = append(paths, file.Path)
	}
	return paths
}

func applyGoogleDriveUploadResult(result exporter.Result, uploadResult gdrive.UploadResult) exporter.Result {
	rootLabel := "Google Drive / " + firstNonEmpty(uploadResult.FolderName, gdrive.DefaultFolderName)
	destinationLabel := rootLabel
	destinationURL := uploadResult.FolderURL

	if strings.TrimSpace(uploadResult.UploadFolderName) != "" {
		destinationLabel = rootLabel + " / " + uploadResult.UploadFolderName
		destinationURL = firstNonEmpty(uploadResult.UploadFolderURL, destinationURL)
	}

	result.StorageKind = config.BackupStorageGDrive
	result.OutputDirectory = firstNonEmpty(destinationURL, destinationLabel)
	result.DestinationLabel = destinationLabel
	result.DestinationURL = destinationURL

	for index := range result.Files {
		result.Files[index].StorageKind = config.BackupStorageGDrive
		if index < len(uploadResult.Files) {
			result.Files[index].Path = destinationLabel + " / " + firstNonEmpty(uploadResult.Files[index].Name, result.Files[index].Path)
		}
	}

	return result
}

func normalizeBackupStorage(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", config.BackupStorageLocal:
		return config.BackupStorageLocal, nil
	case config.BackupStorageGDrive, "google-drive", "google_drive":
		return config.BackupStorageGDrive, nil
	default:
		return "", newUsageError(fmt.Sprintf("unsupported backup storage %q", value))
	}
}

func backupStorageOrDefault(value string) string {
	normalized, err := normalizeBackupStorage(value)
	if err != nil {
		return config.BackupStorageLocal
	}
	return normalized
}

func shouldUseGoogleDriveStorage(settings config.Settings) bool {
	return backupStorageOrDefault(settings.Backup.StorageKind) == config.BackupStorageGDrive
}

func shouldUseGoogleDriveOutput(settings config.Settings, requestedOutput string) bool {
	return shouldUseGoogleDriveStorage(settings) && strings.TrimSpace(requestedOutput) == ""
}

func clearGoogleDriveConnection(settings *config.GoogleDriveSettings) {
	if settings == nil {
		return
	}
	settings.Token = config.OAuthToken{}
	settings.FolderID = ""
	settings.FolderURL = ""
}

func hasGoogleDriveToken(settings config.GoogleDriveSettings) bool {
	return strings.TrimSpace(settings.Token.RefreshToken) != "" || strings.TrimSpace(settings.Token.AccessToken) != ""
}

func googleDriveUploadReady(settings config.GoogleDriveSettings) bool {
	return strings.TrimSpace(settings.ClientID) != "" &&
		strings.TrimSpace(settings.ClientSecret) != "" &&
		hasGoogleDriveToken(settings)
}

func googleDriveFolderName(settings config.GoogleDriveSettings) string {
	return firstNonEmpty(settings.FolderName, gdrive.DefaultFolderName)
}

func backupDestinationPreview(settings config.Settings) string {
	if shouldUseGoogleDriveStorage(settings) {
		return firstNonEmpty(settings.Backup.GoogleDrive.FolderURL, "Google Drive / "+googleDriveFolderName(settings.Backup.GoogleDrive))
	}
	return firstNonEmpty(strings.TrimSpace(settings.ExportDirectory), DefaultExportDirectory())
}

func mergeOAuthTokens(current, next config.OAuthToken) config.OAuthToken {
	if strings.TrimSpace(next.AccessToken) != "" {
		current.AccessToken = next.AccessToken
	}
	if strings.TrimSpace(next.TokenType) != "" {
		current.TokenType = next.TokenType
	}
	if strings.TrimSpace(next.RefreshToken) != "" {
		current.RefreshToken = next.RefreshToken
	}
	if !next.Expiry.IsZero() {
		current.Expiry = next.Expiry
	}
	return current
}

func normalizeMaxBackupAge(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "24h", nil
	}
	if _, ok := parseScheduleDuration(value); !ok {
		return "", newUsageError("max backup age must be a duration such as 12h, 24h, 3d, or 1w")
	}
	return value, nil
}

func scheduleMaxBackupAgeOrDefault(value string) string {
	normalized, err := normalizeMaxBackupAge(value)
	if err != nil {
		return "24h"
	}
	return normalized
}

func parseScheduleDuration(value string) (time.Duration, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = "24h"
	}

	if strings.HasSuffix(value, "d") || strings.HasSuffix(value, "w") {
		multiplier := 24 * time.Hour
		numberPart := strings.TrimSuffix(value, "d")
		if strings.HasSuffix(value, "w") {
			multiplier = 7 * 24 * time.Hour
			numberPart = strings.TrimSuffix(value, "w")
		}
		count, err := strconv.Atoi(strings.TrimSpace(numberPart))
		if err != nil || count <= 0 {
			return 0, false
		}
		return time.Duration(count) * multiplier, true
	}

	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, false
	}
	return duration, true
}

func parseExportedAtOrNow(value string) time.Time {
	if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value)); err == nil {
		return parsed.UTC()
	}
	return time.Now().UTC()
}

func humanDuration(value time.Duration) string {
	if value < time.Minute {
		return "less than 1m"
	}
	if value < time.Hour {
		return fmt.Sprintf("%dm", int(value.Minutes()))
	}
	if value < 24*time.Hour {
		return fmt.Sprintf("%dh", int(value.Hours()))
	}
	return fmt.Sprintf("%dd", int(value.Hours()/24))
}

func normalizeScheduleSettings(input ScheduleSettingsInput) (config.ScheduleSettings, error) {
	frequency, err := normalizeEnum(input.Frequency, []string{"daily", "weekly"}, "daily")
	if err != nil {
		return config.ScheduleSettings{}, err
	}
	outputFormat, err := normalizeEnum(input.OutputFormat, exporter.AllowedOutputFormats(), exporter.FormatCSV)
	if err != nil {
		return config.ScheduleSettings{}, err
	}
	fieldMode, err := normalizeEnum(input.FieldMode, exporter.AllowedFieldModes(), exporter.FieldModeAll)
	if err != nil {
		return config.ScheduleSettings{}, err
	}
	content, err := normalizeValues(input.Content, simkl.AllowedTypes, typeAliases)
	if err != nil {
		return config.ScheduleSettings{}, err
	}
	if len(content) == 0 {
		content = append([]string(nil), simkl.AllowedTypes...)
	}

	maxBackupAge, err := normalizeMaxBackupAge(input.MaxBackupAge)
	if err != nil {
		return config.ScheduleSettings{}, err
	}

	scheduleTime := strings.TrimSpace(input.Time)
	if scheduleTime == "" {
		scheduleTime = "02:00"
	}
	if _, err := time.Parse("15:04", scheduleTime); err != nil {
		return config.ScheduleSettings{}, newUsageError("time must use HH:MM in 24-hour format")
	}

	days := normalizeScheduleDays(input.Days)
	if frequency == "weekly" && len(days) == 0 {
		days = []string{"mon"}
	}

	return config.ScheduleSettings{
		Enabled:            input.Enabled,
		Frequency:          frequency,
		Time:               scheduleTime,
		Days:               days,
		OutputFormat:       outputFormat,
		FieldMode:          fieldMode,
		Content:            content,
		UseActivityCheck:   input.UseActivityCheck,
		MaxBackupAge:       maxBackupAge,
		RunIfBackupIsStale: input.RunIfBackupIsStale,
	}, nil
}

func normalizeScheduleDays(days []string) []string {
	allowed := map[string]struct{}{
		"mon": {}, "tue": {}, "wed": {}, "thu": {}, "fri": {}, "sat": {}, "sun": {},
	}
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(days))
	for _, day := range days {
		value := strings.ToLower(strings.TrimSpace(day))
		if _, ok := allowed[value]; !ok {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	return normalized
}

func (s *Service) buildSchedulerConfig(scheduleSettings config.ScheduleSettings) scheduler.Config {
	args := []string{
		"run",
		"--scheduled",
		"--format", scheduleSettings.OutputFormat,
		"--field-mode", scheduleSettings.FieldMode,
		"--content", strings.Join(scheduleSettings.Content, ","),
	}
	if scheduleSettings.UseActivityCheck {
		args = append(args, "--activity-check")
	}

	days := append([]string(nil), scheduleSettings.Days...)

	return scheduler.Config{
		TaskName:       defaultTaskName,
		Description:    "Recurring backup for SimklExpoGter",
		ExecutablePath: s.executablePath,
		Arguments:      args,
		Frequency:      scheduleSettings.Frequency,
		Time:           scheduleSettings.Time,
		Days:           days,
	}
}

func (s *Service) buildScheduleState(settings config.Settings) ScheduleState {
	scheduleSettings := settings.Schedule
	state := ScheduleState{
		Supported:              s.scheduler != nil && s.scheduler.Supported(),
		Enabled:                scheduleSettings.Enabled,
		Installed:              false,
		Frequency:              scheduleStringOrDefault(scheduleSettings.Frequency, "daily"),
		Time:                   scheduleStringOrDefault(scheduleSettings.Time, "02:00"),
		Days:                   scheduleDaysOrDefault(scheduleSettings.Days),
		OutputFormat:           scheduleStringOrDefault(scheduleSettings.OutputFormat, exporter.FormatCSV),
		FieldMode:              scheduleStringOrDefault(scheduleSettings.FieldMode, exporter.FieldModeAll),
		Content:                scheduleContentOrDefault(scheduleSettings.Content),
		UseActivityCheck:       scheduleSettings.UseActivityCheck,
		MaxBackupAge:           scheduleMaxBackupAgeOrDefault(scheduleSettings.MaxBackupAge),
		RunIfBackupIsStale:     scheduleSettings.RunIfBackupIsStale,
		TaskName:               defaultTaskName,
		UsesSavedOutput:        true,
		OutputDirectoryPreview: backupDestinationPreview(settings),
	}

	if !settings.LastSuccessfulBackupAt.IsZero() {
		state.LastSuccessfulBackupAt = settings.LastSuccessfulBackupAt.UTC().Format(time.RFC3339)
		state.LastSuccessfulBackupKind = settings.LastSuccessfulBackupKind
		age := time.Since(settings.LastSuccessfulBackupAt)
		if age < 0 {
			age = 0
		}
		state.BackupAgeSeconds = int64(age.Seconds())
		if maxAge, ok := parseScheduleDuration(state.MaxBackupAge); ok {
			state.BackupFresh = age < maxAge
		}
	}

	if !state.Supported {
		state.Message = "Recurring scheduling is not supported on this platform."
		return state
	}

	if !scheduleSettings.Enabled {
		state.Message = "Recurring backups are disabled."
		return state
	}

	if !isStableExecutablePath(s.executablePath) {
		state.Message = "Scheduled backups can only be configured from a built desktop executable."
		return state
	}

	info, err := s.scheduler.Query(defaultTaskName)
	if err != nil {
		state.Message = err.Error()
		return state
	}

	state.Installed = info.Installed
	state.Status = info.Status
	state.NextRunAt = info.NextRunAt
	state.LastRunAt = info.LastRunAt
	state.LastResult = info.LastResult
	if info.Message != "" {
		state.Message = info.Message
	} else if info.Installed {
		if shouldUseGoogleDriveStorage(settings) {
			state.Message = "Recurring backup is synced with the system scheduler and uploads to Google Drive."
		} else {
			state.Message = "Recurring backup is synced with the system scheduler."
		}
	} else {
		state.Message = "Recurring backup is enabled in config but no scheduler entry is installed."
	}

	return state
}

func scheduleStringOrDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func scheduleDaysOrDefault(days []string) []string {
	normalized := normalizeScheduleDays(days)
	if len(normalized) == 0 {
		return []string{"mon"}
	}
	return normalized
}

func scheduleContentOrDefault(content []string) []string {
	normalized, err := normalizeValues(content, simkl.AllowedTypes, typeAliases)
	if err != nil || len(normalized) == 0 {
		return append([]string(nil), simkl.AllowedTypes...)
	}
	return normalized
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func isStableExecutablePath(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	lower := strings.ToLower(path)
	return !strings.Contains(lower, "go-build") && !strings.Contains(lower, `\temp\`)
}
