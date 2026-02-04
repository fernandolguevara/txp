package library

import (
	"fmt"
	"os"
)

func fastChecksum(size int64, mtime int64) string {
	return fmt.Sprintf("%d:%d", size, mtime)
}

func fastChecksumFromPath(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	return fastChecksum(info.Size(), info.ModTime().Unix())
}
