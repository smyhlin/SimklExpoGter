package cli

import (
	"bytes"
	"testing"

	"SimklExpoGter/internal/appsvc"
	"SimklExpoGter/internal/config"
	"SimklExpoGter/internal/exporter"
	"SimklExpoGter/internal/simkl"
)

func TestIsCommandRecognizesKnownCommands(t *testing.T) {
	for _, name := range []string{"help", "run", "config", "auth", "schedule", "tui", "-h", "--help"} {
		if !IsCommand(name) {
			t.Fatalf("expected %q to be recognized as a CLI command", name)
		}
	}

	if IsCommand("unknown") {
		t.Fatal("expected unknown command to be rejected")
	}
}

func TestRunUsesModeAndContentAliases(t *testing.T) {
	service := &stubService{
		runResult: exporter.Result{
			OutputDirectory: "C:\\exports",
			ExportedAt:      "2026-04-07T10:00:00Z",
			ItemCounts:      map[string]int{"all": 0},
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"run", "--mode", "compact", "--content", "anime,movie,series"}, &stdout, &stderr, service)
	if code != ExitSuccess {
		t.Fatalf("expected success, got %d (%s)", code, stderr.String())
	}

	if service.lastRequest.FieldMode != "compact" {
		t.Fatalf("expected field mode compact, got %q", service.lastRequest.FieldMode)
	}
	if got := service.lastRequest.Types; len(got) != 3 || got[0] != "anime" || got[1] != "movies" || got[2] != "shows" {
		t.Fatalf("unexpected normalized types: %#v", got)
	}
	if !service.lastRequest.IncludeEpisodeFiles || !service.lastRequest.IncludeMemos || !service.lastRequest.IncludeNextWatchInfo || !service.lastRequest.EpisodeWatchedAt {
		t.Fatal("expected backup-oriented defaults to be enabled")
	}
	if service.lastRequest.UseActivityCheck {
		t.Fatal("expected activity check to default to false")
	}
}

func TestRunRejectsMismatchedFieldModeAliases(t *testing.T) {
	service := &stubService{}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"run", "--mode", "all", "--field-mode", "compact"}, &stdout, &stderr, service)
	if code != ExitUsageError {
		t.Fatalf("expected usage error, got %d", code)
	}
}

func TestRunRejectsUnsupportedContentValue(t *testing.T) {
	service := &stubService{}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"run", "--content", "books"}, &stdout, &stderr, service)
	if code != ExitUsageError {
		t.Fatalf("expected usage error, got %d", code)
	}
}

func TestConfigSetSupportsSecretAlias(t *testing.T) {
	service := &stubService{}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"config", "set", "--client-id", "client-1", "--secret", "secret-1"}, &stdout, &stderr, service)
	if code != ExitSuccess {
		t.Fatalf("expected success, got %d (%s)", code, stderr.String())
	}

	if !service.savedInput.SetClientID || !service.savedInput.SetClientSecret {
		t.Fatalf("expected client ID and secret to be marked as set: %+v", service.savedInput)
	}
	if service.savedInput.ClientSecret != "secret-1" {
		t.Fatalf("expected secret alias to populate client secret, got %q", service.savedInput.ClientSecret)
	}
}

func TestConfigSetSupportsTelegramBackupStorage(t *testing.T) {
	service := &stubService{}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"config", "set",
		"--backup-storage", "telegram",
		"--telegram-bot-token", "token-1",
		"--telegram-chat-id", "-100123",
		"--telegram-thread-id", "42",
		"--telegram-caption", "daily backup",
	}, &stdout, &stderr, service)
	if code != ExitSuccess {
		t.Fatalf("expected success, got %d (%s)", code, stderr.String())
	}

	if !service.savedInput.SetBackupStorage || service.savedInput.BackupStorage != "telegram" {
		t.Fatalf("expected telegram backup storage to be saved: %+v", service.savedInput)
	}
	if !service.savedInput.SetTelegramBotToken || service.savedInput.TelegramBotToken != "token-1" {
		t.Fatalf("expected telegram bot token to be saved: %+v", service.savedInput)
	}
	if !service.savedInput.SetTelegramChatID || service.savedInput.TelegramChatID != "-100123" {
		t.Fatalf("expected telegram chat ID to be saved: %+v", service.savedInput)
	}
	if !service.savedInput.SetTelegramThreadID || service.savedInput.TelegramThreadID != "42" {
		t.Fatalf("expected telegram thread ID to be saved: %+v", service.savedInput)
	}
	if !service.savedInput.SetTelegramCaption || service.savedInput.TelegramCaption != "daily backup" {
		t.Fatalf("expected telegram caption to be saved: %+v", service.savedInput)
	}
}

type stubService struct {
	savedInput  appsvc.SaveSettingsInput
	lastRequest exporter.Request
	runResult   exporter.Result
	runErr      error
}

func (s *stubService) Path() string {
	return "C:\\config\\settings.json"
}

func (s *stubService) ConfigSummary() (appsvc.ConfigSummary, error) {
	return appsvc.ConfigSummary{}, nil
}

func (s *stubService) AuthSummary() (appsvc.AuthSummary, error) {
	return appsvc.AuthSummary{}, nil
}

func (s *stubService) SaveSettings(input appsvc.SaveSettingsInput) (appsvc.SaveSettingsResult, error) {
	s.savedInput = input
	return appsvc.SaveSettingsResult{}, nil
}

func (s *stubService) RequestDeviceCode() (simkl.DeviceCodeResponse, error) {
	return simkl.DeviceCodeResponse{
		UserCode:        "ABCDE",
		VerificationURL: "https://simkl.com/pin/",
		ExpiresIn:       1,
		Interval:        1,
	}, nil
}

func (s *stubService) PollDeviceCode(string) (simkl.DeviceCodeStatusResponse, error) {
	return simkl.DeviceCodeStatusResponse{
		Result:      "OK",
		AccessToken: "token-1",
	}, nil
}

func (s *stubService) SaveAccessToken(string) (config.Settings, error) {
	return config.Settings{}, nil
}

func (s *stubService) StandardAuthURL() (string, error) {
	return "https://simkl.com/oauth/authorize", nil
}

func (s *stubService) ExchangeOAuthCode(string) (config.Settings, error) {
	return config.Settings{}, nil
}

func (s *stubService) ClearAccessToken() (config.Settings, error) {
	return config.Settings{}, nil
}

func (s *stubService) RunExport(request exporter.Request) (exporter.Result, error) {
	s.lastRequest = request
	return s.runResult, s.runErr
}

func (s *stubService) RunScheduledExport(request exporter.Request, maxBackupAge string) (appsvc.ScheduledExportResult, error) {
	s.lastRequest = request
	return appsvc.ScheduledExportResult{
		Ran:          true,
		MaxBackupAge: maxBackupAge,
		Result:       s.runResult,
	}, s.runErr
}

func (s *stubService) ScheduleState() (appsvc.ScheduleState, error) {
	return appsvc.ScheduleState{}, nil
}

func (s *stubService) SaveSchedule(input appsvc.ScheduleSettingsInput) (config.Settings, appsvc.ScheduleState, error) {
	return config.Settings{}, appsvc.ScheduleState{}, nil
}
