//go:build !ffmpeg

package m4a

import "errors"

func Decode(path string) (PCM, error) {
	return PCM{}, errors.New("m4a decode requires FFmpeg (build with tag 'ffmpeg')")
}
