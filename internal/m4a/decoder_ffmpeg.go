//go:build ffmpeg

package m4a

import (
	"bytes"
	"errors"
	"os/exec"

	"txp/internal/media"
)

func Decode(path string) (PCM, error) {
	if tooLarge, err := media.IsTooLargeForDecode(path); err == nil && tooLarge {
		return PCM{}, ErrDecodeTooLarge
	}
	cmd := exec.Command("ffmpeg", "-i", path, "-f", "s16le", "-ac", "2", "-ar", "44100", "pipe:1")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return PCM{}, err
	}
	data := out.Bytes()
	if len(data) < 2 {
		return PCM{}, errors.New("no samples decoded")
	}
	samples := make([]float64, 0, len(data)/2)
	for i := 0; i+1 < len(data); i += 2 {
		value := int16(data[i]) | int16(data[i+1])<<8
		samples = append(samples, float64(value)/32768.0)
	}
	return PCM{Samples: samples, Rate: 44100, Channels: 2}, nil
}
