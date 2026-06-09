package exporter

import "testing"

func TestAllFieldsRowIncludesFlattenedMediaFields(t *testing.T) {
	item := map[string]any{
		"status": "completed",
		"show": map[string]any{
			"title":    "Original Title",
			"en_title": "English Title",
			"ids": map[string]any{
				"simkl": 1234.0,
				"imdb":  "tt1234567",
			},
		},
	}

	row := allFieldsItemRow(item, "shows")

	if got := row["media.title"]; got != "English Title" {
		t.Fatalf("expected normalized media title, got %q", got)
	}
	if got := row["show.ids.simkl"]; got != "1234" {
		t.Fatalf("expected flattened show id, got %q", got)
	}
}

func TestEpisodeRowsCompactIncludesEpisodeMetadata(t *testing.T) {
	items := []map[string]any{
		{
			"status": "watching",
			"show": map[string]any{
				"title": "Series",
				"ids": map[string]any{
					"simkl": 42.0,
				},
			},
			"seasons": []any{
				map[string]any{
					"number": 2.0,
					"episodes": []any{
						map[string]any{
							"number":     5.0,
							"watched_at": "2026-04-06T12:00:00Z",
							"tvdb": map[string]any{
								"season":  7.0,
								"episode": 8.0,
							},
						},
					},
				},
			},
		},
	}

	rows := episodeRowsForItems(items, "shows", FieldModeCompact)
	if len(rows) != 1 {
		t.Fatalf("expected 1 episode row, got %d", len(rows))
	}

	row := rows[0]
	if row["season_number"] != "2" || row["episode_number"] != "5" {
		t.Fatalf("unexpected season/episode numbers: %+v", row)
	}
	if row["episode_watched_at"] != "2026-04-06T12:00:00Z" {
		t.Fatalf("unexpected episode watched timestamp: %+v", row)
	}
	if row["tvdb.season"] != "7" || row["tvdb.episode"] != "8" {
		t.Fatalf("unexpected tvdb mapping: %+v", row)
	}
}
