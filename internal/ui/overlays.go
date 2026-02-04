package ui

import "github.com/rivo/tview"

type overlayEntry struct {
	id    string
	focus tview.Primitive
}

type overlayPage struct {
	primitive     tview.Primitive
	removeOnClose bool
}

func (a *App) registerOverlay(id string, primitive tview.Primitive, removeOnClose bool) {
	if id == "" || primitive == nil {
		return
	}
	a.overlayMu.Lock()
	defer a.overlayMu.Unlock()
	if a.overlayPages == nil {
		a.overlayPages = map[string]overlayPage{}
	}
	a.overlayPages[id] = overlayPage{primitive: primitive, removeOnClose: removeOnClose}
}

func (a *App) pushOverlay(id string, primitive tview.Primitive, removeOnClose bool, focusTarget tview.Primitive) {
	if id == "" || a.Pages == nil {
		return
	}
	if focusTarget == nil && a.App != nil {
		focusTarget = a.App.GetFocus()
	}

	a.overlayMu.Lock()
	if a.overlayPages == nil {
		a.overlayPages = map[string]overlayPage{}
	}
	if primitive != nil {
		a.overlayPages[id] = overlayPage{primitive: primitive, removeOnClose: removeOnClose}
		a.Pages.AddPage(id, primitive, true, true)
	} else if page, ok := a.overlayPages[id]; ok && page.primitive != nil {
		a.Pages.ShowPage(id)
	} else {
		a.overlayMu.Unlock()
		return
	}

	for i := len(a.overlayStack) - 1; i >= 0; i-- {
		if a.overlayStack[i].id == id {
			a.overlayStack = append(a.overlayStack[:i], a.overlayStack[i+1:]...)
			break
		}
	}
	a.overlayStack = append(a.overlayStack, overlayEntry{id: id, focus: focusTarget})

	if id == "palette" {
		a.moveOverlayToTopLocked("palette")
	} else if a.Palette != nil && a.Palette.Active {
		a.moveOverlayToTopLocked("palette")
	}
	a.overlayMu.Unlock()

	if a.Palette != nil && a.Palette.Active && id != "palette" {
		a.bringOverlayToFront("palette")
	}
	if id == "palette" {
		a.bringOverlayToFront("palette")
	}
}

func (a *App) popOverlay(id string) (overlayEntry, bool) {
	if id == "" || a.Pages == nil {
		return overlayEntry{}, false
	}
	a.overlayMu.Lock()
	defer a.overlayMu.Unlock()
	index := -1
	for i := len(a.overlayStack) - 1; i >= 0; i-- {
		if a.overlayStack[i].id == id {
			index = i
			break
		}
	}
	if index == -1 {
		return overlayEntry{}, false
	}
	entry := a.overlayStack[index]
	a.overlayStack = append(a.overlayStack[:index], a.overlayStack[index+1:]...)
	if page, ok := a.overlayPages[id]; ok {
		if page.removeOnClose {
			a.Pages.RemovePage(id)
			delete(a.overlayPages, id)
		} else {
			a.Pages.HidePage(id)
		}
	} else {
		a.Pages.HidePage(id)
	}
	return entry, true
}

func (a *App) closeTopOverlay() bool {
	if a.Pages == nil {
		return false
	}
	a.overlayMu.Lock()
	if len(a.overlayStack) == 0 {
		a.overlayMu.Unlock()
		return false
	}
	top := a.overlayStack[len(a.overlayStack)-1].id
	a.overlayMu.Unlock()
	switch top {
	case "palette":
		a.hidePalette()
		return true
	case "log":
		a.hideLogDialog()
		return true
	case "logfile":
		a.hideLogFileDialog()
		return true
	case "filters":
		a.hideFilterDialog()
		return true
	case "stats":
		a.hideStatsDialog()
		return true
	case "tasks":
		a.hideTaskDetails()
		return true
	case "rescan":
		a.hideRescanDialog()
		return true
	case "tag-editor":
		a.hideTagEditor()
		return true
	case "add-library-path":
		a.hideAddLibraryPathDialog()
		return true
	case "prompt":
		a.hidePromptDialog()
		return true
	default:
		return false
	}
}

func (a *App) closeAllOverlays() {
	for {
		if !a.closeTopOverlay() {
			return
		}
	}
}

func (a *App) moveOverlayToTopLocked(id string) {
	index := -1
	for i := len(a.overlayStack) - 1; i >= 0; i-- {
		if a.overlayStack[i].id == id {
			index = i
			break
		}
	}
	if index == -1 {
		return
	}
	entry := a.overlayStack[index]
	a.overlayStack = append(a.overlayStack[:index], a.overlayStack[index+1:]...)
	a.overlayStack = append(a.overlayStack, entry)
}

func (a *App) bringOverlayToFront(id string) bool {
	if id == "" || a.Pages == nil {
		return false
	}
	a.overlayMu.Lock()
	page, ok := a.overlayPages[id]
	a.overlayMu.Unlock()
	if !ok || page.primitive == nil {
		return false
	}
	a.Pages.RemovePage(id)
	a.Pages.AddPage(id, page.primitive, true, true)
	return true
}

func (a *App) restoreOverlayFocus(entry overlayEntry) {
	if entry.focus != nil && a.App != nil {
		a.App.SetFocus(entry.focus)
		return
	}
	a.focusMainList()
}
