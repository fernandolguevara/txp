package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"txp/internal/config"
	"txp/internal/model"
	"txp/internal/storage"
	"txp/internal/ui/panels"
)

func (a *App) applyLibraryFilter() {
	if a.ViewMode != "library" {
		return
	}
	filter := strings.TrimSpace(a.LibraryFilterText)
	a.refreshLibraryNavFilter(filter)
	a.refreshLibraryTitles()
	if len(a.LibraryAll) == 0 {
		return
	}
	filtered := filterTracks(a.LibraryAll, a.LibraryFilterText, a.AdvancedFilter)
	a.LibraryData = filtered
	a.populateTrackRows(a.Panels.Library.Content, filtered, a.LibrarySelected, a.LibraryErrors)
	if len(filtered) > 0 {
		a.Panels.Library.Content.SetCurrentItem(0)
	}
	a.refreshLibraryTitles()
}

func (a *App) refreshLibraryNavFilter(filter string) {
	if a.db == nil || a.Panels == nil || a.Panels.Library == nil {
		return
	}
	filter = strings.ToLower(strings.TrimSpace(filter))
	roots := a.selectedLibraryPaths()
	if !a.libraryCacheLoaded {
		a.loadLibraryNavCache(filter)
		return
	}
	artists, albums, genres, years, ok := a.libraryCacheSnapshot()
	if !ok {
		a.loadLibraryNavCache(filter)
		return
	}
	go func() {
		counts, _ := storage.GetLibraryCountsInLibraries(a.db.DB, filter, roots)
		a.App.QueueUpdateDraw(func() {
			updateCategoryNode(a.Panels.Library.ArtistsNode, "artist", formatCategoryLabel("Artists", counts.Artists), filterValues(artists, filter))
			updateCategoryNode(a.Panels.Library.AlbumsNode, "album", formatCategoryLabel("Albums", counts.Albums), filterValues(albums, filter))
			updateCategoryNode(a.Panels.Library.FavoritesNode, "favorites", formatCategoryLabel("Favorites", counts.Favorites), nil)
			updateCategoryNode(a.Panels.Library.TracksNode, "tracks", formatCategoryLabel("Tracks", counts.Tracks), nil)
			updateCategoryNode(a.Panels.Library.GenresNode, "genre", formatCategoryLabel("Genres", counts.Genres), filterValues(genres, filter))
			updateCategoryNode(a.Panels.Library.YearsNode, "year", formatCategoryLabel("Years", counts.Years), filterValues(years, filter))
		})
	}()
}

func (a *App) libraryCacheSnapshot() ([]string, []string, []string, []string, bool) {
	a.libraryCacheMu.Lock()
	defer a.libraryCacheMu.Unlock()
	if !a.libraryCacheLoaded {
		return nil, nil, nil, nil, false
	}
	artists := append([]string(nil), a.libraryArtistsCache...)
	albums := append([]string(nil), a.libraryAlbumsCache...)
	genres := append([]string(nil), a.libraryGenresCache...)
	years := append([]string(nil), a.libraryYearsCache...)
	return artists, albums, genres, years, true
}

func (a *App) loadLibraryNavCache(filter string) {
	a.libraryCacheMu.Lock()
	if a.libraryCacheLoading {
		a.libraryCacheMu.Unlock()
		return
	}
	a.libraryCacheLoading = true
	a.libraryCacheMu.Unlock()

	roots := a.selectedLibraryPaths()
	go func() {
		artists, err := storage.ListDistinctArtistsInLibraries(a.db.DB, roots)
		if err != nil {
			a.finishLibraryCacheLoad(err)
			return
		}
		albums, err := storage.ListDistinctAlbumsInLibraries(a.db.DB, roots)
		if err != nil {
			a.finishLibraryCacheLoad(err)
			return
		}
		genres, err := storage.ListDistinctGenresInLibraries(a.db.DB, roots)
		if err != nil {
			a.finishLibraryCacheLoad(err)
			return
		}
		years, err := storage.ListDistinctYearsInLibraries(a.db.DB, roots)
		if err != nil {
			a.finishLibraryCacheLoad(err)
			return
		}
		a.App.QueueUpdateDraw(func() {
			a.libraryCacheMu.Lock()
			a.libraryArtistsCache = artists
			a.libraryAlbumsCache = albums
			a.libraryGenresCache = genres
			a.libraryYearsCache = years
			a.libraryCacheLoaded = true
			a.libraryCacheLoading = false
			a.libraryCacheMu.Unlock()
			a.refreshLibraryNavFilter(filter)
		})
	}()
}

func (a *App) finishLibraryCacheLoad(err error) {
	a.App.QueueUpdateDraw(func() {
		a.libraryCacheMu.Lock()
		a.libraryCacheLoading = false
		a.libraryCacheMu.Unlock()
		a.setStatusMessage("Failed to load library filters")
	})
}

func updateCategoryNode(node *tview.TreeNode, kind string, label string, values []string) {
	if node == nil {
		return
	}
	if label != "" {
		node.SetText(label)
	}
	expanded := node.IsExpanded()
	node.ClearChildren()
	for _, value := range values {
		child := tview.NewTreeNode(value)
		child.SetReference(&panels.NavRef{Kind: kind, Value: value})
		node.AddChild(child)
	}
	node.SetExpanded(expanded)
}

func filterValues(values []string, filter string) []string {
	if filter == "" {
		return values
	}
	if _, ok := extensionFilter(filter); ok {
		return values
	}
	filtered := []string{}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), filter) {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func (a *App) libraryNavTitle() string {
	count := a.TotalTracks
	if count < 0 {
		count = 0
	}
	return fmt.Sprintf("1 Library explorer #%d", count)
}

func (a *App) libraryContentTitle() string {
	count := len(a.LibraryData)
	if count < 0 {
		count = 0
	}
	return fmt.Sprintf("2 Content #%d", count)
}

func (a *App) refreshLibraryTitles() {
	if a.ViewMode != "library" {
		return
	}
	a.setPanelStyle(a.Panels.Library.NavPanel, a.libraryNavTitle(), a.ActivePanel == "nav")
	a.setPanelStyle(a.Panels.Library.ContentPanel, a.libraryContentTitle(), a.ActivePanel == "content")
	if a.Panels.Library.NavHeader != nil {
		a.Panels.Library.NavHeader.SetText(formatTracksHeaderNoSort(a.TotalTracks))
	}
	if a.Panels.Library.ContentHeader != nil {
		a.Panels.Library.ContentHeader.SetText(formatTracksHeaderNoSort(len(a.LibraryData)))
	}
}

func (a *App) refreshLibraryTotals() {
	if a.db == nil {
		return
	}
	roots := a.selectedLibraryPaths()
	go func() {
		counts, err := storage.GetLibraryCountsInLibraries(a.db.DB, "", roots)
		a.App.QueueUpdateDraw(func() {
			if err != nil {
				return
			}
			a.TotalTracks = counts.Tracks
			a.resetLibraryNavCache()
			a.refreshLibraryTitles()
		})
	}()
}

func (a *App) resetLibraryNavCache() {
	a.libraryCacheMu.Lock()
	a.libraryCacheLoaded = false
	a.libraryArtistsCache = nil
	a.libraryAlbumsCache = nil
	a.libraryGenresCache = nil
	a.libraryYearsCache = nil
	a.libraryCacheMu.Unlock()
}

func (a *App) restoreLibraryNavCursor() {
	if a.Config.LibraryNavCursor == nil {
		return
	}
	if a.Panels == nil || a.Panels.Library == nil || a.Panels.Library.Nav == nil {
		return
	}
	cursor := a.Config.LibraryNavCursor
	if cursor.Path != "" {
		if a.Panels.Library.LibraryGroup == nil {
			return
		}
		for _, child := range a.Panels.Library.LibraryGroup.GetChildren() {
			ref, ok := child.GetReference().(*panels.LibraryRef)
			if !ok || ref.Path == "" {
				continue
			}
			if ref.Path == cursor.Path {
				a.NavRestoreInProgress = true
				a.Panels.Library.Nav.SetCurrentNode(child)
				a.NavRestoreInProgress = false
				return
			}
		}
	}
	if cursor.Kind == "" {
		return
	}
	node := a.libraryCategoryNode(cursor.Kind)
	if node == nil {
		return
	}
	if cursor.Value == "" {
		a.NavRestoreInProgress = true
		a.Panels.Library.Nav.SetCurrentNode(node)
		a.NavRestoreInProgress = false
		return
	}
	if len(node.GetChildren()) == 0 {
		kind := cursor.Kind
		value := cursor.Value
		go func() {
			values, err := a.fetchCategoryValues(kind, a.selectedLibraryPaths())
			a.App.QueueUpdateDraw(func() {
				if err != nil {
					return
				}
				node.ClearChildren()
				for _, item := range values {
					child := tview.NewTreeNode(item)
					child.SetReference(&panels.NavRef{Kind: kind, Value: item})
					node.AddChild(child)
				}
				node.SetExpanded(true)
				a.selectLibraryNavChild(node, value)
			})
		}()
		return
	}
	node.SetExpanded(true)
	a.selectLibraryNavChild(node, cursor.Value)
}

func (a *App) libraryCategoryNode(kind string) *tview.TreeNode {
	if a.Panels == nil || a.Panels.Library == nil {
		return nil
	}
	switch kind {
	case "artist":
		return a.Panels.Library.ArtistsNode
	case "album":
		return a.Panels.Library.AlbumsNode
	case "favorites":
		return a.Panels.Library.FavoritesNode
	case "tracks":
		return a.Panels.Library.TracksNode
	case "genre":
		return a.Panels.Library.GenresNode
	case "year":
		return a.Panels.Library.YearsNode
	}
	return nil
}

func (a *App) selectLibraryNavChild(node *tview.TreeNode, value string) {
	if node == nil || value == "" {
		return
	}
	for _, child := range node.GetChildren() {
		if child == nil {
			continue
		}
		if child.GetText() == value {
			a.NavRestoreInProgress = true
			a.Panels.Library.Nav.SetCurrentNode(child)
			a.NavRestoreInProgress = false
			return
		}
	}
	a.NavRestoreInProgress = true
	a.Panels.Library.Nav.SetCurrentNode(node)
	a.NavRestoreInProgress = false
}

func (a *App) reloadLibrarySelection() {
	if a.Panels == nil || a.Panels.Library == nil || a.Panels.Library.Nav == nil {
		return
	}
	node := a.Panels.Library.Nav.GetCurrentNode()
	if node == nil {
		return
	}
	ref, ok := node.GetReference().(*panels.NavRef)
	if !ok {
		return
	}
	a.handleLibrarySelection(node, ref)
}

func (a *App) handleLibrarySelection(node *tview.TreeNode, ref *panels.NavRef) {
	if a.ViewMode != "library" {
		return
	}
	roots := a.selectedLibraryPaths()
	if len(roots) == 0 {
		a.LibraryAll = nil
		a.LibraryData = nil
		a.LibraryErrors = nil
		a.populateTrackRows(a.Panels.Library.Content, nil, a.LibrarySelected, a.LibraryErrors)
		a.refreshLibraryTitles()
		a.setStatusMessage("No libraries selected")
		return
	}
	if ref.Kind == "tracks" {
		go func() {
			tracks, err := storage.ListTracksInLibraries(a.db.DB, roots)
			var trackErrors map[string]string
			if err == nil {
				trackErrors, _ = storage.ListTrackErrors(a.db.DB, tracksToPaths(tracks))
			}
			a.App.QueueUpdateDraw(func() {
				if err != nil {
					a.setStatusMessage(fmt.Sprintf("Failed to load tracks: %v", err))
					return
				}
				a.LibraryAll = tracks
				a.LibraryErrors = trackErrors
				a.applyLibraryFilter()
				a.refreshLibraryTitles()
				a.setStatusMessage(fmt.Sprintf("Tracks loaded: %d", len(tracks)))
			})
		}()
		return
	}
	if ref.Kind == "favorites" {
		go func() {
			tracks, err := storage.ListFavoriteTracksInLibraries(a.db.DB, roots)
			var trackErrors map[string]string
			if err == nil {
				trackErrors, _ = storage.ListTrackErrors(a.db.DB, tracksToPaths(tracks))
			}
			a.App.QueueUpdateDraw(func() {
				if err != nil {
					a.setStatusMessage(fmt.Sprintf("Failed to load favorites: %v", err))
					return
				}
				a.LibraryAll = tracks
				a.LibraryErrors = trackErrors
				a.applyLibraryFilter()
				a.refreshLibraryTitles()
				a.setStatusMessage(fmt.Sprintf("Favorites loaded: %d", len(tracks)))
			})
		}()
		return
	}
	if ref.Value == "" {
		if node.IsExpanded() {
			node.SetExpanded(false)
			a.Panels.Library.Nav.SetCurrentNode(node)
			return
		}
		a.populateCategory(node, ref.Kind)
		a.Panels.Library.Nav.SetCurrentNode(node)
		return
	}
	go func() {
		tracks, err := a.fetchTracksByFilter(ref.Kind, ref.Value, roots)
		var trackErrors map[string]string
		if err == nil {
			trackErrors, _ = storage.ListTrackErrors(a.db.DB, tracksToPaths(tracks))
		}
		a.App.QueueUpdateDraw(func() {
			if err != nil {
				a.setStatusMessage("Failed to load tracks")
				return
			}
			a.LibraryAll = tracks
			a.LibraryErrors = trackErrors
			a.applyLibraryFilter()
			a.refreshLibraryTitles()
		})
	}()
}

func (a *App) enqueueSelectionFromNav(ref *panels.NavRef) {
	if a.db == nil {
		return
	}
	go func() {
		var tracks []model.Track
		var err error
		roots := a.selectedLibraryPaths()
		switch ref.Kind {
		case "tracks":
			tracks, err = storage.ListTracksInLibraries(a.db.DB, roots)
		case "artist", "album", "genre", "year":
			if ref.Value == "" {
				return
			}
			tracks, err = a.fetchTracksByFilter(ref.Kind, ref.Value, roots)
		default:
			return
		}
		a.App.QueueUpdateDraw(func() {
			if err != nil {
				a.setStatusMessage("Failed to add tracks")
				return
			}
			if len(tracks) == 0 {
				a.setStatusMessage("No tracks to add")
				return
			}
			a.addTracksToQueue(tracks)
			if len(tracks) == 1 {
				a.setStatusMessage("Added 1 track to queue")
				return
			}
			a.setStatusMessage(fmt.Sprintf("Added %d tracks to queue", len(tracks)))
		})
	}()
}

func (a *App) populateCategory(node *tview.TreeNode, kind string) {
	if a.db == nil {
		return
	}
	if len(node.GetChildren()) > 0 {
		node.SetExpanded(true)
		return
	}
	go func() {
		values, err := a.fetchCategoryValues(kind, a.selectedLibraryPaths())
		a.App.QueueUpdateDraw(func() {
			if err != nil {
				a.setStatusMessage("Failed to load category")
				return
			}
			node.ClearChildren()
			for _, value := range values {
				child := tview.NewTreeNode(value)
				child.SetReference(&panels.NavRef{Kind: kind, Value: value})
				node.AddChild(child)
			}
			node.SetExpanded(true)
			a.NavRestoreInProgress = true
			a.Panels.Library.Nav.SetCurrentNode(node)
			a.NavRestoreInProgress = false
		})
	}()
}

func (a *App) fetchCategoryValues(kind string, roots []string) ([]string, error) {
	if a.db == nil {
		return nil, fmt.Errorf("db missing")
	}
	switch kind {
	case "artist":
		return storage.ListDistinctArtistsInLibraries(a.db.DB, roots)
	case "album":
		return storage.ListDistinctAlbumsInLibraries(a.db.DB, roots)
	case "genre":
		return storage.ListDistinctGenresInLibraries(a.db.DB, roots)
	case "year":
		return storage.ListDistinctYearsInLibraries(a.db.DB, roots)
	default:
		return storage.ListDistinctArtistsInLibraries(a.db.DB, roots)
	}
}

func (a *App) fetchTracksByFilter(kind string, value string, roots []string) ([]model.Track, error) {
	if a.db == nil {
		return nil, fmt.Errorf("db missing")
	}
	switch kind {
	case "artist":
		return storage.ListTracksByArtistInLibraries(a.db.DB, value, roots)
	case "album":
		return storage.ListTracksByAlbumInLibraries(a.db.DB, value, roots)
	case "favorites":
		return storage.ListFavoriteTracksInLibraries(a.db.DB, roots)
	case "genre":
		return storage.ListTracksByGenreInLibraries(a.db.DB, value, roots)
	case "year":
		return storage.ListTracksByYearInLibraries(a.db.DB, value, roots)
	default:
		return storage.ListTracksInLibraries(a.db.DB, roots)
	}
}

func (a *App) addLibraryFromTree(path string) {
	normalized := config.NormalizeLibraryPath(path)
	if normalized == "" {
		return
	}
	if a.hasLibraryPath(normalized) {
		a.setStatusMessage("Already in list")
		return
	}

	a.Config.Libraries = append([]string{normalized}, a.Config.Libraries...)
	a.Settings.Libraries.InsertItem(0, normalized, "", 0, nil)
	a.addLibraryToNavigation(normalized)
	a.setStatusMessage("Library added")
	a.syncSelectedLibraries()
	a.saveConfig()
}

func (a *App) addLibraryFromPath(path string) error {
	normalized := config.NormalizeLibraryPath(path)
	if normalized == "" {
		return fmt.Errorf("Path is required")
	}
	info, err := os.Stat(normalized)
	if err != nil {
		return fmt.Errorf("Path does not exist")
	}
	if !info.IsDir() {
		return fmt.Errorf("Path is not a directory")
	}
	if a.hasLibraryPath(normalized) {
		return fmt.Errorf("Already in list")
	}

	a.Config.Libraries = append([]string{normalized}, a.Config.Libraries...)
	a.Settings.Libraries.InsertItem(0, normalized, "", 0, nil)
	a.addLibraryToNavigation(normalized)
	a.setStatusMessage("Library added")
	a.syncSelectedLibraries()
	a.saveConfig()
	return nil
}

func (a *App) showAddLibraryPathDialog() {
	form := tview.NewForm()
	input := tview.NewInputField().SetLabel("Full path: ")
	input.SetText("")
	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlV || (event.Rune() == 'v' && event.Modifiers()&tcell.ModMeta != 0) {
			a.appendClipboard(input)
			return nil
		}
		return event
	})
	form.AddFormItem(input)

	errorText := tview.NewTextView().SetText("")

	form.AddButton("Save", func() {
		if err := a.addLibraryFromPath(strings.TrimSpace(input.GetText())); err != nil {
			errorText.SetText(err.Error())
			a.App.SetFocus(input)
			return
		}
		a.hideAddLibraryPathDialog()
	})
	form.AddButton("Cancel", func() {
		a.hideAddLibraryPathDialog()
	})
	form.SetBorder(true).SetTitle("[ Add library path ]")

	modal := tview.NewFlex().SetDirection(tview.FlexRow)
	modal.AddItem(form, 0, 1, true)
	modal.AddItem(errorText, 1, 0, false)
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			a.hideAddLibraryPathDialog()
			return nil
		}
		return event
	})

	a.pushOverlay("add-library-path", centeredOverlay(modal, 8, 70), true, nil)
	a.App.SetFocus(input)
}

func (a *App) hideAddLibraryPathDialog() {
	if entry, ok := a.popOverlay("add-library-path"); ok {
		a.restoreOverlayFocus(entry)
		return
	}
	if a.Settings != nil {
		a.App.SetFocus(a.Settings.Libraries)
	}
}

func (a *App) hasLibraryPath(path string) bool {
	for _, lib := range a.Config.Libraries {
		if lib == path {
			return true
		}
	}
	return false
}

func (a *App) resetLibraryTree(node *tview.TreeNode) {
	if node == nil {
		return
	}
	path, ok := node.GetReference().(string)
	if !ok || path == "" {
		return
	}
	root := tview.NewTreeNode(path)
	root.SetReference(path)
	root.SetExpanded(true)
	panels.PopulateSettingsDirectoryChildren(root)
	a.Settings.LibraryTree.SetRoot(root).SetCurrentNode(root)
}

func (a *App) addLibraryToNavigation(path string) {
	if a.Panels.Library.LibraryGroup == nil {
		return
	}
	node := panels.NewLibraryNode(path, false)
	a.Panels.Library.LibraryGroup.AddChild(node)
	a.Panels.Library.Nav.SetCurrentNode(node)
	a.SelectedLibs[path] = false
}

func (a *App) removeLibraryFromNavigation(path string) {
	if a.Panels.Library.LibraryGroup == nil {
		return
	}
	children := a.Panels.Library.LibraryGroup.GetChildren()
	for _, child := range children {
		ref, ok := child.GetReference().(*panels.LibraryRef)
		if !ok || ref.Path == "" {
			continue
		}
		if ref.Path == path {
			a.Panels.Library.LibraryGroup.RemoveChild(child)
			delete(a.SelectedLibs, path)
			a.syncSelectedLibraries()
			return
		}
	}
}

func (a *App) loadSelectedLibraries() {
	for _, path := range a.Config.SelectedLibraries {
		a.SelectedLibs[path] = true
	}
	if a.Panels.Library.LibraryGroup == nil {
		return
	}
	for _, child := range a.Panels.Library.LibraryGroup.GetChildren() {
		ref, ok := child.GetReference().(*panels.LibraryRef)
		if !ok || ref.Path == "" {
			continue
		}
		panels.UpdateLibraryNodeText(child, ref.Label, a.SelectedLibs[ref.Path])
	}
}

func (a *App) syncSelectedLibraries() {
	selected := []string{}
	for _, lib := range a.Config.Libraries {
		if a.SelectedLibs[lib] {
			selected = append(selected, lib)
		}
	}
	a.Config.SelectedLibraries = selected
}

func (a *App) selectedCount() int {
	count := 0
	for _, selected := range a.SelectedLibs {
		if selected {
			count++
		}
	}
	return count
}

func (a *App) selectedPaths() []string {
	paths := []string{}
	for path, selected := range a.SelectedLibs {
		if selected {
			paths = append(paths, path)
		}
	}
	return paths
}

func (a *App) selectedLibraryPaths() []string {
	raw := a.selectedPaths()
	if len(raw) == 0 {
		return nil
	}
	seen := map[string]bool{}
	paths := []string{}
	for _, path := range raw {
		normalized := config.NormalizeLibraryPath(path)
		if normalized == "" {
			continue
		}
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		paths = append(paths, normalized)
	}
	return paths
}
