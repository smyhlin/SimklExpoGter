package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"SimklExpoGter/internal/config"
)

const DefaultCaption = "SimklExpoGter backup"

type UploadedFile struct {
	Name string
	URL  string
}

type UploadResult struct {
	ChatID  string
	Files   []UploadedFile
	Caption string
}

type Service struct {
	baseURL    string
	httpClient *http.Client
}

func NewService() *Service {
	return NewServiceWithClient("https://api.telegram.org", &http.Client{Timeout: 60 * time.Second})
}

func NewServiceWithClient(baseURL string, httpClient *http.Client) *Service {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Service{baseURL: strings.TrimRight(baseURL, "/"), httpClient: httpClient}
}

func (s *Service) UploadFiles(ctx context.Context, settings config.TelegramBotSettings, localPaths []string) (UploadResult, error) {
	botToken := strings.TrimSpace(settings.BotToken)
	chatID := strings.TrimSpace(settings.ChatID)
	if botToken == "" {
		return UploadResult{}, errors.New("missing Telegram bot token")
	}
	if chatID == "" {
		return UploadResult{}, errors.New("missing Telegram chat ID")
	}

	caption := strings.TrimSpace(settings.Caption)
	if caption == "" {
		caption = DefaultCaption
	}

	result := UploadResult{ChatID: chatID, Caption: caption, Files: make([]UploadedFile, 0, len(localPaths))}
	for _, localPath := range localPaths {
		localPath = strings.TrimSpace(localPath)
		if localPath == "" {
			continue
		}
		name, err := s.sendDocument(ctx, botToken, settings, localPath, caption)
		if err != nil {
			return UploadResult{}, err
		}
		result.Files = append(result.Files, UploadedFile{Name: name, URL: "https://t.me/" + chatID})
	}
	return result, nil
}

func (s *Service) sendDocument(ctx context.Context, botToken string, settings config.TelegramBotSettings, localPath string, caption string) (string, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("chat_id", strings.TrimSpace(settings.ChatID)); err != nil {
		return "", err
	}
	if threadID := strings.TrimSpace(settings.ThreadID); threadID != "" {
		if err := writer.WriteField("message_thread_id", threadID); err != nil {
			return "", err
		}
	}
	if caption != "" {
		if err := writer.WriteField("caption", caption); err != nil {
			return "", err
		}
	}

	part, err := writer.CreateFormFile("document", filepath.Base(localPath))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("%s/bot%s/sendDocument", s.baseURL, botToken)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())

	response, err := s.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	var payload struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || !payload.OK {
		description := strings.TrimSpace(payload.Description)
		if description == "" {
			description = response.Status
		}
		return "", fmt.Errorf("telegram upload failed: %s", description)
	}
	return filepath.Base(localPath), nil
}
