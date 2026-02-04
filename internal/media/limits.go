package media

import (
	"errors"
	"os"
)

const MaxDecodeBytes = 200 * 1024 * 1024

var ErrDecodeTooLarge = errors.New("file too large for full decode")

func IsTooLargeForDecode(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.Size() > MaxDecodeBytes, nil
}
