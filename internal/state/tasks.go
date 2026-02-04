package state

import (
	"context"
	"sync"
	"time"
)

type TaskSnapshot struct {
	Active        bool
	Phase         string
	Pending       int
	Processed     int
	Errors        int
	CurrentFile   string
	ActiveWorkers int
	StartedAt     time.Time
	Discover      StageSnapshot
	Insert        StageSnapshot
	Analyze       StageSnapshot
}

type StageSnapshot struct {
	Pending   int
	Processed int
}

type TaskManager struct {
	mu        sync.Mutex
	active    bool
	phase     string
	pending   int
	processed int
	errors    int
	current   string
	workers   int
	startedAt time.Time
	cancel    context.CancelFunc
	discover  StageSnapshot
	insert    StageSnapshot
	analyze   StageSnapshot
}

func NewTaskManager() *TaskManager {
	return &TaskManager{}
}

func (t *TaskManager) Start(phase string, cancel context.CancelFunc) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cancel != nil {
		t.cancel()
	}
	t.active = true
	t.phase = phase
	t.pending = 0
	t.processed = 0
	t.errors = 0
	t.current = ""
	t.workers = 0
	t.startedAt = time.Now()
	t.cancel = cancel
	t.discover = StageSnapshot{}
	t.insert = StageSnapshot{}
	t.analyze = StageSnapshot{}
}

func (t *TaskManager) Finish() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.active = false
	t.phase = "idle"
	t.pending = 0
	t.current = ""
	t.workers = 0
	t.discover = StageSnapshot{}
	t.insert = StageSnapshot{}
	t.analyze = StageSnapshot{}
}

func (t *TaskManager) Cancel() {
	t.mu.Lock()
	if t.cancel != nil {
		t.cancel()
	}
	t.active = false
	t.phase = "cancelled"
	t.pending = 0
	t.current = ""
	t.workers = 0
	t.discover = StageSnapshot{}
	t.insert = StageSnapshot{}
	t.analyze = StageSnapshot{}
	t.mu.Unlock()
}

func (t *TaskManager) AddPending(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pending += n
}

func (t *TaskManager) ProcessedOne() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.pending > 0 {
		t.pending--
	}
	t.processed++
}

func (t *TaskManager) ErrorOne() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.errors++
}

func (t *TaskManager) SetCurrentFile(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.current = path
}

func (t *TaskManager) SetActiveWorkers(count int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.workers = count
}

func (t *TaskManager) Snapshot() TaskSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	return TaskSnapshot{
		Active:        t.active,
		Phase:         t.phase,
		Pending:       t.pending,
		Processed:     t.processed,
		Errors:        t.errors,
		CurrentFile:   t.current,
		ActiveWorkers: t.workers,
		StartedAt:     t.startedAt,
		Discover:      t.discover,
		Insert:        t.insert,
		Analyze:       t.analyze,
	}
}

func (t *TaskManager) AddDiscover(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.discover.Pending += n
}

func (t *TaskManager) DoneDiscover() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.discover.Pending > 0 {
		t.discover.Pending--
	}
	t.discover.Processed++
}

func (t *TaskManager) AddInsert(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.insert.Pending += n
}

func (t *TaskManager) DoneInsert() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.insert.Pending > 0 {
		t.insert.Pending--
	}
	t.insert.Processed++
}

func (t *TaskManager) AddAnalyze(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.analyze.Pending += n
	t.pending += n
}

func (t *TaskManager) DoneAnalyze() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.analyze.Pending > 0 {
		t.analyze.Pending--
	}
	if t.pending > 0 {
		t.pending--
	}
	t.analyze.Processed++
	t.processed++
}
