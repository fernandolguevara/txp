package player

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dhowden/tag"
	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/vorbis"
	"github.com/faiface/beep/wav"
	"github.com/hajimehoshi/oto/v2"
)

const (
	defaultSampleRate = 44100
	defaultChannels   = 2
)

type Local struct {
	mu            sync.Mutex
	ctx           *oto.Context
	ctxRate       beep.SampleRate
	player        oto.Player
	source        beep.StreamSeekCloser
	streamer      beep.Streamer
	position      *positionTracker
	format        beep.Format
	sourceRate    beep.SampleRate
	volumePercent int
	paused        bool
	requestID     int64
	onEOF         func()
	mediaTitle    string
	mediaArtist   string
	fileName      string
	path          string
	lengthSamples int
}

func Start() (Player, error) {
	return &Local{volumePercent: 60}, nil
}

func (l *Local) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.player != nil {
		_ = l.player.Close()
		l.player = nil
	}
	if l.source != nil {
		_ = l.source.Close()
		l.source = nil
	}
}

func (l *Local) SetOnEOF(fn func()) {
	l.mu.Lock()
	l.onEOF = fn
	l.mu.Unlock()
}

func (l *Local) Load(path string) error {
	if path == "" {
		return errors.New("path is required")
	}
	logf("info", "Load start: %s", path)
	decodeStart := time.Now()

	source, format, err := decodeAudio(path)
	if err != nil {
		logf("error", "Decode failed: %v", err)
		return err
	}
	logf("info", "Decode done in %s (rate=%d)", time.Since(decodeStart).Round(time.Millisecond), format.SampleRate)

	fileName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	title, artist := readTags(path)

	l.mu.Lock()
	defer l.mu.Unlock()

	l.stopPlaybackLocked()

	playRate := format.SampleRate
	if playRate == 0 {
		playRate = beep.SampleRate(defaultSampleRate)
	}
	l.sourceRate = format.SampleRate
	if err := l.ensureContextLocked(int(playRate), defaultChannels); err != nil {
		_ = source.Close()
		return err
	}
	streamer, lengthSamples := l.buildPlaybackStreamLocked(source)
	playRate = l.ctxRate
	if playRate == 0 {
		playRate = beep.SampleRate(defaultSampleRate)
	}

	requestID := atomic.AddInt64(&l.requestID, 1)
	current := requestID

	reader := newStreamReader(streamer, playRate, streamReaderOptions{
		label: "track",
		onEOF: func() {
			if atomic.LoadInt64(&l.requestID) != current {
				return
			}
			l.mu.Lock()
			l.stopPlaybackLocked()
			callback := l.onEOF
			l.mu.Unlock()
			if callback != nil {
				callback()
			}
		},
	})
	if audioDiagEnabled() {
		reader.EnableDiagnostics()
	}

	player := l.ctx.NewPlayer(reader)
	player.SetVolume(volumeFromPercent(l.volumePercent))
	player.Play()

	l.player = player
	l.source = source
	l.streamer = streamer
	playRate = l.ctxRate
	if playRate == 0 {
		playRate = beep.SampleRate(defaultSampleRate)
	}
	l.format = beep.Format{SampleRate: playRate, NumChannels: defaultChannels, Precision: 2}
	l.fileName = fileName
	l.path = path
	l.mediaTitle = title
	l.mediaArtist = artist
	l.lengthSamples = lengthSamples
	l.paused = false

	logf("info", "Playback volume: %d%%", l.volumePercent)
	logf("info", "Oto playback started (rate=%d)", playRate)

	return nil
}

func (l *Local) TestTone(seconds float64) error {
	if seconds <= 0 {
		seconds = 2
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.ensureContextLocked(defaultSampleRate, defaultChannels); err != nil {
		logf("error", "Test tone context init failed: %v", err)
		return err
	}

	streamer := beep.Streamer(newTestTone(beep.SampleRate(defaultSampleRate), seconds, 440))
	reader := newStreamReader(streamer, beep.SampleRate(defaultSampleRate), streamReaderOptions{
		label: "test-tone",
		onEOF: func() { logf("info", "Test tone finished") },
	})
	if audioDiagEnabled() {
		reader.EnableDiagnostics()
	}
	logf("info", "Test tone requested: %.2fs", seconds)

	player := l.ctx.NewPlayer(reader)
	player.SetVolume(volumeFromPercent(l.volumePercent))
	player.Play()

	logf("info", "Test tone play begin (rate=%d)", defaultSampleRate)
	return nil
}

func (l *Local) TogglePause() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.player == nil {
		return nil
	}
	if l.paused {
		l.player.Play()
		l.paused = false
		return nil
	}
	l.player.Pause()
	l.paused = true
	return nil
}

func (l *Local) AddVolume(delta int) error {
	return l.SetVolume(l.volumePercent + delta)
}

func (l *Local) SetVolume(value int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if value < 0 {
		value = 0
	}
	if value > 100 {
		value = 100
	}
	l.volumePercent = value
	if l.player != nil {
		l.player.SetVolume(volumeFromPercent(l.volumePercent))
	}
	logf("info", "SetVolume: %d%%", l.volumePercent)
	return nil
}

func (l *Local) GetVolume() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.volumePercent
}

func (l *Local) Seek(seconds float64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.source == nil {
		return nil
	}
	if l.format.SampleRate == 0 {
		return nil
	}
	if l.path == "" {
		return errors.New("missing track path")
	}

	currentSamples := int(l.position.Position())
	seekDelta := int(math.Round(seconds * float64(l.format.SampleRate)))
	newPos := currentSamples + seekDelta
	if newPos < 0 {
		newPos = 0
	}
	if l.lengthSamples > 0 && newPos > l.lengthSamples {
		newPos = l.lengthSamples
	}

	if l.sourceRate == 0 {
		return errors.New("invalid source sample rate")
	}
	isFlac := strings.EqualFold(filepath.Ext(l.path), ".flac")
	if isFlac {
		if l.player != nil {
			_ = l.player.Close()
			l.player = nil
		}
		if l.source != nil {
			_ = l.source.Close()
			l.source = nil
		}
		source, format, err := decodeAudio(l.path)
		if err != nil {
			return err
		}
		l.source = source
		l.sourceRate = format.SampleRate
		if l.sourceRate == 0 {
			_ = l.source.Close()
			l.source = nil
			return errors.New("invalid source sample rate")
		}
	}

	srcPos := int(math.Round(float64(newPos) * float64(l.sourceRate) / float64(l.format.SampleRate)))
	if srcPos < 0 {
		srcPos = 0
	}
	sourceLen := l.source.Len()
	if !isFlac && sourceLen <= 0 {
		if l.source != nil {
			_ = l.source.Close()
		}
		source, format, err := decodeAudio(l.path)
		if err != nil {
			return err
		}
		l.source = source
		l.sourceRate = format.SampleRate
		if l.sourceRate == 0 {
			_ = l.source.Close()
			l.source = nil
			return errors.New("invalid source sample rate")
		}
		sourceLen = l.source.Len()
		srcPos = int(math.Round(float64(newPos) * float64(l.sourceRate) / float64(l.format.SampleRate)))
		if srcPos < 0 {
			srcPos = 0
		}
	}
	if sourceLen > 0 && srcPos >= sourceLen {
		srcPos = sourceLen - 1
	}
	if isFlac && sourceLen > 0 {
		preRoll := int(math.Round(float64(l.sourceRate) * 0.35))
		if preRoll > 0 {
			if srcPos > preRoll {
				srcPos -= preRoll
			} else {
				srcPos = 0
			}
		}
		backoff := int(math.Round(float64(l.sourceRate) * 0.5))
		if backoff > 0 && srcPos > sourceLen-backoff {
			srcPos = sourceLen - backoff
			if srcPos < 0 {
				srcPos = 0
			}
		}
	}
	if l.format.SampleRate > 0 {
		newPos = int(math.Round(float64(srcPos) * float64(l.format.SampleRate) / float64(l.sourceRate)))
		if newPos < 0 {
			newPos = 0
		}
		if l.lengthSamples > 0 && newPos > l.lengthSamples {
			newPos = l.lengthSamples
		}
	}
	if err := l.source.Seek(srcPos); err != nil {
		return err
	}

	playRate := l.format.SampleRate
	streamer, lengthSamples := l.buildPlaybackStreamLocked(l.source)

	reader := newStreamReader(streamer, playRate, streamReaderOptions{
		label: "track",
		onEOF: func() {
			l.mu.Lock()
			l.stopPlaybackLocked()
			callback := l.onEOF
			l.mu.Unlock()
			if callback != nil {
				callback()
			}
		},
	})
	if audioDiagEnabled() {
		reader.EnableDiagnostics()
	}

	if l.player != nil {
		_ = l.player.Close()
	}
	player := l.ctx.NewPlayer(reader)
	player.SetVolume(volumeFromPercent(l.volumePercent))
	if l.paused {
		player.Pause()
	} else {
		player.Play()
	}

	if lengthSamples > 0 && newPos > lengthSamples {
		newPos = lengthSamples
	}
	if l.position != nil {
		l.position.SetPosition(int64(newPos))
	}
	l.player = player
	l.streamer = streamer
	l.lengthSamples = lengthSamples
	return nil
}

func (l *Local) GetStringProperty(name string) (string, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	switch name {
	case "media-title":
		if strings.TrimSpace(l.mediaTitle) != "" {
			return l.mediaTitle, true, nil
		}
		if l.fileName != "" {
			return l.fileName, true, nil
		}
	case "metadata/by-key/Artist":
		if strings.TrimSpace(l.mediaArtist) != "" {
			return l.mediaArtist, true, nil
		}
	}
	return "", false, nil
}

func (l *Local) GetFloatProperty(name string) (float64, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	switch name {
	case "time-pos":
		if l.format.SampleRate == 0 || l.position == nil {
			return 0, false, nil
		}
		pos := float64(l.position.Position()) / float64(l.format.SampleRate)
		return pos, true, nil
	case "duration":
		if l.format.SampleRate == 0 || l.lengthSamples <= 0 {
			return 0, false, nil
		}
		dur := float64(l.lengthSamples) / float64(l.format.SampleRate)
		return dur, true, nil
	}
	return 0, false, nil
}

func (l *Local) ensureContextLocked(sampleRate int, channels int) error {
	if l.ctx != nil {
		return nil
	}
	ctx, ready, err := oto.NewContext(sampleRate, channels, oto.FormatSignedInt16LE)
	if err != nil {
		return err
	}
	<-ready
	l.ctx = ctx
	l.ctxRate = beep.SampleRate(sampleRate)
	logf("info", "Oto context ready (rate=%d, channels=%d)", sampleRate, channels)
	return nil
}

func (l *Local) stopPlaybackLocked() {
	if l.player != nil {
		_ = l.player.Close()
		l.player = nil
	}
	if l.source != nil {
		_ = l.source.Close()
		l.source = nil
	}
	if l.position != nil {
		l.position.SetPosition(0)
	}
}

func (l *Local) buildPlaybackStreamLocked(source beep.StreamSeekCloser) (beep.Streamer, int) {
	lengthSamples := source.Len()
	if lengthSamples < 0 {
		lengthSamples = 0
	}

	streamer := beep.Streamer(source)
	if l.ctxRate > 0 && l.sourceRate > 0 && l.ctxRate != l.sourceRate {
		streamer = beep.Resample(4, l.sourceRate, l.ctxRate, streamer)
		if lengthSamples > 0 {
			lengthSamples = int(math.Round(float64(lengthSamples) * float64(l.ctxRate) / float64(l.sourceRate)))
		}
	}
	position := newPositionTracker(streamer)
	l.position = position
	streamer = beep.Streamer(position)
	return streamer, lengthSamples
}

func decodeAudio(path string) (beep.StreamSeekCloser, beep.Format, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, beep.Format{}, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mp3":
		streamer, format, err := mp3.Decode(file)
		if err != nil {
			_ = file.Close()
			return nil, beep.Format{}, err
		}
		return streamer, format, nil
	case ".ogg":
		streamer, format, err := vorbis.Decode(file)
		if err != nil {
			_ = file.Close()
			return nil, beep.Format{}, err
		}
		return streamer, format, nil
	case ".flac":
		streamer, format, err := flac.Decode(file)
		if err != nil {
			_ = file.Close()
			return nil, beep.Format{}, err
		}
		return streamer, format, nil
	case ".aif", ".aiff":
		_ = file.Close()
		return decodeAIFF(path)
	case ".m4a":
		_ = file.Close()
		return decodeM4A(path)
	case ".wav":
		streamer, format, err := wav.Decode(file)
		if err != nil {
			_ = file.Close()
			return nil, beep.Format{}, err
		}
		return streamer, format, nil
	default:
		_ = file.Close()
		return nil, beep.Format{}, fmt.Errorf("unsupported format: %s", ext)
	}
}

func readTags(path string) (string, string) {
	file, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer func() {
		_ = file.Close()
	}()

	meta, err := tag.ReadFrom(file)
	if err != nil {
		return "", ""
	}
	return strings.TrimSpace(meta.Title()), strings.TrimSpace(meta.Artist())
}

func volumeFromPercent(percent int) float64 {
	if percent <= 0 {
		return 0
	}
	if percent >= 100 {
		return 1
	}
	return float64(percent) / 100
}

func audioDiagEnabled() bool {
	raw := strings.TrimSpace(os.Getenv("TXP_AUDIO_DIAG"))
	if raw == "" {
		return false
	}
	if raw == "0" || strings.EqualFold(raw, "false") {
		return false
	}
	return true
}

type streamReaderOptions struct {
	label string
	onEOF func()
}

type streamReader struct {
	streamer beep.Streamer
	rate     beep.SampleRate
	samples  [][2]float64
	buf      []byte
	logged   int
	count    int
	label    string
	onEOF    func()
	once     sync.Once
	logDiag  bool
}

func newStreamReader(streamer beep.Streamer, rate beep.SampleRate, opts streamReaderOptions) *streamReader {
	return &streamReader{streamer: streamer, rate: rate, label: opts.label, onEOF: opts.onEOF}
}

func (r *streamReader) EnableDiagnostics() {
	r.logDiag = true
}

func (r *streamReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	frames := len(p) / 4
	if frames == 0 {
		return 0, nil
	}
	if cap(r.samples) < frames {
		r.samples = make([][2]float64, frames)
	} else {
		r.samples = r.samples[:frames]
	}

	n, ok := r.streamer.Stream(r.samples)
	if n == 0 && !ok {
		r.once.Do(func() {
			if r.onEOF != nil {
				r.onEOF()
			}
		})
		return 0, io.EOF
	}

	needed := n * 4
	if cap(r.buf) < needed {
		r.buf = make([]byte, needed)
	}
	buf := r.buf[:needed]

	for i := 0; i < n; i++ {
		l := floatToInt16(r.samples[i][0])
		rv := floatToInt16(r.samples[i][1])
		idx := i * 4
		buf[idx] = byte(l)
		buf[idx+1] = byte(l >> 8)
		buf[idx+2] = byte(rv)
		buf[idx+3] = byte(rv >> 8)
	}

	copy(p, buf)

	if r.logDiag && r.rate > 0 {
		r.count += n
		seconds := r.count / int(r.rate)
		if seconds > r.logged {
			r.logged = seconds
			logf("info", "Audio diagnostics (%s): streamed %ds", r.label, r.logged)
		}
	}

	return needed, nil
}

func floatToInt16(value float64) int16 {
	if value > 1 {
		value = 1
	}
	if value < -1 {
		value = -1
	}
	return int16(value * 32767)
}

type testTone struct {
	rate    beep.SampleRate
	freq    float64
	length  int
	current int
}

func newTestTone(rate beep.SampleRate, seconds float64, freq float64) *testTone {
	if seconds <= 0 {
		seconds = 0.25
	}
	if freq <= 0 {
		freq = 440
	}
	length := int(math.Round(seconds * float64(rate)))
	if length < 1 {
		length = 1
	}
	return &testTone{rate: rate, freq: freq, length: length}
}

func (t *testTone) Stream(samples [][2]float64) (int, bool) {
	for i := 0; i < len(samples); i++ {
		if t.current >= t.length {
			return i, false
		}
		phase := 2 * math.Pi * t.freq * float64(t.current) / float64(t.rate)
		value := math.Sin(phase)
		samples[i][0] = value
		samples[i][1] = value
		t.current++
	}
	return len(samples), true
}

func (t *testTone) Err() error {
	return nil
}

type positionTracker struct {
	streamer beep.Streamer
	position atomic.Int64
}

func newPositionTracker(streamer beep.Streamer) *positionTracker {
	return &positionTracker{streamer: streamer}
}

func (p *positionTracker) Stream(samples [][2]float64) (int, bool) {
	n, ok := p.streamer.Stream(samples)
	if n > 0 {
		p.position.Add(int64(n))
	}
	return n, ok
}

func (p *positionTracker) Err() error {
	if errer, ok := p.streamer.(interface{ Err() error }); ok {
		return errer.Err()
	}
	return nil
}

func (p *positionTracker) Position() int64 {
	return p.position.Load()
}

func (p *positionTracker) SetPosition(pos int64) {
	if pos < 0 {
		pos = 0
	}
	p.position.Store(pos)
}
