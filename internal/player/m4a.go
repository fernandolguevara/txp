package player

import (
	"errors"

	"github.com/faiface/beep"
	"txp/internal/m4a"
)

type m4aStreamer struct {
	samples  []float64
	rate     beep.SampleRate
	channels int
	pos      int
}

func (s *m4aStreamer) Stream(buf [][2]float64) (int, bool) {
	if s.pos >= len(s.samples) {
		return 0, false
	}
	count := 0
	for i := range buf {
		if s.pos >= len(s.samples) {
			return count, false
		}
		if s.channels == 1 {
			value := s.samples[s.pos]
			s.pos++
			buf[i][0] = value
			buf[i][1] = value
		} else {
			if s.pos+1 >= len(s.samples) {
				return count, false
			}
			buf[i][0] = s.samples[s.pos]
			buf[i][1] = s.samples[s.pos+1]
			s.pos += 2
		}
		count++
	}
	return count, true
}

func (s *m4aStreamer) Err() error { return nil }

func (s *m4aStreamer) Len() int {
	if s.channels <= 0 {
		return 0
	}
	return len(s.samples) / s.channels
}

func (s *m4aStreamer) Position() int {
	if s.channels <= 0 {
		return 0
	}
	return s.pos / s.channels
}

func (s *m4aStreamer) Seek(p int) error {
	if s.channels <= 0 {
		return errors.New("invalid channels")
	}
	if p < 0 {
		p = 0
	}
	idx := p * s.channels
	if idx > len(s.samples) {
		idx = len(s.samples)
	}
	s.pos = idx
	return nil
}

func (s *m4aStreamer) Close() error { return nil }

func decodeM4A(path string) (beep.StreamSeekCloser, beep.Format, error) {
	if tooLarge, err := isTooLargeForDecode(path); err == nil && tooLarge {
		return nil, beep.Format{}, ErrDecodeTooLarge
	}
	pcm, err := m4a.Decode(path)
	if err != nil {
		return nil, beep.Format{}, err
	}
	if pcm.Rate <= 0 {
		return nil, beep.Format{}, errors.New("invalid sample rate")
	}
	channels := pcm.Channels
	if channels <= 0 {
		channels = 2
	}
	streamer := &m4aStreamer{samples: pcm.Samples, rate: beep.SampleRate(pcm.Rate), channels: channels}
	format := beep.Format{SampleRate: beep.SampleRate(pcm.Rate), NumChannels: channels, Precision: 2}
	return streamer, format, nil
}
