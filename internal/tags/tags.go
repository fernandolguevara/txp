package tags

import (
	"errors"
	"path/filepath"
	"strings"
)

var ErrUnsupportedFormat = errors.New("tag write unsupported")

type TagFields struct {
	Title    string
	Artist   string
	Album    string
	Genre    string
	Year     string
	TrackNum string
}

func SupportsWrite(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3":
		return true
	case ".flac":
		return true
	default:
		return false
	}
}

func SupportedWriteExtensions() []string {
	return []string{".mp3", ".flac"}
}

func SupportedWriteText() string {
	extensions := SupportedWriteExtensions()
	labels := make([]string, 0, len(extensions))
	for _, ext := range extensions {
		labels = append(labels, strings.TrimPrefix(ext, "."))
	}
	return strings.Join(labels, ", ")
}

func Write(path string, fields TagFields) error {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3":
		return writeMP3Tags(path, fields)
	case ".flac":
		return writeFLACTags(path, fields)
	default:
		return ErrUnsupportedFormat
	}
}
