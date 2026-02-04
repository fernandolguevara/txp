package storage

import (
	"os"
	"strings"
)

func buildLibraryPathFilter(roots []string) (string, []interface{}, bool) {
	return buildLibraryPathFilterForColumn(roots, "path")
}

func buildLibraryPathFilterForColumn(roots []string, column string) (string, []interface{}, bool) {
	if len(roots) == 0 {
		return "", nil, true
	}
	if strings.TrimSpace(column) == "" {
		column = "path"
	}
	clauses := []string{}
	args := []interface{}{}
	sep := string(os.PathSeparator)
	for _, root := range roots {
		clean := strings.TrimSpace(root)
		if clean == "" {
			continue
		}
		clean = strings.TrimRight(clean, sep)
		if clean == "" && strings.TrimSpace(root) == sep {
			clean = sep
		}
		if clean == "" {
			continue
		}
		clauses = append(clauses, "("+column+" = ? OR "+column+" LIKE ?)")
		args = append(args, clean, clean+sep+"%")
	}
	if len(clauses) == 0 {
		return "", nil, true
	}
	return strings.Join(clauses, " OR "), args, false
}
