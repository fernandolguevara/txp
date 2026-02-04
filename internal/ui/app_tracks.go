package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rivo/tview"

	"txp/internal/model"
	"txp/internal/storage"
)

type FilterCriteria struct {
	Text   string
	Artist string
	Album  string
	Genre  string
	Year   string
	Key    string
}

func (a *App) refreshTracks() {
	if a.db == nil {
		return
	}
	roots := a.selectedLibraryPaths()
	if len(roots) == 0 {
		a.TracksAll = nil
		a.TracksData = nil
		a.TracksErrors = nil
		for key := range a.TracksSelected {
			delete(a.TracksSelected, key)
		}
		a.applyTracksFilter()
		a.TotalTracks = 0
		a.Panels.Tracks.Header.SetText(formatTracksHeaderNoSort(0))
		if a.Panels.Tracks.SortInfo != nil {
			a.Panels.Tracks.SortInfo.SetText(formatSortInfo(a.sortMode, a.sortAsc))
		}
		a.refreshLibraryTitles()
		a.setStatusMessage("No libraries selected")
		return
	}
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
			sorted := a.sortTracks(tracks)
			a.TracksAll = sorted
			a.TracksErrors = trackErrors
			a.resetLibraryNavCache()
			a.applyTracksFilter()
			a.TotalTracks = len(sorted)
			a.Panels.Tracks.Header.SetText(formatTracksHeaderNoSort(len(a.TracksData)))
			if a.Panels.Tracks.SortInfo != nil {
				a.Panels.Tracks.SortInfo.SetText(formatSortInfo(a.sortMode, a.sortAsc))
			}
			a.setPanelStyle(a.Panels.Tracks.Container, a.tracksPanelTitle(), a.ViewMode == "tracks" && a.ActivePanel == "tracks")
			a.refreshLibraryTitles()
			a.setStatusMessage(fmt.Sprintf("Tracks loaded: %d", len(sorted)))
		})
	}()
}

func (a *App) populateTrackRows(list *tview.List, tracks []model.Track, selected map[int]bool, errors map[string]string) {
	list.Clear()
	list.ShowSecondaryText(false)
	for key := range selected {
		delete(selected, key)
	}
	a.setIndexWidthForList(list, len(tracks))
	indexWidth := a.indexWidthForList(list)
	for i, track := range tracks {
		highlighted := listItemHighlighted(list, i)
		row := formatTrackRow(list, indexWidth, i, track, selected != nil && selected[i], hasTrackError(errors, track.Path), highlighted, a.Theme.Accent, a.Theme.Fg)
		list.AddItem(row, "", 0, nil)
	}
}

func (a *App) updateTrackListLayout() {
	a.updateTrackListWidth(a.Panels.Tracks.List, a.TracksData, a.TracksSelected, a.TracksErrors, &a.tracksListWidth)
	a.updateTrackListWidth(a.Panels.Library.Content, a.LibraryData, a.LibrarySelected, a.LibraryErrors, &a.libraryListWidth)
}

func (a *App) updateTrackListWidth(list *tview.List, tracks []model.Track, selected map[int]bool, errors map[string]string, widthRef *int) {
	if list == nil {
		return
	}
	_, _, width, _ := list.GetInnerRect()
	if width <= 0 || width == *widthRef {
		return
	}
	*widthRef = width
	a.refreshTrackRowLayout(list, tracks, selected, errors)
}

func (a *App) refreshTrackRowLayout(list *tview.List, tracks []model.Track, selected map[int]bool, errors map[string]string) {
	if list == nil {
		return
	}
	count := min(list.GetItemCount(), len(tracks))
	indexWidth := a.indexWidthForList(list)
	for i := 0; i < count; i++ {
		highlighted := listItemHighlighted(list, i)
		row := formatTrackRow(list, indexWidth, i, tracks[i], selected != nil && selected[i], hasTrackError(errors, tracks[i].Path), highlighted, a.Theme.Accent, a.Theme.Fg)
		_, secondary := list.GetItemText(i)
		list.SetItemText(i, row, secondary)
	}
}

func (a *App) updateTrackScrollBar() {
	if a.Panels.Tracks.Scroll == nil || a.Panels.Tracks.List == nil {
		return
	}
	_, _, _, height := a.Panels.Tracks.List.GetInnerRect()
	if height <= 0 {
		return
	}
	items := a.Panels.Tracks.List.GetItemCount()
	offset, _ := a.Panels.Tracks.List.GetOffset()
	a.Panels.Tracks.Scroll.SetText(buildScrollBar(height, items, offset))
}

func (a *App) updateTrackInfoFromList(list string, index int) {
	if a.TrackInfo == nil {
		return
	}
	_, tracks, _ := a.selectionState(list)
	if index < 0 || index >= len(tracks) {
		a.updateTrackInfo(nil)
		return
	}
	track := tracks[index]
	a.updateTrackInfo(&track)
}

func (a *App) updateTrackInfo(track *model.Track) {
	if a.TrackInfo == nil {
		return
	}
	if track == nil {
		a.TrackInfo.SetText("No track selected")
		return
	}
	name := escapeUserText(formatTrackName(track.Title, track.Artist, track.Path))
	artist := track.Artist
	if strings.TrimSpace(artist) == "" {
		artist = "-"
	}
	album := track.Album
	if strings.TrimSpace(album) == "" {
		album = "-"
	}
	genre := track.Genre
	if strings.TrimSpace(genre) == "" {
		genre = "-"
	}
	key := track.Key
	if strings.TrimSpace(key) == "" {
		key = "-"
	}
	favorite := "No"
	if track.Favorite {
		favorite = "Yes"
	}
	path := track.Path
	if strings.TrimSpace(path) == "" {
		path = "-"
	}
	year := "-"
	if track.Year > 0 {
		year = fmt.Sprintf("%d", track.Year)
	}
	trackNum := "-"
	if track.TrackNum > 0 {
		trackNum = fmt.Sprintf("%d", track.TrackNum)
	}
	length := "-"
	if track.Duration > 0 {
		length = formatDuration(track.Duration)
	}
	bpm := "-"
	if track.BPM > 0 {
		bpm = fmt.Sprintf("%.1f", track.BPM)
	}

	artist = escapeUserText(artist)
	album = escapeUserText(album)
	genre = escapeUserText(genre)
	key = escapeUserText(key)
	path = escapeUserText(path)

	lines := []string{
		fmt.Sprintf("Title: %s", name),
		fmt.Sprintf("Artist: %s", artist),
		fmt.Sprintf("Album: %s", album),
		fmt.Sprintf("Duration: %s", length),
		fmt.Sprintf("Genre: %s", genre),
		fmt.Sprintf("Year: %s", year),
		fmt.Sprintf("Track #: %s", trackNum),
		fmt.Sprintf("BPM: %s", bpm),
		fmt.Sprintf("Key: %s", key),
		fmt.Sprintf("Favorite: %s", favorite),
		fmt.Sprintf("Path: %s", path),
	}
	if errMsg := a.trackErrorMessage(track.Path); errMsg != "" {
		lines = append(lines, fmt.Sprintf("Error: %s", escapeUserText(errMsg)))
	}

	a.TrackInfo.SetText(strings.Join(lines, "\n"))
}

func (a *App) toggleSelection(list string, index int) {
	if index < 0 {
		return
	}
	selected, tracks, targetList := a.selectionState(list)
	if selected == nil || tracks == nil || targetList == nil {
		return
	}
	if index >= len(tracks) {
		return
	}
	if selected[index] {
		delete(selected, index)
	} else {
		selected[index] = true
	}
	track := tracks[index]
	highlighted := listItemHighlighted(targetList, index)
	row := formatTrackRow(targetList, a.indexWidthForList(targetList), index, track, selected[index], hasTrackError(a.trackErrorsForList(list), track.Path), highlighted, a.Theme.Accent, a.Theme.Fg)
	_, secondary := targetList.GetItemText(index)
	targetList.SetItemText(index, row, secondary)
}

func (a *App) handleListDoubleClick(listID string, index int, play func()) {
	if !a.DoubleClickPlayback {
		a.lastClickWasMouse = false
		return
	}
	if !a.lastClickWasMouse {
		return
	}
	threshold := 400 * time.Millisecond
	now := time.Now()
	if a.lastClickList == listID && a.lastClickIndex == index && now.Sub(a.lastClickAt) <= threshold {
		play()
	}
	a.lastClickAt = now
	a.lastClickList = listID
	a.lastClickIndex = index
	a.lastClickWasMouse = false
}

func filterTracks(tracks []model.Track, text string, criteria FilterCriteria) []model.Track {
	filter := strings.ToLower(strings.TrimSpace(text))
	filtered := []model.Track{}
	for _, track := range tracks {
		if trackMatchesFilter(track, filter, criteria) {
			filtered = append(filtered, track)
		}
	}
	return filtered
}

func trackMatchesFilter(track model.Track, filter string, criteria FilterCriteria) bool {
	if filter == "" {
		return matchesAdvancedFilter(track, criteria)
	}
	if ext, ok := extensionFilter(filter); ok {
		if !strings.HasSuffix(strings.ToLower(track.Path), ext) {
			return false
		}
		return matchesAdvancedFilter(track, criteria)
	}
	fields := []string{track.Title, track.Artist, track.Album, track.Genre, track.Path}
	if track.Year > 0 {
		fields = append(fields, fmt.Sprintf("%d", track.Year))
	}
	matched := false
	for _, value := range fields {
		if strings.Contains(strings.ToLower(value), filter) {
			matched = true
			break
		}
	}
	if !matched {
		return false
	}
	return matchesAdvancedFilter(track, criteria)
}

func extensionFilter(filter string) (string, bool) {
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

func (a *App) applyTracksFilter() {
	filtered := filterTracks(a.TracksAll, a.TracksFilterText, a.AdvancedFilter)
	a.TracksData = filtered
	a.populateTrackRows(a.Panels.Tracks.List, filtered, a.TracksSelected, a.TracksErrors)
	if len(filtered) > 0 {
		a.Panels.Tracks.List.SetCurrentItem(0)
	}
	a.Panels.Tracks.Header.SetText(formatTracksHeaderNoSort(len(filtered)))
	if a.Panels.Tracks.SortInfo != nil {
		a.Panels.Tracks.SortInfo.SetText(formatSortInfo(a.sortMode, a.sortAsc))
	}
}

func (a *App) setTrackSort(mode string) {
	if mode == "" {
		return
	}
	if a.sortMode == mode {
		a.sortAsc = !a.sortAsc
	} else {
		a.sortMode = mode
		a.sortAsc = true
	}
	a.applyTrackSort()
}

func (a *App) applyTrackSort() {
	if len(a.TracksAll) == 0 {
		a.Panels.Tracks.Header.SetText(formatTracksHeaderNoSort(0))
		if a.Panels.Tracks.SortInfo != nil {
			a.Panels.Tracks.SortInfo.SetText(formatSortInfo(a.sortMode, a.sortAsc))
		}
		return
	}
	sorted := a.sortTracks(a.TracksAll)
	a.TracksAll = sorted
	a.applyTracksFilter()
	a.setStatusMessage(fmt.Sprintf("Sorted by %s", formatSortLabel(a.sortMode, a.sortAsc)))
}

func (a *App) sortTracks(tracks []model.Track) []model.Track {
	if len(tracks) < 2 {
		return tracks
	}
	sorted := append([]model.Track(nil), tracks...)
	mode := a.sortMode
	asc := a.sortAsc
	sort.SliceStable(sorted, func(i, j int) bool {
		left := sorted[i]
		right := sorted[j]
		switch mode {
		case "bpm":
			return compareFloat(left.BPM, right.BPM, asc)
		case "key":
			return compareString(left.Key, right.Key, asc)
		case "length":
			return compareFloat(left.Duration, right.Duration, asc)
		case "artist":
			return compareString(left.Artist, right.Artist, asc)
		case "title":
			fallthrough
		default:
			return compareString(trackTitleKey(left), trackTitleKey(right), asc)
		}
	})
	return sorted
}

func trackTitleKey(track model.Track) string {
	name := strings.TrimSpace(track.Title)
	if name == "" {
		name = trimExtension(track.Path)
	}
	return strings.ToLower(name)
}

func matchesAdvancedFilter(track model.Track, criteria FilterCriteria) bool {
	text := strings.ToLower(strings.TrimSpace(criteria.Text))
	artist := strings.ToLower(strings.TrimSpace(criteria.Artist))
	album := strings.ToLower(strings.TrimSpace(criteria.Album))
	genre := strings.ToLower(strings.TrimSpace(criteria.Genre))
	key := strings.ToLower(strings.TrimSpace(criteria.Key))
	year := strings.TrimSpace(criteria.Year)

	if text != "" {
		fields := []string{track.Title, track.Artist, track.Album, track.Path}
		matched := false
		for _, value := range fields {
			if strings.Contains(strings.ToLower(value), text) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if artist != "" && !strings.Contains(strings.ToLower(track.Artist), artist) {
		return false
	}
	if album != "" && !strings.Contains(strings.ToLower(track.Album), album) {
		return false
	}
	if genre != "" && !strings.Contains(strings.ToLower(track.Genre), genre) {
		return false
	}
	if key != "" && !strings.Contains(strings.ToLower(track.Key), key) {
		return false
	}
	if year != "" {
		value, err := strconv.Atoi(year)
		if err != nil || track.Year != value {
			return false
		}
	}
	return true
}

func compareFloat(left float64, right float64, asc bool) bool {
	leftMissing := left <= 0
	rightMissing := right <= 0
	if leftMissing != rightMissing {
		return !leftMissing
	}
	if left == right {
		return false
	}
	if asc {
		return left < right
	}
	return left > right
}

func compareString(left string, right string, asc bool) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	leftMissing := left == ""
	rightMissing := right == ""
	if leftMissing != rightMissing {
		return !leftMissing
	}
	left = strings.ToLower(left)
	right = strings.ToLower(right)
	if left == right {
		return false
	}
	if asc {
		return left < right
	}
	return left > right
}

func (a *App) tracksPanelTitle() string {
	count := len(a.TracksData)
	if count < 0 {
		count = 0
	}
	return fmt.Sprintf("1 Tracks #%d", count)
}

func tracksToPaths(tracks []model.Track) []string {
	paths := make([]string, 0, len(tracks))
	for _, track := range tracks {
		if track.Path == "" {
			continue
		}
		paths = append(paths, track.Path)
	}
	return paths
}

func hasTrackError(errors map[string]string, path string) bool {
	if errors == nil || path == "" {
		return false
	}
	value, ok := errors[path]
	if !ok {
		return false
	}
	return strings.TrimSpace(value) != ""
}

func (a *App) trackErrorsForList(list string) map[string]string {
	switch list {
	case "tracks":
		return a.TracksErrors
	case "library":
		return a.LibraryErrors
	default:
		return nil
	}
}

func (a *App) trackErrorMessage(path string) string {
	if path == "" {
		return ""
	}
	if a.TracksErrors != nil {
		if value := strings.TrimSpace(a.TracksErrors[path]); value != "" {
			return value
		}
	}
	if a.LibraryErrors != nil {
		if value := strings.TrimSpace(a.LibraryErrors[path]); value != "" {
			return value
		}
	}
	if a.db == nil {
		return ""
	}
	value, err := storage.GetTrackError(a.db.DB, path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func (a *App) toggleFavorite(track model.Track) {
	if a.db == nil || track.Path == "" {
		return
	}
	value, err := storage.ToggleTrackFavorite(a.db.DB, track.Path)
	if err != nil {
		a.setStatusMessage("Failed to toggle favorite")
		return
	}
	a.updateFavoriteInSlices(track.Path, value)
	label := "Unfavorited"
	if value {
		label = "Favorited"
	}
	a.setStatusMessage(fmt.Sprintf("%s: %s", label, escapeUserText(formatTrackName(track.Title, track.Artist, track.Path))))
}

func (a *App) updateFavoriteInSlices(path string, value bool) {
	for i := range a.TracksAll {
		if a.TracksAll[i].Path == path {
			a.TracksAll[i].Favorite = value
		}
	}
	for i := range a.TracksData {
		if a.TracksData[i].Path == path {
			a.TracksData[i].Favorite = value
		}
	}
	for i := range a.LibraryAll {
		if a.LibraryAll[i].Path == path {
			a.LibraryAll[i].Favorite = value
		}
	}
	for i := range a.LibraryData {
		if a.LibraryData[i].Path == path {
			a.LibraryData[i].Favorite = value
		}
	}
	for i := range a.QueueItems {
		if a.QueueItems[i].Path == path {
			a.QueueItems[i].Favorite = value
		}
	}
	if a.NowPlayingTrack != nil && a.NowPlayingTrack.Path == path {
		a.NowPlayingTrack.Favorite = value
	}
}

func (a *App) updateTrackTagsInSlices(updated model.Track) {
	path := updated.Path
	if path == "" {
		return
	}
	for i := range a.TracksAll {
		if a.TracksAll[i].Path == path {
			a.TracksAll[i].Title = updated.Title
			a.TracksAll[i].Artist = updated.Artist
			a.TracksAll[i].Album = updated.Album
			a.TracksAll[i].Genre = updated.Genre
			a.TracksAll[i].Year = updated.Year
			a.TracksAll[i].TrackNum = updated.TrackNum
		}
	}
	for i := range a.TracksData {
		if a.TracksData[i].Path == path {
			a.TracksData[i].Title = updated.Title
			a.TracksData[i].Artist = updated.Artist
			a.TracksData[i].Album = updated.Album
			a.TracksData[i].Genre = updated.Genre
			a.TracksData[i].Year = updated.Year
			a.TracksData[i].TrackNum = updated.TrackNum
		}
	}
	for i := range a.LibraryAll {
		if a.LibraryAll[i].Path == path {
			a.LibraryAll[i].Title = updated.Title
			a.LibraryAll[i].Artist = updated.Artist
			a.LibraryAll[i].Album = updated.Album
			a.LibraryAll[i].Genre = updated.Genre
			a.LibraryAll[i].Year = updated.Year
			a.LibraryAll[i].TrackNum = updated.TrackNum
		}
	}
	for i := range a.LibraryData {
		if a.LibraryData[i].Path == path {
			a.LibraryData[i].Title = updated.Title
			a.LibraryData[i].Artist = updated.Artist
			a.LibraryData[i].Album = updated.Album
			a.LibraryData[i].Genre = updated.Genre
			a.LibraryData[i].Year = updated.Year
			a.LibraryData[i].TrackNum = updated.TrackNum
		}
	}
	for i := range a.QueueItems {
		if a.QueueItems[i].Path == path {
			a.QueueItems[i].Title = updated.Title
			a.QueueItems[i].Artist = updated.Artist
			a.QueueItems[i].Album = updated.Album
			a.QueueItems[i].Genre = updated.Genre
			a.QueueItems[i].Year = updated.Year
			a.QueueItems[i].TrackNum = updated.TrackNum
		}
	}
	if a.NowPlayingTrack != nil && a.NowPlayingTrack.Path == path {
		a.NowPlayingTrack.Title = updated.Title
		a.NowPlayingTrack.Artist = updated.Artist
		a.NowPlayingTrack.Album = updated.Album
		a.NowPlayingTrack.Genre = updated.Genre
		a.NowPlayingTrack.Year = updated.Year
		a.NowPlayingTrack.TrackNum = updated.TrackNum
	}
}

func (a *App) updateTrackRow(listID string, index int) {
	if index < 0 {
		return
	}
	var (
		list     *tview.List
		tracks   []model.Track
		selected map[int]bool
		errors   map[string]string
	)
	switch listID {
	case "tracks":
		list = a.Panels.Tracks.List
		tracks = a.TracksData
		selected = a.TracksSelected
		errors = a.TracksErrors
	case "library":
		list = a.Panels.Library.Content
		tracks = a.LibraryData
		selected = a.LibrarySelected
		errors = a.LibraryErrors
	default:
		return
	}
	if list == nil || index >= len(tracks) {
		return
	}
	if index >= list.GetItemCount() {
		return
	}
	highlighted := listItemHighlighted(list, index)
	row := formatTrackRow(list, a.indexWidthForList(list), index, tracks[index], selected != nil && selected[index], hasTrackError(errors, tracks[index].Path), highlighted, a.Theme.Accent, a.Theme.Fg)
	_, secondary := list.GetItemText(index)
	list.SetItemText(index, row, secondary)
}

func (a *App) setIndexWidthForList(list *tview.List, count int) {
	width := trackRowIndexDigits(count)
	if list == a.Panels.Tracks.List {
		a.tracksIndexWidth = width
		return
	}
	if list == a.Panels.Library.Content {
		a.libraryIndexWidth = width
	}
}

func (a *App) indexWidthForList(list *tview.List) int {
	if list == a.Panels.Tracks.List && a.tracksIndexWidth > 0 {
		return a.tracksIndexWidth
	}
	if list == a.Panels.Library.Content && a.libraryIndexWidth > 0 {
		return a.libraryIndexWidth
	}
	return trackRowIndexDigits(1)
}
