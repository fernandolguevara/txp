package library

import (
	"context"
	"runtime"
	"sync"

	"txp/internal/model"
	"txp/internal/state"
	"txp/internal/storage"
)

type AnalyzeError struct {
	Path  string
	Stage string
	Err   error
}

func AnalyzeTracks(ctx context.Context, paths []string, db *storage.Database, tasks *state.TaskManager, analysisSettings AnalysisSettings) (ScanResult, []AnalyzeError) {
	metaWorkers := max(4, runtime.NumCPU()*2)
	analysisWorkers := max(2, runtime.NumCPU())
	tasks.SetActiveWorkers(metaWorkers + analysisWorkers)

	metaQueue := make(chan string, 256)
	analysisQueue := make(chan model.Track, 128)
	var wgMeta sync.WaitGroup
	var wgAnalyze sync.WaitGroup
	var dbMu sync.Mutex
	var errMu sync.Mutex
	errors := []AnalyzeError{}
	addError := func(path string, stage string, err error) {
		if err == nil {
			return
		}
		errMu.Lock()
		errors = append(errors, AnalyzeError{Path: path, Stage: stage, Err: err})
		errMu.Unlock()
	}

	metaWorker := func() {
		for path := range metaQueue {
			select {
			case <-ctx.Done():
				return
			default:
			}
			tasks.SetCurrentFile(path)
			track, err := ReadMetadata(path)
			if err != nil {
				addError(path, "read", err)
				dbMu.Lock()
				_ = storage.UpsertTrackError(db.DB, path, "read: "+err.Error())
				dbMu.Unlock()
				tasks.ErrorOne()
				tasks.DoneAnalyze()
				continue
			}
			track.Duration = DetectDuration(path)
			track.Checksum = fastChecksumFromPath(track.Path)
			if !sendWithContext(ctx, analysisQueue, track) {
				return
			}
		}
	}

	analyzeWorker := func() {
		for track := range analysisQueue {
			select {
			case <-ctx.Done():
				return
			default:
			}
			track.Key = DetectKey(track.Path, analysisSettings)
			track.BPM = DetectBPM(track.Path, analysisSettings)
			dbMu.Lock()
			if err := storage.UpsertTrack(db.DB, track); err != nil {
				dbMu.Unlock()
				addError(track.Path, "upsert", err)
				_ = storage.UpsertTrackError(db.DB, track.Path, "upsert: "+err.Error())
				tasks.ErrorOne()
				tasks.DoneAnalyze()
				continue
			}
			_ = storage.EnsureTrackStats(db.DB, track.Path)
			_ = storage.ClearTrackError(db.DB, track.Path)
			dbMu.Unlock()
			tasks.DoneAnalyze()
		}
	}

	for i := 0; i < metaWorkers; i++ {
		wgMeta.Add(1)
		go func() {
			defer wgMeta.Done()
			metaWorker()
		}()
	}
	for i := 0; i < analysisWorkers; i++ {
		wgAnalyze.Add(1)
		go func() {
			defer wgAnalyze.Done()
			analyzeWorker()
		}()
	}

	for _, path := range paths {
		if path == "" {
			continue
		}
		tasks.AddAnalyze(1)
		if !sendWithContext(ctx, metaQueue, path) {
			break
		}
	}
	close(metaQueue)
	if len(paths) == 0 {
		tasks.SetActiveWorkers(0)
	}
	wgMeta.Wait()
	close(analysisQueue)
	wgAnalyze.Wait()

	snapshot := tasks.Snapshot()
	return ScanResult{Processed: snapshot.Processed, Errors: snapshot.Errors}, errors
}
