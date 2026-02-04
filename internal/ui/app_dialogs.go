package ui

import (
	"strings"

	"github.com/atotto/clipboard"
	"github.com/rivo/tview"
)

func (a *App) toggleLogDialog() {
	if a.LogDialog.Active {
		a.hideLogDialog()
		return
	}
	a.showLogDialog()
}

func (a *App) showLogDialog() {
	a.LogDialog.Active = true
	a.pushOverlay("log", nil, false, nil)
	a.setPanelStyle(a.LogDialog.Root, "Log (debug)", true)
	a.flushLogView(true)
	a.App.SetFocus(a.LogDialog.View)
	a.updateBottomHints()
}

func (a *App) hideLogDialog() {
	a.LogDialog.Active = false
	if entry, ok := a.popOverlay("log"); ok {
		a.restoreOverlayFocus(entry)
	}
	a.setPanelStyle(a.LogDialog.Root, "Log (debug)", false)
	a.updateBottomHints()
}

func (a *App) clearLog() {
	a.logMu.Lock()
	a.LogBuffer = nil
	a.logDirty = true
	a.logMu.Unlock()
	a.flushLogView(true)
	a.setStatusMessage("Log cleared")
}

func (a *App) copyLogToClipboard() {
	a.logMu.Lock()
	text := strings.Join(a.LogBuffer, "\n")
	a.logMu.Unlock()
	if strings.TrimSpace(text) == "" {
		a.setStatusMessage("Log empty")
		return
	}
	if err := clipboard.WriteAll(text); err != nil {
		a.setStatusMessage("Clipboard unavailable")
		return
	}
	a.setStatusMessage("Log copied to clipboard")
}

func (a *App) toggleLogAutoScroll() {
	a.LogDialog.AutoScroll = !a.LogDialog.AutoScroll
	if a.LogDialog.AutoScroll {
		a.setStatusMessage("Log auto-scroll on")
		a.flushLogView(true)
		return
	}
	a.setStatusMessage("Log auto-scroll off")
}

func (a *App) showFilterDialog() {
	a.FilterDialog.Active = true
	a.pushOverlay("filters", nil, false, nil)
	a.setPanelStyle(a.FilterDialog.Form, "Filters", true)
	a.syncFilterDialog()
	a.App.SetFocus(a.FilterDialog.Form)
	a.updateBottomHints()
}

func (a *App) hideFilterDialog() {
	a.FilterDialog.Active = false
	if entry, ok := a.popOverlay("filters"); ok {
		a.restoreOverlayFocus(entry)
	}
	a.setPanelStyle(a.FilterDialog.Form, "Filters", false)
	a.updateBottomHints()
}

func (a *App) applyFilterDialog() {
	a.AdvancedFilter = a.readFilterDialogCriteria()
	a.applyTracksFilter()
	a.applyLibraryFilter()
	a.setStatusMessage("Filters applied")
	a.hideFilterDialog()
}

func (a *App) clearFilterDialog() {
	a.AdvancedFilter = FilterCriteria{}
	a.setFilterDialogCriteria(a.AdvancedFilter)
	a.applyTracksFilter()
	a.applyLibraryFilter()
	a.setStatusMessage("Filters cleared")
}

func (a *App) syncFilterDialog() {
	a.setFilterDialogCriteria(a.AdvancedFilter)
}

func (a *App) readFilterDialogCriteria() FilterCriteria {
	form := a.FilterDialog.Form
	if form == nil {
		return FilterCriteria{}
	}
	getText := func(index int) string {
		if field, ok := form.GetFormItem(index).(*tview.InputField); ok {
			return field.GetText()
		}
		return ""
	}
	return FilterCriteria{
		Text:   getText(0),
		Artist: getText(1),
		Album:  getText(2),
		Genre:  getText(3),
		Year:   getText(4),
		Key:    getText(5),
	}
}

func (a *App) setFilterDialogCriteria(criteria FilterCriteria) {
	form := a.FilterDialog.Form
	if form == nil {
		return
	}
	setText := func(index int, value string) {
		if field, ok := form.GetFormItem(index).(*tview.InputField); ok {
			field.SetText(value)
		}
	}
	setText(0, criteria.Text)
	setText(1, criteria.Artist)
	setText(2, criteria.Album)
	setText(3, criteria.Genre)
	setText(4, criteria.Year)
	setText(5, criteria.Key)
}

func (a *App) showStatsDialog() {
	a.StatsDialog.Active = true
	a.pushOverlay("stats", nil, false, nil)
	a.setPanelStyle(a.StatsDialog.Root, "Stats", true)
	a.App.SetFocus(a.StatsDialog.Root)
	a.updateBottomHints()
}

func (a *App) hideStatsDialog() {
	a.StatsDialog.Active = false
	if entry, ok := a.popOverlay("stats"); ok {
		a.restoreOverlayFocus(entry)
	}
	a.setPanelStyle(a.StatsDialog.Root, "Stats", false)
	a.updateBottomHints()
}

func (a *App) showTaskDetails() {
	a.TaskDetails.Active = true
	a.pushOverlay("tasks", nil, false, nil)
	a.setPanelStyle(a.TaskDetails.Root, "Task Details", true)
	a.App.SetFocus(a.TaskDetails.Root)
	a.updateBottomHints()
}

func (a *App) hideTaskDetails() {
	a.TaskDetails.Active = false
	if entry, ok := a.popOverlay("tasks"); ok {
		a.restoreOverlayFocus(entry)
	}
	a.setPanelStyle(a.TaskDetails.Root, "Task Details", false)
	a.updateBottomHints()
}

func (a *App) showRescanDialog() {
	count := a.selectedCount()
	a.RescanDialog.Configure(count, func(force bool) {
		a.hideRescanDialog()
		a.rescanLibraries(a.Config.Libraries, force)
	}, func(force bool) {
		a.hideRescanDialog()
		paths := a.selectedPaths()
		if len(paths) == 0 {
			a.setStatusMessage("No libraries selected")
			return
		}
		a.rescanLibraries(paths, force)
	}, func() {
		a.hideRescanDialog()
	})

	a.RescanDialog.Active = true
	a.pushOverlay("rescan", nil, false, nil)
	a.setPanelStyle(a.RescanDialog.Root, "Rescan", true)
	a.App.SetFocus(a.RescanDialog.Root)
	a.updateBottomHints()
}

func (a *App) hideRescanDialog() {
	a.RescanDialog.Active = false
	if entry, ok := a.popOverlay("rescan"); ok {
		a.restoreOverlayFocus(entry)
	}
	a.setPanelStyle(a.RescanDialog.Root, "Rescan", false)
	a.updateBottomHints()
}

func (a *App) showSettings() {
	a.SettingsOpen = true
	a.Pages.ShowPage("settings")
	a.App.SetFocus(a.Settings.Libraries)
	a.setActiveSection("libraries")
	a.setLibrariesColumnFocus("left")
	a.updateBottomHints()
}

func (a *App) hideSettings() {
	a.SettingsOpen = false
	a.Pages.HidePage("settings")
	a.focusMainList()
	a.updateBottomHints()
}
