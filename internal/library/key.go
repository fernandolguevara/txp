package library

import (
	"math"

	"github.com/madelynnblue/go-dsp/fft"
	"github.com/madelynnblue/go-dsp/window"
)

func DetectKey(path string, settings AnalysisSettings) string {
	samples, rate, err := readAnalysisSamples(path, settings)
	if err != nil || len(samples) < 4096 || rate <= 0 {
		return ""
	}
	chroma := computeChroma(samples, rate)
	if len(chroma) != 12 {
		return ""
	}
	return detectKeyFromChroma(chroma)
}

func computeChroma(samples []float64, rate int) []float64 {
	frameSize := 4096
	hop := 2048
	if len(samples) < frameSize {
		return nil
	}
	win := window.Hann(frameSize)
	chroma := make([]float64, 12)
	frames := 0
	for start := 0; start+frameSize <= len(samples); start += hop {
		frame := make([]float64, frameSize)
		for i := 0; i < frameSize; i++ {
			frame[i] = samples[start+i] * win[i]
		}
		spectrum := fft.FFTReal(frame)
		for bin := 1; bin < frameSize/2; bin++ {
			freq := float64(bin) * float64(rate) / float64(frameSize)
			if freq < 40 || freq > 5000 {
				continue
			}
			re := real(spectrum[bin])
			im := imag(spectrum[bin])
			mag := math.Sqrt(re*re + im*im)
			midi := 69 + 12*math.Log2(freq/440.0)
			pc := int(math.Round(midi)) % 12
			if pc < 0 {
				pc += 12
			}
			chroma[pc] += mag
		}
		frames++
	}
	if frames == 0 {
		return nil
	}
	sum := 0.0
	for i := 0; i < 12; i++ {
		sum += chroma[i]
	}
	if sum > 0 {
		for i := 0; i < 12; i++ {
			chroma[i] /= sum
		}
	}
	return chroma
}

func detectKeyFromChroma(chroma []float64) string {
	if len(chroma) != 12 {
		return ""
	}
	major := []float64{6.35, 2.23, 3.48, 2.33, 4.38, 4.09, 2.52, 5.19, 2.39, 3.66, 2.29, 2.88}
	minor := []float64{6.33, 2.68, 3.52, 5.38, 2.60, 3.53, 2.54, 4.75, 3.98, 2.69, 3.34, 3.17}
	names := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

	bestScore := 0.0
	bestKey := ""
	for shift := 0; shift < 12; shift++ {
		score := 0.0
		for i := 0; i < 12; i++ {
			score += chroma[(i+shift)%12] * major[i]
		}
		if score > bestScore {
			bestScore = score
			bestKey = names[shift]
		}
	}
	for shift := 0; shift < 12; shift++ {
		score := 0.0
		for i := 0; i < 12; i++ {
			score += chroma[(i+shift)%12] * minor[i]
		}
		if score > bestScore {
			bestScore = score
			bestKey = names[shift] + "m"
		}
	}
	return bestKey
}
