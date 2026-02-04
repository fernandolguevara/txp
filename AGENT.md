# Project: txp

## Overview
- Terminal UI music player/library manager written in Go (tview/tcell).
- Scans audio libraries into SQLite, analyzes duration/key/BPM, and provides browsing, queueing, and playback.

## Repo layout
- `cmd/txp/main.go`: CLI entrypoint, config/db resolution, logging, start UI/player.
- `internal/config/`: config schema, defaults, load/save, path resolution.
- `internal/storage/`: SQLite access, schema/migrations, queue/errors/stats.
- `internal/library/`: library scan, metadata read, duration/key/BPM analysis.
- `internal/player/`: local audio playback via beep/oto.
- `internal/ui/`: TUI layout, panels, dialogs, shortcuts, palette, settings.
- `internal/model/`: core models (e.g., `Track`).

## Build and run
- Build: `go build ./cmd/txp`
- Run: `go run ./cmd/txp`
- Run with config: `go run ./cmd/txp --config-path /path/to/config.json` (directory paths resolve to `config.json`).

## Configuration
- Config is JSON; schema in `internal/config/config.go`.
- Default config path: `~/.txp/config.json` (auto-created if missing).
- `--config-path` uses the provided file and sets scope to `current` (for queue).
- Key fields:
  - `libraries` / `selectedLibraries`
  - `library_nav_cursor`
  - `main_view`: `track_view` or `library_view`
  - `theme`, `panel.libraryWidth`, `panel.queueWidth`, `panel.showTrackInfo`
  - `shortcuts` map (customizable)
  - `analysis_*` settings (window seconds, sample rate, BPM range)
  - `volume`, `enable_double_click_playback`

## Environment variables
- `TXP_LOG_PATH`: override log file path.
- `TXP_LOG_LEVEL`: `error` (default), `warn`, `info`, `debug`.
- `TXP_AUDIO_DIAG`: enable periodic audio diagnostics logs (any non-empty value except `0`/`false`).

## Database
- SQLite created automatically:
  - Default: `~/.txp/txp.db`
  - With `--config-path`: `txp.db` next to that config.
- WAL + pragmas are enabled in `internal/storage/db.go`.

## Database schema (summary)
- `tracks`: `path` (PK), `title`, `artist`, `album`, `genre`, `year`, `track_no`, `duration`, `mtime`, `checksum`, `key`, `bpm`, `favorite`.
- `queue`: `scope`, `position`, `path` (PK: `scope`, `position`).
- `track_stats`: `path` (PK), `play_count`, `skip_count`, `last_played`, `last_position`.
- `stats_cache`: `key` (PK), `value_json`, `updated_at`.
- `track_errors`: `path` (PK), `last_error`, `updated_at`.

## Audio + metadata
- Metadata via `github.com/dhowden/tag`.
- Playback via `beep` + `oto` (`internal/player/local.go`).
- Supported playback formats: `.mp3 .ogg .flac .wav .aif .aiff .m4a`.
- Analysis sampling supports `.mp3 .ogg .flac .wav`, plus `.aif/.aiff` and `.m4a`.

## M4A decoding (FFmpeg)
- `.m4a` decoding requires FFmpeg and build tag `ffmpeg`.
- Without the tag, m4a decode returns an error.

## Notes for agents
- Workspace may not be a git repo; avoid destructive git operations.
- Escape user-provided strings before rendering in tview (use `tview.Escape`).
