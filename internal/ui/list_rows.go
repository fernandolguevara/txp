package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"

	"txp/internal/model"
)

func formatQueueRow(list *tview.List, indexWidth int, index int, track model.Track, played bool, accent tcell.Color, contrast tcell.Color, highlighted bool) string {
	name := formatQueueName(track)
	length := formatDuration(track.Duration)
	bpm := formatBpm(track.BPM)
	key := formatKey(track.Key)
	favorite := formatFavoriteMarker(track.Favorite, highlighted, accent, contrast)
	indexLabel := formatTrackIndex(index, false, indexWidth)
	prefix := padTaggedPrefix(fmt.Sprintf("%s%s", favorite, indexLabel), trackRowFavWidth+indexWidth)
	nameWidth := trackRowNameWidth(list, indexWidth)
	name = sanitizeRowName(name)
	name = padToWidth(name, nameWidth)
	name = escapeUserText(name)
	key = escapeUserText(padToWidth(key, trackRowKeyWidth))
	line := fmt.Sprintf("%s %s %5s %5s %s", prefix, name, length, bpm, key)
	if !played {
		return line
	}
	return "[gray]" + line + "[-]"
}

const (
	trackRowFavWidth     = 1
	trackRowIndexWidth   = 3
	trackRowLengthWidth  = 5
	trackRowBpmWidth     = 5
	trackRowKeyWidth     = 4
	trackRowSpacingWidth = 1
	trackRowMinNameWidth = 1
	trackRowDefaultWidth = 40
)

func formatTrackRow(list *tview.List, indexWidth int, index int, track model.Track, selected bool, hasError bool, highlighted bool, accent tcell.Color, contrast tcell.Color) string {
	name := formatTrackName(track.Title, track.Artist, track.Path)
	length := formatDuration(track.Duration)
	bpm := formatBpm(track.BPM)
	key := formatKey(track.Key)
	marker := " "
	if selected {
		marker = "*"
	}
	favorite := formatFavoriteMarker(track.Favorite, highlighted, accent, contrast)
	indexLabel := formatTrackIndex(index, hasError, indexWidth)
	prefix := padTaggedPrefix(fmt.Sprintf("%s%s%s", favorite, marker, indexLabel), trackRowFavWidth+1+indexWidth)
	nameWidth := trackRowNameWidth(list, indexWidth)
	name = sanitizeRowName(name)
	name = padToWidth(name, nameWidth)
	name = escapeUserText(name)
	key = escapeUserText(padToWidth(key, trackRowKeyWidth))
	return fmt.Sprintf("%s %s %5s %5s %s", prefix, name, length, bpm, key)
}

func padTaggedPrefix(prefix string, width int) string {
	visible := runewidth.StringWidth(stripColorTags(prefix))
	if visible >= width {
		return prefix
	}
	return prefix + strings.Repeat(" ", width-visible)
}

func stripColorTags(text string) string {
	if text == "" {
		return ""
	}
	var out strings.Builder
	out.Grow(len(text))
	inTag := false
	for _, r := range text {
		switch r {
		case '[':
			if !inTag {
				inTag = true
				continue
			}
		case ']':
			if inTag {
				inTag = false
				continue
			}
		}
		if inTag {
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}

func trackRowNameWidth(list *tview.List, indexWidth int) int {
	width := trackRowDefaultWidth
	if list != nil {
		_, _, innerWidth, _ := list.GetInnerRect()
		if innerWidth > 0 {
			width = innerWidth - trackRowFixedWidth(indexWidth)
		}
	}
	if width < trackRowMinNameWidth {
		width = trackRowMinNameWidth
	}
	return width
}

func trackRowFixedWidth(indexWidth int) int {
	return trackRowFavWidth + 1 + indexWidth + trackRowSpacingWidth + trackRowLengthWidth + trackRowSpacingWidth + trackRowBpmWidth + trackRowSpacingWidth + trackRowKeyWidth
}

func trackRowIndexDigits(count int) int {
	if count < 1 {
		count = 1
	}
	return len(fmt.Sprintf("%d", count))
}

func truncateText(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if runewidth.StringWidth(text) <= width {
		return text
	}
	if width <= 3 {
		return runewidth.Truncate(text, width, "")
	}
	return runewidth.Truncate(text, width, "...")
}

func padToWidth(text string, width int) string {
	if width <= 0 {
		return ""
	}
	padded := truncateText(text, width)
	current := runewidth.StringWidth(padded)
	if current >= width {
		return padded
	}
	return padded + strings.Repeat(" ", width-current)
}

func sanitizeRowName(text string) string {
	replacer := strings.NewReplacer("\t", " ", "\n", " ", "\r", " ")
	return replacer.Replace(text)
}

func listItemHighlighted(list *tview.List, index int) bool {
	if list == nil {
		return false
	}
	return list.HasFocus() && index == list.GetCurrentItem()
}

func buildScrollBar(height int, total int, offset int) string {
	if height <= 0 {
		return ""
	}
	if total <= 0 {
		return strings.Repeat(".", height)
	}
	thumbSize := height
	thumbTop := 0
	if total > height {
		thumbSize = int(math.Round(float64(height*height) / float64(total)))
		if thumbSize < 1 {
			thumbSize = 1
		}
		maxOffset := max(1, total-height)
		position := float64(min(max(offset, 0), maxOffset)) / float64(maxOffset)
		thumbTop = int(math.Round(position * float64(height-thumbSize)))
		if thumbTop < 0 {
			thumbTop = 0
		}
		if thumbTop > height-thumbSize {
			thumbTop = height - thumbSize
		}
	}
	var out strings.Builder
	for i := 0; i < height; i++ {
		if i > 0 {
			out.WriteByte('\n')
		}
		if i >= thumbTop && i < thumbTop+thumbSize {
			out.WriteByte('|')
		} else {
			out.WriteByte('.')
		}
	}
	return out.String()
}
