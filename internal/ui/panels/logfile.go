package panels

import "github.com/rivo/tview"

type LogFileDialog struct {
	Root        *tview.Flex
	View        *tview.TextView
	Footer      *tview.TextView
	Active      bool
	Path        string
	Lines       []string
	StartLine   int
	EndLine     int
	TotalLines  int
	HasNewLines bool
	LastSize    int64
	LastOffset  int64
	TailBuffer  string
	StopCh      chan struct{}
}

func NewLogFileDialog() *LogFileDialog {
	view := tview.NewTextView().SetDynamicColors(true)
	view.SetScrollable(true)
	view.SetBorder(false)

	footer := tview.NewTextView().SetDynamicColors(true)
	footer.SetBorder(false)
	footer.SetTextAlign(tview.AlignRight)

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.SetBorder(true).SetTitle("[ Log File ]")
	root.AddItem(view, 0, 1, true)
	root.AddItem(footer, 1, 0, false)

	return &LogFileDialog{Root: root, View: view, Footer: footer}
}
