package player

import (
	"errors"
	"math"
	"os"

	"github.com/faiface/beep"
	"github.com/go-audio/aiff"
)

func decodeAIFF(path string) (beep.StreamSeekCloser, beep.Format, error) {
	if tooLarge, err := isTooLargeForDecode(path); err == nil && tooLarge {
		return nil, beep.Format{}, ErrDecodeTooLarge
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, beep.Format{}, err
	}
	defer func() {
		_ = file.Close()
	}()

	decoder := aiff.NewDecoder(file)
	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		return nil, beep.Format{}, err
	}
	if buf == nil || buf.Format == nil {
		return nil, beep.Format{}, errors.New("invalid aiff buffer")
	}
	samples := interleavedToFloat(buf.Data, buf.SourceBitDepth)
	if len(samples) == 0 {
		return nil, beep.Format{}, errors.New("invalid aiff samples")
	}
	channels := buf.Format.NumChannels
	if channels <= 0 {
		channels = 2
	}
	format := beep.Format{SampleRate: beep.SampleRate(buf.Format.SampleRate), NumChannels: channels, Precision: 2}
	streamer := &m4aStreamer{samples: samples, rate: format.SampleRate, channels: channels}
	return streamer, format, nil
}

func interleavedToFloat(samples []int, bitDepth int) []float64 {
	if len(samples) == 0 {
		return nil
	}
	max := maxIntForBitDepth(bitDepth)
	if max <= 0 {
		max = math.MaxInt16
	}
	denom := float64(max)
	converted := make([]float64, len(samples))
	for i, value := range samples {
		converted[i] = float64(value) / denom
	}
	return converted
}

func maxIntForBitDepth(bitDepth int) int {
	if bitDepth <= 0 {
		return 0
	}
	return (1 << (bitDepth - 1)) - 1
}
