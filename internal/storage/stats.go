package storage

import (
	"database/sql"
	"encoding/json"
	"time"
)

const statsKeyTotals = "library_totals"

type StatsTotals struct {
	TotalTracks   int     `json:"totalTracks"`
	TotalDuration float64 `json:"totalDuration"`
	TotalPlays    int     `json:"totalPlays"`
	Artists       int     `json:"artists"`
	Albums        int     `json:"albums"`
}

func ComputeTotals(db *sql.DB) (StatsTotals, error) {
	var totals StatsTotals
	row := db.QueryRow(
		"SELECT COUNT(*), COALESCE(SUM(duration), 0), COUNT(DISTINCT NULLIF(artist, '')), COUNT(DISTINCT NULLIF(album, '')) FROM tracks",
	)
	if err := row.Scan(&totals.TotalTracks, &totals.TotalDuration, &totals.Artists, &totals.Albums); err != nil {
		return totals, err
	}
	row = db.QueryRow("SELECT COALESCE(SUM(play_count), 0) FROM track_stats")
	if err := row.Scan(&totals.TotalPlays); err != nil {
		return totals, err
	}
	return totals, nil
}

func ComputeTotalsInLibraries(db *sql.DB, roots []string) (StatsTotals, error) {
	var totals StatsTotals
	where, args, empty := buildLibraryPathFilter(roots)
	if empty {
		return totals, nil
	}
	query := "SELECT COUNT(*), COALESCE(SUM(duration), 0), COUNT(DISTINCT NULLIF(artist, '')), COUNT(DISTINCT NULLIF(album, '')) FROM tracks WHERE " + where
	row := db.QueryRow(query, args...)
	if err := row.Scan(&totals.TotalTracks, &totals.TotalDuration, &totals.Artists, &totals.Albums); err != nil {
		return totals, err
	}
	joinWhere, joinArgs, _ := buildLibraryPathFilterForColumn(roots, "t.path")
	query = "SELECT COALESCE(SUM(ts.play_count), 0) FROM track_stats ts JOIN tracks t ON t.path = ts.path WHERE " + joinWhere
	row = db.QueryRow(query, joinArgs...)
	if err := row.Scan(&totals.TotalPlays); err != nil {
		return totals, err
	}
	return totals, nil
}

func GetTotals(db *sql.DB) (StatsTotals, bool, error) {
	row := db.QueryRow("SELECT value_json FROM stats_cache WHERE key = ?", statsKeyTotals)
	var payload string
	if err := row.Scan(&payload); err != nil {
		if err == sql.ErrNoRows {
			return StatsTotals{}, false, nil
		}
		return StatsTotals{}, false, err
	}

	var totals StatsTotals
	if err := json.Unmarshal([]byte(payload), &totals); err != nil {
		return StatsTotals{}, false, err
	}

	return totals, true, nil
}

func SaveTotals(db *sql.DB, totals StatsTotals) error {
	payload, err := json.Marshal(totals)
	if err != nil {
		return err
	}

	_, err = db.Exec(
		"INSERT OR REPLACE INTO stats_cache (key, value_json, updated_at) VALUES (?, ?, ?)",
		statsKeyTotals,
		string(payload),
		time.Now().Unix(),
	)
	return err
}
