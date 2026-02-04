package library

import (
	"errors"
	"math"

	"txp/internal/m4a"
)

func readM4ASamples(path string, settings AnalysisSettings) ([]float64, int, error) {
	if tooLarge, err := isTooLargeForDecode(path); err == nil && tooLarge {
		return nil, 0, ErrDecodeTooLarge
	}
	pcm, err := m4a.Decode(path)
	if err != nil {
		return nil, 0, err
	}
	if pcm.Rate <= 0 {
		return nil, 0, errors.New("invalid sample rate")
	}
	settings = normalizeAnalysis(settings)
	mono := interleavedToMono(pcm.Samples, pcm.Channels)
	if settings.SampleRate != pcm.Rate {
		mono = resampleLinear(mono, pcm.Rate, settings.SampleRate)
	}
	return mono, settings.SampleRate, nil
}

func interleavedToMono(samples []float64, channels int) []float64 {
	if channels <= 1 {
		return samples
	}
	frames := len(samples) / channels
	mono := make([]float64, 0, frames)
	for i := 0; i < frames; i++ {
		offset := i * channels
		sum := 0.0
		for c := 0; c < channels; c++ {
			sum += samples[offset+c]
		}
		mono = append(mono, sum/float64(channels))
	}
	return mono
}

func resampleLinear(samples []float64, srcRate int, dstRate int) []float64 {
	if srcRate <= 0 || dstRate <= 0 || len(samples) == 0 {
		return samples
	}
	if srcRate == dstRate {
		return samples
	}
	ratio := float64(dstRate) / float64(srcRate)
	newLen := int(math.Round(float64(len(samples)) * ratio))
	if newLen <= 1 {
		return samples
	}
	resampled := make([]float64, newLen)
	for i := 0; i < newLen; i++ {
		pos := float64(i) / ratio
		idx := int(pos)
		frac := pos - float64(idx)
		if idx+1 >= len(samples) {
			resampled[i] = samples[len(samples)-1]
			continue
		}
		resampled[i] = samples[idx]*(1-frac) + samples[idx+1]*frac
	}
	return resampled
}
