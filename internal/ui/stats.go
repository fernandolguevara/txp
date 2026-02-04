package ui

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"

	"txp/internal/storage"
)

type StatsDialog struct {
	Root   *tview.Flex
	View   *tview.TextView
	Active bool
}

func NewStatsDialog() *StatsDialog {
	view := tview.NewTextView().SetDynamicColors(true)
	view.SetBorder(false)
	view.SetText("No stats available yet.")

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.SetBorder(true).SetTitle("[ Stats ]")
	root.AddItem(view, 0, 1, true)

	return &StatsDialog{Root: root, View: view}
}

func (s *StatsDialog) SetTotals(totals storage.StatsTotals, hasData bool) {
	if !hasData {
		s.View.SetText("No stats available yet.")
		return
	}

	lines := []string{
		"Library Stats",
		"",
		fmt.Sprintf("Tracks: %d", totals.TotalTracks),
		fmt.Sprintf("Duration: %.1f min", totals.TotalDuration/60.0),
		fmt.Sprintf("Plays: %d", totals.TotalPlays),
		fmt.Sprintf("Artists: %d", totals.Artists),
		fmt.Sprintf("Albums: %d", totals.Albums),
		"",
		"Press Esc to close.",
	}

	s.View.SetText(strings.Join(lines, "\n"))
}
