package storage

import (
	"database/sql"
	"fmt"

	"txp/internal/model"
)

func UpsertTrack(db *sql.DB, track model.Track) error {
	_, err := db.Exec(
		"INSERT INTO tracks (path, title, artist, album, genre, year, track_no, duration, mtime, checksum, key, bpm) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) "+
			"ON CONFLICT(path) DO UPDATE SET title=excluded.title, artist=excluded.artist, album=excluded.album, genre=excluded.genre, year=excluded.year, track_no=excluded.track_no, duration=excluded.duration, mtime=excluded.mtime, checksum=excluded.checksum, key=excluded.key, bpm=excluded.bpm",
		track.Path,
		track.Title,
		track.Artist,
		track.Album,
		track.Genre,
		track.Year,
		track.TrackNum,
		track.Duration,
		track.Mtime,
		track.Checksum,
		track.Key,
		track.BPM,
	)
	return err
}

func EnsureTrackStats(db *sql.DB, path string) error {
	_, err := db.Exec(
		"INSERT OR IGNORE INTO track_stats (path, play_count, skip_count, last_played, last_position) VALUES (?, 0, 0, 0, 0)",
		path,
	)
	return err
}

func InsertTrackStub(db *sql.DB, path string, mtime int64, checksum string) error {
	_, err := db.Exec(
		"INSERT OR IGNORE INTO tracks (path, mtime, checksum) VALUES (?, ?, ?)",
		path,
		mtime,
		checksum,
	)
	return err
}

func GetTrackScanMeta(db *sql.DB, path string) (int64, string, bool, error) {
	row := db.QueryRow("SELECT mtime, checksum FROM tracks WHERE path = ?", path)
	var mtime sql.NullInt64
	var checksum sql.NullString
	if err := row.Scan(&mtime, &checksum); err != nil {
		if err == sql.ErrNoRows {
			return 0, "", false, nil
		}
		return 0, "", false, err
	}
	if !mtime.Valid {
		return 0, checksum.String, true, nil
	}
	if checksum.Valid {
		return mtime.Int64, checksum.String, true, nil
	}
	return mtime.Int64, "", true, nil
}

func ListTracks(db *sql.DB) ([]model.Track, error) {
	rows, err := db.Query("SELECT path, title, artist, album, genre, year, track_no, duration, mtime, checksum, key, bpm, favorite FROM tracks ORDER BY path")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	tracks := []model.Track{}
	for rows.Next() {
		t, err := scanTrackRow(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tracks, nil
}

func ListTracksInLibraries(db *sql.DB, roots []string) ([]model.Track, error) {
	where, args, empty := buildLibraryPathFilter(roots)
	if empty {
		return []model.Track{}, nil
	}
	query := "SELECT path, title, artist, album, genre, year, track_no, duration, mtime, checksum, key, bpm, favorite FROM tracks WHERE " + where + " ORDER BY path"
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	tracks := []model.Track{}
	for rows.Next() {
		t, err := scanTrackRow(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tracks, nil
}

func ListUnanalyzedPaths(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT path FROM tracks WHERE duration <= 0 OR bpm <= 0 OR key IS NULL OR key = '' ORDER BY path")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	paths := []string{}
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		if path == "" {
			continue
		}
		paths = append(paths, path)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return paths, nil
}

func CountTracks(db *sql.DB) (int, error) {
	row := db.QueryRow("SELECT COUNT(*) FROM tracks")
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func ListDistinctArtists(db *sql.DB) ([]string, error) {
	return listDistinct(db, "artist")
}

func ListDistinctArtistsInLibraries(db *sql.DB, roots []string) ([]string, error) {
	return listDistinctInLibraries(db, "artist", roots)
}

func ListDistinctAlbums(db *sql.DB) ([]string, error) {
	return listDistinct(db, "album")
}

func ListDistinctAlbumsInLibraries(db *sql.DB, roots []string) ([]string, error) {
	return listDistinctInLibraries(db, "album", roots)
}

func ListDistinctGenres(db *sql.DB) ([]string, error) {
	return listDistinct(db, "genre")
}

func ListDistinctGenresInLibraries(db *sql.DB, roots []string) ([]string, error) {
	return listDistinctInLibraries(db, "genre", roots)
}

func ListDistinctYears(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT DISTINCT year FROM tracks WHERE year IS NOT NULL AND year != 0 ORDER BY year")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	values := []string{}
	for rows.Next() {
		var year int
		if err := rows.Scan(&year); err != nil {
			return nil, err
		}
		values = append(values, fmt.Sprintf("%d", year))
	}
	return values, rows.Err()
}

func ListDistinctYearsInLibraries(db *sql.DB, roots []string) ([]string, error) {
	where, args, empty := buildLibraryPathFilter(roots)
	if empty {
		return []string{}, nil
	}
	query := "SELECT DISTINCT year FROM tracks WHERE year IS NOT NULL AND year != 0 AND (" + where + ") ORDER BY year"
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	values := []string{}
	for rows.Next() {
		var year int
		if err := rows.Scan(&year); err != nil {
			return nil, err
		}
		values = append(values, fmt.Sprintf("%d", year))
	}
	return values, rows.Err()
}

func ListTracksByArtist(db *sql.DB, artist string) ([]model.Track, error) {
	return listTracksByField(db, "artist", artist)
}

func ListTracksByArtistInLibraries(db *sql.DB, artist string, roots []string) ([]model.Track, error) {
	return listTracksByFieldInLibraries(db, "artist", artist, roots)
}

func ListTracksByAlbum(db *sql.DB, album string) ([]model.Track, error) {
	return listTracksByField(db, "album", album)
}

func ListTracksByAlbumInLibraries(db *sql.DB, album string, roots []string) ([]model.Track, error) {
	return listTracksByFieldInLibraries(db, "album", album, roots)
}

func ListTracksByGenre(db *sql.DB, genre string) ([]model.Track, error) {
	return listTracksByField(db, "genre", genre)
}

func ListTracksByGenreInLibraries(db *sql.DB, genre string, roots []string) ([]model.Track, error) {
	return listTracksByFieldInLibraries(db, "genre", genre, roots)
}

func ListTracksByYear(db *sql.DB, year string) ([]model.Track, error) {
	rows, err := db.Query("SELECT path, title, artist, album, genre, year, track_no, duration, mtime, checksum, key, bpm, favorite FROM tracks WHERE year = ? ORDER BY album, track_no, title", year)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	tracks := []model.Track{}
	for rows.Next() {
		t, err := scanTrackRow(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

func ListTracksByYearInLibraries(db *sql.DB, year string, roots []string) ([]model.Track, error) {
	where, args, empty := buildLibraryPathFilter(roots)
	if empty {
		return []model.Track{}, nil
	}
	query := "SELECT path, title, artist, album, genre, year, track_no, duration, mtime, checksum, key, bpm, favorite FROM tracks WHERE year = ? AND (" + where + ") ORDER BY album, track_no, title"
	args = append([]interface{}{year}, args...)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	tracks := []model.Track{}
	for rows.Next() {
		t, err := scanTrackRow(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

func listDistinct(db *sql.DB, field string) ([]string, error) {
	query := fmt.Sprintf("SELECT DISTINCT %s FROM tracks WHERE %s IS NOT NULL AND %s != '' ORDER BY %s", field, field, field, field)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	values := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func listDistinctInLibraries(db *sql.DB, field string, roots []string) ([]string, error) {
	where, args, empty := buildLibraryPathFilter(roots)
	if empty {
		return []string{}, nil
	}
	query := fmt.Sprintf("SELECT DISTINCT %s FROM tracks WHERE %s IS NOT NULL AND %s != '' AND (%s) ORDER BY %s", field, field, field, where, field)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	values := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func listTracksByFieldInLibraries(db *sql.DB, field string, value string, roots []string) ([]model.Track, error) {
	where, args, empty := buildLibraryPathFilter(roots)
	if empty {
		return []model.Track{}, nil
	}
	query := fmt.Sprintf("SELECT path, title, artist, album, genre, year, track_no, duration, mtime, checksum, key, bpm, favorite FROM tracks WHERE %s = ? AND (%s) ORDER BY album, track_no, title", field, where)
	args = append([]interface{}{value}, args...)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	tracks := []model.Track{}
	for rows.Next() {
		t, err := scanTrackRow(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

func listTracksByField(db *sql.DB, field string, value string) ([]model.Track, error) {
	query := fmt.Sprintf("SELECT path, title, artist, album, genre, year, track_no, duration, mtime, checksum, key, bpm, favorite FROM tracks WHERE %s = ? ORDER BY album, track_no, title", field)
	rows, err := db.Query(query, value)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	tracks := []model.Track{}
	for rows.Next() {
		t, err := scanTrackRow(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

func scanTrackRow(rows *sql.Rows) (model.Track, error) {
	var t model.Track
	var title sql.NullString
	var artist sql.NullString
	var album sql.NullString
	var genre sql.NullString
	var year sql.NullInt64
	var trackNum sql.NullInt64
	var duration sql.NullFloat64
	var mtime sql.NullInt64
	var checksum sql.NullString
	var key sql.NullString
	var bpm sql.NullFloat64
	var favorite sql.NullInt64
	if err := rows.Scan(
		&t.Path,
		&title,
		&artist,
		&album,
		&genre,
		&year,
		&trackNum,
		&duration,
		&mtime,
		&checksum,
		&key,
		&bpm,
		&favorite,
	); err != nil {
		return t, err
	}
	if title.Valid {
		t.Title = title.String
	}
	if artist.Valid {
		t.Artist = artist.String
	}
	if album.Valid {
		t.Album = album.String
	}
	if genre.Valid {
		t.Genre = genre.String
	}
	if year.Valid {
		t.Year = int(year.Int64)
	}
	if trackNum.Valid {
		t.TrackNum = int(trackNum.Int64)
	}
	if duration.Valid {
		t.Duration = duration.Float64
	}
	if mtime.Valid {
		t.Mtime = mtime.Int64
	}
	if checksum.Valid {
		t.Checksum = checksum.String
	}
	if key.Valid {
		t.Key = key.String
	}
	if bpm.Valid {
		t.BPM = bpm.Float64
	}
	if favorite.Valid {
		t.Favorite = favorite.Int64 != 0
	}
	return t, nil
}

func SetTrackFavorite(db *sql.DB, path string, value bool) error {
	if path == "" {
		return nil
	}
	flag := 0
	if value {
		flag = 1
	}
	_, err := db.Exec("UPDATE tracks SET favorite = ? WHERE path = ?", flag, path)
	return err
}

func ToggleTrackFavorite(db *sql.DB, path string) (bool, error) {
	if path == "" {
		return false, nil
	}
	_, err := db.Exec("UPDATE tracks SET favorite = CASE WHEN favorite = 1 THEN 0 ELSE 1 END WHERE path = ?", path)
	if err != nil {
		return false, err
	}
	row := db.QueryRow("SELECT favorite FROM tracks WHERE path = ?", path)
	var favorite sql.NullInt64
	if err := row.Scan(&favorite); err != nil {
		return false, err
	}
	return favorite.Valid && favorite.Int64 != 0, nil
}

func UpdateTrackTags(db *sql.DB, path string, title string, artist string, album string, genre string, year int, trackNum int) error {
	if path == "" {
		return nil
	}
	_, err := db.Exec(
		"UPDATE tracks SET title = ?, artist = ?, album = ?, genre = ?, year = ?, track_no = ? WHERE path = ?",
		title,
		artist,
		album,
		genre,
		year,
		trackNum,
		path,
	)
	return err
}

func ListFavoriteTracks(db *sql.DB) ([]model.Track, error) {
	rows, err := db.Query("SELECT path, title, artist, album, genre, year, track_no, duration, mtime, checksum, key, bpm, favorite FROM tracks WHERE favorite = 1 ORDER BY path")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	tracks := []model.Track{}
	for rows.Next() {
		t, err := scanTrackRow(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tracks, nil
}

func ListFavoriteTracksInLibraries(db *sql.DB, roots []string) ([]model.Track, error) {
	where, args, empty := buildLibraryPathFilter(roots)
	if empty {
		return []model.Track{}, nil
	}
	query := "SELECT path, title, artist, album, genre, year, track_no, duration, mtime, checksum, key, bpm, favorite FROM tracks WHERE favorite = 1 AND (" + where + ") ORDER BY path"
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	tracks := []model.Track{}
	for rows.Next() {
		t, err := scanTrackRow(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tracks, nil
}
