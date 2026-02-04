# txp

Terminal UI music player and library manager written in Go.

## Overview
- Scan audio libraries into SQLite.
- Analyze duration, key, and BPM.
- Browse tracks, manage a queue, and play locally in a TUI.

## Features
- Library scanning with incremental updates.
- Track filtering, sorting, and favorites.
- Queue management and persistence.
- Built-in log view and log file viewer.
- Tag editing for MP3 and FLAC.

## Install / Build / Run
Build:
```bash
go build ./cmd/txp
```

Run:
```bash
go run ./cmd/txp
```

Run with a custom config path:
```bash
go run ./cmd/txp --config-path /path/to/config.json
```

If you pass a directory, it resolves to `config.json` inside that directory.

## Configuration
Config is JSON (schema in `internal/config/config.go`). The default path is `~/.txp/config.json`.

Key fields:
- `libraries` / `selectedLibraries`
- `library_nav_cursor`
- `main_view`: `track_view` or `library_view`
- `theme`, `panel.libraryWidth`, `panel.queueWidth`, `panel.showTrackInfo`
- `shortcuts` map
- `analysis_*` (window seconds, sample rate, BPM range)
- `volume`, `enable_double_click_playback`

Library paths are normalized to absolute paths and deduped.

## Environment variables
- `TXP_LOG_PATH`: override log file path.
- `TXP_LOG_LEVEL`: `error` (default), `warn`, `info`, `debug`.
- `TXP_AUDIO_DIAG`: enable periodic audio diagnostics logs (any non-empty value except `0`/`false`).

## Supported formats
Playback:
- `.mp3 .ogg .flac .wav .aif .aiff .m4a`

Analysis sampling:
- `.mp3 .ogg .flac .wav .aif .aiff .m4a`

Tag writing:
- MP3 (ID3v2)
- FLAC (Vorbis comments)

## M4A decoding (FFmpeg)
`.m4a` decoding requires FFmpeg and build tag `ffmpeg`.

Example build with FFmpeg decoder:
```bash
go build -tags ffmpeg ./cmd/txp
```

## Keybindings (defaults)
- `Ctrl+K`: command palette
- `Ctrl+C`: quit
- `Ctrl+R`: resize mode
- `/` or `Ctrl+F`: focus filter
- `Space`: play/pause
- `n` / `p`: next/previous track

Keybindings are configurable in the settings view and persisted in config.

## Database
SQLite is created automatically:
- Default: `~/.txp/txp.db`
- With `--config-path`: `txp.db` next to that config

Schema summary:
- `tracks`, `queue`, `track_stats`, `stats_cache`, `track_errors`

## Troubleshooting
- No audio: check device output and verify supported formats.
- `.m4a` errors: build with `-tags ffmpeg` and ensure `ffmpeg` is on `PATH`.
- Large libraries: filtering and log tailing may be slower; let the scan finish first.
