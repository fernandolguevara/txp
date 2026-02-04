package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/rivo/tview"

	"txp/internal/state"
)

type TaskDetailsDialog struct {
	Root   *tview.TextView
	Active bool
}

func NewTaskDetailsDialog() *TaskDetailsDialog {
	view := tview.NewTextView().SetDynamicColors(true)
	view.SetBorder(true).SetTitle("[ Task Details ]")
	view.SetText("No active tasks.")

	return &TaskDetailsDialog{Root: view}
}

func (t *TaskDetailsDialog) SetSnapshot(snapshot state.TaskSnapshot) {
	lines := []string{}
	if !snapshot.Active && snapshot.Pending == 0 {
		lines = append(lines, "Idle")
	} else {
		lines = append(lines, fmt.Sprintf("Phase: %s", snapshot.Phase))
		lines = append(lines, fmt.Sprintf("Pending: %d", snapshot.Pending))
		lines = append(lines, fmt.Sprintf("Processed: %d", snapshot.Processed))
		lines = append(lines, fmt.Sprintf("Errors: %d", snapshot.Errors))
		lines = append(lines, fmt.Sprintf("Workers: %d", snapshot.ActiveWorkers))
		lines = append(lines, fmt.Sprintf("Discover: %d/%d", snapshot.Discover.Processed, snapshot.Discover.Pending))
		lines = append(lines, fmt.Sprintf("Insert: %d/%d", snapshot.Insert.Processed, snapshot.Insert.Pending))
		lines = append(lines, fmt.Sprintf("Analyze: %d/%d", snapshot.Analyze.Processed, snapshot.Analyze.Pending))
		if snapshot.CurrentFile != "" {
			lines = append(lines, "")
			lines = append(lines, fmt.Sprintf("Current: %s", snapshot.CurrentFile))
		}
		if !snapshot.StartedAt.IsZero() {
			lines = append(lines, "")
			lines = append(lines, fmt.Sprintf("Elapsed: %s", time.Since(snapshot.StartedAt).Truncate(time.Second)))
		}
	}

	lines = append(lines, "")
	lines = append(lines, "Press Esc to close.")
	t.Root.SetText(strings.Join(lines, "\n"))
}
