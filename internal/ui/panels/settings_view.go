package panels

import (
	"github.com/rivo/tview"

	"txp/internal/config"
)

type SettingsView struct {
	Root           *tview.Flex
	Header         *tview.TextView
	LibrariesGrp   *tview.Flex
	LibrariesLeft  *tview.Flex
	LibrariesRight *tview.Flex
	ShortcutsGrp   *tview.Flex
	AnalysisGrp    *tview.Flex
	ThemesGrp      *tview.Flex
	Libraries      *tview.List
	LibraryTree    *tview.TreeView
	Shortcuts      *tview.List
	Analysis       *tview.List
	Themes         *tview.List
	LibrariesMsg   *tview.TextView
	TreeMsg        *tview.TextView
	ShortcutsMsg   *tview.TextView
	AnalysisMsg    *tview.TextView
	ThemeMsg       *tview.TextView
}

func NewSettingsView(cfg config.Config, headerText string, themeNames []string) *SettingsView {
	header := tview.NewTextView()
	header.SetBorder(true).SetTitle("[ Settings ]")
	header.SetText(headerText)

	libsGrp, leftColumn, rightColumn, librariesList, libraryTree, librariesMsg, treeMsg := buildLibrariesGroup(cfg)
	shortcutsGrp, shortcutsList, shortcutsMsg := buildShortcutsGroup(cfg)
	analysisGrp, analysisList, analysisMsg := buildAnalysisGroup(cfg)
	themesGrp, themesList, themeMsg := buildThemesGroup(themeNames)

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(header, 5, 0, false)
	root.AddItem(libsGrp, 0, 2, true)
	root.AddItem(shortcutsGrp, 0, 1, false)
	root.AddItem(analysisGrp, 0, 1, false)
	root.AddItem(themesGrp, 0, 1, false)

	return &SettingsView{
		Root:           root,
		Header:         header,
		LibrariesGrp:   libsGrp,
		LibrariesLeft:  leftColumn,
		LibrariesRight: rightColumn,
		ShortcutsGrp:   shortcutsGrp,
		AnalysisGrp:    analysisGrp,
		ThemesGrp:      themesGrp,
		Libraries:      librariesList,
		LibraryTree:    libraryTree,
		Shortcuts:      shortcutsList,
		Analysis:       analysisList,
		Themes:         themesList,
		LibrariesMsg:   librariesMsg,
		TreeMsg:        treeMsg,
		ShortcutsMsg:   shortcutsMsg,
		AnalysisMsg:    analysisMsg,
		ThemeMsg:       themeMsg,
	}
}
