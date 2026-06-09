package gdrive

import (
	"path/filepath"
	"testing"
	"time"
)

func TestBuildUploadFolderNameUsesExportTimestamp(t *testing.T) {
	got := buildUploadFolderName([]string{
		filepath.Join("tmp", "simkl-full-backup-shows-items-20260409-101112.csv"),
	}, time.Date(2026, time.April, 9, 12, 13, 14, 0, time.UTC))

	if want := "backup-20260409-101112"; got != want {
		t.Fatalf("expected upload folder name %q, got %q", want, got)
	}
}

func TestBuildUploadFolderNameFallsBackToCurrentTime(t *testing.T) {
	got := buildUploadFolderName([]string{
		filepath.Join("tmp", "backup.csv"),
	}, time.Date(2026, time.April, 9, 12, 13, 14, 0, time.UTC))

	if want := "backup-20260409-121314"; got != want {
		t.Fatalf("expected fallback upload folder name %q, got %q", want, got)
	}
}
