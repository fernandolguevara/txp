package panels

import "github.com/rivo/tview"

type Tracks struct {
	Container *tview.Flex
	Header    *tview.TextView
	Filter    *tview.InputField
	List      *tview.List
	Scroll    *tview.TextView
	SortInfo  *tview.TextView
}

func NewTracks(headerText string) *Tracks {
	header := tview.NewTextView().SetDynamicColors(true)
	header.SetText(headerText)
	header.SetWrap(false)

	filter := tview.NewInputField().SetLabel("Filter: ")
	filter.SetFieldWidth(0)

	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(false)
	list.SetBorderPadding(0, 0, 0, 1)

	scroll := tview.NewTextView()
	scroll.SetBorder(false)
	scroll.SetDynamicColors(false)
	scroll.SetWrap(false)

	sortInfo := tview.NewTextView().SetDynamicColors(true)
	sortInfo.SetWrap(false)

	listRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	listRow.AddItem(list, 0, 1, true)
	listRow.AddItem(scroll, 1, 0, false)

	headerRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	headerRow.AddItem(header, 0, 1, false)
	headerRow.AddItem(tview.NewTextView(), 1, 0, false)

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBorder(true).SetTitle("[ (1) Tracks #0 ]")
	container.AddItem(filter, 1, 0, false)
	container.AddItem(headerRow, 1, 0, false)
	container.AddItem(listRow, 0, 1, true)
	container.AddItem(sortInfo, 1, 0, false)

	return &Tracks{Container: container, Header: header, Filter: filter, List: list, Scroll: scroll, SortInfo: sortInfo}
}
