package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"txp/internal/model"
)

const (
	playbackSourceQueue = "queue"
	playbackSourceOther = "other"
)

func defaultEqualizerFrames() []string {
	return []string{
		"▁▂▃▄▃▂",
		"▂▄▅▆▅▄",
		"▃▅▆▇▆▅",
		"▄▆▇▆▅▄",
		"▃▅▆▇▆▅",
		"▂▄▅▆▅▄",
	}
}

func (a *App) equalizerFrame() string {
	a.eqMu.Lock()
	defer a.eqMu.Unlock()
	return a.eqFrame
}

func (a *App) advanceEqualizer(active bool) {
	a.eqMu.Lock()
	defer a.eqMu.Unlock()
	if len(a.eqFrames) == 0 {
		return
	}
	if !active {
		if a.eqFrame == "" {
			a.eqFrame = a.eqFrames[0]
		}
		return
	}
	a.eqIndex = (a.eqIndex + 1) % len(a.eqFrames)
	a.eqFrame = a.eqFrames[a.eqIndex]
}

func (a *App) setEqualizerActive() {
	a.eqMu.Lock()
	defer a.eqMu.Unlock()
	if len(a.eqFrames) == 0 {
		return
	}
	if a.eqIndex < 0 || a.eqIndex >= len(a.eqFrames) {
		a.eqIndex = 0
	}
	a.eqFrame = a.eqFrames[a.eqIndex]
}

func (a *App) updateNowPlaying() {
	text, timeline, vol := a.buildNowPlayingText()
	a.setNowPlayingText(text, timeline, vol)
}

func (a *App) startNowPlayingTicker() {
	if a.Player == nil {
		return
	}
	ticker := time.NewTicker(400 * time.Millisecond)
	go func() {
		for range ticker.C {
			active := a.NowPlayingTrack != nil && !a.Paused
			a.advanceEqualizer(active)
			text, timeline, vol := a.buildNowPlayingText()
			if !a.shouldUpdateNowPlaying(text, timeline, vol) {
				continue
			}
			a.App.QueueUpdateDraw(func() {
				a.setNowPlayingText(text, timeline, vol)
			})
		}
	}()
}

func (a *App) buildNowPlayingText() (string, string, string) {
	volText := a.volumeDisplay()
	timelineWidth := a.timelineAvailableWidth()
	if a.NowPlayingTrack == nil {
		name := ""
		position := 0.0
		duration := 0.0
		path := ""
		if a.Player != nil {
			title, _, _ := a.Player.GetStringProperty("media-title")
			artist, _, _ := a.Player.GetStringProperty("metadata/by-key/Artist")
			path, _, _ = a.Player.GetStringProperty("path")
			name = formatTrackName(strings.TrimSpace(title), strings.TrimSpace(artist), path)
			if pos, ok, _ := a.Player.GetFloatProperty("time-pos"); ok {
				position = pos
			}
			if dur, ok, _ := a.Player.GetFloatProperty("duration"); ok && dur > 0 {
				duration = dur
			}
		}
		if strings.TrimSpace(name) == "" {
			idle := "Idle"
			if eq := a.equalizerFrame(); eq != "" {
				idle = fmt.Sprintf("%s  %s", eq, idle)
			}
			return idle, formatTimelineLineAdaptive(0, 0, timelineWidth), volText
		}
		formatLabel := formatTrackFormat(path)
		if formatLabel != "" {
			name = fmt.Sprintf("%s [%s]", name, formatLabel)
		}
		if eq := a.equalizerFrame(); eq != "" {
			name = fmt.Sprintf("%s  %s", eq, name)
		}
		return escapeUserText(name), formatTimelineLineAdaptive(position, duration, timelineWidth), volText
	}

	name := formatTrackName(a.NowPlayingTrack.Title, a.NowPlayingTrack.Artist, a.NowPlayingTrack.Path)
	formatLabel := formatTrackFormat(a.NowPlayingTrack.Path)
	position := 0.0
	duration := a.NowPlayingTrack.Duration
	if a.Player != nil {
		if title, ok, _ := a.Player.GetStringProperty("media-title"); ok && strings.TrimSpace(title) != "" {
			artist := ""
			if art, ok, _ := a.Player.GetStringProperty("metadata/by-key/Artist"); ok {
				artist = strings.TrimSpace(art)
			}
			if artist != "" {
				name = fmt.Sprintf("%s - %s", artist, title)
			} else {
				name = title
			}
		}
		if pos, ok, _ := a.Player.GetFloatProperty("time-pos"); ok {
			position = pos
		}
		if dur, ok, _ := a.Player.GetFloatProperty("duration"); ok && dur > 0 {
			duration = dur
		}
	}

	if formatLabel != "" {
		name = fmt.Sprintf("%s [%s]", name, formatLabel)
	}
	if eq := a.equalizerFrame(); eq != "" {
		name = fmt.Sprintf("%s  %s", eq, name)
	}
	return escapeUserText(name), formatTimelineLineAdaptive(position, duration, timelineWidth), volText
}

func (a *App) volumeDisplay() string {
	a.seekMu.Lock()
	defer a.seekMu.Unlock()
	if a.seekText != "" {
		if time.Now().Before(a.seekUntil) {
			return a.seekText
		}
		a.seekText = ""
	}
	return formatVolumeBar(a.Volume, 12)
}

func (a *App) timelineAvailableWidth() int {
	if a.Panels == nil || a.Panels.TopBar == nil || a.Panels.TopBar.Timeline == nil {
		return 0
	}
	_, _, width, _ := a.Panels.TopBar.Timeline.GetInnerRect()
	if width <= 0 {
		return 0
	}
	return width
}

func (a *App) showSeekIndicator(delta float64) {
	text := formatSeekIndicator(delta)
	a.seekMu.Lock()
	a.seekText = text
	a.seekUntil = time.Now().Add(900 * time.Millisecond)
	a.seekMu.Unlock()
}

func (a *App) requestSeek(delta float64) {
	if a.Player == nil || a.NowPlayingTrack == nil {
		return
	}
	if delta == 0 {
		return
	}

	a.showSeekIndicator(delta)
	a.scheduleSeek(delta)
}

func (a *App) scheduleSeek(delta float64) {
	a.seekMu.Lock()
	a.seekPendingDelta += delta
	a.seekToken++
	token := a.seekToken
	if a.seekTimer != nil {
		a.seekTimer.Stop()
	}
	a.seekTimer = time.AfterFunc(180*time.Millisecond, func() {
		a.flushSeek(token)
	})
	a.seekMu.Unlock()
}

func (a *App) flushSeek(token uint64) {
	a.seekMu.Lock()
	if token != a.seekToken {
		a.seekMu.Unlock()
		return
	}
	if a.seekInFlight {
		a.seekMu.Unlock()
		return
	}
	delta := a.seekPendingDelta
	a.seekPendingDelta = 0
	a.seekInFlight = true
	a.seekMu.Unlock()

	if a.Player != nil && a.NowPlayingTrack != nil && delta != 0 {
		_ = a.Player.Seek(delta)
	}
	text, timeline, vol := a.buildNowPlayingText()
	a.App.QueueUpdateDraw(func() {
		a.setNowPlayingText(text, timeline, vol)
	})

	a.seekMu.Lock()
	a.seekInFlight = false
	a.seekMu.Unlock()
}

func (a *App) shouldUpdateNowPlaying(text string, timeline string, volume string) bool {
	a.nowPlayingMu.Lock()
	defer a.nowPlayingMu.Unlock()
	if text == a.nowPlayingText && timeline == a.nowPlayingTimeline && volume == a.nowPlayingVolume {
		return false
	}
	return true
}

func (a *App) setNowPlayingText(text string, timeline string, volume string) {
	a.nowPlayingMu.Lock()
	if text == a.nowPlayingText && timeline == a.nowPlayingTimeline && volume == a.nowPlayingVolume {
		a.nowPlayingMu.Unlock()
		return
	}
	a.nowPlayingText = text
	a.nowPlayingTimeline = timeline
	a.nowPlayingVolume = volume
	a.nowPlayingMu.Unlock()

	a.Panels.TopBar.Song.SetText(text)
	if a.Panels.TopBar.Timeline != nil {
		a.Panels.TopBar.Timeline.SetText(timeline)
	}
	if a.Panels.TopBar.Volume != nil {
		a.Panels.TopBar.Volume.SetText(volume)
	}
}

func (a *App) playTrack(track model.Track) {
	if a.Player == nil {
		a.setStatusMessage("Player not available")
		return
	}
	trackName := escapeUserText(formatTrackName(track.Title, track.Artist, track.Path))
	a.setStatusMessage(fmt.Sprintf("Loading track: %s", trackName))
	go func() {
		err := a.Player.Load(track.Path)
		a.App.QueueUpdateDraw(func() {
			if err != nil {
				a.setStatusMessage(fmt.Sprintf("Failed to play track: %v", err))
				return
			}
			a.NowPlayingTrack = &track
			a.Paused = false
			a.setEqualizerActive()
			a.syncVolumeFromPlayer()
			a.updateNowPlaying()
			a.setStatusMessage(fmt.Sprintf("Playing: %s", trackName))
		})
	}()
}

func (a *App) playTrackFromSource(track model.Track, source string) {
	a.playbackSource = source
	a.playTrack(track)
}

func (a *App) handleTrackEnd() {
	a.App.QueueUpdateDraw(func() {
		if a.playbackSource == playbackSourceQueue && a.QueueIndex >= 0 && a.QueueIndex+1 < len(a.QueueItems) {
			a.playQueueIndex(a.QueueIndex + 1)
			return
		}
		a.NowPlayingTrack = nil
		a.Paused = false
		a.updateNowPlaying()
		a.setStatusMessage("Playback finished")
	})
}

func (a *App) togglePause() {
	if a.Player == nil {
		a.setStatusMessage("Player not available")
		return
	}
	if a.NowPlayingTrack == nil {
		a.setStatusMessage("No track playing")
		return
	}
	_ = a.Player.TogglePause()
	a.Paused = !a.Paused
	if a.Paused {
		a.setStatusMessage("Paused")
		return
	}
	a.setStatusMessage("Resumed")
}

func (a *App) adjustVolume(delta int) {
	if a.Player != nil {
		_ = a.Player.AddVolume(delta)
	}
	a.Volume = clampPercent(a.Volume+delta, 0, 100)
	a.Config.Volume = &a.Volume
	a.saveConfig()
	a.updateNowPlaying()
	a.setStatusMessage(fmt.Sprintf("Volume: %d%%", a.Volume))
}

func (a *App) syncVolumeFromPlayer() {
	if a.Player == nil {
		return
	}
	value := a.Player.GetVolume()
	a.Volume = clampPercent(value, 0, 100)
}

func (a *App) bindTopBarMouse() {
	if a.Panels == nil || a.Panels.TopBar == nil {
		return
	}
	if a.Panels.TopBar.Timeline != nil {
		a.Panels.TopBar.Timeline.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
			if action != tview.MouseLeftClick {
				return action, event
			}
			if a.Player == nil || a.NowPlayingTrack == nil {
				return action, event
			}
			x, y := event.Position()
			if !a.Panels.TopBar.Timeline.InRect(x, y) {
				return action, event
			}
			rectX, rectY, _, _ := a.Panels.TopBar.Timeline.GetInnerRect()
			localX := x - rectX
			localY := y - rectY
			if localY != 0 {
				return action, event
			}
			line := firstLine(a.Panels.TopBar.Timeline.GetText(false))
			barStart, barLen, ok := barRangeFromLine(line)
			if !ok || barLen <= 1 {
				return action, event
			}
			if localX < barStart || localX >= barStart+barLen {
				return action, event
			}
			duration, ok, _ := a.Player.GetFloatProperty("duration")
			if !ok || duration <= 0 {
				return action, event
			}
			pos, _, _ := a.Player.GetFloatProperty("time-pos")
			fraction := float64(localX-barStart) / float64(barLen-1)
			if fraction < 0 {
				fraction = 0
			}
			if fraction > 1 {
				fraction = 1
			}
			target := duration * fraction
			delta := target - pos
			a.requestSeek(delta)
			return action, event
		})
	}
	if a.Panels.TopBar.Volume != nil {
		a.Panels.TopBar.Volume.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
			if action != tview.MouseLeftClick {
				return action, event
			}
			if a.Player == nil {
				return action, event
			}
			x, y := event.Position()
			if !a.Panels.TopBar.Volume.InRect(x, y) {
				return action, event
			}
			rectX, rectY, _, _ := a.Panels.TopBar.Volume.GetInnerRect()
			localX := x - rectX
			localY := y - rectY
			if localY != 0 {
				return action, event
			}
			line := firstLine(a.Panels.TopBar.Volume.GetText(false))
			barStart, barLen, ok := barRangeFromLine(line)
			if !ok || barLen <= 1 {
				return action, event
			}
			if localX < barStart || localX >= barStart+barLen {
				return action, event
			}
			fraction := float64(localX-barStart) / float64(barLen-1)
			if fraction < 0 {
				fraction = 0
			}
			if fraction > 1 {
				fraction = 1
			}
			value := int(math.Round(fraction * 100))
			_ = a.Player.SetVolume(value)
			a.syncVolumeFromPlayer()
			a.Config.Volume = &a.Volume
			a.saveConfig()
			text, timeline, vol := a.buildNowPlayingText()
			a.App.QueueUpdateDraw(func() {
				a.setNowPlayingText(text, timeline, vol)
			})
			return action, event
		})
	}
}

func (a *App) isSeekFocus() bool {
	focus := a.App.GetFocus()
	return focus == a.Panels.TopBar.Timeline
}

func (a *App) testTone() {
	if a.Player == nil {
		a.setStatusMessage("Player not available")
		return
	}
	a.setStatusMessage("Playing test tone (2s)")
	go func() {
		err := a.Player.TestTone(2)
		a.App.QueueUpdateDraw(func() {
			if err != nil {
				a.setStatusMessage(fmt.Sprintf("Test tone failed: %v", err))
				return
			}
			a.setStatusMessage("Test tone finished")
		})
	}()
}
