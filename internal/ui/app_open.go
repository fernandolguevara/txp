package ui

import (
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"

	"txp/internal/model"
)

func (a *App) openFocusedTrackInExplorer() {
	path := a.focusedTrackPath()
	if path == "" {
		a.setStatusMessage("No track selected")
		return
	}
	if err := revealInFileExplorer(path); err != nil {
		a.setStatusMessage("Failed to open file explorer")
		return
	}
	a.setStatusMessage("Opened in file explorer")
}

func (a *App) focusedTrackPath() string {
	focus := a.App.GetFocus()
	switch focus {
	case a.Panels.Tracks.List:
		return currentTrackPath(a.Panels.Tracks.List, a.TracksData)
	case a.Panels.Library.Content:
		return currentTrackPath(a.Panels.Library.Content, a.LibraryData)
	case a.Panels.Queue.List:
		return currentTrackPath(a.Panels.Queue.List, a.QueueItems)
	default:
		return ""
	}
}

func currentTrackPath(list interface{ GetCurrentItem() int }, tracks []model.Track) string {
	if list == nil {
		return ""
	}
	index := list.GetCurrentItem()
	if index < 0 || index >= len(tracks) {
		return ""
	}
	return tracks[index].Path
}

func revealInFileExplorer(path string) error {
	if path == "" {
		return errors.New("missing path")
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-R", path).Start()
	case "windows":
		return exec.Command("explorer.exe", "/select,", path).Start()
	default:
		return exec.Command("xdg-open", filepath.Dir(path)).Start()
	}
}
