package panels

import "github.com/rivo/tview"

type TopBar struct {
	Root     *tview.Flex
	Song     *tview.TextView
	Timeline *tview.TextView
	Volume   *tview.TextView
}

func NewTopBar() *TopBar {
	song := tview.NewTextView().SetDynamicColors(true)
	song.SetText("Idle")

	timeline := tview.NewTextView().SetDynamicColors(true)
	timeline.SetText("00:00 ─────■── 00:00")

	volume := tview.NewTextView().SetDynamicColors(true)

	root := tview.NewFlex().SetDirection(tview.FlexColumn)
	root.SetBorder(true).SetTitle("[ (0) Now Playing ]")
	volume.SetTextAlign(tview.AlignRight)

	right := tview.NewFlex().SetDirection(tview.FlexColumn)
	right.AddItem(timeline, 0, 1, false)
	right.AddItem(volume, 15, 0, false)

	root.AddItem(song, 0, 1, false)
	root.AddItem(right, 0, 1, false)

	return &TopBar{
		Root:     root,
		Song:     song,
		Timeline: timeline,
		Volume:   volume,
	}
}
