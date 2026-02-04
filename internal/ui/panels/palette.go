package panels

import "github.com/rivo/tview"

type CommandPalette struct {
	Root   *tview.Flex
	Input  *tview.InputField
	List   *tview.List
	Active bool
}

func NewCommandPalette() *CommandPalette {
	input := tview.NewInputField().SetLabel("Command: ")
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(true).SetTitle("[ Palette ]")
	list.SetSelectedFunc(func(index int, main string, secondary string, shortcut rune) {})

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.SetBorder(true).SetTitle("[ Command Palette ]")
	root.AddItem(input, 1, 0, true)
	root.AddItem(list, 0, 1, false)

	return &CommandPalette{Root: root, Input: input, List: list}
}
