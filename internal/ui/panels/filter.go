package panels

import "github.com/rivo/tview"

type FilterDialog struct {
	Root   *tview.Flex
	Form   *tview.Form
	Active bool
}

func NewFilterDialog() *FilterDialog {
	form := tview.NewForm()
	form.AddInputField("Text", "", 30, nil, nil)
	form.AddInputField("Artist", "", 30, nil, nil)
	form.AddInputField("Album", "", 30, nil, nil)
	form.AddInputField("Genre", "", 30, nil, nil)
	form.AddInputField("Year", "", 8, nil, nil)
	form.AddInputField("Key", "", 8, nil, nil)
	form.AddButton("Apply", nil)
	form.AddButton("Clear", nil)
	form.AddButton("Close", nil)
	form.SetBorder(true).SetTitle("[ Filters ]")

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(form, 0, 1, true)

	return &FilterDialog{Root: root, Form: form}
}
