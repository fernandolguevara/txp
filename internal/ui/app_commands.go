package ui

import (
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
)

func (a *App) bindPaletteCommands() {
	commands := []string{
		"app:quit",
		"audio:test-tone",
		"filters:open",
		"library:add-path",
		"library:analyze-missing",
		"library:reload-tracks",
		"library:rescan",
		"library:update",
		"log:clear",
		"log:open-file",
		"log:toggle-autoscroll",
		"log:view",
		"settings:shortcuts",
		"settings:view",
		"stats:view",
		"tasks:details",
		"theme:switch",
		"tracks:view",
		"library:view",
	}
	sort.SliceStable(commands, func(i, j int) bool {
		leftScope, leftAction := splitCommandKey(commands[i])
		rightScope, rightAction := splitCommandKey(commands[j])
		if leftScope == rightScope {
			return leftAction < rightAction
		}
		return leftScope < rightScope
	})
	a.paletteCommands = commands
	a.resetPaletteList()

	a.Palette.Input.SetChangedFunc(func(text string) {
		a.Palette.List.Clear()
		for _, cmd := range commands {
			if strings.Contains(strings.ToLower(cmd), strings.ToLower(text)) {
				label := cmd
				a.Palette.List.AddItem(label, "", 0, func() { a.handleCommand(label) })
			}
		}
		if a.Palette.List.GetItemCount() > 0 {
			a.Palette.List.SetCurrentItem(0)
		}
	})

	a.Palette.Input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			index := a.Palette.List.GetCurrentItem()
			count := a.Palette.List.GetItemCount()
			if index >= 0 && index < count {
				main, _ := a.Palette.List.GetItemText(index)
				if main != "" {
					a.handleCommand(main)
				}
			}
		}
	})

	a.Palette.Input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyDown || event.Key() == tcell.KeyUp {
			if a.Palette.List.GetItemCount() == 0 {
				return nil
			}
			a.App.SetFocus(a.Palette.List)
			if event.Key() == tcell.KeyDown {
				a.Palette.List.SetCurrentItem(min(a.Palette.List.GetCurrentItem()+1, a.Palette.List.GetItemCount()-1))
			} else {
				a.Palette.List.SetCurrentItem(max(a.Palette.List.GetCurrentItem()-1, 0))
			}
			return nil
		}
		return event
	})

	a.Palette.List.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			index := a.Palette.List.GetCurrentItem()
			count := a.Palette.List.GetItemCount()
			if index >= 0 && index < count {
				main, _ := a.Palette.List.GetItemText(index)
				if main != "" {
					a.handleCommand(main)
					return nil
				}
			}
		case tcell.KeyEsc:
			a.hidePalette()
			return nil
		case tcell.KeyUp, tcell.KeyDown:
			return event
		case tcell.KeyRune:
			if event.Rune() != 0 {
				a.App.SetFocus(a.Palette.Input)
				return event
			}
		}
		return event
	})
}

func (a *App) togglePalette() {
	if a.Palette.Active {
		a.hidePalette()
		return
	}
	a.Palette.Active = true
	a.pushOverlay("palette", nil, false, nil)
	a.setPanelStyle(a.Palette.Root, "Command Palette", true)
	a.App.SetFocus(a.Palette.Input)
}

func (a *App) hidePalette() {
	a.Palette.Active = false
	if entry, ok := a.popOverlay("palette"); ok {
		a.restoreOverlayFocus(entry)
	}
	a.setPanelStyle(a.Palette.Root, "Command Palette", false)
	a.updateBottomHints()
}

func (a *App) resetPaletteList() {
	if a.Palette == nil || a.Palette.List == nil {
		return
	}
	a.Palette.List.Clear()
	for _, cmd := range a.paletteCommands {
		label := cmd
		a.Palette.List.AddItem(label, "", 0, func() {
			a.handleCommand(label)
		})
	}
}

func (a *App) handleCommand(command string) {
	a.hidePalette()
	a.closeAllOverlays()
	if a.Palette != nil && a.Palette.Input != nil {
		a.Palette.Input.SetText("")
	}
	a.resetPaletteList()
	switch command {
	case "library:update":
		a.showSettings()
		a.App.SetFocus(a.Settings.Libraries)
		a.setStatusMessage("Focus: libraries")
	case "library:rescan":
		a.showRescanDialog()
	case "library:analyze-missing":
		a.analyzeMissingTracks()
	case "tasks:details":
		a.showTaskDetails()
	case "library:reload-tracks":
		a.refreshTracks()
		a.refreshStatsTotals()
	case "log:open-file":
		a.openLogFileCommand()
	case "audio:test-tone":
		a.testTone()
	case "tracks:view":
		if a.SettingsOpen {
			a.hideSettings()
		}
		a.updateViewMode("tracks")
	case "library:view":
		if a.SettingsOpen {
			a.hideSettings()
		}
		a.updateViewMode("library")
	case "log:view":
		a.showLogDialog()
	case "filters:open":
		a.showFilterDialog()
	case "stats:view":
		a.showStatsDialog()
	case "settings:view":
		a.showSettings()
	case "theme:switch":
		a.showSettings()
		a.App.SetFocus(a.Settings.Themes)
	case "settings:shortcuts":
		a.showSettings()
		a.App.SetFocus(a.Settings.Shortcuts)
	case "log:clear":
		a.clearLog()
	case "log:toggle-autoscroll":
		a.toggleLogAutoScroll()
	case "app:quit":
		a.App.Stop()
	case "library:add-path":
		a.showAddLibraryPathDialog()
	}
}

func splitCommandKey(value string) (string, string) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}
