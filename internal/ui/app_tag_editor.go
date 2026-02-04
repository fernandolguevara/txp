package ui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"txp/internal/model"
	"txp/internal/storage"
	"txp/internal/tags"
)

func (a *App) openTagEditor() {
	track, listID, ok := a.focusedTrack()
	if !ok {
		a.setStatusMessage("No track selected")
		return
	}
	if a.TagEditor == nil {
		return
	}
	a.TagEditor.Configure(track, func(update TagUpdate) bool {
		return a.saveTagEdits(update, listID, track)
	}, func() {
		a.hideTagEditor()
	})
	a.TagEditor.SetSupportedText(tags.SupportedWriteText())
	a.pushOverlay("tag-editor", nil, false, nil)
	a.TagEditor.Active = true
	a.setPanelStyle(a.TagEditor.Root, "Tag Editor", true)
	a.App.SetFocus(a.TagEditor.Title)
	a.setStatusMessage(fmt.Sprintf("Tag write supported: %s", tags.SupportedWriteText()))
	a.updateBottomHints()
}

func (a *App) hideTagEditor() {
	if a.TagEditor == nil {
		return
	}
	a.TagEditor.Active = false
	if entry, ok := a.popOverlay("tag-editor"); ok {
		a.restoreOverlayFocus(entry)
	}
	a.setPanelStyle(a.TagEditor.Root, "Tag Editor", false)
	a.updateBottomHints()
}

func (a *App) saveTagEdits(update TagUpdate, listID string, track model.Track) bool {
	if update.Path == "" {
		a.setStatusMessage("No track selected")
		return false
	}
	if !tags.SupportsWrite(update.Path) {
		a.setStatusMessage(fmt.Sprintf("Tag write not supported for %s", strings.ToLower(filepath.Ext(update.Path))))
		return false
	}
	year, err := parseOptionalInt(update.Year)
	if err != nil {
		a.setStatusMessage("Invalid year")
		return false
	}
	trackNum, err := parseOptionalInt(update.TrackNum)
	if err != nil {
		a.setStatusMessage("Invalid track #")
		return false
	}

	fields := tags.TagFields{
		Title:    update.Title,
		Artist:   update.Artist,
		Album:    update.Album,
		Genre:    update.Genre,
		Year:     update.Year,
		TrackNum: update.TrackNum,
	}
	if err := tags.Write(update.Path, fields); err != nil {
		if err == tags.ErrUnsupportedFormat {
			a.setStatusMessage(fmt.Sprintf("Tag write not supported for %s", strings.ToLower(filepath.Ext(update.Path))))
			return false
		}
		a.setStatusMessage("Failed to write tags")
		return false
	}

	if a.db != nil {
		if err := storage.UpdateTrackTags(a.db.DB, update.Path, update.Title, update.Artist, update.Album, update.Genre, year, trackNum); err != nil {
			a.setStatusMessage("Failed to save tags")
			return false
		}
	}

	updated := track
	updated.Title = update.Title
	updated.Artist = update.Artist
	updated.Album = update.Album
	updated.Genre = update.Genre
	updated.Year = year
	updated.TrackNum = trackNum
	a.updateTrackTagsInSlices(updated)

	switch listID {
	case "tracks":
		a.updateTrackRow("tracks", a.Panels.Tracks.List.GetCurrentItem())
	case "library":
		a.updateTrackRow("library", a.Panels.Library.Content.GetCurrentItem())
	case "queue":
		a.updateQueueRow(a.Panels.Queue.List.GetCurrentItem())
	}
	a.refreshQueueRows()
	a.updateTrackInfo(&updated)
	a.setStatusMessage("Tags saved")
	a.hideTagEditor()
	return true
}

func parseOptionalInt(value string) (int, error) {
	text := strings.TrimSpace(value)
	if text == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(text)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func (a *App) focusedTrack() (model.Track, string, bool) {
	focus := a.App.GetFocus()
	switch focus {
	case a.Panels.Tracks.List:
		track := currentTrack(a.Panels.Tracks.List, a.TracksData)
		return track, "tracks", track.Path != ""
	case a.Panels.Library.Content:
		track := currentTrack(a.Panels.Library.Content, a.LibraryData)
		return track, "library", track.Path != ""
	case a.Panels.Queue.List:
		track := currentTrack(a.Panels.Queue.List, a.QueueItems)
		return track, "queue", track.Path != ""
	default:
		return model.Track{}, "", false
	}
}

func currentTrack(list interface{ GetCurrentItem() int }, tracks []model.Track) model.Track {
	index := list.GetCurrentItem()
	if index < 0 || index >= len(tracks) {
		return model.Track{}
	}
	return tracks[index]
}
