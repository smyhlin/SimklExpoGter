package exporter

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

const (
	FormatCSV  = "csv"
	FormatJSON = "json"
	FormatBoth = "both"

	FieldModeCompact = "compact"
	FieldModeAll     = "all"

	GroupingSingle   = "single-file"
	GroupingSeparate = "separate-files"
)

type Request struct {
	Types                []string `json:"types"`
	Statuses             []string `json:"statuses"`
	DateFrom             string   `json:"dateFrom"`
	Extended             string   `json:"extended"`
	EpisodeWatchedAt     bool     `json:"episodeWatchedAt"`
	IncludeMemos         bool     `json:"includeMemos"`
	IncludeNextWatchInfo bool     `json:"includeNextWatchInfo"`
	OutputFormat         string   `json:"outputFormat"`
	FieldMode            string   `json:"fieldMode"`
	Grouping             string   `json:"grouping"`
	IncludeEpisodeFiles  bool     `json:"includeEpisodeFiles"`
	UseActivityCheck     bool     `json:"useActivityCheck"`
	ExportDirectory      string   `json:"exportDirectory"`
	FilenamePrefix       string   `json:"filenamePrefix"`
}

type Result struct {
	ExportedAt        string         `json:"exportedAt"`
	OutputDirectory   string         `json:"outputDirectory"`
	StorageKind       string         `json:"storageKind,omitempty"`
	DestinationLabel  string         `json:"destinationLabel,omitempty"`
	DestinationURL    string         `json:"destinationUrl,omitempty"`
	Files             []ExportedFile `json:"files"`
	ItemCounts        map[string]int `json:"itemCounts"`
	Warnings          []string       `json:"warnings,omitempty"`
	ActivitiesChecked bool           `json:"activitiesChecked"`
	EffectiveDateFrom string         `json:"effectiveDateFrom,omitempty"`
}

type ExportedFile struct {
	Path        string `json:"path"`
	StorageKind string `json:"storageKind,omitempty"`
	Format      string `json:"format"`
	MediaType   string `json:"mediaType"`
	Kind        string `json:"kind"`
	Rows        int    `json:"rows"`
}

type Service struct{}

type jsonDocument struct {
	ExportedAt string                         `json:"exportedAt"`
	Request    Request                        `json:"request"`
	Counts     map[string]int                 `json:"counts"`
	Activities map[string]any                 `json:"activities,omitempty"`
	Items      map[string][]map[string]any    `json:"items"`
	Episodes   map[string][]map[string]string `json:"episodes,omitempty"`
}

func NewService() *Service {
	return &Service{}
}

func AllowedOutputFormats() []string {
	return []string{FormatCSV, FormatJSON, FormatBoth}
}

func AllowedFieldModes() []string {
	return []string{FieldModeCompact, FieldModeAll}
}

func AllowedGroupings() []string {
	return []string{GroupingSingle, GroupingSeparate}
}

func (s *Service) WriteSnapshot(request Request, payload map[string][]map[string]any, activities map[string]any) (Result, error) {
	if err := os.MkdirAll(request.ExportDirectory, 0o755); err != nil {
		return Result{}, err
	}

	now := time.Now().UTC()
	result := Result{
		ExportedAt:       now.Format(time.RFC3339),
		OutputDirectory:  request.ExportDirectory,
		StorageKind:      "local",
		DestinationLabel: request.ExportDirectory,
		ItemCounts:       map[string]int{},
		Files:            []ExportedFile{},
		Warnings:         []string{},
	}

	mediaTypes := requestedMediaTypes(request.Types)

	total := 0
	for _, mediaType := range []string{"movies", "shows", "anime"} {
		count := len(payload[mediaType])
		result.ItemCounts[mediaType] = count
		total += count
	}
	result.ItemCounts["all"] = total

	episodeRows := map[string][]map[string]string{}
	if request.IncludeEpisodeFiles {
		for _, mediaType := range mediaTypes {
			episodeRows[mediaType] = episodeRowsForItems(payload[mediaType], mediaType, request.FieldMode)
		}
	}

	if request.OutputFormat == FormatCSV || request.OutputFormat == FormatBoth {
		files, err := s.writeCSVOutputs(request, payload, episodeRows, mediaTypes, now)
		if err != nil {
			return Result{}, err
		}
		result.Files = append(result.Files, files...)
	}

	if request.OutputFormat == FormatJSON || request.OutputFormat == FormatBoth {
		files, err := s.writeJSONOutputs(request, payload, episodeRows, activities, result.ItemCounts, mediaTypes, now)
		if err != nil {
			return Result{}, err
		}
		result.Files = append(result.Files, files...)
	}

	if len(result.Files) == 0 {
		result.Warnings = append(result.Warnings, "No files were written because the export selection returned no items.")
	}

	return result, nil
}

func (s *Service) writeCSVOutputs(request Request, payload map[string][]map[string]any, episodeRows map[string][]map[string]string, mediaTypes []string, now time.Time) ([]ExportedFile, error) {
	files := make([]ExportedFile, 0)

	if request.Grouping == GroupingSingle {
		rows := make([]map[string]string, 0)
		for _, mediaType := range mediaTypes {
			rows = append(rows, rowsForItems(payload[mediaType], mediaType, request.FieldMode)...)
		}

		headers := headersForRows(rows, defaultHeaders(request.FieldMode, "items"))
		path := filepath.Join(request.ExportDirectory, buildFilename(request.FilenamePrefix, "all", "items", "csv", now))
		if err := writeCSV(path, headers, rows); err != nil {
			return nil, err
		}
		files = append(files, ExportedFile{
			Path:        path,
			StorageKind: "local",
			Format:      FormatCSV,
			MediaType:   "all",
			Kind:        "items",
			Rows:        len(rows),
		})

		if request.IncludeEpisodeFiles {
			rows = make([]map[string]string, 0)
			for _, mediaType := range mediaTypes {
				rows = append(rows, episodeRows[mediaType]...)
			}
			headers = headersForRows(rows, defaultHeaders(request.FieldMode, "episodes"))
			path = filepath.Join(request.ExportDirectory, buildFilename(request.FilenamePrefix, "all", "episodes", "csv", now))
			if err := writeCSV(path, headers, rows); err != nil {
				return nil, err
			}
			files = append(files, ExportedFile{
				Path:        path,
				StorageKind: "local",
				Format:      FormatCSV,
				MediaType:   "all",
				Kind:        "episodes",
				Rows:        len(rows),
			})
		}

		return files, nil
	}

	for _, mediaType := range mediaTypes {
		rows := rowsForItems(payload[mediaType], mediaType, request.FieldMode)
		headers := headersForRows(rows, defaultHeaders(request.FieldMode, "items"))
		path := filepath.Join(request.ExportDirectory, buildFilename(request.FilenamePrefix, mediaType, "items", "csv", now))
		if err := writeCSV(path, headers, rows); err != nil {
			return nil, err
		}
		files = append(files, ExportedFile{
			Path:        path,
			StorageKind: "local",
			Format:      FormatCSV,
			MediaType:   mediaType,
			Kind:        "items",
			Rows:        len(rows),
		})

		if request.IncludeEpisodeFiles {
			rows = episodeRows[mediaType]
			headers = headersForRows(rows, defaultHeaders(request.FieldMode, "episodes"))
			path = filepath.Join(request.ExportDirectory, buildFilename(request.FilenamePrefix, mediaType, "episodes", "csv", now))
			if err := writeCSV(path, headers, rows); err != nil {
				return nil, err
			}
			files = append(files, ExportedFile{
				Path:        path,
				StorageKind: "local",
				Format:      FormatCSV,
				MediaType:   mediaType,
				Kind:        "episodes",
				Rows:        len(rows),
			})
		}
	}

	return files, nil
}

func (s *Service) writeJSONOutputs(request Request, payload map[string][]map[string]any, episodeRows map[string][]map[string]string, activities map[string]any, counts map[string]int, mediaTypes []string, now time.Time) ([]ExportedFile, error) {
	files := make([]ExportedFile, 0)

	if request.Grouping == GroupingSingle {
		items := map[string][]map[string]any{}
		for _, mediaType := range mediaTypes {
			items[mediaType] = payload[mediaType]
		}
		document := jsonDocument{
			ExportedAt: now.Format(time.RFC3339),
			Request:    request,
			Counts:     counts,
			Activities: activities,
			Items:      items,
		}
		if request.IncludeEpisodeFiles {
			document.Episodes = map[string][]map[string]string{}
			for _, mediaType := range mediaTypes {
				document.Episodes[mediaType] = episodeRows[mediaType]
			}
		}

		path := filepath.Join(request.ExportDirectory, buildFilename(request.FilenamePrefix, "all", "snapshot", "json", now))
		if err := writeJSON(path, document); err != nil {
			return nil, err
		}
		files = append(files, ExportedFile{
			Path:        path,
			StorageKind: "local",
			Format:      FormatJSON,
			MediaType:   "all",
			Kind:        "snapshot",
			Rows:        counts["all"],
		})
		return files, nil
	}

	for _, mediaType := range mediaTypes {
		document := jsonDocument{
			ExportedAt: now.Format(time.RFC3339),
			Request:    request,
			Counts: map[string]int{
				mediaType: len(payload[mediaType]),
			},
			Activities: activities,
			Items: map[string][]map[string]any{
				mediaType: payload[mediaType],
			},
		}
		if request.IncludeEpisodeFiles {
			document.Episodes = map[string][]map[string]string{
				mediaType: episodeRows[mediaType],
			}
		}

		path := filepath.Join(request.ExportDirectory, buildFilename(request.FilenamePrefix, mediaType, "snapshot", "json", now))
		if err := writeJSON(path, document); err != nil {
			return nil, err
		}
		files = append(files, ExportedFile{
			Path:        path,
			StorageKind: "local",
			Format:      FormatJSON,
			MediaType:   mediaType,
			Kind:        "snapshot",
			Rows:        len(payload[mediaType]),
		})
	}

	return files, nil
}

func requestedMediaTypes(selected []string) []string {
	if len(selected) == 0 {
		return []string{"movies", "shows", "anime"}
	}
	return append([]string(nil), selected...)
}

func writeCSV(path string, headers []string, rows []map[string]string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err := writer.Write(headers); err != nil {
		return err
	}

	for _, row := range rows {
		record := make([]string, 0, len(headers))
		for _, header := range headers {
			record = append(record, row[header])
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func buildFilename(prefix, mediaType, kind, extension string, now time.Time) string {
	base := sanitizePart(prefix)
	if base == "" {
		base = "simkl-export"
	}
	stamp := now.Format("20060102-150405")
	parts := []string{base, sanitizePart(mediaType), sanitizePart(kind), stamp}
	return strings.Join(parts, "-") + "." + extension
}

func sanitizePart(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}

	var builder strings.Builder
	lastDash := false
	for _, char := range value {
		switch {
		case unicode.IsLetter(char), unicode.IsDigit(char):
			builder.WriteRune(char)
			lastDash = false
		case char == '-', char == '_':
			builder.WriteRune(char)
			lastDash = false
		default:
			if lastDash {
				continue
			}
			builder.WriteByte('-')
			lastDash = true
		}
	}

	clean := strings.Trim(builder.String(), "-_")
	if clean == "" {
		return "export"
	}
	return clean
}
