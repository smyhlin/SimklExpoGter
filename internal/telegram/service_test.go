package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"SimklExpoGter/internal/config"
)

func TestUploadFilesSendsDocument(t *testing.T) {
	var gotPath string
	var gotChatID string
	var gotThreadID string
	var gotCaption string
	var gotFilename string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm returned error: %v", err)
		}
		gotChatID = r.FormValue("chat_id")
		gotThreadID = r.FormValue("message_thread_id")
		gotCaption = r.FormValue("caption")
		file, header, err := r.FormFile("document")
		if err != nil {
			t.Fatalf("FormFile returned error: %v", err)
		}
		_ = file.Close()
		gotFilename = header.Filename
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "backup.json")
	if err := os.WriteFile(path, []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	service := NewServiceWithClient(server.URL, server.Client())
	result, err := service.UploadFiles(context.Background(), config.TelegramBotSettings{
		BotToken: "bot-token",
		ChatID:   "-100123",
		ThreadID: "42",
		Caption:  "daily backup",
	}, []string{path})
	if err != nil {
		t.Fatalf("UploadFiles returned error: %v", err)
	}

	if gotPath != "/botbot-token/sendDocument" {
		t.Fatalf("path = %q, want %q", gotPath, "/botbot-token/sendDocument")
	}
	if gotChatID != "-100123" || gotThreadID != "42" || gotCaption != "daily backup" || gotFilename != "backup.json" {
		t.Fatalf("unexpected multipart fields chat=%q thread=%q caption=%q filename=%q", gotChatID, gotThreadID, gotCaption, gotFilename)
	}
	if len(result.Files) != 1 || result.Files[0].Name != "backup.json" {
		t.Fatalf("unexpected upload result: %#v", result)
	}
}

func TestUploadFilesDoesNotLeakTokenInAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "description": "chat not found"})
	}))
	defer server.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "backup.json")
	if err := os.WriteFile(path, []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	service := NewServiceWithClient(server.URL, server.Client())
	_, err := service.UploadFiles(context.Background(), config.TelegramBotSettings{
		BotToken: "secret-token",
		ChatID:   "-100123",
	}, []string{path})
	if err == nil {
		t.Fatal("expected upload error")
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("error leaked bot token: %v", err)
	}
}
