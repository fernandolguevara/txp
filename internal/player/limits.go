package player

import "txp/internal/media"

var ErrDecodeTooLarge = media.ErrDecodeTooLarge

func isTooLargeForDecode(path string) (bool, error) {
	return media.IsTooLargeForDecode(path)
}
