package library

import (
	"errors"
	"math"
	"os"

	"github.com/go-audio/aiff"
)

func readAIFFSamples(path string, settings AnalysisSettings) ([]float64, int, error) {
	if tooLarge, err := isTooLargeForDecode(path); err == nil && tooLarge {
		return nil, 0, ErrDecodeTooLarge
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		_ = file.Close()
	}()

	decoder := aiff.NewDecoder(file)
	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		return nil, 0, err
	}
	if buf == nil || buf.Format == nil {
		return nil, 0, errors.New("invalid aiff buffer")
	}
	mono := interleavedToMonoInt(buf.Data, buf.Format.NumChannels, buf.SourceBitDepth)
	settings = normalizeAnalysis(settings)
	if settings.SampleRate != buf.Format.SampleRate {
		mono = resampleLinear(mono, buf.Format.SampleRate, settings.SampleRate)
	}
	return mono, settings.SampleRate, nil
}

func interleavedToMonoInt(samples []int, channels int, bitDepth int) []float64 {
	if len(samples) == 0 {
		return nil
	}
	if channels <= 0 {
		channels = 1
	}
	frames := len(samples) / channels
	mono := make([]float64, 0, frames)
	max := maxIntForBitDepth(bitDepth)
	if max <= 0 {
		max = math.MaxInt16
	}
	denom := float64(max)
	for i := 0; i < frames; i++ {
		offset := i * channels
		sum := 0.0
		for c := 0; c < channels; c++ {
			sum += float64(samples[offset+c]) / denom
		}
		mono = append(mono, sum/float64(channels))
	}
	return mono
}

func maxIntForBitDepth(bitDepth int) int {
	if bitDepth <= 0 {
		return 0
	}
	return (1 << (bitDepth - 1)) - 1
}
