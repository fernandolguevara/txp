package storage

import (
	"database/sql"
	"fmt"

	"txp/internal/model"
)

func LoadQueue(db *sql.DB, scope string) ([]model.Track, error) {
	rows, err := db.Query(
		"SELECT q.path, t.title, t.artist, t.album, t.genre, t.year, t.track_no, t.duration, t.mtime, t.key, t.bpm, t.favorite "+
			"FROM queue q LEFT JOIN tracks t ON t.path = q.path WHERE q.scope = ? ORDER BY q.position",
		scope,
	)
	if err != nil {
		return nil, fmt.Errorf("load queue: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	tracks := []model.Track{}
	for rows.Next() {
		var path string
		var title sql.NullString
		var artist sql.NullString
		var album sql.NullString
		var genre sql.NullString
		var year sql.NullInt64
		var trackNo sql.NullInt64
		var duration sql.NullFloat64
		var mtime sql.NullInt64
		var key sql.NullString
		var bpm sql.NullFloat64
		var favorite sql.NullInt64
		if err := rows.Scan(&path, &title, &artist, &album, &genre, &year, &trackNo, &duration, &mtime, &key, &bpm, &favorite); err != nil {
			return nil, fmt.Errorf("load queue: %w", err)
		}
		track := model.Track{Path: path}
		if title.Valid {
			track.Title = title.String
		}
		if artist.Valid {
			track.Artist = artist.String
		}
		if album.Valid {
			track.Album = album.String
		}
		if genre.Valid {
			track.Genre = genre.String
		}
		if year.Valid {
			track.Year = int(year.Int64)
		}
		if trackNo.Valid {
			track.TrackNum = int(trackNo.Int64)
		}
		if duration.Valid {
			track.Duration = duration.Float64
		}
		if mtime.Valid {
			track.Mtime = mtime.Int64
		}
		if key.Valid {
			track.Key = key.String
		}
		if bpm.Valid {
			track.BPM = bpm.Float64
		}
		if favorite.Valid {
			track.Favorite = favorite.Int64 != 0
		}
		tracks = append(tracks, track)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load queue: %w", err)
	}
	return tracks, nil
}

func ReplaceQueue(db *sql.DB, scope string, tracks []model.Track) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("save queue: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM queue WHERE scope = ?", scope); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("save queue: %w", err)
	}

	stmt, err := tx.Prepare("INSERT INTO queue (scope, position, path) VALUES (?, ?, ?)")
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("save queue: %w", err)
	}
	defer func() {
		_ = stmt.Close()
	}()

	position := 0
	for _, track := range tracks {
		if track.Path == "" {
			continue
		}
		if _, err := stmt.Exec(scope, position, track.Path); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("save queue: %w", err)
		}
		position++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("save queue: %w", err)
	}
	return nil
}
