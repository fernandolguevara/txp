package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"txp/internal/model"
)

func formatTrackName(title string, artist string, path string) string {
	if title == "" && artist == "" {
		return trimExtension(path)
	}
	if artist == "" {
		return title
	}
	if title == "" {
		return artist
	}
	return fmt.Sprintf("%s - %s", artist, title)
}

func escapeUserText(text string) string {
	if text == "" {
		return ""
	}
	return tview.Escape(text)
}

func formatQueueName(track model.Track) string {
	if strings.TrimSpace(track.Title) == "" || strings.TrimSpace(track.Artist) == "" {
		return trimExtension(track.Path)
	}
	return fmt.Sprintf("%s - %s", track.Artist, track.Title)
}

func trimExtension(path string) string {
	base := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		base = path[idx+1:]
	}
	if idx := strings.LastIndex(base, "."); idx > 0 {
		return base[:idx]
	}
	return base
}

func formatVolumeBar(volume int, slots int) string {
	clamped := clampPercent(volume, 0, 100)
	bar := formatTimeline(float64(clamped), 100, slots)
	return fmt.Sprintf("Vol %d%% %s", clamped, bar)
}

func formatDuration(seconds float64) string {
	if seconds <= 0 {
		return "--:--"
	}
	secs := int(seconds + 0.5)
	mins := secs / 60
	secs = secs % 60
	return fmt.Sprintf("%02d:%02d", mins, secs)
}

func formatBpm(value float64) string {
	if value <= 0 {
		return "--.-"
	}
	return fmt.Sprintf("%5.1f", value)
}

func formatKey(value string) string {
	key := strings.TrimSpace(value)
	if key == "" {
		return "----"
	}
	return truncateText(key, trackRowKeyWidth)
}

func formatTrackIndex(index int, hasError bool, width int) string {
	if width < 1 {
		width = 1
	}
	label := fmt.Sprintf("%*d", width, index+1)
	if !hasError {
		return label
	}
	return fmt.Sprintf("[red]%s[-]", label)
}

func formatFavoriteMarker(favorite bool, highlighted bool, accent tcell.Color, _ tcell.Color) string {
	if !favorite {
		return " "
	}
	if highlighted {
		return "♥"
	}
	return fmt.Sprintf("[#%06x]♥[-]", accent.Hex())
}

func formatTrackFormat(path string) string {
	if path == "" {
		return ""
	}
	idx := strings.LastIndex(path, ".")
	if idx < 0 || idx == len(path)-1 {
		return ""
	}
	return strings.ToUpper(path[idx+1:])
}

func formatCategoryLabel(label string, count int) string {
	return fmt.Sprintf("%s (#%d)", label, count)
}

func formatTracksHeader(count int, sortMode string, sortAsc bool) string {
	if count < 0 {
		count = 0
	}
	sortLabel := formatSortLabel(sortMode, sortAsc)
	return fmt.Sprintf("#    Name                                  Length BPM  Key (%d)  [Sort: %s | b:BPM k:Key]", count, sortLabel)
}

func formatTracksHeaderNoSort(count int) string {
	if count < 0 {
		count = 0
	}
	return fmt.Sprintf("#    Name                                  Length BPM  Key (%d)", count)
}

func formatQueueHeader(count int, indexWidth int) string {
	if count < 0 {
		count = 0
	}
	if indexWidth < 1 {
		indexWidth = 1
	}
	indexLabel := fmt.Sprintf("%*s", indexWidth, "#")
	prefix := fmt.Sprintf("♥%s", indexLabel)
	return fmt.Sprintf("%s Name                                  Length BPM  Key (%d)", prefix, count)
}

func formatSortLabel(mode string, asc bool) string {
	label := "Title"
	switch mode {
	case "bpm":
		label = "BPM"
	case "key":
		label = "Key"
	case "length":
		label = "Length"
	case "artist":
		label = "Artist"
	case "title":
		label = "Title"
	}
	if asc {
		return label + " ASC"
	}
	return label + " DESC"
}

func formatSortInfo(sortMode string, sortAsc bool) string {
	label := formatSortLabel(sortMode, sortAsc)
	return fmt.Sprintf("Sort: %s | b:BPM k:Key", label)
}

func formatTimestamp(seconds float64) string {
	if seconds <= 0 {
		return "00:00"
	}
	secs := int(seconds + 0.5)
	mins := secs / 60
	secs = secs % 60
	return fmt.Sprintf("%02d:%02d", mins, secs)
}

func formatTimeline(position float64, duration float64, slots int) string {
	if slots < 4 {
		slots = 4
	}
	if duration <= 0 {
		duration = 1
	}
	pos := position
	if pos < 0 {
		pos = 0
	}
	index := int(math.Round(pos / duration * float64(slots-1)))
	if index < 0 {
		index = 0
	}
	if index > slots-1 {
		index = slots - 1
	}
	var bar strings.Builder
	bar.Grow(slots)
	for i := 0; i < slots; i++ {
		if i == index {
			bar.WriteRune('■')
			continue
		}
		bar.WriteRune('─')
	}
	return bar.String()
}

func formatTimelineLine(position float64, duration float64, slots int) string {
	pos := formatTimestamp(position)
	dur := formatTimestamp(duration)
	bar := formatTimeline(position, duration, slots)
	return fmt.Sprintf("%s %s %s", pos, bar, dur)
}

func formatTimelineLineAdaptive(position float64, duration float64, width int) string {
	if width > 0 && width < 24 {
		return formatTimelineCompact(position, duration)
	}
	if width <= 0 {
		width = 12
	}
	slots := width - len(formatTimestamp(0)) - len(formatTimestamp(0)) - 2
	if slots < 3 {
		return formatTimelineCompact(position, duration)
	}
	return formatTimelineLine(position, duration, slots)
}

func formatTimelineCompact(position float64, duration float64) string {
	left := formatTimestamp(position)
	if left == "--:--" {
		left = "00:00"
	}
	right := formatTimestamp(duration)
	if right == "--:--" {
		right = "00:00"
	}
	return fmt.Sprintf("%s/%s", left, right)
}

func formatSeekIndicator(delta float64) string {
	seconds := int(delta + 0.5)
	if seconds == 0 {
		return "Seek 0s"
	}
	sign := "+"
	if seconds < 0 {
		sign = "-"
		seconds = -seconds
	}
	return fmt.Sprintf("Seek %s%ds", sign, seconds)
}

func formatTitle(base string, bold bool) string {
	label := base
	parts := strings.SplitN(base, " ", 2)
	if len(parts) == 2 && len(parts[0]) == 1 {
		label = fmt.Sprintf("(%s) %s", parts[0], parts[1])
	}
	title := fmt.Sprintf("[ %s ]", label)
	if bold {
		return "[::b]" + title
	}
	return title
}
