package library

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/vorbis"
	"github.com/faiface/beep/wav"

	"txp/internal/m4a"
)

func DetectDuration(path string) float64 {
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer func() {
		_ = file.Close()
	}()

	ext := strings.ToLower(filepath.Ext(path))
	var (
		streamer beep.StreamSeekCloser
		format   beep.Format
	)
	switch ext {
	case ".mp3":
		streamer, format, err = mp3.Decode(file)
	case ".ogg":
		streamer, format, err = vorbis.Decode(file)
	case ".flac":
		streamer, format, err = flac.Decode(file)
	case ".wav":
		streamer, format, err = wav.Decode(file)
	case ".aif", ".aiff":
		if tooLarge, err := isTooLargeForDecode(path); err == nil && tooLarge {
			return 0
		}
		_ = file.Close()
		samples, rate, err := readAIFFSamples(path, AnalysisSettings{SampleRate: 44100})
		if err != nil || rate <= 0 || len(samples) == 0 {
			return 0
		}
		return float64(len(samples)) / float64(rate)
	case ".m4a":
		if tooLarge, err := isTooLargeForDecode(path); err == nil && tooLarge {
			return 0
		}
		_ = file.Close()
		pcm, err := m4a.Decode(path)
		if err != nil || pcm.Rate <= 0 || len(pcm.Samples) == 0 {
			return 0
		}
		channels := pcm.Channels
		if channels <= 0 {
			channels = 2
		}
		frames := len(pcm.Samples) / channels
		return float64(frames) / float64(pcm.Rate)
	default:
		return 0
	}
	if err != nil {
		return 0
	}
	defer func() {
		_ = streamer.Close()
	}()

	if format.SampleRate == 0 {
		return 0
	}
	length := streamer.Len()
	if length <= 0 {
		return 0
	}
	return float64(length) / float64(format.SampleRate)
}
