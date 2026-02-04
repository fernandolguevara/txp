package panels

import "github.com/rivo/tview"

func NewTrackInfo() *tview.TextView {
	trackInfo := tview.NewTextView().SetDynamicColors(true)
	trackInfo.SetBorder(true).SetTitle("[ (3) Track Info ]")
	trackInfo.SetText("No track selected")
	return trackInfo
}
