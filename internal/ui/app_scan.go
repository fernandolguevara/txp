package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"txp/internal/config"
	"txp/internal/library"
	"txp/internal/state"
	"txp/internal/storage"
)

func (a *App) rescanLibraries(paths []string, force bool) {
	if len(paths) == 0 {
		a.setStatusMessage("No libraries configured")
		return
	}
	if a.tasks == nil || a.db == nil {
		a.setStatusMessage("Scan unavailable")
		return
	}
	if !a.beginScan() {
		a.setStatusMessage("Scan already running")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.tasks.Start("scan", cancel)
	a.setStatusMessage("Rescanning libraries...")
	stopProgress := make(chan struct{})
	var stopOnce sync.Once
	stopProgressLoop := func() {
		stopOnce.Do(func() { close(stopProgress) })
	}
	go a.scanProgressLoop(stopProgress)

	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				stopProgressLoop()
				a.tasks.Finish()
				a.endScan()
				a.App.QueueUpdateDraw(func() {
					a.setStatusMessage("Scan failed: unexpected error")
					a.AppendLogf("error", "Scan panic: %v", recovered)
				})
			}
		}()

		resolved := config.ResolveAnalysis(a.Config)
		analysis := library.AnalysisSettings{
			WindowSeconds: resolved.WindowSeconds,
			SampleRate:    resolved.SampleRate,
			BPMMin:        resolved.BPMMin,
			BPMMax:        resolved.BPMMax,
		}
		result := library.ScanLibraries(ctx, paths, a.db, a.tasks, analysis, force)
		stopProgressLoop()
		a.tasks.Finish()
		a.endScan()
		a.App.QueueUpdateDraw(func() {
			a.setStatusMessage(fmt.Sprintf("Scan complete: %d tracks (%d errors)", result.Processed, result.Errors))
			a.refreshTracks()
			a.refreshStatsTotals()
		})
	}()
}

func (a *App) analyzeMissingTracks() {
	if a.db == nil || a.tasks == nil {
		a.setStatusMessage("Scan unavailable")
		return
	}
	if !a.beginScan() {
		a.setStatusMessage("Scan already running")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.tasks.Start("analyze-missing", cancel)
	a.setStatusMessage("Analyzing missing metadata...")
	stopProgress := make(chan struct{})
	var stopOnce sync.Once
	stopProgressLoop := func() {
		stopOnce.Do(func() { close(stopProgress) })
	}
	go a.scanProgressLoop(stopProgress)

	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				stopProgressLoop()
				a.tasks.Finish()
				a.endScan()
				a.App.QueueUpdateDraw(func() {
					a.setStatusMessage("Analyze failed: unexpected error")
					a.AppendLogf("error", "Analyze panic: %v", recovered)
				})
			}
		}()

		paths, err := storage.ListUnanalyzedPaths(a.db.DB)
		if err != nil {
			stopProgressLoop()
			a.tasks.Finish()
			a.endScan()
			a.App.QueueUpdateDraw(func() {
				a.setStatusMessage("Failed to load unanalyzed tracks")
			})
			return
		}
		if len(paths) == 0 {
			stopProgressLoop()
			a.tasks.Finish()
			a.endScan()
			a.App.QueueUpdateDraw(func() {
				a.setStatusMessage("No unanalyzed tracks")
			})
			return
		}

		resolved := config.ResolveAnalysis(a.Config)
		analysis := library.AnalysisSettings{
			WindowSeconds: resolved.WindowSeconds,
			SampleRate:    resolved.SampleRate,
			BPMMin:        resolved.BPMMin,
			BPMMax:        resolved.BPMMax,
		}
		result, analyzeErrors := library.AnalyzeTracks(ctx, paths, a.db, a.tasks, analysis)
		stopProgressLoop()
		a.tasks.Finish()
		a.endScan()
		a.App.QueueUpdateDraw(func() {
			a.setStatusMessage(fmt.Sprintf("Analyze complete: %d tracks (%d errors)", result.Processed, result.Errors))
			for _, errItem := range analyzeErrors {
				name := filepath.Base(errItem.Path)
				a.AppendLogf("error", "Analyze %s: %s: %v", errItem.Stage, name, errItem.Err)
			}
			a.refreshTracks()
			a.refreshStatsTotals()
		})
	}()
}

func (a *App) beginScan() bool {
	a.scanMu.Lock()
	defer a.scanMu.Unlock()
	if a.scanRunning {
		return false
	}
	a.scanRunning = true
	return true
}

func (a *App) endScan() {
	a.scanMu.Lock()
	a.scanRunning = false
	a.scanMu.Unlock()
}

func (a *App) scanProgressLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(350 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			snapshot := a.tasks.Snapshot()
			status := formatScanStatus(snapshot)
			a.App.QueueUpdateDraw(func() {
				a.setStatusMessage(status)
				if a.TaskDetails != nil && a.TaskDetails.Active {
					a.TaskDetails.SetSnapshot(snapshot)
				}
			})
		}
	}
}

func formatScanStatus(snapshot state.TaskSnapshot) string {
	if !snapshot.Active && snapshot.Pending == 0 {
		return "Scan idle"
	}
	return fmt.Sprintf(
		"Scanning… Discover %d/%d | Analyze %d/%d | Errors %d",
		snapshot.Discover.Processed,
		snapshot.Discover.Pending,
		snapshot.Analyze.Processed,
		snapshot.Analyze.Pending,
		snapshot.Errors,
	)
}
