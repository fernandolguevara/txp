package storage

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"txp/internal/config"
)

type Database struct {
	DB    *sql.DB
	Path  string
	Scope string
}

func Open() (*Database, error) {
	resolved, err := config.ResolveDbPath()
	if err != nil {
		return nil, err
	}
	return openWithPath(resolved.Path, resolved.Scope)
}

func OpenWithPath(path string) (*Database, error) {
	return openWithPath(path, config.ScopeCurrent)
}

func openWithPath(path string, scope string) (*Database, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err := initSchema(db); err != nil {
		return nil, err
	}

	applySqlitePragmas(db)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	return &Database{DB: db, Path: path, Scope: scope}, nil
}

func initSchema(db *sql.DB) error {
	statements := []string{
		"CREATE TABLE IF NOT EXISTS tracks (path TEXT PRIMARY KEY, title TEXT, artist TEXT, album TEXT, track_no INTEGER, duration REAL, mtime INTEGER)",
		"CREATE TABLE IF NOT EXISTS queue (scope TEXT, position INTEGER, path TEXT, PRIMARY KEY(scope, position))",
		"CREATE TABLE IF NOT EXISTS track_stats (path TEXT PRIMARY KEY, play_count INTEGER, skip_count INTEGER, last_played INTEGER, last_position REAL)",
		"CREATE TABLE IF NOT EXISTS stats_cache (key TEXT PRIMARY KEY, value_json TEXT, updated_at INTEGER)",
		"CREATE TABLE IF NOT EXISTS track_errors (path TEXT PRIMARY KEY, last_error TEXT, updated_at INTEGER)",
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	if err := ensureTrackKeyColumn(db); err != nil {
		return err
	}
	if err := ensureTrackBpmColumn(db); err != nil {
		return err
	}
	if err := ensureTrackGenreColumn(db); err != nil {
		return err
	}
	if err := ensureTrackYearColumn(db); err != nil {
		return err
	}
	if err := ensureTrackFavoriteColumn(db); err != nil {
		return err
	}
	if err := ensureTrackChecksumColumn(db); err != nil {
		return err
	}
	return nil
}

func applySqlitePragmas(db *sql.DB) {
	_, _ = db.Exec("PRAGMA journal_mode=WAL")
	_, _ = db.Exec("PRAGMA synchronous=NORMAL")
	_, _ = db.Exec("PRAGMA busy_timeout=5000")
}

func ensureTrackKeyColumn(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(tracks)")
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "key" {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec("ALTER TABLE tracks ADD COLUMN key TEXT")
	return err
}

func ensureTrackBpmColumn(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(tracks)")
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "bpm" {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec("ALTER TABLE tracks ADD COLUMN bpm REAL")
	return err
}

func ensureTrackGenreColumn(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(tracks)")
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "genre" {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec("ALTER TABLE tracks ADD COLUMN genre TEXT")
	return err
}

func ensureTrackYearColumn(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(tracks)")
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "year" {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec("ALTER TABLE tracks ADD COLUMN year INTEGER")
	return err
}

func ensureTrackFavoriteColumn(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(tracks)")
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "favorite" {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec("ALTER TABLE tracks ADD COLUMN favorite INTEGER DEFAULT 0")
	return err
}

func ensureTrackChecksumColumn(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(tracks)")
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "checksum" {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec("ALTER TABLE tracks ADD COLUMN checksum TEXT")
	return err
}
