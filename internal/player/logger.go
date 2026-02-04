package player

import "sync"

var (
	loggerMu sync.RWMutex
	logger   func(level string, format string, args ...any)
)

// SetLogger configures a logger for player events.
// The logger receives a level and printf-style format string.
func SetLogger(fn func(level string, format string, args ...any)) {
	loggerMu.Lock()
	logger = fn
	loggerMu.Unlock()
}

func logf(level string, format string, args ...any) {
	loggerMu.RLock()
	fn := logger
	loggerMu.RUnlock()
	if fn == nil {
		return
	}
	fn(level, format, args...)
}
