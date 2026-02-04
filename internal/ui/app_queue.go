package ui

import (
	"fmt"
	"sort"
	"time"

	"github.com/rivo/tview"

	"txp/internal/config"
	"txp/internal/model"
	"txp/internal/storage"
)

func (a *App) addToQueue(track model.Track) {
	a.addTracksToQueue([]model.Track{track})
}

func (a *App) addTracksToQueue(tracks []model.Track) {
	if len(tracks) == 0 {
		return
	}
	a.setQueueIndexWidth(len(a.QueueItems) + len(tracks))
	indexWidth := a.queueIndexWidth
	startIndex := len(a.QueueItems)
	for _, track := range tracks {
		if track.Path == "" {
			continue
		}
		a.QueueItems = append(a.QueueItems, track)
		index := len(a.QueueItems) - 1
		highlighted := listItemHighlighted(a.Panels.Queue.List, index)
		a.Panels.Queue.List.AddItem(formatQueueRow(a.Panels.Queue.List, indexWidth, index, track, false, a.Theme.Accent, a.Theme.Fg, highlighted), "", 0, nil)
	}
	if len(a.QueueItems) > startIndex {
		for i := startIndex; i < len(a.QueueItems); i++ {
			delete(a.QueuePlayed, i)
		}
	}
	if a.QueueIndex == -1 && len(a.QueueItems) > 0 {
		a.QueueIndex = 0
		a.Panels.Queue.List.SetCurrentItem(0)
		a.updateQueueRow(0)
	}
	a.refreshQueueTitle()
	a.persistQueue()
}

func (a *App) moveQueueItem(delta int) {
	if len(a.QueueItems) < 2 {
		return
	}
	index := a.Panels.Queue.List.GetCurrentItem()
	if index < 0 || index >= len(a.QueueItems) {
		return
	}
	newIndex := index + delta
	if newIndex < 0 || newIndex >= len(a.QueueItems) {
		return
	}
	a.QueueItems[index], a.QueueItems[newIndex] = a.QueueItems[newIndex], a.QueueItems[index]
	if a.QueuePlayed[index] || a.QueuePlayed[newIndex] {
		a.QueuePlayed[index], a.QueuePlayed[newIndex] = a.QueuePlayed[newIndex], a.QueuePlayed[index]
	}
	a.refreshQueueRows()
	a.Panels.Queue.List.SetCurrentItem(newIndex)
	switch a.QueueIndex {
	case index:
		a.QueueIndex = newIndex
	case newIndex:
		a.QueueIndex = index
	}
	a.persistQueue()
	if delta < 0 {
		a.setStatusMessage("Moved up in queue")
		return
	}
	a.setStatusMessage("Moved down in queue")
}

func (a *App) removeQueueItem(index int) {
	if index < 0 || index >= len(a.QueueItems) {
		return
	}
	a.QueueItems = append(a.QueueItems[:index], a.QueueItems[index+1:]...)
	played := map[int]bool{}
	for i, value := range a.QueuePlayed {
		if i == index {
			continue
		}
		if i > index {
			played[i-1] = value
			continue
		}
		played[i] = value
	}
	a.QueuePlayed = played
	if a.QueueIndex > index {
		a.QueueIndex--
	}
	if len(a.QueueItems) == 0 {
		a.QueueIndex = -1
	} else if a.QueueIndex >= len(a.QueueItems) {
		a.QueueIndex = len(a.QueueItems) - 1
	}
	a.refreshQueueRows()
	if a.QueueIndex >= 0 {
		a.Panels.Queue.List.SetCurrentItem(a.QueueIndex)
	}
	a.persistQueue()
	a.refreshQueueTitle()
	a.setStatusMessage("Removed from queue")
}

func (a *App) clearQueue() {
	if len(a.QueueItems) == 0 {
		return
	}
	a.QueueItems = nil
	a.QueuePlayed = map[int]bool{}
	a.QueueIndex = -1
	a.Panels.Queue.List.Clear()
	a.persistQueue()
	a.refreshQueueTitle()
	a.setStatusMessage("Queue cleared")
}

func (a *App) refreshQueueRows() {
	current := a.Panels.Queue.List.GetCurrentItem()
	focused := a.Panels.Queue.List.HasFocus()
	a.setQueueIndexWidth(len(a.QueueItems))
	indexWidth := a.queueIndexWidth
	a.Panels.Queue.List.Clear()
	for i, track := range a.QueueItems {
		highlighted := focused && i == current
		a.Panels.Queue.List.AddItem(formatQueueRow(a.Panels.Queue.List, indexWidth, i, track, a.QueuePlayed[i], a.Theme.Accent, a.Theme.Fg, highlighted), "", 0, nil)
	}
	if current >= 0 && current < len(a.QueueItems) {
		a.Panels.Queue.List.SetCurrentItem(current)
	}
}

func (a *App) playQueueIndex(index int) {
	if index < 0 || index >= len(a.QueueItems) {
		return
	}
	a.QueueIndex = index
	a.QueuePlayed[index] = true
	a.Panels.Queue.List.SetCurrentItem(index)
	a.playTrackFromSource(a.QueueItems[index], playbackSourceQueue)
	a.refreshQueueRows()
}

func (a *App) loadQueue() {
	if a.db == nil {
		return
	}
	scope := a.QueueScope
	if scope == "" {
		scope = config.ScopeCurrent
	}
	go func() {
		tracks, err := storage.LoadQueue(a.db.DB, scope)
		a.App.QueueUpdateDraw(func() {
			if err != nil {
				a.setStatusMessage("Failed to load queue")
				return
			}
			a.QueueItems = tracks
			a.Panels.Queue.List.Clear()
			a.QueuePlayed = map[int]bool{}
			a.setQueueIndexWidth(len(tracks))
			indexWidth := a.queueIndexWidth
			for i, track := range tracks {
				highlighted := listItemHighlighted(a.Panels.Queue.List, i)
				a.Panels.Queue.List.AddItem(formatQueueRow(a.Panels.Queue.List, indexWidth, i, track, false, a.Theme.Accent, a.Theme.Fg, highlighted), "", 0, nil)
			}
			if len(tracks) > 0 {
				a.QueueIndex = 0
				a.Panels.Queue.List.SetCurrentItem(0)
				a.updateQueueRow(0)
			} else {
				a.QueueIndex = -1
			}
			a.refreshQueueTitle()
		})
	}()
}

func (a *App) persistQueue() {
	if a.db == nil {
		return
	}
	scope := a.QueueScope
	if scope == "" {
		scope = config.ScopeCurrent
	}
	items := append([]model.Track(nil), a.QueueItems...)
	a.scheduleQueuePersist(scope, items)
}

func (a *App) scheduleQueuePersist(scope string, items []model.Track) {
	a.queuePersistMu.Lock()
	a.queuePersistItems = items
	a.queuePersistScope = scope
	if a.queuePersistTimer == nil {
		a.queuePersistTimer = time.AfterFunc(350*time.Millisecond, a.flushQueuePersist)
	} else {
		a.queuePersistTimer.Reset(350 * time.Millisecond)
	}
	a.queuePersistMu.Unlock()
}

func (a *App) flushQueuePersist() {
	a.queuePersistMu.Lock()
	scope := a.queuePersistScope
	items := append([]model.Track(nil), a.queuePersistItems...)
	a.queuePersistTimer = nil
	a.queuePersistMu.Unlock()
	if a.db == nil {
		return
	}
	if err := storage.ReplaceQueue(a.db.DB, scope, items); err != nil {
		a.App.QueueUpdateDraw(func() {
			a.setStatusMessage("Failed to save queue")
		})
	}
}

func (a *App) nextTrack() {
	if len(a.QueueItems) == 0 {
		return
	}
	if a.QueueIndex < 0 {
		a.QueueIndex = 0
	}
	a.QueueIndex = (a.QueueIndex + 1) % len(a.QueueItems)
	a.playQueueIndex(a.QueueIndex)
}

func (a *App) prevTrack() {
	if len(a.QueueItems) == 0 {
		return
	}
	if a.QueueIndex <= 0 {
		a.QueueIndex = len(a.QueueItems) - 1
	} else {
		a.QueueIndex--
	}
	a.playQueueIndex(a.QueueIndex)
}

func (a *App) updateQueueRow(index int) {
	if index < 0 || index >= len(a.QueueItems) {
		return
	}
	if a.Panels.Queue.List == nil || index >= a.Panels.Queue.List.GetItemCount() {
		return
	}
	indexWidth := a.queueIndexWidth
	highlighted := listItemHighlighted(a.Panels.Queue.List, index)
	row := formatQueueRow(a.Panels.Queue.List, indexWidth, index, a.QueueItems[index], a.QueuePlayed[index], a.Theme.Accent, a.Theme.Fg, highlighted)
	_, secondary := a.Panels.Queue.List.GetItemText(index)
	a.Panels.Queue.List.SetItemText(index, row, secondary)
}

func (a *App) setQueueIndexWidth(count int) {
	width := trackRowIndexDigits(count)
	a.queueIndexWidth = width
}

func (a *App) queuePanelTitle() string {
	count := len(a.QueueItems)
	if count < 0 {
		count = 0
	}
	if a.ViewMode == "library" {
		return fmt.Sprintf("3 Queue View #%d", count)
	}
	return fmt.Sprintf("2 Queue #%d", count)
}

func (a *App) refreshQueueTitle() {
	if a.Panels.Queue.Header != nil {
		a.setQueueIndexWidth(len(a.QueueItems))
		a.Panels.Queue.Header.SetText(formatQueueHeader(len(a.QueueItems), a.queueIndexWidth))
	}
	if a.Panels.Queue.Container != nil {
		a.setPanelStyle(a.Panels.Queue.Container, a.queuePanelTitle(), a.ActivePanel == "queue")
	}
}

func (a *App) addSelectedFromList(list string) {
	selected, tracks, targetList := a.selectionState(list)
	if selected == nil || tracks == nil || targetList == nil {
		return
	}
	indices := []int{}
	for index := range selected {
		if index >= 0 && index < len(tracks) {
			indices = append(indices, index)
		}
	}
	if len(indices) == 0 {
		index := targetList.GetCurrentItem()
		if index >= 0 && index < len(tracks) {
			a.addToQueue(tracks[index])
			a.setStatusMessage("Added to queue")
		}
		return
	}
	sort.Ints(indices)
	batch := []model.Track{}
	for _, index := range indices {
		batch = append(batch, tracks[index])
	}
	a.addTracksToQueue(batch)
	count := len(indices)
	if count == 1 {
		a.setStatusMessage("Added 1 track to queue")
		return
	}
	a.setStatusMessage(fmt.Sprintf("Added %d tracks to queue", count))
}

func (a *App) selectionState(list string) (map[int]bool, []model.Track, *tview.List) {
	switch list {
	case "tracks":
		return a.TracksSelected, a.TracksData, a.Panels.Tracks.List
	case "library":
		return a.LibrarySelected, a.LibraryData, a.Panels.Library.Content
	default:
		return nil, nil, nil
	}
}
