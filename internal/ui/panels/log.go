package panels

import "github.com/rivo/tview"

type LogDialog struct {
	Root       *tview.Flex
	View       *tview.TextView
	Active     bool
	AutoScroll bool
}

func NewLogDialog() *LogDialog {
	view := tview.NewTextView().SetDynamicColors(true)
	view.SetScrollable(true)
	view.SetBorder(false)

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.SetBorder(true).SetTitle("[ Log (debug) ]")
	root.AddItem(view, 0, 1, true)

	return &LogDialog{Root: root, View: view, AutoScroll: true}
}
