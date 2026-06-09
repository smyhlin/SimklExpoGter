package exporter

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var compactHeaders = []string{
	"media_type",
	"title",
	"original_title",
	"year",
	"runtime",
	"status",
	"user_rating",
	"user_rated_at",
	"added_to_watchlist_at",
	"last_watched_at",
	"last_watched",
	"next_to_watch",
	"watched_episodes_count",
	"total_episodes_count",
	"not_aired_episodes_count",
	"anime_type",
	"poster",
	"ids.simkl",
	"ids.slug",
	"ids.imdb",
	"ids.tmdb",
	"ids.tvdb",
	"ids.mal",
	"ids.anidb",
	"memo.text",
	"memo.is_private",
}

var compactEpisodeHeaders = []string{
	"media_type",
	"title",
	"status",
	"season_number",
	"episode_number",
	"episode_watched_at",
	"tvdb.season",
	"tvdb.episode",
	"ids.simkl",
	"ids.imdb",
	"ids.tmdb",
	"ids.tvdb",
	"ids.mal",
	"ids.anidb",
}

func rowsForItems(items []map[string]any, mediaType, fieldMode string) []map[string]string {
	rows := make([]map[string]string, 0, len(items))
	for _, item := range items {
		if fieldMode == FieldModeCompact {
			rows = append(rows, compactItemRow(item, mediaType))
			continue
		}
		rows = append(rows, allFieldsItemRow(item, mediaType))
	}
	return rows
}

func episodeRowsForItems(items []map[string]any, mediaType, fieldMode string) []map[string]string {
	rows := make([]map[string]string, 0)
	for _, item := range items {
		seasons := mapSlice(item["seasons"])
		if len(seasons) == 0 {
			continue
		}

		media := mediaObject(item, mediaType)
		parentTitle := firstNonEmpty(stringValue(media["en_title"]), stringValue(media["title"]))
		parentIDs := mapValue(media["ids"])

		for _, season := range seasons {
			episodes := mapSlice(season["episodes"])
			for _, episode := range episodes {
				if fieldMode == FieldModeCompact {
					row := map[string]string{
						"media_type":         mediaType,
						"title":              parentTitle,
						"status":             stringValue(item["status"]),
						"season_number":      stringValue(season["number"]),
						"episode_number":     stringValue(episode["number"]),
						"episode_watched_at": stringValue(episode["watched_at"]),
						"tvdb.season":        nestedString(episode, "tvdb", "season"),
						"tvdb.episode":       nestedString(episode, "tvdb", "episode"),
						"ids.simkl":          stringValue(parentIDs["simkl"]),
						"ids.imdb":           stringValue(parentIDs["imdb"]),
						"ids.tmdb":           stringValue(parentIDs["tmdb"]),
						"ids.tvdb":           stringValue(parentIDs["tvdb"]),
						"ids.mal":            stringValue(parentIDs["mal"]),
						"ids.anidb":          stringValue(parentIDs["anidb"]),
					}
					rows = append(rows, row)
					continue
				}

				row := map[string]string{
					"media_type":   mediaType,
					"parent.title": parentTitle,
				}
				parent := cloneMap(item)
				delete(parent, "seasons")
				flattenInto("parent", parent, row)
				flattenInto("season", map[string]any{"number": season["number"]}, row)
				flattenInto("episode", episode, row)
				rows = append(rows, row)
			}
		}
	}
	return rows
}

func defaultHeaders(fieldMode, kind string) []string {
	if kind == "episodes" {
		return append([]string(nil), compactEpisodeHeaders...)
	}
	if fieldMode == FieldModeCompact {
		return append([]string(nil), compactHeaders...)
	}
	return []string{"media_type"}
}

func headersForRows(rows []map[string]string, fallback []string) []string {
	if len(rows) == 0 {
		return append([]string(nil), fallback...)
	}

	headerSet := map[string]struct{}{}
	headers := make([]string, 0, len(fallback))
	for _, header := range fallback {
		headerSet[header] = struct{}{}
		headers = append(headers, header)
	}

	extra := make([]string, 0)
	for _, row := range rows {
		for key := range row {
			if _, ok := headerSet[key]; ok {
				continue
			}
			headerSet[key] = struct{}{}
			extra = append(extra, key)
		}
	}
	sort.Strings(extra)
	headers = append(headers, extra...)
	return headers
}

func compactItemRow(item map[string]any, mediaType string) map[string]string {
	media := mediaObject(item, mediaType)
	ids := mapValue(media["ids"])

	title := firstNonEmpty(stringValue(media["en_title"]), stringValue(media["title"]))
	original := stringValue(media["title"])
	if original == title {
		original = ""
	}

	row := map[string]string{
		"media_type":               mediaType,
		"title":                    title,
		"original_title":           original,
		"year":                     stringValue(media["year"]),
		"runtime":                  stringValue(media["runtime"]),
		"status":                   stringValue(item["status"]),
		"user_rating":              stringValue(item["user_rating"]),
		"user_rated_at":            stringValue(item["user_rated_at"]),
		"added_to_watchlist_at":    stringValue(item["added_to_watchlist_at"]),
		"last_watched_at":          stringValue(item["last_watched_at"]),
		"last_watched":             stringValue(item["last_watched"]),
		"next_to_watch":            stringValue(item["next_to_watch"]),
		"watched_episodes_count":   stringValue(item["watched_episodes_count"]),
		"total_episodes_count":     stringValue(item["total_episodes_count"]),
		"not_aired_episodes_count": stringValue(item["not_aired_episodes_count"]),
		"anime_type":               stringValue(item["anime_type"]),
		"poster":                   stringValue(media["poster"]),
		"ids.simkl":                stringValue(ids["simkl"]),
		"ids.slug":                 stringValue(ids["slug"]),
		"ids.imdb":                 stringValue(ids["imdb"]),
		"ids.tmdb":                 stringValue(ids["tmdb"]),
		"ids.tvdb":                 stringValue(ids["tvdb"]),
		"ids.mal":                  stringValue(ids["mal"]),
		"ids.anidb":                stringValue(ids["anidb"]),
	}

	if memo := mapValue(item["memo"]); len(memo) > 0 {
		row["memo.text"] = stringValue(memo["text"])
		row["memo.is_private"] = stringValue(memo["is_private"])
	}

	return row
}

func allFieldsItemRow(item map[string]any, mediaType string) map[string]string {
	row := map[string]string{
		"media_type": mediaType,
	}

	media := mediaObject(item, mediaType)
	row["media.title"] = firstNonEmpty(stringValue(media["en_title"]), stringValue(media["title"]))
	row["media.original_title"] = stringValue(media["title"])

	flattenInto("", item, row)
	return row
}

func flattenInto(prefix string, value any, row map[string]string) {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			childPrefix := key
			if prefix != "" {
				childPrefix = prefix + "." + key
			}
			flattenInto(childPrefix, typed[key], row)
		}
	case []any:
		if prefix != "" {
			row[prefix] = marshalString(typed)
		}
	case string:
		if prefix != "" {
			row[prefix] = typed
		}
	case float64:
		if prefix != "" {
			row[prefix] = formatNumber(typed)
		}
	case bool:
		if prefix != "" {
			row[prefix] = strconv.FormatBool(typed)
		}
	case nil:
		if prefix != "" {
			row[prefix] = ""
		}
	default:
		if prefix != "" {
			row[prefix] = fmt.Sprint(typed)
		}
	}
}

func mediaObject(item map[string]any, mediaType string) map[string]any {
	switch mediaType {
	case "movies":
		return mapValue(item["movie"])
	default:
		return mapValue(item["show"])
	}
}

func nestedString(value map[string]any, path ...string) string {
	current := value
	for index, key := range path {
		raw, ok := current[key]
		if !ok {
			return ""
		}
		if index == len(path)-1 {
			return stringValue(raw)
		}
		next, ok := raw.(map[string]any)
		if !ok {
			return ""
		}
		current = next
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		return formatNumber(typed)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return fmt.Sprint(typed)
	}
}

func formatNumber(value float64) string {
	if value == float64(int64(value)) {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func marshalString(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func mapValue(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func mapSlice(value any) []map[string]any {
	list, ok := value.([]any)
	if !ok {
		return nil
	}

	rows := make([]map[string]any, 0, len(list))
	for _, item := range list {
		if typed, ok := item.(map[string]any); ok {
			rows = append(rows, typed)
		}
	}
	return rows
}

func cloneMap(source map[string]any) map[string]any {
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
