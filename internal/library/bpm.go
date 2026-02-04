package library

import (
	"math"

	"github.com/madelynnblue/go-dsp/fft"
	"github.com/madelynnblue/go-dsp/window"
)

func DetectBPM(path string, settings AnalysisSettings) float64 {
	samples, rate, err := readAnalysisSamples(path, settings)
	if err != nil || len(samples) < 2048 || rate <= 0 {
		return 0
	}
	settings = normalizeAnalysis(settings)
	return estimateBPM(samples, rate, settings.BPMMin, settings.BPMMax)
}

func estimateBPM(samples []float64, rate int, bpmMin int, bpmMax int) float64 {
	if bpmMin <= 0 || bpmMax <= bpmMin {
		return 0
	}
	frameSize := 1024
	hop := 512
	flux := spectralFlux(samples, frameSize, hop)
	if len(flux) < 8 {
		return 0
	}
	minLag := int(math.Round(float64(rate) * 60 / float64(bpmMax) / float64(hop)))
	maxLag := int(math.Round(float64(rate) * 60 / float64(bpmMin) / float64(hop)))
	if minLag < 1 {
		minLag = 1
	}
	if maxLag >= len(flux) {
		maxLag = len(flux) - 1
	}
	if minLag >= maxLag {
		return 0
	}

	bestLag := 0
	bestScore := 0.0
	for lag := minLag; lag <= maxLag; lag++ {
		score := 0.0
		for i := lag; i < len(flux); i++ {
			score += flux[i] * flux[i-lag]
		}
		if score > bestScore {
			bestScore = score
			bestLag = lag
		}
	}
	if bestLag == 0 || bestScore <= 0 {
		return 0
	}
	return 60 * float64(rate) / (float64(hop) * float64(bestLag))
}

func spectralFlux(samples []float64, frameSize int, hop int) []float64 {
	if frameSize <= 0 || hop <= 0 || len(samples) < frameSize {
		return nil
	}
	win := window.Hann(frameSize)
	prev := make([]float64, frameSize)
	flux := []float64{}
	for start := 0; start+frameSize <= len(samples); start += hop {
		frame := make([]float64, frameSize)
		for i := 0; i < frameSize; i++ {
			frame[i] = samples[start+i] * win[i]
		}
		spectrum := fft.FFTReal(frame)
		mag := make([]float64, frameSize)
		for i := 0; i < frameSize; i++ {
			re := real(spectrum[i])
			im := imag(spectrum[i])
			mag[i] = math.Sqrt(re*re + im*im)
		}
		sum := 0.0
		for i := 0; i < frameSize; i++ {
			diff := mag[i] - prev[i]
			if diff > 0 {
				sum += diff
			}
			prev[i] = mag[i]
		}
		flux = append(flux, sum)
	}
	return flux
}
