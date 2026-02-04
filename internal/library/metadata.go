package library

import (
	"os"
	"path/filepath"

	"github.com/dhowden/tag"

	"txp/internal/model"
)

func ReadMetadata(path string) (model.Track, error) {
	file, err := os.Open(path)
	if err != nil {
		return model.Track{}, err
	}
	defer func() {
		_ = file.Close()
	}()

	meta, err := tag.ReadFrom(file)
	if err != nil {
		return model.Track{}, err
	}

	info, err := file.Stat()
	if err != nil {
		return model.Track{}, err
	}

	title := meta.Title()
	if title == "" {
		title = filepath.Base(path)
	}

	trackNum, _ := meta.Track()

	return model.Track{
		Path:     path,
		Title:    title,
		Artist:   meta.Artist(),
		Album:    meta.Album(),
		Genre:    meta.Genre(),
		Year:     meta.Year(),
		TrackNum: trackNum,
		Duration: 0,
		Mtime:    info.ModTime().Unix(),
	}, nil
}
