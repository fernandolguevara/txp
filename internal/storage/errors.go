package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func UpsertTrackError(db *sql.DB, path string, message string) error {
	if path == "" {
		return nil
	}
	_, err := db.Exec(
		"INSERT INTO track_errors (path, last_error, updated_at) VALUES (?, ?, ?) ON CONFLICT(path) DO UPDATE SET last_error=excluded.last_error, updated_at=excluded.updated_at",
		path,
		strings.TrimSpace(message),
		time.Now().Unix(),
	)
	return err
}

func ClearTrackError(db *sql.DB, path string) error {
	if path == "" {
		return nil
	}
	_, err := db.Exec("DELETE FROM track_errors WHERE path = ?", path)
	return err
}

func ListTrackErrors(db *sql.DB, paths []string) (map[string]string, error) {
	result := map[string]string{}
	if len(paths) == 0 {
		return result, nil
	}
	placeholders := make([]string, 0, len(paths))
	args := make([]any, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		placeholders = append(placeholders, "?")
		args = append(args, path)
	}
	if len(placeholders) == 0 {
		return result, nil
	}
	query := fmt.Sprintf("SELECT path, last_error FROM track_errors WHERE path IN (%s)", strings.Join(placeholders, ","))
	rows, err := db.Query(query, args...)
	if err != nil {
		return result, err
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		var path string
		var message sql.NullString
		if err := rows.Scan(&path, &message); err != nil {
			return result, err
		}
		if path == "" {
			continue
		}
		if message.Valid {
			result[path] = message.String
		} else {
			result[path] = ""
		}
	}
	return result, rows.Err()
}

func GetTrackError(db *sql.DB, path string) (string, error) {
	if path == "" {
		return "", nil
	}
	row := db.QueryRow("SELECT last_error FROM track_errors WHERE path = ?", path)
	var message sql.NullString
	if err := row.Scan(&message); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	if message.Valid {
		return message.String, nil
	}
	return "", nil
}
