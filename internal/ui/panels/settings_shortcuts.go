package panels

import (
	"strings"

	"github.com/rivo/tview"

	"txp/internal/config"
)

func buildShortcutsGroup(cfg config.Config) (*tview.Flex, *tview.List, *tview.TextView) {
	shortcutsList := tview.NewList().ShowSecondaryText(false)
	shortcutsList.SetBorder(false)
	for action, bindings := range cfg.Shortcuts {
		shortcutsList.AddItem(action, strings.Join(bindings, ", "), 0, nil)
	}
	shortcutsMsg := tview.NewTextView().SetText("Enter: rebind  x: reset")

	shortcutsGrp := tview.NewFlex().SetDirection(tview.FlexRow)
	shortcutsGrp.SetBorder(true).SetTitle("[ (2) Shortcuts ]")
	shortcutsGrp.AddItem(shortcutsList, 0, 1, true)
	shortcutsGrp.AddItem(shortcutsMsg, 1, 0, false)

	return shortcutsGrp, shortcutsList, shortcutsMsg
}
