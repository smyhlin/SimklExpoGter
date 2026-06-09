//go:build !cli

package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"SimklExpoGter/internal/appsvc"
	"SimklExpoGter/internal/config"
	"SimklExpoGter/internal/exporter"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/oauth2"
)

type App struct {
	ctx context.Context
	svc *appsvc.Service

	authMu      sync.Mutex
	authSession *deviceAuthSession

	gdriveMu      sync.Mutex
	gdriveSession *googleDriveAuthSession
}

type AppState struct {
	Settings               SettingsState               `json:"settings"`
	Schedule               ScheduleStateView           `json:"schedule"`
	LastActivities         map[string]any              `json:"lastActivities,omitempty"`
	PendingAuth            *DeviceAuthSessionView      `json:"pendingAuth,omitempty"`
	PendingGoogleDriveAuth *GoogleDriveAuthSessionView `json:"pendingGoogleDriveAuth,omitempty"`
}

type SettingsState struct {
	ClientID                   string `json:"clientId"`
	HasClientSecret            bool   `json:"hasClientSecret"`
	ExportDirectory            string `json:"exportDirectory"`
	SuggestedExportDirectory   string `json:"suggestedExportDirectory"`
	HasAccessToken             bool   `json:"hasAccessToken"`
	ConfigPath                 string `json:"configPath"`
	BackupStorage              string `json:"backupStorage"`
	GoogleDriveClientID        string `json:"googleDriveClientId"`
	HasGoogleDriveClientSecret bool   `json:"hasGoogleDriveClientSecret"`
	HasGoogleDriveToken        bool   `json:"hasGoogleDriveToken"`
	GoogleDriveFolderName      string `json:"googleDriveFolderName"`
	GoogleDriveFolderURL       string `json:"googleDriveFolderUrl"`
}

type SaveSettingsInput struct {
	ClientID                string `json:"clientId"`
	ClientSecret            string `json:"clientSecret"`
	ExportDirectory         string `json:"exportDirectory"`
	BackupStorage           string `json:"backupStorage"`
	GoogleDriveClientID     string `json:"googleDriveClientId"`
	GoogleDriveClientSecret string `json:"googleDriveClientSecret"`
	GoogleDriveFolderName   string `json:"googleDriveFolderName"`
}

type SaveScheduleInput struct {
	Enabled            bool     `json:"enabled"`
	Frequency          string   `json:"frequency"`
	Time               string   `json:"time"`
	Days               []string `json:"days"`
	OutputFormat       string   `json:"outputFormat"`
	FieldMode          string   `json:"fieldMode"`
	Content            []string `json:"content"`
	UseActivityCheck   bool     `json:"useActivityCheck"`
	MaxBackupAge       string   `json:"maxBackupAge"`
	RunIfBackupIsStale bool     `json:"runIfBackupIsStale"`
}

type ScheduleStateView struct {
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

type DeviceAuthSessionView struct {
	UserCode        string `json:"userCode"`
	VerificationURL string `json:"verificationUrl"`
	PinURL          string `json:"pinUrl"`
	IntervalSeconds int    `json:"intervalSeconds"`
	ExpiresAt       string `json:"expiresAt"`
}

type DeviceAuthStatus struct {
	State   string                 `json:"state"`
	Message string                 `json:"message"`
	Session *DeviceAuthSessionView `json:"session,omitempty"`
}

type GoogleDriveAuthSessionView struct {
	AuthURL     string `json:"authUrl"`
	ExpiresAt   string `json:"expiresAt"`
	RedirectURI string `json:"redirectUri"`
}

type GoogleDriveAuthStatus struct {
	State   string                      `json:"state"`
	Message string                      `json:"message"`
	Session *GoogleDriveAuthSessionView `json:"session,omitempty"`
}

type deviceAuthSession struct {
	UserCode               string
	VerificationURL        string
	IntervalSeconds        int
	CurrentIntervalSeconds int
	ExpiresAt              time.Time
}

type googleDriveAuthSession struct {
	mu          sync.Mutex
	AuthURL     string
	RedirectURI string
	State       string
	Verifier    string
	ResultState string
	Message     string
	ExpiresAt   time.Time
	Server      *http.Server
	Listener    net.Listener
}

func NewApp(service *appsvc.Service) *App {
	return &App{
		svc: service,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetAppState() (AppState, error) {
	settings, err := a.svc.LoadSettings()
	if err != nil {
		return AppState{}, err
	}

	return a.buildState(settings), nil
}

func (a *App) SaveSettings(input SaveSettingsInput) (AppState, error) {
	result, err := a.svc.SaveSettings(appsvc.SaveSettingsInput{
		ClientID:                   input.ClientID,
		ClientSecret:               input.ClientSecret,
		ExportDirectory:            input.ExportDirectory,
		BackupStorage:              input.BackupStorage,
		GoogleDriveClientID:        input.GoogleDriveClientID,
		GoogleDriveClientSecret:    input.GoogleDriveClientSecret,
		GoogleDriveFolderName:      input.GoogleDriveFolderName,
		SetClientID:                true,
		SetClientSecret:            true,
		SetExportDirectory:         true,
		SetBackupStorage:           true,
		SetGoogleDriveClientID:     true,
		SetGoogleDriveClientSecret: true,
		SetGoogleDriveFolderName:   true,
	})
	if err != nil {
		return AppState{}, err
	}
	if result.ClientIDChanged {
		a.clearAuthSession()
	}
	a.clearGoogleDriveSession()

	return a.buildState(result.Settings), nil
}

func (a *App) SaveSchedule(input SaveScheduleInput) (AppState, error) {
	settings, scheduleState, err := a.svc.SaveSchedule(appsvc.ScheduleSettingsInput{
		Enabled:            input.Enabled,
		Frequency:          input.Frequency,
		Time:               input.Time,
		Days:               input.Days,
		OutputFormat:       input.OutputFormat,
		FieldMode:          input.FieldMode,
		Content:            input.Content,
		UseActivityCheck:   input.UseActivityCheck,
		MaxBackupAge:       input.MaxBackupAge,
		RunIfBackupIsStale: input.RunIfBackupIsStale,
	})
	if err != nil {
		return AppState{}, err
	}

	return a.buildStateWithSchedule(settings, scheduleState), nil
}

func (a *App) ChooseExportDirectory() (string, error) {
	if a.ctx == nil {
		return "", errors.New("application is not ready yet")
	}

	selected, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose an export directory",
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(selected) == "" {
		return "", nil
	}

	_, err = a.svc.SaveSettings(appsvc.SaveSettingsInput{
		ExportDirectory:    selected,
		SetExportDirectory: true,
	})
	if err != nil {
		return "", err
	}

	return selected, nil
}

func (a *App) ExportSettingsBackup(password string) (string, error) {
	if a.ctx == nil {
		return "", errors.New("application is not ready yet")
	}

	defaultDirectory := settingsDialogDirectory(a.svc.Path())
	destination, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultDirectory: defaultDirectory,
		DefaultFilename:  appsvc.DefaultSettingsBackupFilename(time.Now().UTC()),
		Title:            "Export encrypted settings backup",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Encrypted Settings Backup",
				Pattern:     "*" + appsvc.SettingsBackupExtension,
			},
		},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(destination) == "" {
		return "", nil
	}

	return a.svc.ExportEncryptedSettings(destination, password)
}

func (a *App) ImportSettingsBackup(password string) (string, error) {
	if a.ctx == nil {
		return "", errors.New("application is not ready yet")
	}

	defaultDirectory := settingsDialogDirectory(a.svc.Path())
	source, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		DefaultDirectory: defaultDirectory,
		Title:            "Import encrypted settings backup",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Encrypted Settings Backup",
				Pattern:     "*" + appsvc.SettingsBackupExtension,
			},
		},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(source) == "" {
		return "", nil
	}

	if err := a.svc.ImportEncryptedSettings(source, password); err != nil {
		return "", err
	}

	a.clearAuthSession()
	a.clearGoogleDriveSession()

	return source, nil
}

func (a *App) StartDeviceAuth() (DeviceAuthSessionView, error) {
	response, err := a.svc.RequestDeviceCode()
	if err != nil {
		return DeviceAuthSessionView{}, err
	}

	session := &deviceAuthSession{
		UserCode:               response.UserCode,
		VerificationURL:        response.VerificationURL,
		IntervalSeconds:        response.Interval,
		CurrentIntervalSeconds: response.Interval,
		ExpiresAt:              time.Now().UTC().Add(time.Duration(response.ExpiresIn) * time.Second),
	}
	a.setAuthSession(session)

	view := session.view()
	if view == nil {
		return DeviceAuthSessionView{}, errors.New("failed to create authorization session")
	}

	return *view, nil
}

func (a *App) CheckDeviceAuth() (DeviceAuthStatus, error) {
	session := a.getAuthSession()
	if session == nil {
		return DeviceAuthStatus{}, errors.New("start authorization first")
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		a.clearAuthSession()
		return DeviceAuthStatus{
			State:   "expired",
			Message: "The Simkl authorization code expired. Start a new login flow.",
		}, nil
	}

	response, err := a.svc.PollDeviceCode(session.UserCode)
	if err != nil {
		return DeviceAuthStatus{}, err
	}

	switch {
	case strings.EqualFold(strings.TrimSpace(response.Result), "OK") || strings.TrimSpace(response.AccessToken) != "":
		if _, err := a.svc.SaveAccessToken(response.AccessToken); err != nil {
			return DeviceAuthStatus{}, err
		}
		a.clearAuthSession()
		return DeviceAuthStatus{
			State:   "authorized",
			Message: "Simkl access token saved.",
		}, nil

	case strings.EqualFold(strings.TrimSpace(response.Message), "Slow down"):
		a.authMu.Lock()
		if a.authSession != nil {
			newInterval := a.authSession.CurrentIntervalSeconds * 2
			if newInterval > 30 {
				newInterval = 30
			}
			a.authSession.CurrentIntervalSeconds = newInterval
		}
		a.authMu.Unlock()
		session = a.getAuthSession()
		return DeviceAuthStatus{
			State:   "slow-down",
			Message: fmt.Sprintf("Simkl asked the app to slow down. Wait at least %d seconds before polling again.", session.CurrentIntervalSeconds),
			Session: session.view(),
		}, nil

	case strings.EqualFold(strings.TrimSpace(response.Message), "Authorization pending"),
		strings.EqualFold(strings.TrimSpace(response.Result), "KO") && strings.TrimSpace(response.Message) == "":
		return DeviceAuthStatus{
			State:   "pending",
			Message: "Authorization pending - approve the request on the Simkl pin page.",
			Session: session.view(),
		}, nil

	case strings.EqualFold(strings.TrimSpace(response.Result), "KO"):
		message := strings.TrimSpace(response.Message)
		if message == "" {
			message = "Authorization was rejected by Simkl."
		}
		a.clearAuthSession()
		return DeviceAuthStatus{
			State:   "error",
			Message: message,
		}, nil

	default:
		message := strings.TrimSpace(response.Message)
		if message == "" {
			message = "Authorization is still pending."
		}
		return DeviceAuthStatus{
			State:   "pending",
			Message: message,
			Session: session.view(),
		}, nil
	}
}

func (a *App) Logout() (AppState, error) {
	settings, err := a.svc.ClearAccessToken()
	if err != nil {
		return AppState{}, err
	}
	a.clearAuthSession()

	return a.buildState(settings), nil
}

func (a *App) GetOAuthRedirectURI() string {
	return a.svc.OAuthRedirectURI()
}

func (a *App) GetStandardAuthURL() (string, error) {
	return a.svc.StandardAuthURL()
}

func (a *App) ExchangeOAuthCode(code string) (AppState, error) {
	settings, err := a.svc.ExchangeOAuthCode(code)
	if err != nil {
		return AppState{}, err
	}

	return a.buildState(settings), nil
}

func (a *App) StartGoogleDriveAuth() (GoogleDriveAuthSessionView, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return GoogleDriveAuthSessionView{}, err
	}

	redirectURI := fmt.Sprintf("http://%s/google-drive/callback", listener.Addr().String())
	state := uuid.NewString()
	verifier := oauth2.GenerateVerifier()
	authURL, err := a.svc.GoogleDriveAuthURL(redirectURI, state, verifier)
	if err != nil {
		_ = listener.Close()
		return GoogleDriveAuthSessionView{}, err
	}

	session := &googleDriveAuthSession{
		AuthURL:     authURL,
		RedirectURI: redirectURI,
		State:       state,
		Verifier:    verifier,
		ResultState: "pending",
		Message:     "Waiting for Google Drive approval.",
		ExpiresAt:   time.Now().UTC().Add(10 * time.Minute),
		Listener:    listener,
	}

	mux := http.NewServeMux()
	session.Server = &http.Server{Handler: mux}
	mux.HandleFunc("/google-drive/callback", func(w http.ResponseWriter, r *http.Request) {
		a.handleGoogleDriveCallback(session, w, r)
	})

	a.setGoogleDriveSession(session)
	go func() {
		_ = session.Server.Serve(listener)
	}()

	view := session.view()
	if view == nil {
		return GoogleDriveAuthSessionView{}, errors.New("failed to start Google Drive authorization")
	}

	return *view, nil
}

func (a *App) CheckGoogleDriveAuth() (GoogleDriveAuthStatus, error) {
	session := a.getGoogleDriveSession()
	if session == nil {
		return GoogleDriveAuthStatus{}, errors.New("start Google Drive authorization first")
	}

	resultState, message, expiresAt := session.status()
	if resultState == "pending" && time.Now().UTC().After(expiresAt) {
		a.clearGoogleDriveSession()
		return GoogleDriveAuthStatus{
			State:   "expired",
			Message: "The Google Drive approval window expired. Start a new login flow.",
		}, nil
	}

	switch resultState {
	case "authorized":
		a.clearGoogleDriveSession()
		return GoogleDriveAuthStatus{
			State:   "authorized",
			Message: firstNonEmpty(message, "Google Drive connected."),
		}, nil
	case "error":
		a.clearGoogleDriveSession()
		return GoogleDriveAuthStatus{
			State:   "error",
			Message: firstNonEmpty(message, "Google Drive authorization failed."),
		}, nil
	default:
		return GoogleDriveAuthStatus{
			State:   "pending",
			Message: firstNonEmpty(message, "Waiting for Google Drive approval."),
			Session: session.view(),
		}, nil
	}
}

func (a *App) DisconnectGoogleDrive() (AppState, error) {
	settings, err := a.svc.ClearGoogleDriveToken()
	if err != nil {
		return AppState{}, err
	}
	a.clearGoogleDriveSession()

	return a.buildState(settings), nil
}

func (a *App) RunExport(request exporter.Request) (exporter.Result, error) {
	return a.svc.RunExport(request)
}

func (a *App) buildState(settings config.Settings) AppState {
	return a.buildStateWithSchedule(settings, a.svc.ScheduleStateForSettings(settings))
}

func (a *App) buildStateWithSchedule(settings config.Settings, scheduleState appsvc.ScheduleState) AppState {
	return AppState{
		Settings: SettingsState{
			ClientID:                   settings.ClientID,
			HasClientSecret:            strings.TrimSpace(settings.ClientSecret) != "",
			ExportDirectory:            settings.ExportDirectory,
			SuggestedExportDirectory:   appsvc.DefaultExportDirectory(),
			HasAccessToken:             strings.TrimSpace(settings.AccessToken) != "",
			ConfigPath:                 a.svc.Path(),
			BackupStorage:              firstNonEmpty(settings.Backup.StorageKind, config.BackupStorageLocal),
			GoogleDriveClientID:        settings.Backup.GoogleDrive.ClientID,
			HasGoogleDriveClientSecret: strings.TrimSpace(settings.Backup.GoogleDrive.ClientSecret) != "",
			HasGoogleDriveToken:        strings.TrimSpace(settings.Backup.GoogleDrive.Token.RefreshToken) != "" || strings.TrimSpace(settings.Backup.GoogleDrive.Token.AccessToken) != "",
			GoogleDriveFolderName:      googleDriveFolderName(settings.Backup.GoogleDrive),
			GoogleDriveFolderURL:       settings.Backup.GoogleDrive.FolderURL,
		},
		Schedule:               mapScheduleState(scheduleState),
		LastActivities:         settings.LastActivities,
		PendingAuth:            a.getAuthSession().viewOrNil(),
		PendingGoogleDriveAuth: a.getGoogleDriveSession().viewOrNil(),
	}
}

func (a *App) setAuthSession(session *deviceAuthSession) {
	a.authMu.Lock()
	defer a.authMu.Unlock()
	a.authSession = session
}

func (a *App) clearAuthSession() {
	a.authMu.Lock()
	defer a.authMu.Unlock()
	a.authSession = nil
}

func (a *App) getAuthSession() *deviceAuthSession {
	a.authMu.Lock()
	defer a.authMu.Unlock()
	return a.authSession
}

func (a *App) setGoogleDriveSession(session *googleDriveAuthSession) {
	previous := a.getGoogleDriveSession()
	a.gdriveMu.Lock()
	a.gdriveSession = session
	a.gdriveMu.Unlock()
	if previous != nil {
		previous.close()
	}
}

func (a *App) clearGoogleDriveSession() {
	a.gdriveMu.Lock()
	session := a.gdriveSession
	a.gdriveSession = nil
	a.gdriveMu.Unlock()
	if session != nil {
		session.close()
	}
}

func (a *App) getGoogleDriveSession() *googleDriveAuthSession {
	a.gdriveMu.Lock()
	defer a.gdriveMu.Unlock()
	return a.gdriveSession
}

func (a *App) handleGoogleDriveCallback(session *googleDriveAuthSession, w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if errValue := strings.TrimSpace(query.Get("error")); errValue != "" {
		message := query.Get("error_description")
		if strings.TrimSpace(message) == "" {
			message = errValue
		}
		session.setResult("error", message)
		writeGoogleDriveAuthPage(w, "Google Drive authorization cancelled", message)
		go session.close()
		return
	}

	if strings.TrimSpace(query.Get("state")) != session.stateValue() {
		session.setResult("error", "Google Drive returned an invalid authorization state.")
		writeGoogleDriveAuthPage(w, "Google Drive authorization failed", "The returned authorization state did not match the pending request.")
		go session.close()
		return
	}

	code := strings.TrimSpace(query.Get("code"))
	if code == "" {
		session.setResult("error", "Google Drive did not return an authorization code.")
		writeGoogleDriveAuthPage(w, "Google Drive authorization failed", "The browser callback did not include an authorization code.")
		go session.close()
		return
	}

	settings, err := a.svc.ExchangeGoogleDriveCode(code, session.redirectURI(), session.verifier())
	if err != nil {
		session.setResult("error", err.Error())
		writeGoogleDriveAuthPage(w, "Google Drive authorization failed", err.Error())
		go session.close()
		return
	}

	successMessage := "Google Drive connected. Backup folder is ready."
	if strings.TrimSpace(settings.Backup.GoogleDrive.FolderURL) == "" {
		successMessage = "Google Drive connected."
	}
	session.setResult("authorized", successMessage)
	writeGoogleDriveAuthPage(w, "Google Drive connected", "The Google Drive connection is ready. You can close this browser tab and return to SimklExpoGter.")
	go session.close()
}

func (s *deviceAuthSession) view() *DeviceAuthSessionView {
	if s == nil {
		return nil
	}

	return &DeviceAuthSessionView{
		UserCode:        s.UserCode,
		VerificationURL: s.VerificationURL,
		PinURL:          "https://simkl.com/pin/" + s.UserCode,
		IntervalSeconds: s.CurrentIntervalSeconds,
		ExpiresAt:       s.ExpiresAt.UTC().Format(time.RFC3339),
	}
}

func (s *deviceAuthSession) viewOrNil() *DeviceAuthSessionView {
	if s == nil {
		return nil
	}
	return s.view()
}

func (s *googleDriveAuthSession) view() *GoogleDriveAuthSessionView {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return &GoogleDriveAuthSessionView{
		AuthURL:     s.AuthURL,
		ExpiresAt:   s.ExpiresAt.UTC().Format(time.RFC3339),
		RedirectURI: s.RedirectURI,
	}
}

func (s *googleDriveAuthSession) viewOrNil() *GoogleDriveAuthSessionView {
	if s == nil {
		return nil
	}
	return s.view()
}

func (s *googleDriveAuthSession) status() (string, string, time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ResultState, s.Message, s.ExpiresAt
}

func (s *googleDriveAuthSession) stateValue() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.State
}

func (s *googleDriveAuthSession) redirectURI() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.RedirectURI
}

func (s *googleDriveAuthSession) verifier() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Verifier
}

func (s *googleDriveAuthSession) setResult(state, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ResultState = state
	s.Message = message
}

func (s *googleDriveAuthSession) close() {
	if s == nil {
		return
	}

	s.mu.Lock()
	server := s.Server
	listener := s.Listener
	s.Server = nil
	s.Listener = nil
	s.mu.Unlock()

	if server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = server.Shutdown(ctx)
		cancel()
	}
	if listener != nil {
		_ = listener.Close()
	}
}

func googleDriveFolderName(settings config.GoogleDriveSettings) string {
	if strings.TrimSpace(settings.FolderName) != "" {
		return strings.TrimSpace(settings.FolderName)
	}
	return "SimklExpoGter Backups"
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

func settingsDialogDirectory(configPath string) string {
	directory := strings.TrimSpace(filepath.Dir(configPath))
	if directory == "" || directory == "." {
		return ""
	}

	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() {
		return ""
	}

	return directory
}

func writeGoogleDriveAuthPage(w http.ResponseWriter, title, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(
		w,
		`<!doctype html><html><head><meta charset="utf-8"><title>%s</title></head><body style="font-family:Segoe UI,Arial,sans-serif;padding:32px;background:#101010;color:#f5f5f5"><h1 style="font-size:24px;margin:0 0 12px">%s</h1><p style="font-size:16px;line-height:1.5;max-width:42rem">%s</p></body></html>`,
		title,
		title,
		message,
	)
}

func mapScheduleState(state appsvc.ScheduleState) ScheduleStateView {
	return ScheduleStateView{
		Supported:                state.Supported,
		Enabled:                  state.Enabled,
		Installed:                state.Installed,
		Frequency:                state.Frequency,
		Time:                     state.Time,
		Days:                     append([]string(nil), state.Days...),
		OutputFormat:             state.OutputFormat,
		FieldMode:                state.FieldMode,
		Content:                  append([]string(nil), state.Content...),
		UseActivityCheck:         state.UseActivityCheck,
		MaxBackupAge:             state.MaxBackupAge,
		RunIfBackupIsStale:       state.RunIfBackupIsStale,
		LastSuccessfulBackupAt:   state.LastSuccessfulBackupAt,
		LastSuccessfulBackupKind: state.LastSuccessfulBackupKind,
		BackupFresh:              state.BackupFresh,
		BackupAgeSeconds:         state.BackupAgeSeconds,
		TaskName:                 state.TaskName,
		Status:                   state.Status,
		NextRunAt:                state.NextRunAt,
		LastRunAt:                state.LastRunAt,
		LastResult:               state.LastResult,
		Message:                  state.Message,
		UsesSavedOutput:          state.UsesSavedOutput,
		OutputDirectoryPreview:   state.OutputDirectoryPreview,
	}
}
