package panels

import (
	"fmt"
	"os/exec"

	"github.com/rivo/tview"

	"txp/internal/config"
)

func buildAnalysisGroup(cfg config.Config) (*tview.Flex, *tview.List, *tview.TextView) {
	analysisList := tview.NewList().ShowSecondaryText(true)
	analysisList.SetBorder(false)
	resolved := config.ResolveAnalysis(cfg)
	analysisList.AddItem("Analysis window (seconds)", fmt.Sprintf("%d", resolved.WindowSeconds), 0, nil)
	analysisList.AddItem("Sample rate", fmt.Sprintf("%d", resolved.SampleRate), 0, nil)
	analysisList.AddItem("BPM min", fmt.Sprintf("%d", resolved.BPMMin), 0, nil)
	analysisList.AddItem("BPM max", fmt.Sprintf("%d", resolved.BPMMax), 0, nil)
	analysisList.AddItem("Double-click to play", boolLabel(config.ResolveDoubleClickPlay(cfg)), 0, nil)
	analysisMsg := tview.NewTextView().SetText("Enter: edit")
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		analysisMsg.SetText("Enter: edit | M4A decode requires FFmpeg (CGO)")
	}

	analysisGrp := tview.NewFlex().SetDirection(tview.FlexRow)
	analysisGrp.SetBorder(true).SetTitle("[ (3) Analysis ]")
	analysisGrp.AddItem(analysisList, 0, 1, true)
	analysisGrp.AddItem(analysisMsg, 1, 0, false)

	return analysisGrp, analysisList, analysisMsg
}

func boolLabel(value bool) string {
	if value {
		return "On"
	}
	return "Off"
}
