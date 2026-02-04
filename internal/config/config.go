package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PanelSizes struct {
	LibraryWidth  int   `json:"libraryWidth"`
	QueueWidth    int   `json:"queueWidth"`
	ShowTrackInfo *bool `json:"showTrackInfo"`
}

type AnalysisSettings struct {
	WindowSeconds *int `json:"analysis_seconds"`
	SampleRate    *int `json:"analysis_sample_rate"`
	BPMMin        *int `json:"analysis_bpm_min"`
	BPMMax        *int `json:"analysis_bpm_max"`
}

type Config struct {
	Libraries         []string            `json:"libraries"`
	SelectedLibraries []string            `json:"selectedLibraries"`
	LibraryNavCursor  *LibraryNavCursor   `json:"library_nav_cursor,omitempty"`
	MainView          string              `json:"main_view"`
	Theme             string              `json:"theme"`
	Panel             PanelSizes          `json:"panel"`
	Shortcuts         map[string][]string `json:"shortcuts"`
	Analysis          AnalysisSettings    `json:"analysis"`
	Volume            *int                `json:"volume"`
	DoubleClickPlay   *bool               `json:"enable_double_click_playback"`
}

type LibraryNavCursor struct {
	Kind  string `json:"kind,omitempty"`
	Value string `json:"value,omitempty"`
	Path  string `json:"path,omitempty"`
}

type ResolvedConfig struct {
	Config Config
	Path   string
	Scope  string
	Exists bool
}

const (
	ScopeCurrent = "current"
	ScopeGlobal  = "global"
)

func DefaultConfig() Config {
	windowSeconds := 60
	sampleRate := 22050
	bpmMin := 70
	bpmMax := 180
	showTrackInfo := true
	volume := 60
	doubleClickPlay := false
	return Config{
		Libraries:         []string{},
		SelectedLibraries: []string{},
		MainView:          "track_view",
		Theme:             "Mono",
		Panel: PanelSizes{
			LibraryWidth:  60,
			QueueWidth:    30,
			ShowTrackInfo: &showTrackInfo,
		},
		Shortcuts: map[string][]string{
			"commandPalette": {"Ctrl+K", "Cmd+K"},
			"quit":           {"Ctrl+C"},
			"resizeMode":     {"Ctrl+R"},
			"openFilters":    {"/"},
			"togglePlay":     {"Space"},
			"nextTrack":      {"n"},
			"prevTrack":      {"p"},
		},
		Analysis: AnalysisSettings{
			WindowSeconds: &windowSeconds,
			SampleRate:    &sampleRate,
			BPMMin:        &bpmMin,
			BPMMax:        &bpmMax,
		},
		Volume:          &volume,
		DoubleClickPlay: &doubleClickPlay,
	}
}

func Load() (ResolvedConfig, error) {
	resolved, err := ResolveConfigPath()
	if err != nil {
		return ResolvedConfig{}, err
	}

	if !resolved.Exists {
		return ResolvedConfig{Config: DefaultConfig(), Path: resolved.Path, Scope: resolved.Scope, Exists: false}, nil
	}
	return loadFromPath(resolved.Path, resolved.Scope, true)
}

func LoadWithPath(path string) (ResolvedConfig, error) {
	return loadFromPath(path, ScopeCurrent, false)
}

func loadFromPath(path string, scope string, exists bool) (ResolvedConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			_ = Save(path, cfg)
			return ResolvedConfig{Config: cfg, Path: path, Scope: scope, Exists: false}, nil
		}
		return ResolvedConfig{Config: DefaultConfig(), Path: path, Scope: scope, Exists: exists}, fmt.Errorf("read config %s: %w", path, err)
	}

	var partial Config
	if err := json.Unmarshal(data, &partial); err != nil {
		return ResolvedConfig{Config: DefaultConfig(), Path: path, Scope: scope, Exists: true}, fmt.Errorf("parse config %s: %w", path, err)
	}

	merged := mergeConfig(DefaultConfig(), partial)
	normalized, libsChanged := NormalizeLibraryList(merged.Libraries)
	selected, selectedChanged := NormalizeSelectedLibraries(normalized, merged.SelectedLibraries)
	if libsChanged || selectedChanged {
		merged.Libraries = normalized
		merged.SelectedLibraries = selected
		_ = Save(path, merged)
	}
	return ResolvedConfig{Config: merged, Path: path, Scope: scope, Exists: true}, nil
}

func Save(path string, config Config) error {
	if path == "" {
		return errors.New("config path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

func mergeConfig(base Config, override Config) Config {
	if len(override.Libraries) > 0 {
		base.Libraries = override.Libraries
	}
	if len(override.SelectedLibraries) > 0 {
		base.SelectedLibraries = override.SelectedLibraries
	}
	if override.LibraryNavCursor != nil {
		base.LibraryNavCursor = override.LibraryNavCursor
	}
	if override.MainView != "" {
		base.MainView = override.MainView
	}
	if override.Theme != "" {
		base.Theme = override.Theme
	}
	if override.Panel.LibraryWidth != 0 {
		base.Panel.LibraryWidth = override.Panel.LibraryWidth
	}
	if override.Panel.QueueWidth != 0 {
		base.Panel.QueueWidth = override.Panel.QueueWidth
	}
	if override.Panel.ShowTrackInfo != nil {
		base.Panel.ShowTrackInfo = override.Panel.ShowTrackInfo
	}
	if len(override.Shortcuts) > 0 {
		if base.Shortcuts == nil {
			base.Shortcuts = map[string][]string{}
		}
		for key, values := range override.Shortcuts {
			if len(values) == 0 {
				continue
			}
			if key == "quit" {
				filtered := make([]string, 0, len(values))
				for _, value := range values {
					if !strings.EqualFold(strings.TrimSpace(value), "q") {
						filtered = append(filtered, value)
					}
				}
				values = filtered
				if len(values) == 0 {
					continue
				}
			}
			base.Shortcuts[key] = values
		}
	}
	if override.Analysis.WindowSeconds != nil {
		base.Analysis.WindowSeconds = override.Analysis.WindowSeconds
	}
	if override.Analysis.SampleRate != nil {
		base.Analysis.SampleRate = override.Analysis.SampleRate
	}
	if override.Analysis.BPMMin != nil {
		base.Analysis.BPMMin = override.Analysis.BPMMin
	}
	if override.Analysis.BPMMax != nil {
		base.Analysis.BPMMax = override.Analysis.BPMMax
	}
	if override.Volume != nil {
		base.Volume = override.Volume
	}
	if override.DoubleClickPlay != nil {
		base.DoubleClickPlay = override.DoubleClickPlay
	}
	return base
}

type AnalysisResolved struct {
	WindowSeconds int
	SampleRate    int
	BPMMin        int
	BPMMax        int
}

func ResolveAnalysis(cfg Config) AnalysisResolved {
	defaults := DefaultConfig().Analysis
	window := 0
	if defaults.WindowSeconds != nil {
		window = *defaults.WindowSeconds
	}
	sample := 0
	if defaults.SampleRate != nil {
		sample = *defaults.SampleRate
	}
	bpmMin := 0
	if defaults.BPMMin != nil {
		bpmMin = *defaults.BPMMin
	}
	bpmMax := 0
	if defaults.BPMMax != nil {
		bpmMax = *defaults.BPMMax
	}

	if cfg.Analysis.WindowSeconds != nil {
		window = *cfg.Analysis.WindowSeconds
	}
	if cfg.Analysis.SampleRate != nil {
		sample = *cfg.Analysis.SampleRate
	}
	if cfg.Analysis.BPMMin != nil {
		bpmMin = *cfg.Analysis.BPMMin
	}
	if cfg.Analysis.BPMMax != nil {
		bpmMax = *cfg.Analysis.BPMMax
	}

	if sample <= 0 {
		sample = 22050
	}
	if bpmMin <= 0 {
		bpmMin = 70
	}
	if bpmMax <= 0 {
		bpmMax = 180
	}
	if bpmMax <= bpmMin {
		bpmMax = bpmMin + 10
	}

	return AnalysisResolved{WindowSeconds: window, SampleRate: sample, BPMMin: bpmMin, BPMMax: bpmMax}
}

func ResolveVolume(cfg Config) int {
	if cfg.Volume != nil {
		return clampVolume(*cfg.Volume)
	}
	defaultVolume := 60
	if DefaultConfig().Volume != nil {
		defaultVolume = *DefaultConfig().Volume
	}
	return clampVolume(defaultVolume)
}

func ResolveDoubleClickPlay(cfg Config) bool {
	if cfg.DoubleClickPlay != nil {
		return *cfg.DoubleClickPlay
	}
	if DefaultConfig().DoubleClickPlay != nil {
		return *DefaultConfig().DoubleClickPlay
	}
	return false
}

func clampVolume(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func NormalizeLibraryPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}

	if strings.HasPrefix(trimmed, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			if trimmed == "~" {
				trimmed = home
			} else if strings.HasPrefix(trimmed, "~/") {
				trimmed = filepath.Join(home, strings.TrimPrefix(trimmed, "~/"))
			} else if strings.HasPrefix(trimmed, "~\\") {
				trimmed = filepath.Join(home, strings.TrimPrefix(trimmed, "~\\"))
			}
		}
	}

	cleaned := filepath.Clean(trimmed)
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return cleaned
	}
	return abs
}

func NormalizeLibraryList(paths []string) ([]string, bool) {
	seen := map[string]bool{}
	normalized := []string{}
	changed := false

	for _, path := range paths {
		value := NormalizeLibraryPath(path)
		if value == "" {
			changed = true
			continue
		}
		if seen[value] {
			changed = true
			continue
		}
		seen[value] = true
		normalized = append(normalized, value)
		if value != path {
			changed = true
		}
	}

	if len(normalized) != len(paths) {
		changed = true
	}

	return normalized, changed
}

func NormalizeSelectedLibraries(libraries []string, selected []string) ([]string, bool) {
	allowed := map[string]bool{}
	for _, lib := range libraries {
		allowed[lib] = true
	}

	seen := map[string]bool{}
	result := []string{}
	changed := false

	for _, sel := range selected {
		value := NormalizeLibraryPath(sel)
		if value == "" || !allowed[value] {
			changed = true
			continue
		}
		if seen[value] {
			changed = true
			continue
		}
		seen[value] = true
		result = append(result, value)
		if value != sel {
			changed = true
		}
	}

	if len(result) != len(selected) {
		changed = true
	}

	return result, changed
}
