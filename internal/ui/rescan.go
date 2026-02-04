package ui

import (
	"fmt"

	"github.com/rivo/tview"
)

type RescanDialog struct {
	Root       *tview.Form
	Active     bool
	Force      bool
	onAll      func(bool)
	onSelected func(bool)
	onCancel   func()
}

func NewRescanDialog() *RescanDialog {
	form := tview.NewForm()
	form.SetBorder(true).SetTitle("[ Rescan ]")
	form.AddTextView("", "Rescan libraries", 0, 1, false, false)
	form.AddCheckbox("Force rescan (ignore checksum)", false, func(checked bool) {})

	dialog := &RescanDialog{Root: form}

	return dialog
}

func (d *RescanDialog) Configure(selectedCount int, onAll func(bool), onSelected func(bool), onCancel func()) {
	label := "Rescan selected"
	if selectedCount >= 0 {
		label = fmt.Sprintf("Rescan selected (%d)", selectedCount)
	}

	form := d.Root
	form.Clear(true)
	form.SetBorder(true).SetTitle("[ Rescan ]")
	form.AddTextView("", "Rescan libraries", 0, 1, false, false)
	form.AddCheckbox("Force rescan (ignore checksum)", d.Force, func(checked bool) {
		d.Force = checked
	})
	form.AddButton("Rescan all", func() {
		if d.onAll != nil {
			d.onAll(d.Force)
		}
	})
	form.AddButton(label, func() {
		if d.onSelected != nil {
			d.onSelected(d.Force)
		}
	})
	form.AddButton("Cancel", func() {
		if d.onCancel != nil {
			d.onCancel()
		}
	})
	d.onAll = onAll
	d.onSelected = onSelected
	d.onCancel = onCancel
}
