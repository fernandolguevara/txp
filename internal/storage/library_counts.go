package storage

import (
	"database/sql"
	"strings"
)

type LibraryCounts struct {
	Artists   int
	Albums    int
	Tracks    int
	Genres    int
	Years     int
	Favorites int
}

func GetLibraryCounts(db *sql.DB, filter string) (LibraryCounts, error) {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return libraryCountsAll(db)
	}
	if ext, ok := extensionFilter(filter); ok {
		return libraryCountsExtension(db, "%"+ext)
	}
	return libraryCountsFiltered(db, "%"+filter+"%")
}

func GetLibraryCountsInLibraries(db *sql.DB, filter string, roots []string) (LibraryCounts, error) {
	filter = strings.TrimSpace(filter)
	where, args, empty := buildLibraryPathFilter(roots)
	if empty {
		return LibraryCounts{}, nil
	}
	if filter == "" {
		return libraryCountsAllInLibraries(db, where, args)
	}
	if ext, ok := extensionFilter(filter); ok {
		return libraryCountsExtensionInLibraries(db, where, args, "%"+ext)
	}
	return libraryCountsFilteredInLibraries(db, where, args, "%"+filter+"%")
}

func libraryCountsAll(db *sql.DB) (LibraryCounts, error) {
	var counts LibraryCounts
	row := db.QueryRow("SELECT COUNT(DISTINCT NULLIF(TRIM(artist), '')), COUNT(DISTINCT NULLIF(TRIM(album), '')), COUNT(*), COUNT(DISTINCT NULLIF(TRIM(genre), '')), COUNT(DISTINCT NULLIF(year, 0)), SUM(CASE WHEN favorite = 1 THEN 1 ELSE 0 END) FROM tracks")
	if err := row.Scan(&counts.Artists, &counts.Albums, &counts.Tracks, &counts.Genres, &counts.Years, &counts.Favorites); err != nil {
		return counts, err
	}
	return counts, nil
}

func libraryCountsAllInLibraries(db *sql.DB, where string, args []interface{}) (LibraryCounts, error) {
	var counts LibraryCounts
	query := "SELECT COUNT(DISTINCT NULLIF(TRIM(artist), '')), COUNT(DISTINCT NULLIF(TRIM(album), '')), COUNT(*), COUNT(DISTINCT NULLIF(TRIM(genre), '')), COUNT(DISTINCT NULLIF(year, 0)), SUM(CASE WHEN favorite = 1 THEN 1 ELSE 0 END) FROM tracks WHERE " + where
	row := db.QueryRow(query, args...)
	if err := row.Scan(&counts.Artists, &counts.Albums, &counts.Tracks, &counts.Genres, &counts.Years, &counts.Favorites); err != nil {
		return counts, err
	}
	return counts, nil
}

func libraryCountsFiltered(db *sql.DB, filter string) (LibraryCounts, error) {
	var counts LibraryCounts
	row := db.QueryRow(
		"SELECT COUNT(DISTINCT NULLIF(TRIM(artist), '')), COUNT(DISTINCT NULLIF(TRIM(album), '')), COUNT(*), COUNT(DISTINCT NULLIF(TRIM(genre), '')), COUNT(DISTINCT NULLIF(year, 0)), SUM(CASE WHEN favorite = 1 THEN 1 ELSE 0 END) FROM tracks WHERE lower(artist) LIKE lower(?) OR lower(album) LIKE lower(?) OR lower(title) LIKE lower(?) OR lower(genre) LIKE lower(?) OR CAST(year AS TEXT) LIKE lower(?) OR lower(path) LIKE lower(?)",
		filter,
		filter,
		filter,
		filter,
		filter,
		filter,
	)
	if err := row.Scan(&counts.Artists, &counts.Albums, &counts.Tracks, &counts.Genres, &counts.Years, &counts.Favorites); err != nil {
		return counts, err
	}
	return counts, nil
}

func libraryCountsFilteredInLibraries(db *sql.DB, where string, args []interface{}, filter string) (LibraryCounts, error) {
	var counts LibraryCounts
	query := "SELECT COUNT(DISTINCT NULLIF(TRIM(artist), '')), COUNT(DISTINCT NULLIF(TRIM(album), '')), COUNT(*), COUNT(DISTINCT NULLIF(TRIM(genre), '')), COUNT(DISTINCT NULLIF(year, 0)), SUM(CASE WHEN favorite = 1 THEN 1 ELSE 0 END) FROM tracks WHERE (" + where + ") AND (lower(artist) LIKE lower(?) OR lower(album) LIKE lower(?) OR lower(title) LIKE lower(?) OR lower(genre) LIKE lower(?) OR CAST(year AS TEXT) LIKE lower(?) OR lower(path) LIKE lower(?))"
	args = append(args, filter, filter, filter, filter, filter, filter)
	row := db.QueryRow(query, args...)
	if err := row.Scan(&counts.Artists, &counts.Albums, &counts.Tracks, &counts.Genres, &counts.Years, &counts.Favorites); err != nil {
		return counts, err
	}
	return counts, nil
}

func libraryCountsExtension(db *sql.DB, filter string) (LibraryCounts, error) {
	var counts LibraryCounts
	row := db.QueryRow(
		"SELECT COUNT(DISTINCT NULLIF(TRIM(artist), '')), COUNT(DISTINCT NULLIF(TRIM(album), '')), COUNT(*), COUNT(DISTINCT NULLIF(TRIM(genre), '')), COUNT(DISTINCT NULLIF(year, 0)), SUM(CASE WHEN favorite = 1 THEN 1 ELSE 0 END) FROM tracks WHERE lower(path) LIKE lower(?)",
		filter,
	)
	if err := row.Scan(&counts.Artists, &counts.Albums, &counts.Tracks, &counts.Genres, &counts.Years, &counts.Favorites); err != nil {
		return counts, err
	}
	return counts, nil
}

func libraryCountsExtensionInLibraries(db *sql.DB, where string, args []interface{}, filter string) (LibraryCounts, error) {
	var counts LibraryCounts
	query := "SELECT COUNT(DISTINCT NULLIF(TRIM(artist), '')), COUNT(DISTINCT NULLIF(TRIM(album), '')), COUNT(*), COUNT(DISTINCT NULLIF(TRIM(genre), '')), COUNT(DISTINCT NULLIF(year, 0)), SUM(CASE WHEN favorite = 1 THEN 1 ELSE 0 END) FROM tracks WHERE (" + where + ") AND lower(path) LIKE lower(?)"
	args = append(args, filter)
	row := db.QueryRow(query, args...)
	if err := row.Scan(&counts.Artists, &counts.Albums, &counts.Tracks, &counts.Genres, &counts.Years, &counts.Favorites); err != nil {
		return counts, err
	}
	return counts, nil
}

func extensionFilter(filter string) (string, bool) {
	filter = strings.TrimSpace(strings.ToLower(filter))
	if filter == "" {
		return "", false
	}
	if strings.HasPrefix(filter, "*.") && len(filter) > 2 {
		return filter[1:], true
	}
	if strings.HasPrefix(filter, ".") && len(filter) > 1 {
		return filter, true
	}
	return "", false
}
