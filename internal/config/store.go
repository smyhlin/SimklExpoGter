package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	BackupStorageLocal  = "local"
	BackupStorageGDrive = "gdrive"
)

type Settings struct {
	ClientID                 string           `json:"clientId"`
	ClientSecret             string           `json:"clientSecret,omitempty"`
	AccessToken              string           `json:"accessToken,omitempty"`
	ExportDirectory          string           `json:"exportDirectory,omitempty"`
	Backup                   BackupSettings   `json:"backup,omitempty"`
	LastActivities           map[string]any   `json:"lastActivities,omitempty"`
	LastSuccessfulBackupAt   time.Time        `json:"lastSuccessfulBackupAt,omitempty"`
	LastSuccessfulBackupKind string           `json:"lastSuccessfulBackupKind,omitempty"`
	Schedule                 ScheduleSettings `json:"schedule,omitempty"`
	UpdatedAt                time.Time        `json:"updatedAt,omitempty"`
}

type BackupSettings struct {
	StorageKind string              `json:"storageKind,omitempty"`
	GoogleDrive GoogleDriveSettings `json:"googleDrive,omitempty"`
}

type GoogleDriveSettings struct {
	ClientID     string     `json:"clientId,omitempty"`
	ClientSecret string     `json:"clientSecret,omitempty"`
	Token        OAuthToken `json:"token,omitempty"`
	FolderID     string     `json:"folderId,omitempty"`
	FolderName   string     `json:"folderName,omitempty"`
	FolderURL    string     `json:"folderUrl,omitempty"`
}

type OAuthToken struct {
	AccessToken  string    `json:"accessToken,omitempty"`
	TokenType    string    `json:"tokenType,omitempty"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
}

type ScheduleSettings struct {
	Enabled            bool     `json:"enabled,omitempty"`
	Frequency          string   `json:"frequency,omitempty"`
	Time               string   `json:"time,omitempty"`
	Days               []string `json:"days,omitempty"`
	OutputFormat       string   `json:"outputFormat,omitempty"`
	FieldMode          string   `json:"fieldMode,omitempty"`
	Content            []string `json:"content,omitempty"`
	UseActivityCheck   bool     `json:"useActivityCheck,omitempty"`
	MaxBackupAge       string   `json:"maxBackupAge,omitempty"`
	RunIfBackupIsStale bool     `json:"runIfBackupIsStale"`
}

type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(appName string) (*Store, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	return NewStoreAtPath(filepath.Join(configDir, appName, "settings.json")), nil
}

func NewStoreAtPath(path string) *Store {
	return &Store{
		path: path,
	}
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Load() (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.loadLocked()
}

func (s *Store) Save(settings Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	settings.UpdatedAt = time.Now().UTC()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0o600)
}

func (s *Store) loadLocked() (Settings, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return Settings{}, nil
		}
		return Settings{}, err
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, err
	}

	if settings.LastActivities == nil {
		settings.LastActivities = map[string]any{}
	}
	if settings.Backup.StorageKind == "" {
		settings.Backup.StorageKind = BackupStorageLocal
	}
	if settings.Schedule.Enabled && settings.Schedule.MaxBackupAge == "" {
		settings.Schedule.MaxBackupAge = "24h"
		settings.Schedule.RunIfBackupIsStale = true
	}

	return settings, nil
}
