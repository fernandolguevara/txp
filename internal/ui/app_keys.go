package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"txp/internal/config"
	"txp/internal/ui/panels"
)

func (a *App) bindKeyHandlers() {
	a.Panels.TopBar.Timeline.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			a.seekMu.Lock()
			a.seekMetaPrefixAt = time.Now()
			a.seekMu.Unlock()
			return nil
		case tcell.KeyF1, tcell.KeyF2, tcell.KeyF3, tcell.KeyF4, tcell.KeyF5, tcell.KeyF6, tcell.KeyF7, tcell.KeyF8, tcell.KeyF9, tcell.KeyF10:
			if a.Player == nil || a.NowPlayingTrack == nil {
				return nil
			}
			duration, ok, _ := a.Player.GetFloatProperty("duration")
			if !ok || duration <= 0 {
				return nil
			}
			position, _, _ := a.Player.GetFloatProperty("time-pos")
			segment := int(event.Key() - tcell.KeyF1)
			if segment < 0 {
				segment = 0
			}
			if segment > 9 {
				segment = 9
			}
			fraction := float64(segment) / 10.0
			target := duration * fraction
			a.requestSeek(target - position)
			return nil
		case tcell.KeyLeft:
			if a.Player != nil && a.NowPlayingTrack != nil {
				a.requestSeek(-5)
				return nil
			}
		case tcell.KeyRight:
			if a.Player != nil && a.NowPlayingTrack != nil {
				a.requestSeek(5)
				return nil
			}
		}
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case '+', '=':
				a.adjustVolume(5)
				return nil
			case '-', '_':
				a.adjustVolume(-5)
				return nil
			}
		}
		return event
	})

	a.LogDialog.View.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			a.hideLogDialog()
			return nil
		}
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'c':
				a.clearLog()
				return nil
			case 'a':
				a.toggleLogAutoScroll()
				return nil
			case 'y':
				a.copyLogToClipboard()
				return nil
			}
		}
		return event
	})

	a.LogFileDialog.View.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			a.hideLogFileDialog()
			return nil
		case tcell.KeyF5:
			a.appendLogLines()
			return nil
		case tcell.KeyPgUp, tcell.KeyUp:
			row, _ := a.LogFileDialog.View.GetScrollOffset()
			if row == 0 {
				a.prependLogLines()
			}
			return event
		case tcell.KeyPgDn, tcell.KeyDown:
			if a.logViewAtBottom() {
				a.appendLogLines()
			}
			return event
		}
		return event
	})

	a.Panels.Tracks.List.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Modifiers() != 0 {
			return event
		}
		if event.Key() == tcell.KeyEnter {
			index := a.Panels.Tracks.List.GetCurrentItem()
			if index >= 0 && index < len(a.TracksData) {
				a.playTrackFromSource(a.TracksData[index], playbackSourceOther)
				return nil
			}
		}
		if event.Key() == tcell.KeyRune && event.Rune() == ' ' {
			a.toggleSelection("tracks", a.Panels.Tracks.List.GetCurrentItem())
			return nil
		}
		if event.Key() == tcell.KeyRune && event.Rune() == 'f' {
			index := a.Panels.Tracks.List.GetCurrentItem()
			if index >= 0 && index < len(a.TracksData) {
				a.toggleFavorite(a.TracksData[index])
				a.updateTrackRow("tracks", index)
				a.updateTrackInfoFromList("tracks", index)
				return nil
			}
		}
		if event.Rune() == 'a' {
			a.addSelectedFromList("tracks")
			return nil
		}
		if event.Rune() == 'b' {
			a.setTrackSort("bpm")
			return nil
		}
		if event.Rune() == 'k' {
			a.setTrackSort("key")
			return nil
		}
		return event
	})

	a.Panels.Library.Content.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Modifiers() != 0 {
			return event
		}
		if event.Key() == tcell.KeyEnter {
			index := a.Panels.Library.Content.GetCurrentItem()
			if index >= 0 && index < len(a.LibraryData) {
				a.playTrackFromSource(a.LibraryData[index], playbackSourceOther)
				return nil
			}
		}
		if event.Key() == tcell.KeyRune && event.Rune() == ' ' {
			a.toggleSelection("library", a.Panels.Library.Content.GetCurrentItem())
			return nil
		}
		if event.Key() == tcell.KeyRune && event.Rune() == 'f' {
			index := a.Panels.Library.Content.GetCurrentItem()
			if index >= 0 && index < len(a.LibraryData) {
				a.toggleFavorite(a.LibraryData[index])
				a.updateTrackRow("library", index)
				a.updateTrackInfoFromList("library", index)
				return nil
			}
		}
		if event.Rune() == 'a' {
			a.addSelectedFromList("library")
			return nil
		}
		return event
	})

	a.Panels.Queue.List.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlD {
			a.clearQueue()
			return nil
		}
		if event.Key() == tcell.KeyRune && event.Rune() == 'd' {
			index := a.Panels.Queue.List.GetCurrentItem()
			a.removeQueueItem(index)
			return nil
		}
		if event.Key() == tcell.KeyRune && event.Rune() == 'f' {
			index := a.Panels.Queue.List.GetCurrentItem()
			if index >= 0 && index < len(a.QueueItems) {
				a.toggleFavorite(a.QueueItems[index])
				a.updateQueueRow(index)
				return nil
			}
		}
		if event.Modifiers()&tcell.ModAlt != 0 {
			switch event.Key() {
			case tcell.KeyUp:
				a.moveQueueItem(-1)
				return nil
			case tcell.KeyDown:
				a.moveQueueItem(1)
				return nil
			}
		}
		if event.Key() == tcell.KeyRune && event.Rune() == ' ' {
			if a.NowPlayingTrack == nil {
				index := a.Panels.Queue.List.GetCurrentItem()
				if index >= 0 {
					a.playQueueIndex(index)
					return nil
				}
			}
			a.togglePause()
			return nil
		}
		if event.Key() == tcell.KeyEnter {
			index := a.Panels.Queue.List.GetCurrentItem()
			if index >= 0 && index < len(a.QueueItems) {
				a.playQueueIndex(index)
				return nil
			}
		}
		return event
	})
}

func (a *App) bindFilters() {
	if a.Panels != nil && a.Panels.Tracks != nil {
		a.Panels.Tracks.Filter.SetChangedFunc(func(text string) {
			a.TracksFilterText = text
			a.applyTracksFilter()
		})
		a.Panels.Tracks.Filter.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEnter, tcell.KeyDown:
				if a.Panels.Tracks.List.GetItemCount() > 0 {
					a.App.SetFocus(a.Panels.Tracks.List)
					return nil
				}
			case tcell.KeyEsc:
				a.Panels.Tracks.Filter.SetText("")
				a.TracksFilterText = ""
				a.applyTracksFilter()
				if a.Panels.Tracks.List.GetItemCount() > 0 {
					a.App.SetFocus(a.Panels.Tracks.List)
					return nil
				}
			}
			return event
		})
	}
	if a.Panels != nil && a.Panels.Library != nil {
		a.Panels.Library.NavFilter.SetChangedFunc(func(text string) {
			a.LibraryFilterText = text
			a.applyLibraryFilter()
		})
		a.Panels.Library.NavFilter.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEnter, tcell.KeyDown:
				a.App.SetFocus(a.Panels.Library.Nav)
				return nil
			case tcell.KeyEsc:
				a.Panels.Library.NavFilter.SetText("")
				a.LibraryFilterText = ""
				a.applyLibraryFilter()
				a.App.SetFocus(a.Panels.Library.Nav)
				return nil
			}
			return event
		})
	}

	a.FilterDialog.Form.GetButton(0).SetSelectedFunc(func() {
		a.applyFilterDialog()
	})
	a.FilterDialog.Form.GetButton(1).SetSelectedFunc(func() {
		a.clearFilterDialog()
	})
	a.FilterDialog.Form.GetButton(2).SetSelectedFunc(func() { a.hideFilterDialog() })
	a.FilterDialog.Form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if (event.Key() == tcell.KeyBackspace || event.Key() == tcell.KeyBackspace2) && event.Modifiers()&tcell.ModCtrl != 0 {
			a.clearFilterDialog()
			return nil
		}
		return event
	})
}

func (a *App) bindSettings() {
	a.Settings.Libraries.SetFocusFunc(func() {
		a.setActiveSection("libraries")
		a.setLibrariesColumnFocus("left")
	})
	a.Settings.LibraryTree.SetFocusFunc(func() {
		a.setActiveSection("libraries")
		a.setLibrariesColumnFocus("right")
	})
	a.Settings.Shortcuts.SetFocusFunc(func() {
		a.setActiveSection("shortcuts")
	})
	a.Settings.Analysis.SetFocusFunc(func() {
		a.setActiveSection("analysis")
	})
	a.Settings.Themes.SetFocusFunc(func() {
		a.setActiveSection("themes")
	})

	a.Settings.Themes.SetSelectedFunc(func(index int, main string, secondary string, shortcut rune) {
		theme := ThemeByName(main)
		a.Theme = theme
		a.Config.Theme = theme.Name
		a.applyTheme(theme)
		a.setStatusMessage(fmt.Sprintf("Theme set to %s", theme.Name))
		a.saveConfig()
	})

	a.Settings.Shortcuts.SetSelectedFunc(func(index int, main string, secondary string, shortcut rune) {
		a.capturing = true
		a.captureAction = main
		a.setStatusMessage(fmt.Sprintf("Rebinding %s (press keys)...", main))
	})

	a.Settings.Analysis.SetSelectedFunc(func(index int, main string, secondary string, shortcut rune) {
		a.handleAnalysisSelection(index)
	})

	a.Settings.Libraries.SetSelectedFunc(func(index int, main string, secondary string, shortcut rune) {
		a.setStatusMessage(fmt.Sprintf("Library: %s", main))
	})

	a.Settings.Libraries.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'p':
			a.showAddLibraryPathDialog()
			return nil
		case 'd':
			index := a.Settings.Libraries.GetCurrentItem()
			if index >= 0 {
				main, _ := a.Settings.Libraries.GetItemText(index)
				a.Settings.Libraries.RemoveItem(index)
				a.removeLibrary(main)
				a.removeLibraryFromNavigation(main)
				a.setStatusMessage("Library removed")
				a.saveConfig()
			}
			return nil
		case 'r':
			a.showRescanDialog()
			return nil
		}
		if event.Key() == tcell.KeyRight {
			a.App.SetFocus(a.Settings.LibraryTree)
			return nil
		}
		return event
	})

	a.Settings.LibraryTree.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyLeft {
			a.App.SetFocus(a.Settings.Libraries)
			return nil
		}
		if event.Key() == tcell.KeyEnter {
			node := a.Settings.LibraryTree.GetCurrentNode()
			if node != nil && node.GetText() == ".." {
				a.resetLibraryTree(node)
				return nil
			}
		}
		if event.Key() == tcell.KeyBackspace || event.Key() == tcell.KeyBackspace2 {
			a.resetLibraryTree(a.Settings.LibraryTree.GetCurrentNode())
			return nil
		}
		if event.Key() == tcell.KeyRune && event.Rune() == ' ' {
			node := a.Settings.LibraryTree.GetCurrentNode()
			if node == nil || node.GetText() == ".." {
				return nil
			}
			path, ok := node.GetReference().(string)
			if !ok || path == "" {
				return nil
			}
			a.addLibraryFromTree(path)
			return nil
		}
		return event
	})

	a.Settings.Shortcuts.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() == 'x' {
			index := a.Settings.Shortcuts.GetCurrentItem()
			if index < 0 {
				return nil
			}
			action, _ := a.Settings.Shortcuts.GetItemText(index)
			defaults := config.DefaultConfig().Shortcuts[action]
			if len(defaults) == 0 {
				a.setStatusMessage("No default shortcut for action")
				return nil
			}
			if a.Config.Shortcuts == nil {
				a.Config.Shortcuts = map[string][]string{}
			}
			a.Config.Shortcuts[action] = append([]string(nil), defaults...)
			a.Settings.Shortcuts.SetItemText(index, action, strings.Join(defaults, ", "))
			a.setStatusMessage(fmt.Sprintf("Reset shortcut for %s", action))
			a.saveConfig()
			return nil
		}
		return event
	})

	a.Settings.Analysis.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		return event
	})

	a.Settings.Themes.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		return event
	})
}

func (a *App) bindNavigation() {
	a.Panels.Library.Nav.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() == ' ' {
			node := a.Panels.Library.Nav.GetCurrentNode()
			if node == nil {
				return nil
			}
			ref, ok := node.GetReference().(*panels.LibraryRef)
			if !ok || ref.Path == "" {
				return nil
			}

			selected := !a.SelectedLibs[ref.Path]
			a.SelectedLibs[ref.Path] = selected
			panels.UpdateLibraryNodeText(node, ref.Label, selected)
			a.syncSelectedLibraries()
			a.saveConfig()
			a.resetLibraryNavCache()
			a.refreshLibraryTotals()
			a.refreshLibraryNavFilter(a.LibraryFilterText)
			a.refreshStatsTotals()
			if a.ViewMode == "library" {
				a.reloadLibrarySelection()
			}
			if a.ViewMode == "tracks" {
				a.refreshTracks()
			}
			a.setStatusMessage(fmt.Sprintf("Selected libraries: %d", a.selectedCount()))
			return nil
		}
		if event.Key() == tcell.KeyRune && event.Rune() == 'f' {
			a.App.SetFocus(a.Panels.Library.NavFilter)
			return nil
		}
		if event.Key() == tcell.KeyRune && event.Rune() == 'a' {
			node := a.Panels.Library.Nav.GetCurrentNode()
			if node == nil {
				return nil
			}
			ref, ok := node.GetReference().(*panels.NavRef)
			if !ok {
				return nil
			}
			a.enqueueSelectionFromNav(ref)
			return nil
		}
		return event
	})

	a.Panels.Library.Nav.SetSelectedFunc(func(node *tview.TreeNode) {
		if a.NavRestoreInProgress {
			return
		}
		if node == nil {
			return
		}
		if a.NavPersistOnNextSelect {
			a.NavPersistOnNextSelect = false
			if ref, ok := node.GetReference().(*panels.LibraryRef); ok {
				a.Config.LibraryNavCursor = &config.LibraryNavCursor{Path: ref.Path}
				a.saveConfig()
			} else if ref, ok := node.GetReference().(*panels.NavRef); ok {
				a.Config.LibraryNavCursor = &config.LibraryNavCursor{Kind: ref.Kind, Value: ref.Value}
				a.saveConfig()
			}
		}
		ref, ok := node.GetReference().(*panels.NavRef)
		if !ok {
			if a.Panels != nil && a.Panels.Library != nil && a.Panels.Library.Nav != nil {
				a.App.SetFocus(a.Panels.Library.Nav)
			}
			return
		}
		a.handleLibrarySelection(node, ref)
		if a.Panels != nil && a.Panels.Library != nil && a.Panels.Library.Nav != nil {
			a.App.SetFocus(a.Panels.Library.Nav)
		}
	})
}

func (a *App) bindGlobalKeys(layout *tview.Pages) {
	a.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if a.capturing {
			binding := formatKeybinding(event)
			if binding != "" {
				a.Config.Shortcuts[a.captureAction] = []string{binding}
				updateShortcutList(a.Settings.Shortcuts, a.captureAction, binding)
				a.setStatusMessage(fmt.Sprintf("Bound %s to %s", a.captureAction, binding))
				a.capturing = false
				a.captureAction = ""
				a.saveConfig()
				return nil
			}
		}
		if a.App.GetFocus() == a.Panels.TopBar.Timeline {
			if strings.TrimSpace(os.Getenv("TXP_KEY_DEBUG")) != "" {
				a.AppendLogf("info", "Key debug: key=%v rune=%q mods=%v", event.Key(), event.Rune(), event.Modifiers())
			}
		}
		if event.Key() == tcell.KeyEnter || (event.Key() == tcell.KeyRune && (event.Rune() == '\n' || event.Rune() == '\r')) {
			if a.App.GetFocus() == a.Panels.Library.Nav {
				a.NavPersistOnNextSelect = true
			}
		}

		if a.ResizeMode {
			switch event.Key() {
			case tcell.KeyLeft:
				a.resizeFocused(-5)
				return nil
			case tcell.KeyRight:
				a.resizeFocused(5)
				return nil
			case tcell.KeyEsc:
				a.ResizeMode = false
				a.setStatusMessage("Resize mode off")
				return nil
			}
		}

		if event.Key() == tcell.KeyCtrlK {
			a.togglePalette()
			return nil
		}

		if a.matchesShortcut("commandPalette", event) {
			a.togglePalette()
			return nil
		}
		if a.matchesShortcut("quit", event) {
			a.App.Stop()
			return nil
		}
		if event.Key() == tcell.KeyCtrlC {
			a.App.Stop()
			return nil
		}
		if a.matchesShortcut("resizeMode", event) {
			a.ResizeMode = !a.ResizeMode
			if a.ResizeMode {
				a.setStatusMessage("Resize mode: use left/right, Esc to exit")
			} else {
				a.setStatusMessage("Resize mode off")
			}
			return nil
		}

		if event.Key() == tcell.KeyCtrlG {
			a.toggleLogDialog()
			return nil
		}

		if event.Key() == tcell.KeyCtrlE {
			a.openTagEditor()
			return nil
		}

		if event.Key() == tcell.KeyCtrlF {
			if a.ViewMode == "tracks" {
				a.App.SetFocus(a.Panels.Tracks.Filter)
				return nil
			}
			if a.ViewMode == "library" {
				a.App.SetFocus(a.Panels.Library.NavFilter)
				return nil
			}
		}

		if event.Key() == tcell.KeyCtrlO {
			a.openFocusedTrackInExplorer()
			return nil
		}

		if event.Key() == tcell.KeyEsc {
			if a.closeTopOverlay() {
				return nil
			}
			if a.SettingsOpen {
				a.hideSettings()
				return nil
			}
		}

		if a.SettingsOpen && event.Key() == tcell.KeyRune {
			focus := a.App.GetFocus()
			if _, ok := focus.(*tview.InputField); ok {
				return event
			}
			switch event.Rune() {
			case '1':
				a.App.SetFocus(a.Settings.Libraries)
				a.setActiveSection("libraries")
				a.setLibrariesColumnFocus("left")
				return nil
			case '2':
				a.App.SetFocus(a.Settings.Shortcuts)
				a.setActiveSection("shortcuts")
				return nil
			case '3':
				a.App.SetFocus(a.Settings.Analysis)
				a.setActiveSection("analysis")
				return nil
			case '4':
				a.App.SetFocus(a.Settings.Themes)
				a.setActiveSection("themes")
				return nil
			}
		}

		if shouldBypassGlobalKeys(a, a.App.GetFocus()) {
			return event
		}

		if a.matchesShortcut("openFilters", event) {
			if a.ViewMode == "tracks" {
				a.App.SetFocus(a.Panels.Tracks.Filter)
				return nil
			}
			a.App.SetFocus(a.Panels.Library.NavFilter)
			return nil
		}
		if a.matchesShortcut("togglePlay", event) {
			focus := a.App.GetFocus()
			if focus == a.Panels.Library.Nav || focus == a.Panels.Tracks.List || focus == a.Panels.Library.Content {
				return event
			}
			a.togglePause()
			return nil
		}
		if a.matchesShortcut("nextTrack", event) {
			a.nextTrack()
			return nil
		}
		if a.matchesShortcut("prevTrack", event) {
			a.prevTrack()
			return nil
		}

		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case '+':
				a.adjustVolume(5)
				return nil
			case '=':
				a.adjustVolume(5)
				return nil
			case '-':
				a.adjustVolume(-5)
				return nil
			case '_':
				a.adjustVolume(-5)
				return nil
			}
		}

		if event.Key() == tcell.KeyCtrlT {
			a.updateViewMode("tracks")
			return nil
		}

		if event.Key() == tcell.KeyCtrlL {
			a.updateViewMode("library")
			return nil
		}

		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case '1':
				if a.SettingsOpen {
					a.App.SetFocus(a.Settings.Libraries)
					a.setActiveSection("libraries")
					a.setLibrariesColumnFocus("left")
					return nil
				}
				if a.ViewMode == "tracks" {
					a.App.SetFocus(a.Panels.Tracks.List)
					a.setMainFocus("tracks")
					return nil
				}
				a.App.SetFocus(a.Panels.Library.Nav)
				a.setMainFocus("nav")
				return nil
			case '2':
				if a.SettingsOpen {
					a.App.SetFocus(a.Settings.Shortcuts)
					a.setActiveSection("shortcuts")
					return nil
				}
				if a.ViewMode == "tracks" {
					a.App.SetFocus(a.Panels.Queue.List)
					a.setMainFocus("queue")
					return nil
				}
				a.App.SetFocus(a.Panels.Library.Content)
				a.setMainFocus("content")
				return nil
			case '3':
				if a.SettingsOpen {
					a.App.SetFocus(a.Settings.Analysis)
					a.setActiveSection("analysis")
					return nil
				}
				if a.ViewMode == "tracks" {
					if a.ShowTrackInfo {
						a.App.SetFocus(a.TrackInfo)
						a.setMainFocus("trackinfo")
						return nil
					}
					a.setStatusMessage("Track info hidden (press t)")
					return nil
				}
				if a.ViewMode == "library" {
					a.App.SetFocus(a.Panels.Queue.List)
					a.setMainFocus("queue")
					return nil
				}
			case '4':
				if a.SettingsOpen {
					a.App.SetFocus(a.Settings.Themes)
					a.setActiveSection("themes")
					return nil
				}
				if a.ViewMode == "library" {
					if a.ShowTrackInfo {
						a.App.SetFocus(a.TrackInfo)
						a.setMainFocus("trackinfo")
						return nil
					}
					a.setStatusMessage("Track info hidden (press t)")
					return nil
				}
			case '0':
				if !a.SettingsOpen {
					a.focusNowPlaying()
					return nil
				}
			}
		}

		if event.Rune() == '/' {
			if a.ViewMode == "tracks" {
				a.App.SetFocus(a.Panels.Tracks.Filter)
				return nil
			}
			a.App.SetFocus(a.Panels.Library.NavFilter)
			return nil
		}

		if event.Rune() == 't' {
			a.toggleRightPane()
			return nil
		}

		if event.Key() == tcell.KeyCtrlT {
			a.updateViewMode("tracks")
			return nil
		}

		if event.Key() == tcell.KeyCtrlL {
			a.updateViewMode("library")
			return nil
		}

		if event.Key() == tcell.KeyTab {
			a.cycleFocus()
			return nil
		}

		if event.Key() == tcell.KeyLeft {
			if a.Player != nil && a.NowPlayingTrack != nil && a.isSeekFocus() {
				a.requestSeek(-5)
				return nil
			}
		}
		if event.Key() == tcell.KeyRight {
			if a.Player != nil && a.NowPlayingTrack != nil && a.isSeekFocus() {
				a.requestSeek(5)
				return nil
			}
		}

		return event
	})
}

func shouldBypassGlobalKeys(a *App, focus tview.Primitive) bool {
	if a.Palette.Active || a.FilterDialog.Active || a.LogDialog.Active || a.LogFileDialog.Active || a.StatsDialog.Active || a.RescanDialog.Active || a.TaskDetails.Active || a.SettingsOpen || (a.TagEditor != nil && a.TagEditor.Active) {
		return true
	}
	if focus == nil {
		return false
	}
	if _, ok := focus.(*tview.InputField); ok {
		return true
	}
	return false
}

func formatKeybinding(event *tcell.EventKey) string {
	if event.Key() == tcell.KeyEsc {
		return ""
	}

	parts := []string{}
	if event.Modifiers()&tcell.ModMeta != 0 {
		parts = append(parts, "Cmd")
	}
	if event.Modifiers()&tcell.ModCtrl != 0 {
		parts = append(parts, "Ctrl")
	}
	if event.Modifiers()&tcell.ModAlt != 0 {
		parts = append(parts, "Alt")
	}
	if event.Modifiers()&tcell.ModShift != 0 {
		parts = append(parts, "Shift")
	}

	if event.Key() == tcell.KeyRune {
		if event.Rune() == ' ' {
			parts = append(parts, "Space")
		} else {
			parts = append(parts, strings.ToUpper(string(event.Rune())))
		}
		return strings.Join(parts, "+")
	}

	key := event.Key()
	switch key {
	case tcell.KeyCtrlA, tcell.KeyCtrlB, tcell.KeyCtrlC, tcell.KeyCtrlD, tcell.KeyCtrlE,
		tcell.KeyCtrlF, tcell.KeyCtrlG, tcell.KeyCtrlH, tcell.KeyCtrlI, tcell.KeyCtrlJ,
		tcell.KeyCtrlK, tcell.KeyCtrlL, tcell.KeyCtrlM, tcell.KeyCtrlN, tcell.KeyCtrlO,
		tcell.KeyCtrlP, tcell.KeyCtrlQ, tcell.KeyCtrlR, tcell.KeyCtrlS, tcell.KeyCtrlT,
		tcell.KeyCtrlU, tcell.KeyCtrlV, tcell.KeyCtrlW, tcell.KeyCtrlX, tcell.KeyCtrlY,
		tcell.KeyCtrlZ:
		if len(parts) == 0 || parts[0] != "Ctrl" {
			parts = append(parts, "Ctrl")
		}
		name := strings.TrimPrefix(tcell.KeyNames[key], "Ctrl+")
		parts = append(parts, strings.ToUpper(name))
	case tcell.KeyEnter:
		parts = append(parts, "Enter")
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		parts = append(parts, "Backspace")
	case tcell.KeyTab:
		parts = append(parts, "Tab")
	default:
		parts = append(parts, tcell.KeyNames[key])
	}
	return strings.Join(parts, "+")
}

func (a *App) matchesShortcut(action string, event *tcell.EventKey) bool {
	if a.Config.Shortcuts == nil {
		return false
	}
	values := a.Config.Shortcuts[action]
	if len(values) == 0 {
		return false
	}
	binding := formatKeybinding(event)
	if binding == "" {
		return false
	}
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), binding) {
			return true
		}
	}
	return false
}

func updateShortcutList(list *tview.List, action string, binding string) {
	count := list.GetItemCount()
	for i := 0; i < count; i++ {
		main, _ := list.GetItemText(i)
		if main == action {
			list.SetItemText(i, main, binding)
			return
		}
	}
}
