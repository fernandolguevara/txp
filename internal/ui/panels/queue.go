package panels

import "github.com/rivo/tview"

type Queue struct {
	Container *tview.Flex
	Header    *tview.TextView
	List      *tview.List
}

func NewQueue() *Queue {
	header := tview.NewTextView().SetDynamicColors(true)
	header.SetWrap(false)

	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(false)

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBorder(true).SetTitle("[ (2) Queue #0 ]")
	container.AddItem(header, 1, 0, false)
	container.AddItem(list, 0, 1, true)

	return &Queue{Container: container, Header: header, List: list}
}
