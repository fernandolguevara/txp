package library

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/vorbis"
	"github.com/faiface/beep/wav"
)

type AnalysisSettings struct {
	WindowSeconds int
	SampleRate    int
	BPMMin        int
	BPMMax        int
}

func normalizeAnalysis(settings AnalysisSettings) AnalysisSettings {
	if settings.SampleRate <= 0 {
		settings.SampleRate = 22050
	}
	if settings.BPMMin <= 0 {
		settings.BPMMin = 70
	}
	if settings.BPMMax <= 0 {
		settings.BPMMax = 180
	}
	if settings.BPMMax <= settings.BPMMin {
		settings.BPMMax = settings.BPMMin + 10
	}
	return settings
}

func readAnalysisSamples(path string, settings AnalysisSettings) ([]float64, int, error) {
	settings = normalizeAnalysis(settings)
	if strings.EqualFold(filepath.Ext(path), ".m4a") {
		return readM4ASamples(path, settings)
	}
	if strings.EqualFold(filepath.Ext(path), ".aif") || strings.EqualFold(filepath.Ext(path), ".aiff") {
		return readAIFFSamples(path, settings)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
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
	default:
		return nil, 0, errors.New("unsupported audio format")
	}
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		_ = streamer.Close()
	}()

	stream := beep.Streamer(streamer)
	if format.SampleRate > 0 && int(format.SampleRate) != settings.SampleRate {
		stream = beep.Resample(4, format.SampleRate, beep.SampleRate(settings.SampleRate), streamer)
	}

	maxSamples := 0
	if settings.WindowSeconds > 0 {
		maxSamples = settings.SampleRate * settings.WindowSeconds
	}

	buf := make([][2]float64, 1024)
	samples := make([]float64, 0, 8192)
	for {
		n, ok := stream.Stream(buf)
		if n > 0 {
			for i := 0; i < n; i++ {
				mono := (buf[i][0] + buf[i][1]) * 0.5
				samples = append(samples, mono)
				if maxSamples > 0 && len(samples) >= maxSamples {
					return samples[:maxSamples], settings.SampleRate, nil
				}
			}
		}
		if !ok {
			break
		}
	}
	if len(samples) == 0 {
		return nil, 0, errors.New("no samples")
	}
	return samples, settings.SampleRate, nil
}
