package panels

import "github.com/rivo/tview"

type BottomBar struct {
	Root  *tview.Flex
	Hints *tview.TextView
}

func NewBottomBar() *BottomBar {
	hints := tview.NewTextView().SetDynamicColors(true)
	hints.SetText("── [ :now-playing ] ─ [ ]")

	root := tview.NewFlex().SetDirection(tview.FlexColumn)
	root.AddItem(hints, 0, 1, false)

	return &BottomBar{Root: root, Hints: hints}
}
