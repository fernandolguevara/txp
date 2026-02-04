package panels

import "github.com/rivo/tview"

func buildThemesGroup(themeNames []string) (*tview.Flex, *tview.List, *tview.TextView) {
	themesList := tview.NewList().ShowSecondaryText(false)
	themesList.SetBorder(false)
	for _, name := range themeNames {
		themesList.AddItem(name, "", 0, nil)
	}
	themeMsg := tview.NewTextView().SetText("Enter: apply theme")

	themesGrp := tview.NewFlex().SetDirection(tview.FlexRow)
	themesGrp.SetBorder(true).SetTitle("[ (4) Themes ]")
	themesGrp.AddItem(themesList, 0, 1, true)
	themesGrp.AddItem(themeMsg, 1, 0, false)

	return themesGrp, themesList, themeMsg
}
