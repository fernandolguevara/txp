package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"

	"txp/internal/config"
	"txp/internal/model"
	"txp/internal/player"
	"txp/internal/state"
	"txp/internal/storage"
	"txp/internal/ui/panels"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type App struct {
	App                    *tview.Application
	Pages                  *tview.Pages
	Root                   *tview.Flex
	Panels                 *UIPanels
	BodyPages              *tview.Pages
	RightPane              *tview.Flex
	TrackInfo              *tview.TextView
	Settings               *panels.SettingsView
	ShowTrackInfo          bool
	ViewMode               string
	TracksData             []model.Track
	TracksAll              []model.Track
	LibraryData            []model.Track
	LibraryAll             []model.Track
	TotalTracks            int
	QueueItems             []model.Track
	QueueIndex             int
	QueueScope             string
	QueuePlayed            map[int]bool
	TracksSelected         map[int]bool
	LibrarySelected        map[int]bool
	TracksCursor           int
	LibraryCursor          int
	QueueCursor            int
	Volume                 int
	Paused                 bool
	NowPlayingTrack        *model.Track
	BottomBar              *tview.Flex
	BottomHints            *tview.TextView
	Palette                *panels.CommandPalette
	FilterDialog           *panels.FilterDialog
	LogDialog              *panels.LogDialog
	LogFileDialog          *panels.LogFileDialog
	StatsDialog            *StatsDialog
	RescanDialog           *RescanDialog
	TagEditor              *TagEditorDialog
	SelectedLibs           map[string]bool
	Theme                  Theme
	Config                 config.Config
	ConfigPath             string
	ResizeMode             bool
	SettingsOpen           bool
	ActiveSection          string
	ActivePanel            string
	ActiveLibrariesColumn  string
	NavRestoreInProgress   bool
	NavPersistOnNextSelect bool
	capturing              bool
	captureAction          string
	statusMessage          string
	bottomHint             string
	bottomTitle            string
	bottomMarqueeOffset    int
	bottomMarqueeDir       int
	bottomMarqueeHold      int
	appRunning             bool
	tickersStarted         bool
	tracksListWidth        int
	libraryListWidth       int
	tracksIndexWidth       int
	libraryIndexWidth      int
	queueIndexWidth        int
	queuePersistMu         sync.Mutex
	queuePersistTimer      *time.Timer
	queuePersistItems      []model.Track
	queuePersistScope      string
	libraryCacheMu         sync.Mutex
	libraryCacheLoaded     bool
	libraryCacheLoading    bool
	libraryArtistsCache    []string
	libraryAlbumsCache     []string
	libraryGenresCache     []string
	libraryYearsCache      []string
	LogBuffer              []string
	LogMax                 int
	logDirty               bool
	logMu                  sync.Mutex
	tasks                  *state.TaskManager
	db                     *storage.Database
	TaskDetails            *TaskDetailsDialog
	Player                 player.Player
	overlayMu              sync.Mutex
	overlayStack           []overlayEntry
	overlayPages           map[string]overlayPage
	nowPlayingMu           sync.Mutex
	nowPlayingText         string
	nowPlayingTimeline     string
	nowPlayingVolume       string
	seekMu                 sync.Mutex
	seekText               string
	seekUntil              time.Time
	seekInFlight           bool
	seekTimer              *time.Timer
	seekPendingDelta       float64
	seekToken              uint64
	seekMetaPrefixAt       time.Time
	eqMu                   sync.Mutex
	eqFrames               []string
	eqIndex                int
	eqFrame                string
	sortMode               string
	sortAsc                bool
	scanMu                 sync.Mutex
	scanRunning            bool
	paletteCommands        []string
	logger                 *zap.Logger
	statusMu               sync.RWMutex
	TracksFilterText       string
	AdvancedFilter         FilterCriteria
	LibraryFilterText      string
	TracksErrors           map[string]string
	LibraryErrors          map[string]string
	DoubleClickPlayback    bool
	lastClickAt            time.Time
	lastClickIndex         int
	lastClickList          string
	lastClickWasMouse      bool
	playbackSource         string
}

type UIPanels struct {
	TopBar  *panels.TopBar
	Tracks  *panels.Tracks
	Library *panels.Library
	Queue   *panels.Queue
}

type boxStyler interface {
	SetBorderColor(tcell.Color) *tview.Box
	SetTitleColor(tcell.Color) *tview.Box
	SetTitle(string) *tview.Box
}

type titleAligner interface {
	SetTitleAlign(int) *tview.Box
}

func NewApp(cfg config.Config, configPath string, configScope string, dbPath string, db *storage.Database, tasks *state.TaskManager, playerClient player.Player) *App {
	app := tview.NewApplication()
	app.EnableMouse(true)
	pages := tview.NewPages()

	panelsGroup := &UIPanels{
		TopBar:  panels.NewTopBar(),
		Tracks:  panels.NewTracks(formatTracksHeader(0, "title", true)),
		Library: panels.NewLibrary(cfg),
		Queue:   panels.NewQueue(),
	}

	trackInfo := panels.NewTrackInfo()

	queueRatio := 3
	infoRatio := 2
	showTrackInfo := true
	if cfg.Panel.ShowTrackInfo != nil {
		showTrackInfo = *cfg.Panel.ShowTrackInfo
	}

	rightPane := tview.NewFlex().SetDirection(tview.FlexRow)
	if showTrackInfo {
		rightPane.AddItem(panelsGroup.Queue.Container, 0, queueRatio, false)
		rightPane.AddItem(trackInfo, 0, infoRatio, false)
	} else {
		rightPane.AddItem(panelsGroup.Queue.Container, 0, 1, false)
		rightPane.AddItem(trackInfo, 0, 0, false)
	}

	leftWidth := clampPercent(cfg.Panel.LibraryWidth, 15, 40)
	rightWidth := clampPercent(cfg.Panel.QueueWidth, 15, 40)
	centerWidth := 100 - leftWidth - rightWidth
	if centerWidth < 20 {
		centerWidth = 20
		leftWidth = 40
		rightWidth = 40
	}

	libraryLayout := tview.NewFlex().SetDirection(tview.FlexColumn)
	libraryLayout.AddItem(panelsGroup.Library.NavPanel, 0, leftWidth, true)
	libraryLayout.AddItem(panelsGroup.Library.ContentPanel, 0, centerWidth, true)
	libraryLayout.AddItem(rightPane, 0, rightWidth, false)

	tracksLayout := tview.NewFlex().SetDirection(tview.FlexColumn)
	tracksLayout.AddItem(panelsGroup.Tracks.Container, 0, 2, true)
	tracksLayout.AddItem(rightPane, 0, 1, false)

	bodyPages := tview.NewPages()
	bodyPages.AddPage("tracks", tracksLayout, true, true)
	bodyPages.AddPage("library", libraryLayout, true, false)

	bottomBar := panels.NewBottomBar()
	bottomHints := bottomBar.Hints

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(panelsGroup.TopBar.Root, 3, 0, false)
	root.AddItem(bodyPages, 0, 1, true)
	root.AddItem(bottomBar.Root, 1, 0, false)

	palette := panels.NewCommandPalette()
	filterDialog := panels.NewFilterDialog()
	statsDialog := NewStatsDialog()
	rescanDialog := NewRescanDialog()
	tagEditor := NewTagEditorDialog()
	taskDetails := NewTaskDetailsDialog()
	logDialog := panels.NewLogDialog()
	logFileDialog := panels.NewLogFileDialog()
	headerText := fmt.Sprintf("Scope: %s\nConfig: %s\nDB: %s", configScope, configPath, dbPath)
	themeNames := make([]string, 0, len(Themes))
	for name := range Themes {
		themeNames = append(themeNames, name)
	}
	sort.Strings(themeNames)
	settings := panels.NewSettingsView(cfg, headerText, themeNames)

	pages.AddPage("root", root, true, true)
	filterOverlay := centeredOverlayPercent(filterDialog.Root, 60, 70)
	pages.AddPage("filters", filterOverlay, true, false)
	statsOverlay := centeredOverlay(statsDialog.Root, 12, 70)
	pages.AddPage("stats", statsOverlay, true, false)
	rescanOverlay := centeredOverlay(rescanDialog.Root, 12, 70)
	pages.AddPage("rescan", rescanOverlay, true, false)
	tagEditorOverlay := centeredOverlay(tagEditor.Root, 16, 80)
	pages.AddPage("tag-editor", tagEditorOverlay, true, false)
	tasksOverlay := centeredOverlay(taskDetails.Root, 18, 90)
	pages.AddPage("tasks", tasksOverlay, true, false)
	pages.AddPage("settings", settings.Root, true, false)
	paletteOverlay := centeredOverlay(palette.Root, 12, 70)
	pages.AddPage("palette", paletteOverlay, true, false)
	logOverlay := centeredOverlay(logDialog.Root, 18, 90)
	pages.AddPage("log", logOverlay, true, false)
	logFileOverlay := centeredOverlay(logFileDialog.Root, 20, 100)
	pages.AddPage("logfile", logFileOverlay, true, false)

	ui := &App{
		App:                 app,
		Pages:               pages,
		Root:                root,
		Panels:              panelsGroup,
		BodyPages:           bodyPages,
		RightPane:           rightPane,
		TrackInfo:           trackInfo,
		Settings:            settings,
		ShowTrackInfo:       showTrackInfo,
		ViewMode:            "tracks",
		QueueIndex:          -1,
		Volume:              config.ResolveVolume(cfg),
		DoubleClickPlayback: config.ResolveDoubleClickPlay(cfg),
		eqFrames:            defaultEqualizerFrames(),
		BottomBar:           bottomBar.Root,
		BottomHints:         bottomHints,
		Palette:             palette,
		FilterDialog:        filterDialog,
		LogDialog:           logDialog,
		LogFileDialog:       logFileDialog,
		StatsDialog:         statsDialog,
		RescanDialog:        rescanDialog,
		TagEditor:           tagEditor,
		SelectedLibs:        map[string]bool{},
		Theme:               ThemeByName(cfg.Theme),
		Config:              cfg,
		ConfigPath:          configPath,
		tasks:               tasks,
		db:                  db,
		TaskDetails:         taskDetails,
		Player:              playerClient,
		QueueScope:          queueScope(db),
		QueuePlayed:         map[int]bool{},
		TracksSelected:      map[int]bool{},
		LibrarySelected:     map[int]bool{},
		TracksCursor:        -1,
		LibraryCursor:       -1,
		QueueCursor:         -1,
		LogMax:              2000,
		sortMode:            "title",
		sortAsc:             true,
	}
	ui.registerOverlay("filters", filterOverlay, false)
	ui.registerOverlay("stats", statsOverlay, false)
	ui.registerOverlay("rescan", rescanOverlay, false)
	ui.registerOverlay("tag-editor", tagEditorOverlay, false)
	ui.registerOverlay("tasks", tasksOverlay, false)
	ui.registerOverlay("palette", paletteOverlay, false)
	ui.registerOverlay("log", logOverlay, false)
	ui.registerOverlay("logfile", logFileOverlay, false)
	if len(ui.eqFrames) > 0 {
		ui.eqFrame = ui.eqFrames[0]
	}
	ui.Panels.Tracks.Header.SetText(formatTracksHeaderNoSort(0))
	if ui.Panels.Tracks.SortInfo != nil {
		ui.Panels.Tracks.SortInfo.SetText(formatSortInfo(ui.sortMode, ui.sortAsc))
	}
	if ui.Panels.Library.NavHeader != nil {
		ui.Panels.Library.NavHeader.SetText(formatTracksHeaderNoSort(0))
	}
	if ui.Panels.Library.ContentHeader != nil {
		ui.Panels.Library.ContentHeader.SetText(formatTracksHeaderNoSort(0))
	}
	if ui.Panels.Queue.Header != nil {
		ui.Panels.Queue.Header.SetText(formatTracksHeaderNoSort(0))
	}
	player.SetLogger(ui.AppendLogf)
	if playerClient != nil {
		if notifier, ok := playerClient.(player.EOFNotifier); ok {
			notifier.SetOnEOF(ui.handleTrackEnd)
		}
		_ = playerClient.SetVolume(ui.Volume)
	}
	ui.syncVolumeFromPlayer()
	ui.bindTopBarMouse()
	ui.bottomMarqueeDir = 1
	ui.updateBottomHints()

	ui.Panels.Library.Nav.SetFocusFunc(func() {
		ui.setMainFocus("nav")
	})
	ui.Panels.TopBar.Timeline.SetFocusFunc(func() {
		ui.setMainFocus("top")
	})
	ui.Panels.Tracks.Filter.SetFocusFunc(func() {
		ui.setMainFocus("tracks")
	})
	ui.Panels.Tracks.List.SetFocusFunc(func() {
		ui.setMainFocus("tracks")
		ui.updateTrackInfoFromList("tracks", ui.Panels.Tracks.List.GetCurrentItem())
	})
	ui.Panels.Library.NavFilter.SetFocusFunc(func() {
		ui.setMainFocus("nav")
	})
	ui.Panels.Library.Content.SetFocusFunc(func() {
		ui.setMainFocus("content")
		ui.updateTrackInfoFromList("library", ui.Panels.Library.Content.GetCurrentItem())
	})
	ui.Panels.Queue.List.SetFocusFunc(func() {
		ui.setMainFocus("queue")
	})
	ui.TrackInfo.SetFocusFunc(func() {
		ui.setMainFocus("trackinfo")
	})

	ui.Panels.Tracks.List.SetSelectedFunc(func(index int, main string, secondary string, shortcut rune) {
		if index < 0 || index >= len(ui.TracksData) {
			return
		}
		ui.handleListDoubleClick("tracks", index, func() {
			ui.playTrackFromSource(ui.TracksData[index], playbackSourceOther)
		})
	})
	ui.Panels.Tracks.List.SetChangedFunc(func(index int, main string, secondary string, shortcut rune) {
		previous := ui.TracksCursor
		if ui.Panels.Tracks.List.GetItemCount() == 0 {
			ui.TracksCursor = index
			ui.updateTrackInfoFromList("tracks", index)
			return
		}
		if previous != index {
			ui.updateTrackRow("tracks", previous)
			ui.TracksCursor = index
			ui.updateTrackRow("tracks", index)
		}
		ui.updateTrackInfoFromList("tracks", index)
	})
	ui.Panels.Tracks.List.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if action == tview.MouseLeftClick {
			ui.lastClickWasMouse = true
		}
		return action, event
	})

	ui.Panels.Library.Content.SetSelectedFunc(func(index int, main string, secondary string, shortcut rune) {
		if index < 0 || index >= len(ui.LibraryData) {
			return
		}
		ui.handleListDoubleClick("library", index, func() {
			ui.playTrackFromSource(ui.LibraryData[index], playbackSourceOther)
		})
	})
	ui.Panels.Library.Content.SetChangedFunc(func(index int, main string, secondary string, shortcut rune) {
		previous := ui.LibraryCursor
		if ui.Panels.Library.Content.GetItemCount() == 0 {
			ui.LibraryCursor = index
			ui.updateTrackInfoFromList("library", index)
			return
		}
		if previous != index {
			ui.updateTrackRow("library", previous)
			ui.LibraryCursor = index
			ui.updateTrackRow("library", index)
		}
		ui.updateTrackInfoFromList("library", index)
	})
	ui.Panels.Library.Content.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if action == tview.MouseLeftClick {
			ui.lastClickWasMouse = true
		}
		return action, event
	})

	ui.Panels.Queue.List.SetSelectedFunc(func(index int, main string, secondary string, shortcut rune) {
		if index < 0 || index >= len(ui.QueueItems) {
			return
		}
		ui.handleListDoubleClick("queue", index, func() {
			ui.playQueueIndex(index)
		})
	})
	ui.Panels.Queue.List.SetChangedFunc(func(index int, main string, secondary string, shortcut rune) {
		previous := ui.QueueCursor
		if ui.Panels.Queue.List.GetItemCount() == 0 {
			ui.QueueCursor = index
			return
		}
		if previous != index {
			ui.updateQueueRow(previous)
			ui.QueueCursor = index
			ui.updateQueueRow(index)
		}
	})
	ui.Panels.Queue.List.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if action == tview.MouseLeftClick {
			ui.lastClickWasMouse = true
		}
		return action, event
	})

	ui.setMainFocus("content")
	ui.setActiveSection("libraries")
	ui.setLibrariesColumnFocus("left")
	ui.setStatusMessage("Ready")
	ui.updateViewMode(viewModeFromConfig(cfg.MainView))
	ui.updateNowPlaying()
	ui.applyTheme(ui.Theme)
	ui.loadSelectedLibraries()
	ui.loadQueue()
	// Tick-driven UI updates start in Run().
	ui.bindPaletteCommands()
	ui.bindFilters()
	ui.bindSettings()
	ui.bindNavigation()
	ui.bindKeyHandlers()
	ui.restoreLibraryNavCursor()
	ui.bindGlobalKeys(bodyPages)
	ui.setActiveSection("libraries")

	app.SetAfterDrawFunc(func(screen tcell.Screen) {
		if ui.tickersStarted {
			ui.updateTrackListLayout()
			ui.updateTrackScrollBar()
			return
		}
		ui.tickersStarted = true
		ui.appRunning = true
		ui.startStatusTicker()
		ui.startNowPlayingTicker()
		ui.updateTrackListLayout()
		ui.updateTrackScrollBar()
		go func() {
			time.Sleep(150 * time.Millisecond)
			ui.refreshTracks()
		}()
	})
	app.SetRoot(pages, true)
	return ui
}

func (a *App) Run() error {
	err := a.App.Run()
	a.appRunning = false
	return err
}

func (a *App) SetStatsTotals(totals storage.StatsTotals, hasData bool) {
	a.StatsDialog.SetTotals(totals, hasData)
}

func (a *App) refreshStatsTotals() {
	if a.db == nil {
		return
	}
	roots := a.selectedLibraryPaths()
	if len(roots) == 0 {
		go func() {
			a.App.QueueUpdateDraw(func() {
				a.SetStatsTotals(storage.StatsTotals{}, false)
			})
		}()
		return
	}
	go func() {
		totals, err := storage.ComputeTotalsInLibraries(a.db.DB, roots)
		if err != nil {
			a.App.QueueUpdateDraw(func() {
				a.setStatusMessage("Failed to refresh stats")
			})
			return
		}
		_ = storage.SaveTotals(a.db.DB, totals)
		a.App.QueueUpdateDraw(func() {
			a.SetStatsTotals(totals, true)
		})
	}()
}

func (a *App) applyTheme(theme Theme) {
	widgets := []interface{}{a.Panels.TopBar.Song, a.Panels.TopBar.Timeline, a.Panels.TopBar.Volume, a.TrackInfo}
	for _, widget := range widgets {
		if view, ok := widget.(*tview.TextView); ok {
			view.SetTextColor(theme.Fg).SetBackgroundColor(theme.Bg).SetBorderColor(theme.Border)
		}
	}
	if a.BottomHints != nil {
		a.BottomHints.SetTextColor(theme.Border).SetBackgroundColor(theme.Bg)
	}

	a.Panels.Tracks.List.SetMainTextColor(theme.Fg).SetSelectedTextColor(theme.Bg).SetSelectedBackgroundColor(theme.Accent)
	a.Panels.Tracks.Filter.SetFieldTextColor(theme.Fg).SetFieldBackgroundColor(theme.Bg)
	a.Panels.Tracks.Filter.SetLabelColor(theme.Muted)
	a.Panels.Tracks.Filter.SetBorderColor(theme.Border)
	if a.Panels.Tracks.SortInfo != nil {
		a.Panels.Tracks.SortInfo.SetTextColor(theme.Muted).SetBackgroundColor(theme.Bg)
	}
	if a.Panels.Tracks.Scroll != nil {
		a.Panels.Tracks.Scroll.SetTextColor(theme.Muted).SetBackgroundColor(theme.Bg)
	}
	a.Panels.Library.Content.SetMainTextColor(theme.Fg).SetSelectedTextColor(theme.Bg).SetSelectedBackgroundColor(theme.Accent)
	a.Panels.Library.NavFilter.SetFieldTextColor(theme.Fg).SetFieldBackgroundColor(theme.Bg)
	a.Panels.Library.NavFilter.SetLabelColor(theme.Muted)
	a.Panels.Library.NavFilter.SetBorderColor(theme.Border)
	a.Panels.Library.ContentPanel.SetBorderColor(theme.Border)
	a.Panels.Queue.List.SetMainTextColor(theme.Fg).SetSelectedTextColor(theme.Bg).SetSelectedBackgroundColor(theme.Accent)
	if a.Panels.Queue.Container != nil {
		a.Panels.Queue.Container.SetBorderColor(theme.Border)
	}
	a.Panels.Library.Nav.SetGraphicsColor(theme.Muted)
	a.Panels.Library.Nav.SetBorderColor(theme.Border)

	a.Settings.Root.SetBorderColor(theme.Border)
	a.Settings.Header.SetTextColor(theme.Fg).SetBackgroundColor(theme.Bg).SetBorderColor(theme.Border)
	a.setPanelStyle(a.Settings.Header, "Settings", false)
	a.Settings.LibrariesMsg.SetTextColor(theme.Muted)
	a.Settings.TreeMsg.SetTextColor(theme.Muted)
	a.Settings.ShortcutsMsg.SetTextColor(theme.Muted)
	a.Settings.AnalysisMsg.SetTextColor(theme.Muted)
	a.Settings.ThemeMsg.SetTextColor(theme.Muted)
	a.Settings.Libraries.SetMainTextColor(theme.Fg).SetBorderColor(theme.Border)
	a.Settings.LibraryTree.SetGraphicsColor(theme.Muted)
	a.Settings.LibraryTree.SetBorderColor(theme.Border)
	a.Settings.Shortcuts.SetMainTextColor(theme.Fg).SetBorderColor(theme.Border)
	a.Settings.Analysis.SetMainTextColor(theme.Fg).SetBorderColor(theme.Border)
	a.Settings.Themes.SetMainTextColor(theme.Fg).SetBorderColor(theme.Border)
	a.Palette.Input.SetFieldTextColor(theme.Fg).SetFieldBackgroundColor(theme.Bg)
	a.Palette.List.SetMainTextColor(theme.Fg).SetBorderColor(theme.Border)
	if a.TagEditor != nil {
		a.TagEditor.applyTextColors(theme)
	}
	if a.LogDialog != nil {
		a.LogDialog.View.SetTextColor(theme.Fg).SetBackgroundColor(theme.Bg)
		a.LogDialog.Root.SetBorderColor(theme.Border)
		a.setPanelStyle(a.LogDialog.Root, "Log (debug)", a.LogDialog.Active)
	}
	if a.LogFileDialog != nil {
		a.LogFileDialog.View.SetTextColor(theme.Fg).SetBackgroundColor(theme.Bg)
		a.LogFileDialog.Footer.SetTextColor(theme.Muted).SetBackgroundColor(theme.Bg)
		a.LogFileDialog.Root.SetBorderColor(theme.Border)
		a.setPanelStyle(a.LogFileDialog.Root, "Log File", a.LogFileDialog.Active)
	}
	a.FilterDialog.Form.SetLabelColor(theme.Fg).SetButtonTextColor(theme.Bg)
	a.FilterDialog.Form.SetButtonBackgroundColor(theme.Accent)
	a.StatsDialog.View.SetTextColor(theme.Fg).SetBackgroundColor(theme.Bg)
	a.StatsDialog.Root.SetBorderColor(theme.Border)
	a.RescanDialog.Root.SetFieldTextColor(theme.Fg).SetFieldBackgroundColor(theme.Bg)
	a.RescanDialog.Root.SetLabelColor(theme.Fg).SetBorderColor(theme.Border)
	a.Panels.TopBar.Root.SetBorderColor(theme.Border)

	a.setPanelStyle(a.Palette.Root, "Command Palette", a.Palette.Active)
	a.setPanelStyle(a.FilterDialog.Form, "Filters", a.FilterDialog.Active)
	a.setPanelStyle(a.StatsDialog.Root, "Stats", a.StatsDialog.Active)
	a.setPanelStyle(a.RescanDialog.Root, "Rescan", a.RescanDialog.Active)
	a.setPanelStyle(a.TaskDetails.Root, "Task Details", a.TaskDetails.Active)
	a.setMainFocus(a.ActivePanel)
	a.setActiveSection(a.ActiveSection)
	a.setLibrariesColumnFocus(a.ActiveLibrariesColumn)
}

func (a *App) startStatusTicker() {
	ticker := time.NewTicker(150 * time.Millisecond)
	go func() {
		for range ticker.C {
			if !a.appRunning {
				continue
			}
			snapshot := a.statusSnapshot()
			a.App.QueueUpdateDraw(func() {
				a.advanceBottomMarquee()
				a.updateBottomBarLine()
				a.flushLogView(false)
				if a.TaskDetails.Active {
					a.TaskDetails.SetSnapshot(snapshot)
				}
			})
		}
	}()
}

func (a *App) setStatusMessage(message string) {
	a.statusMu.Lock()
	a.statusMessage = message
	a.statusMu.Unlock()
	a.AppendLog("info", message)
}

func (a *App) AppendLog(level string, message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	if level == "" {
		level = "info"
	}
	line := fmt.Sprintf("%s [%s] %s", time.Now().Format("15:04:05"), strings.ToUpper(level), message)
	if a.logger != nil {
		if zapLevel, ok := parseZapLevel(level); ok {
			if ce := a.logger.Check(zapLevel, message); ce != nil {
				ce.Write(zap.String("line", line))
			}
		}
	}

	a.logMu.Lock()
	if a.LogMax <= 0 {
		a.LogMax = 2000
	}
	a.LogBuffer = append(a.LogBuffer, line)
	if len(a.LogBuffer) > a.LogMax {
		a.LogBuffer = a.LogBuffer[len(a.LogBuffer)-a.LogMax:]
	}
	a.logDirty = true
	a.logMu.Unlock()
}

func (a *App) SetLogger(logger *zap.Logger) {
	a.logger = logger
}

func parseZapLevel(level string) (zapcore.Level, bool) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zapcore.DebugLevel, true
	case "info":
		return zapcore.InfoLevel, true
	case "warn", "warning":
		return zapcore.WarnLevel, true
	case "error":
		return zapcore.ErrorLevel, true
	default:
		return zapcore.ErrorLevel, false
	}
}

func (a *App) AppendLogf(level string, format string, args ...any) {
	a.AppendLog(level, fmt.Sprintf(format, args...))
}

func (a *App) flushLogView(force bool) {
	if a.LogDialog == nil {
		return
	}
	a.logMu.Lock()
	if !force && !a.logDirty {
		a.logMu.Unlock()
		return
	}
	text := strings.Join(a.LogBuffer, "\n")
	a.logDirty = false
	a.logMu.Unlock()

	a.LogDialog.View.SetText(text)
	if a.LogDialog.AutoScroll {
		a.LogDialog.View.ScrollToEnd()
	}
}

func (a *App) statusSnapshot() state.TaskSnapshot {
	if a.tasks == nil {
		return state.TaskSnapshot{}
	}
	return a.tasks.Snapshot()
}

func (a *App) focusMainList() {
	if a.ViewMode == "tracks" {
		a.App.SetFocus(a.Panels.Tracks.List)
		a.setMainFocus("tracks")
		return
	}
	a.App.SetFocus(a.Panels.Library.Content)
	a.setMainFocus("content")
}

func (a *App) focusNowPlaying() {
	if a.Panels == nil || a.Panels.TopBar == nil {
		return
	}
	a.App.SetFocus(a.Panels.TopBar.Timeline)
	a.setMainFocus("top")
}

func (a *App) updateTrackInfoTitle() {
	if a.TrackInfo == nil {
		return
	}
	a.TrackInfo.SetTitle(formatTitle(a.trackInfoPanelTitle(), a.ActivePanel == "trackinfo"))
}

func (a *App) trackInfoPanelTitle() string {
	if a.ViewMode == "library" {
		return "4 Track Info"
	}
	return "3 Track Info"
}

func (a *App) setPanelStyle(box boxStyler, base string, focused bool) {
	color := a.Theme.Border
	title := formatTitle(base, false)
	if focused {
		color = a.Theme.Accent
		title = formatTitle(base, true)
	}
	box.SetBorderColor(color)
	box.SetTitleColor(color)
	box.SetTitle(title)
	if aligner, ok := box.(titleAligner); ok {
		aligner.SetTitleAlign(tview.AlignLeft)
	}
}

func queueScope(db *storage.Database) string {
	if db == nil || db.Scope == "" {
		return config.ScopeCurrent
	}
	return db.Scope
}

func viewModeFromConfig(value string) string {
	switch value {
	case "library_view":
		return "library"
	case "track_view":
		return "tracks"
	default:
		return "tracks"
	}
}

func (a *App) setMainFocus(panel string) {
	if panel == "" {
		panel = "content"
	}
	a.ActivePanel = panel

	a.setPanelStyle(a.Panels.TopBar.Root, "0 Now Playing", panel == "top")
	if a.ViewMode == "tracks" {
		a.setPanelStyle(a.Panels.Tracks.Container, a.tracksPanelTitle(), panel == "tracks")
		if a.Panels.Queue.Container != nil {
			a.setPanelStyle(a.Panels.Queue.Container, a.queuePanelTitle(), panel == "queue")
		}
	} else {
		a.setPanelStyle(a.Panels.Library.NavPanel, a.libraryNavTitle(), panel == "nav")
		a.setPanelStyle(a.Panels.Library.ContentPanel, a.libraryContentTitle(), panel == "content")
		if a.Panels.Queue.Container != nil {
			a.setPanelStyle(a.Panels.Queue.Container, a.queuePanelTitle(), panel == "queue")
		}
	}
	if a.TrackInfo != nil && a.ShowTrackInfo {
		a.setPanelStyle(a.TrackInfo, a.trackInfoPanelTitle(), panel == "trackinfo")
	}

	a.updateBottomHints()
	a.refreshHighlightedRows()
}

func (a *App) refreshHighlightedRows() {
	if a.Panels != nil && a.Panels.Tracks != nil && a.Panels.Tracks.List != nil {
		a.updateTrackRow("tracks", a.Panels.Tracks.List.GetCurrentItem())
	}
	if a.Panels != nil && a.Panels.Library != nil && a.Panels.Library.Content != nil {
		a.updateTrackRow("library", a.Panels.Library.Content.GetCurrentItem())
	}
	if a.Panels != nil && a.Panels.Queue != nil && a.Panels.Queue.List != nil {
		a.updateQueueRow(a.Panels.Queue.List.GetCurrentItem())
	}
}

func (a *App) updateViewMode(mode string) {
	if mode == "" {
		mode = "tracks"
	}
	a.ViewMode = mode
	switch mode {
	case "tracks":
		a.Config.MainView = "track_view"
	case "library":
		a.Config.MainView = "library_view"
	}
	a.updateTrackInfoTitle()
	a.saveConfig()
	switch mode {
	case "tracks":
		a.BodyPages.SwitchToPage("tracks")
		a.setMainFocus("tracks")
		a.refreshTracks()
	case "library":
		a.BodyPages.SwitchToPage("library")
		a.setMainFocus("nav")
		a.refreshLibraryTotals()
	}
}

func (a *App) updateBottomHints() {
	text := a.bottomHintText()
	title := a.bottomTitleText()
	if text == a.bottomHint && title == a.bottomTitle {
		return
	}
	a.bottomHint = text
	a.bottomTitle = title
	a.bottomMarqueeOffset = 0
	if a.bottomMarqueeDir == 0 {
		a.bottomMarqueeDir = 1
	}
	a.bottomMarqueeHold = 6
	if a.BottomHints != nil {
		a.updateBottomBarLine()
	}
}

func (a *App) updateBottomBarLine() {
	if a.BottomHints == nil {
		return
	}
	_, _, width, _ := a.BottomHints.GetInnerRect()
	if width <= 0 {
		return
	}
	line := a.buildBottomBarLine(width)
	a.BottomHints.SetText(line)
}

func (a *App) advanceBottomMarquee() {
	if a.BottomHints == nil {
		return
	}
	_, _, width, _ := a.BottomHints.GetInnerRect()
	if width <= 0 {
		return
	}
	if a.bottomMarqueeHold > 0 {
		a.bottomMarqueeHold--
		return
	}
	_, available := a.bottomBarParts(width)
	if available <= 0 {
		return
	}
	hintWidth := runewidth.StringWidth(a.bottomHint)
	if hintWidth <= available {
		a.bottomMarqueeOffset = 0
		return
	}
	maxOffset := hintWidth - available
	if a.bottomMarqueeDir == 0 {
		a.bottomMarqueeDir = 1
	}
	a.bottomMarqueeOffset += a.bottomMarqueeDir
	if a.bottomMarqueeOffset <= 0 {
		a.bottomMarqueeOffset = 0
		a.bottomMarqueeDir = 1
	}
	if a.bottomMarqueeOffset >= maxOffset {
		a.bottomMarqueeOffset = maxOffset
		a.bottomMarqueeDir = -1
	}
}

func (a *App) buildBottomBarLine(width int) string {
	if width <= 0 {
		return ""
	}
	line := ""
	panel := a.bottomTitle
	hints := a.bottomHint
	prefix := fmt.Sprintf("── [ %s ] ─ [ ", panel)
	suffix := " ]"
	if runewidth.StringWidth(prefix)+runewidth.StringWidth(suffix) > width {
		line = runewidth.Truncate(prefix, width, "")
		return line
	}
	available := width - runewidth.StringWidth(prefix) - runewidth.StringWidth(suffix)
	if available < 0 {
		available = 0
	}
	content := hints
	if available > 0 {
		hintWidth := runewidth.StringWidth(hints)
		if hintWidth > available {
			content = sliceByWidth(hints, a.bottomMarqueeOffset, available)
		} else {
			content = runewidth.Truncate(hints, available, "")
		}
	}
	line = prefix + padRight(content, available) + suffix
	return line
}

func (a *App) bottomBarParts(width int) (int, int) {
	panel := a.bottomTitle
	prefix := fmt.Sprintf("── [ %s ] ─ [ ", panel)
	suffix := " ]"
	available := width - runewidth.StringWidth(prefix) - runewidth.StringWidth(suffix)
	return runewidth.StringWidth(prefix) + runewidth.StringWidth(suffix), available
}

func padRight(value string, width int) string {
	current := runewidth.StringWidth(value)
	if current >= width {
		return value
	}
	return value + strings.Repeat(" ", width-current)
}

func sliceByWidth(value string, start int, width int) string {
	if width <= 0 {
		return ""
	}
	if start < 0 {
		start = 0
	}
	var b strings.Builder
	current := 0
	limit := start + width
	for _, r := range value {
		rw := runewidth.RuneWidth(r)
		if current+rw <= start {
			current += rw
			continue
		}
		if current >= limit {
			break
		}
		b.WriteRune(r)
		current += rw
		if current >= limit {
			break
		}
	}
	return b.String()
}

func (a *App) bottomHintText() string {
	infoHintTracks := ""
	infoHintLibrary := ""
	if a.ShowTrackInfo {
		infoHintTracks = " :3(info)"
		infoHintLibrary = " :4(info)"
	}
	switch {
	case a.Palette != nil && a.Palette.Active:
		return ":enter(run) :esc(close) :↑/↓(navigate)"
	case a.LogDialog != nil && a.LogDialog.Active:
		return ":esc(close) :c(clear) :a(auto-scroll) :y(copy)"
	case a.LogFileDialog != nil && a.LogFileDialog.Active:
		return ":esc(close) :f5(refresh) :pgup(load more)"
	case a.FilterDialog != nil && a.FilterDialog.Active:
		return ":enter(apply) :esc(close) :ctrl+v(paste)"
	case a.SettingsOpen:
		return a.settingsHintText()
	case a.RescanDialog != nil && a.RescanDialog.Active:
		return ":enter(confirm) :esc(close)"
	case a.StatsDialog != nil && a.StatsDialog.Active:
		return ":esc(close)"
	case a.TagEditor != nil && a.TagEditor.Active:
		return ":enter(save) :esc(close)"
	case a.TaskDetails != nil && a.TaskDetails.Active:
		return ":esc(close)"
	}

	if a.ViewMode == "library" {
		switch a.ActivePanel {
		case "nav":
			return ":enter(play) :a(add) :/(filter) :ctrl+f(filter)" + infoHintLibrary + " :ctrl+t(tracks) :ctrl+l(library)"
		case "content":
			return ":enter(play) :a(add) :/(filter) :ctrl+f(filter) :ctrl+o(open) :ctrl+e(edit tags)" + infoHintLibrary + " :ctrl+t(tracks) :ctrl+l(library)"
		case "queue":
			return ":enter(play) :d(remove) :ctrl+d(clear) :alt+up/down(move) :t(toggle info) :ctrl+o(open) :ctrl+e(edit tags)" + infoHintLibrary + " :ctrl+t(tracks) :ctrl+l(library)"
		case "top":
			return ":space(play/pause) :←/→(seek) :F1-F10(seek part) :+/- (volume) :t(toggle info)" + infoHintLibrary + " :ctrl+t(tracks) :ctrl+l(library)"
		}
	} else {
		switch a.ActivePanel {
		case "tracks":
			return ":enter(play) :a(add) :/(filter) :ctrl+f(filter) :ctrl+o(open) :ctrl+e(edit tags)" + infoHintTracks + " :ctrl+t(tracks) :ctrl+l(library)"
		case "queue":
			return ":enter(play) :d(remove) :ctrl+d(clear) :alt+up/down(move) :t(toggle info) :ctrl+o(open) :ctrl+e(edit tags)" + infoHintTracks + " :ctrl+t(tracks) :ctrl+l(library)"
		case "top":
			return ":space(play/pause) :←/→(seek) :F1-F10(seek part) :+/- (volume) :t(toggle info)" + infoHintTracks + " :ctrl+t(tracks) :ctrl+l(library)"
		}
	}

	return ":space(play/pause) :←/→(seek) :F1-F10(seek part) :+/- (volume) :ctrl+t(tracks) :ctrl+l(library)"
}

func (a *App) settingsHintText() string {
	section := a.ActiveSection
	if section == "" {
		section = "libraries"
	}
	switch section {
	case "shortcuts":
		return ":enter(rebind) :esc(close) :↑/↓(navigate)"
	case "analysis":
		return ":enter(edit) :esc(close) :↑/↓(navigate)"
	case "themes":
		return ":enter(apply) :esc(close) :↑/↓(navigate)"
	default:
		if a.ActiveLibrariesColumn == "right" {
			return ":space(add) :tab(switch) :esc(close)"
		}
		return ":d(remove) :r(rescan) :tab(switch) :esc(close)"
	}
}

func (a *App) bottomTitleText() string {
	switch {
	case a.Palette != nil && a.Palette.Active:
		return ":palette"
	case a.LogDialog != nil && a.LogDialog.Active:
		return ":log"
	case a.FilterDialog != nil && a.FilterDialog.Active:
		return ":filters"
	case a.SettingsOpen:
		return ":settings"
	case a.RescanDialog != nil && a.RescanDialog.Active:
		return ":rescan"
	case a.StatsDialog != nil && a.StatsDialog.Active:
		return ":stats"
	case a.TagEditor != nil && a.TagEditor.Active:
		return ":tags"
	case a.TaskDetails != nil && a.TaskDetails.Active:
		return ":tasks"
	}

	if a.ViewMode == "library" {
		switch a.ActivePanel {
		case "queue":
			return ":queue"
		case "top":
			return ":now-playing"
		default:
			return ":library"
		}
	}

	switch a.ActivePanel {
	case "tracks":
		return ":tracks"
	case "queue":
		return ":queue"
	case "top":
		return ":now-playing"
	default:
		return ":tracks"
	}
}

func firstLine(text string) string {
	if idx := strings.Index(text, "\n"); idx >= 0 {
		return text[:idx]
	}
	return text
}

func barRangeFromLine(line string) (int, int, bool) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return 0, 0, false
	}
	bar := ""
	if len(fields) == 3 && isBarField(fields[1]) {
		bar = fields[1]
	} else {
		last := fields[len(fields)-1]
		if isBarField(last) {
			bar = last
		}
	}
	if bar == "" {
		return 0, 0, false
	}
	start := strings.Index(line, bar)
	if start < 0 {
		return 0, 0, false
	}
	return start, len(bar), true
}

func isBarField(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r != '─' && r != '■' {
			return false
		}
	}
	return true
}

type transparentSpacer struct {
	*tview.Box
}

func newTransparentSpacer() *transparentSpacer {
	return &transparentSpacer{Box: tview.NewBox()}
}

func (s *transparentSpacer) Draw(_ tcell.Screen) {
}

func centeredOverlay(content tview.Primitive, height int, width int) *tview.Flex {
	if height <= 0 {
		height = 12
	}
	if width <= 0 {
		width = 70
	}

	row := tview.NewFlex().SetDirection(tview.FlexColumn)
	row.AddItem(newTransparentSpacer(), 0, 1, false)
	row.AddItem(content, width, 0, true)
	row.AddItem(newTransparentSpacer(), 0, 1, false)

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(newTransparentSpacer(), 0, 1, false)
	root.AddItem(row, height, 0, true)
	root.AddItem(newTransparentSpacer(), 0, 1, false)

	return root
}

type percentOverlay struct {
	*tview.Box
	content   tview.Primitive
	heightPct int
	widthPct  int
	minWidth  int
	minHeight int
	maxWidth  int
	maxHeight int
}

func centeredOverlayPercent(content tview.Primitive, heightPct int, widthPct int) *percentOverlay {
	if heightPct <= 0 {
		heightPct = 60
	}
	if widthPct <= 0 {
		widthPct = 70
	}
	return &percentOverlay{
		Box:       tview.NewBox(),
		content:   content,
		heightPct: heightPct,
		widthPct:  widthPct,
		minWidth:  40,
		minHeight: 10,
	}
}

func (o *percentOverlay) Draw(screen tcell.Screen) {
	width, height := screen.Size()
	if width <= 0 || height <= 0 {
		return
	}
	w := int(float64(width)*float64(o.widthPct)/100.0 + 0.5)
	h := int(float64(height)*float64(o.heightPct)/100.0 + 0.5)
	if o.minWidth > 0 && w < o.minWidth {
		w = o.minWidth
	}
	if o.minHeight > 0 && h < o.minHeight {
		h = o.minHeight
	}
	if o.maxWidth > 0 && w > o.maxWidth {
		w = o.maxWidth
	}
	if o.maxHeight > 0 && h > o.maxHeight {
		h = o.maxHeight
	}
	if w > width {
		w = width
	}
	if h > height {
		h = height
	}
	x := (width - w) / 2
	y := (height - h) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	o.SetRect(0, 0, width, height)
	o.content.SetRect(x, y, w, h)
	o.content.Draw(screen)
}

func (o *percentOverlay) Focus(delegate func(p tview.Primitive)) {
	if o.content != nil {
		delegate(o.content)
	}
}

func (o *percentOverlay) HasFocus() bool {
	if o.content == nil {
		return false
	}
	return o.content.HasFocus()
}

func (o *percentOverlay) Blur() {
	if o.content != nil {
		o.content.Blur()
	}
}

func (o *percentOverlay) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	if o.content == nil {
		return nil
	}
	return o.content.InputHandler()
}

func (o *percentOverlay) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	if o.content == nil {
		return nil
	}
	return o.content.MouseHandler()
}

func (a *App) resizeFocused(delta int) {
	focus := a.App.GetFocus()
	if a.ViewMode == "tracks" {
		switch focus {
		case a.Panels.Tracks.List, a.Panels.Tracks.Container:
			a.Config.Panel.LibraryWidth = clampPercent(a.Config.Panel.LibraryWidth+delta, 20, 80)
			a.setStatusMessage(fmt.Sprintf("Tracks width: %d%%", a.Config.Panel.LibraryWidth))
		case a.Panels.Queue.List, a.TrackInfo:
			a.Config.Panel.QueueWidth = clampPercent(a.Config.Panel.QueueWidth+delta, 20, 80)
			a.setStatusMessage(fmt.Sprintf("Queue width: %d%%", a.Config.Panel.QueueWidth))
		default:
			return
		}
		a.saveConfig()
		return
	}

	switch focus {
	case a.Panels.Library.Nav:
		a.Config.Panel.LibraryWidth = clampPercent(a.Config.Panel.LibraryWidth+delta, 15, 40)
		a.setStatusMessage(fmt.Sprintf("Nav width: %d%%", a.Config.Panel.LibraryWidth))
	case a.Panels.Queue.List, a.TrackInfo, a.Panels.Library.Content:
		a.Config.Panel.QueueWidth = clampPercent(a.Config.Panel.QueueWidth+delta, 15, 40)
		a.setStatusMessage(fmt.Sprintf("Right pane width: %d%%", a.Config.Panel.QueueWidth))
	default:
		return
	}

	a.saveConfig()
}

func (a *App) setActiveSection(section string) {
	if section == "" {
		section = "libraries"
	}
	a.ActiveSection = section

	a.setPanelStyle(a.Settings.LibrariesGrp, "1 Libraries", section == "libraries")
	a.setPanelStyle(a.Settings.ShortcutsGrp, "2 Shortcuts", section == "shortcuts")
	a.setPanelStyle(a.Settings.AnalysisGrp, "3 Analysis", section == "analysis")
	a.setPanelStyle(a.Settings.ThemesGrp, "4 Themes", section == "themes")
	a.updateBottomHints()
}

func (a *App) setLibrariesColumnFocus(column string) {
	if column == "" {
		column = "left"
	}
	a.ActiveLibrariesColumn = column

	a.setPanelStyle(a.Settings.LibrariesLeft, "Added library paths", column == "left")
	a.setPanelStyle(a.Settings.LibrariesRight, "Folder tree (add new)", column == "right")
	a.updateBottomHints()
}

func (a *App) toggleRightPane() {
	a.ShowTrackInfo = !a.ShowTrackInfo
	value := a.ShowTrackInfo
	a.Config.Panel.ShowTrackInfo = &value
	if a.ShowTrackInfo {
		if a.Panels.Queue.Container != nil {
			a.RightPane.ResizeItem(a.Panels.Queue.Container, 0, 3)
		}
		a.RightPane.ResizeItem(a.TrackInfo, 0, 2)
		a.setStatusMessage("Track info shown")
	} else {
		if a.Panels.Queue.Container != nil {
			a.RightPane.ResizeItem(a.Panels.Queue.Container, 0, 1)
		}
		a.RightPane.ResizeItem(a.TrackInfo, 0, 0)
		a.setStatusMessage("Track info hidden")
	}
	a.saveConfig()
}

func (a *App) cycleFocus() {
	focus := a.App.GetFocus()
	switch focus {
	case a.Panels.Library.Nav:
		if a.ViewMode == "library" {
			a.App.SetFocus(a.Panels.Library.Content)
			a.setMainFocus("content")
			return
		}
		a.App.SetFocus(a.Panels.Tracks.List)
		a.setMainFocus("tracks")
	case a.Panels.Tracks.List:
		if a.ViewMode == "tracks" {
			a.App.SetFocus(a.Panels.Queue.List)
			a.setMainFocus("queue")
			return
		}
	case a.Panels.Queue.List:
		if a.ShowTrackInfo {
			a.App.SetFocus(a.TrackInfo)
			a.setMainFocus("trackinfo")
			return
		}
	case a.Panels.Library.Content:
		if a.ViewMode == "library" {
			a.App.SetFocus(a.Panels.Queue.List)
			a.setMainFocus("queue")
			return
		}
		a.App.SetFocus(a.Panels.Queue.List)
		a.setMainFocus("queue")
	case a.TrackInfo:
		if a.ViewMode == "tracks" {
			a.App.SetFocus(a.Panels.Tracks.List)
			a.setMainFocus("tracks")
			return
		}
		a.App.SetFocus(a.Panels.Library.Nav)
		a.setMainFocus("nav")
	default:
		if a.ViewMode == "tracks" {
			a.App.SetFocus(a.Panels.Tracks.List)
			a.setMainFocus("tracks")
			return
		}
		a.App.SetFocus(a.Panels.Library.Nav)
		a.setMainFocus("nav")
	}
}

func (a *App) promptInput(title string, label string, onSubmit func(string)) {
	form := tview.NewForm()
	input := tview.NewInputField().SetLabel(label + ": ")
	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlV || (event.Rune() == 'v' && event.Modifiers()&tcell.ModMeta != 0) {
			a.appendClipboard(input)
			return nil
		}
		return event
	})
	form.AddFormItem(input)
	form.AddButton("Save", func() {
		onSubmit(strings.TrimSpace(input.GetText()))
		a.hidePromptDialog()
	})
	form.AddButton("Cancel", func() {
		a.hidePromptDialog()
	})
	form.SetBorder(true).SetTitle(fmt.Sprintf("[ %s ]", title))

	hint := tview.NewTextView().SetText("Ctrl+V to paste")

	modal := tview.NewFlex().SetDirection(tview.FlexRow)
	modal.AddItem(form, 0, 1, true)
	modal.AddItem(hint, 1, 0, false)
	a.pushOverlay("prompt", modal, true, nil)
	a.App.SetFocus(input)
}

func (a *App) hidePromptDialog() {
	if entry, ok := a.popOverlay("prompt"); ok {
		a.restoreOverlayFocus(entry)
		return
	}
	if a.Settings != nil {
		a.App.SetFocus(a.Settings.Libraries)
	}
}

func (a *App) removeLibrary(path string) {
	updated := []string{}
	for _, lib := range a.Config.Libraries {
		if lib != path {
			updated = append(updated, lib)
		}
	}
	a.Config.Libraries = updated
}

func (a *App) saveConfig() {
	_ = config.Save(a.ConfigPath, a.Config)
}

func (a *App) appendClipboard(input *tview.InputField) {
	text, err := clipboard.ReadAll()
	if err != nil {
		a.setStatusMessage("Clipboard unavailable")
		return
	}

	line := firstNonEmptyLine(text)
	if line == "" {
		a.setStatusMessage("Clipboard empty")
		return
	}

	current := input.GetText()
	input.SetText(current + line)
	a.setStatusMessage("Pasted from clipboard")
}

func firstNonEmptyLine(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (a *App) updateAnalysisList() {
	if a.Settings == nil || a.Settings.Analysis == nil {
		return
	}
	resolved := config.ResolveAnalysis(a.Config)
	list := a.Settings.Analysis
	list.SetItemText(0, "Analysis window (seconds)", fmt.Sprintf("%d", resolved.WindowSeconds))
	list.SetItemText(1, "Sample rate", fmt.Sprintf("%d", resolved.SampleRate))
	list.SetItemText(2, "BPM min", fmt.Sprintf("%d", resolved.BPMMin))
	list.SetItemText(3, "BPM max", fmt.Sprintf("%d", resolved.BPMMax))
	list.SetItemText(4, "Double-click to play", boolLabel(config.ResolveDoubleClickPlay(a.Config)))
}

func (a *App) handleAnalysisSelection(index int) {
	resolved := config.ResolveAnalysis(a.Config)
	applyValue := func(value int, set func(int) error) {
		if err := set(value); err != nil {
			a.setStatusMessage(err.Error())
			return
		}
		a.updateAnalysisList()
		a.saveConfig()
	}

	switch index {
	case 0:
		a.promptInput("Analysis window", "Seconds (0 = full)", func(text string) {
			value, err := strconv.Atoi(text)
			if err != nil || value < 0 {
				a.setStatusMessage("Invalid analysis window")
				return
			}
			applyValue(value, func(val int) error {
				a.Config.Analysis.WindowSeconds = &val
				return nil
			})
		})
	case 1:
		a.promptInput("Sample rate", "Hz", func(text string) {
			value, err := strconv.Atoi(text)
			if err != nil || value < 8000 {
				a.setStatusMessage("Invalid sample rate")
				return
			}
			applyValue(value, func(val int) error {
				a.Config.Analysis.SampleRate = &val
				return nil
			})
		})
	case 2:
		a.promptInput("BPM min", "Min BPM", func(text string) {
			value, err := strconv.Atoi(text)
			if err != nil || value <= 0 {
				a.setStatusMessage("Invalid BPM min")
				return
			}
			applyValue(value, func(val int) error {
				if resolved.BPMMax <= val {
					return fmt.Errorf("BPM min must be < BPM max")
				}
				a.Config.Analysis.BPMMin = &val
				return nil
			})
		})
	case 3:
		a.promptInput("BPM max", "Max BPM", func(text string) {
			value, err := strconv.Atoi(text)
			if err != nil || value <= 0 {
				a.setStatusMessage("Invalid BPM max")
				return
			}
			applyValue(value, func(val int) error {
				if val <= resolved.BPMMin {
					return fmt.Errorf("BPM max must be > BPM min")
				}
				a.Config.Analysis.BPMMax = &val
				return nil
			})
		})
	case 4:
		value := !a.DoubleClickPlayback
		a.DoubleClickPlayback = value
		a.Config.DoubleClickPlay = &value
		a.updateAnalysisList()
		a.saveConfig()
		a.setStatusMessage(fmt.Sprintf("Double-click to play: %s", boolLabel(value)))
	default:
		return
	}
}

func clampPercent(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
