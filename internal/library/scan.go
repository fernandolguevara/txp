package library

import (
	"context"
	"hash/fnv"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"txp/internal/model"
	"txp/internal/state"
	"txp/internal/storage"
)

type ScanResult struct {
	Processed int
	Errors    int
}

func ScanLibraries(ctx context.Context, paths []string, db *storage.Database, tasks *state.TaskManager, analysisSettings AnalysisSettings, force bool) ScanResult {
	discoverWorkers := max(2, runtime.NumCPU())
	metaWorkers := max(4, runtime.NumCPU()*2)
	analysisWorkers := max(2, runtime.NumCPU())
	activeWorkers := discoverWorkers + metaWorkers + analysisWorkers
	tasks.SetActiveWorkers(activeWorkers)

	discoverQueues := make([]chan string, discoverWorkers)
	metaQueue := make(chan string, 512)
	analysisQueue := make(chan model.Track, 256)
	var wgDiscover sync.WaitGroup
	var wgMeta sync.WaitGroup
	var wgAnalyze sync.WaitGroup
	var dbMu sync.Mutex

	for i := 0; i < discoverWorkers; i++ {
		discoverQueues[i] = make(chan string, 256)
	}

	insertWorker := func(path string, mtime int64, checksum string) {
		dbMu.Lock()
		_ = storage.InsertTrackStub(db.DB, path, mtime, checksum)
		_ = storage.EnsureTrackStats(db.DB, path)
		dbMu.Unlock()
	}

	discoverWorker := func(id int) {
		for path := range discoverQueues[id] {
			select {
			case <-ctx.Done():
				return
			default:
			}
			tasks.DoneDiscover()
			storedMtime, storedChecksum, exists, lookupErr := storage.GetTrackScanMeta(db.DB, path)
			if lookupErr != nil {
				tasks.ErrorOne()
				continue
			}
			info, statErr := os.Stat(path)
			if statErr != nil {
				tasks.ErrorOne()
				continue
			}
			mtime := info.ModTime().Unix()
			size := info.Size()
			checksum := fastChecksum(size, mtime)
			if !exists {
				tasks.AddInsert(1)
				insertWorker(path, mtime, checksum)
				tasks.DoneInsert()
				tasks.AddAnalyze(1)
				if !sendWithContext(ctx, metaQueue, path) {
					return
				}
				continue
			}
			if !force {
				if storedChecksum != "" && storedChecksum == checksum {
					continue
				}
				if storedMtime == mtime && storedChecksum == "" {
					continue
				}
			}
			tasks.AddAnalyze(1)
			if !sendWithContext(ctx, metaQueue, path) {
				return
			}
		}
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

	for i := 0; i < discoverWorkers; i++ {
		wgDiscover.Add(1)
		go func(id int) {
			defer wgDiscover.Done()
			discoverWorker(id)
		}(i)
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

	var walkWg sync.WaitGroup
	for _, root := range paths {
		root := root
		walkWg.Add(1)
		go func() {
			defer walkWg.Done()
			if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					tasks.ErrorOne()
					return nil
				}
				select {
				case <-ctx.Done():
					return context.Canceled
				default:
				}
				if d.IsDir() {
					return nil
				}
				if !isAudioFile(path) {
					return nil
				}
				idx := hashIndex(path, discoverWorkers)
				tasks.AddDiscover(1)
				if !sendWithContext(ctx, discoverQueues[idx], path) {
					return context.Canceled
				}
				return nil
			}); err != nil && err != context.Canceled {
				tasks.ErrorOne()
			}
		}()
	}

	walkWg.Wait()
	for _, queue := range discoverQueues {
		close(queue)
	}
	wgDiscover.Wait()
	close(metaQueue)
	wgMeta.Wait()
	close(analysisQueue)
	wgAnalyze.Wait()

	snapshot := tasks.Snapshot()
	return ScanResult{Processed: snapshot.Processed, Errors: snapshot.Errors}
}

func isAudioFile(path string) bool {
	name := strings.ToLower(filepath.Ext(path))
	switch name {
	case ".mp3", ".flac", ".m4a", ".wav", ".ogg", ".aac", ".aiff", ".alac":
		return true
	default:
		return false
	}
}

func hashIndex(path string, size int) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(path))
	return int(h.Sum32() % uint32(size))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
