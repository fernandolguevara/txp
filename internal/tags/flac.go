package tags

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	flacSignature          = "fLaC"
	flacBlockVorbisComment = 4
)

type flacBlock struct {
	Type   byte
	Data   []byte
	IsLast bool
}

func writeFLACTags(path string, fields TagFields) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	sig := make([]byte, 4)
	if _, err := io.ReadFull(file, sig); err != nil {
		return err
	}
	if string(sig) != flacSignature {
		return fmt.Errorf("invalid flac signature")
	}

	blocks, err := readFlacBlocks(file)
	if err != nil {
		return err
	}
	blocks = updateFlacTags(blocks, fields)

	info, _ := file.Stat()
	perm := os.FileMode(0o644)
	if info != nil {
		perm = info.Mode().Perm()
	}

	trimmed := strings.TrimSuffix(path, filepath.Ext(path))
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(trimmed)+"-*.flac")
	if err != nil {
		return err
	}

	if _, err := tmp.Write([]byte(flacSignature)); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := writeFlacBlocks(tmp, blocks); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}

	if _, err := io.Copy(tmp, file); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	_ = os.Chmod(tmp.Name(), perm)
	return os.Rename(tmp.Name(), path)
}

func readFlacBlocks(r io.Reader) ([]flacBlock, error) {
	blocks := []flacBlock{}
	for {
		header := make([]byte, 4)
		if _, err := io.ReadFull(r, header); err != nil {
			return nil, err
		}
		isLast := header[0]&0x80 != 0
		blockType := header[0] & 0x7f
		length := int(header[1])<<16 | int(header[2])<<8 | int(header[3])
		if length < 0 {
			return nil, fmt.Errorf("invalid block length")
		}
		data := make([]byte, length)
		if length > 0 {
			if _, err := io.ReadFull(r, data); err != nil {
				return nil, err
			}
		}
		blocks = append(blocks, flacBlock{Type: blockType, Data: data, IsLast: isLast})
		if isLast {
			break
		}
	}
	return blocks, nil
}

func writeFlacBlocks(w io.Writer, blocks []flacBlock) error {
	for i, block := range blocks {
		isLast := i == len(blocks)-1
		header := []byte{block.Type & 0x7f, 0, 0, 0}
		if isLast {
			header[0] |= 0x80
		}
		length := len(block.Data)
		header[1] = byte((length >> 16) & 0xff)
		header[2] = byte((length >> 8) & 0xff)
		header[3] = byte(length & 0xff)
		if _, err := w.Write(header); err != nil {
			return err
		}
		if length > 0 {
			if _, err := w.Write(block.Data); err != nil {
				return err
			}
		}
	}
	return nil
}

func updateFlacTags(blocks []flacBlock, fields TagFields) []flacBlock {
	updates := buildTagUpdates(fields)
	if len(updates) == 0 {
		return blocks
	}
	for i, block := range blocks {
		if block.Type != flacBlockVorbisComment {
			continue
		}
		vendor, tags, err := parseVorbisComment(block.Data)
		if err != nil {
			return blocks
		}
		updated := applyTagUpdates(tags, updates)
		blocks[i].Data = encodeVorbisComment(vendor, updated)
		return blocks
	}
	blocks = append(blocks, flacBlock{Type: flacBlockVorbisComment, Data: encodeVorbisComment("txp", applyTagUpdates(nil, updates))})
	return blocks
}

func buildTagUpdates(fields TagFields) map[string]string {
	updates := map[string]string{}
	set := func(key string, value string) {
		updates[key] = strings.TrimSpace(value)
	}
	set("TITLE", fields.Title)
	set("ARTIST", fields.Artist)
	set("ALBUM", fields.Album)
	set("GENRE", fields.Genre)
	if strings.TrimSpace(fields.Year) != "" {
		set("DATE", fields.Year)
		set("YEAR", fields.Year)
	} else {
		set("DATE", "")
		set("YEAR", "")
	}
	set("TRACKNUMBER", fields.TrackNum)
	return updates
}

func applyTagUpdates(tags []string, updates map[string]string) []string {
	if tags == nil {
		tags = []string{}
	}
	used := map[string]bool{}
	updated := []string{}
	for _, tag := range tags {
		parts := strings.SplitN(tag, "=", 2)
		if len(parts) != 2 {
			updated = append(updated, tag)
			continue
		}
		key := strings.ToUpper(parts[0])
		value, ok := updates[key]
		if !ok {
			updated = append(updated, tag)
			continue
		}
		used[key] = true
		if value == "" {
			continue
		}
		updated = append(updated, fmt.Sprintf("%s=%s", key, value))
	}
	for key, value := range updates {
		if used[key] || value == "" {
			continue
		}
		updated = append(updated, fmt.Sprintf("%s=%s", key, value))
	}
	return updated
}

func parseVorbisComment(data []byte) (string, []string, error) {
	reader := bytes.NewReader(data)
	var vendorLen uint32
	if err := binary.Read(reader, binary.LittleEndian, &vendorLen); err != nil {
		return "", nil, err
	}
	vendor := make([]byte, vendorLen)
	if _, err := io.ReadFull(reader, vendor); err != nil {
		return "", nil, err
	}
	var count uint32
	if err := binary.Read(reader, binary.LittleEndian, &count); err != nil {
		return string(vendor), nil, err
	}
	tags := make([]string, 0, count)
	for i := uint32(0); i < count; i++ {
		var length uint32
		if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
			return string(vendor), tags, err
		}
		buf := make([]byte, length)
		if _, err := io.ReadFull(reader, buf); err != nil {
			return string(vendor), tags, err
		}
		tags = append(tags, string(buf))
	}
	return string(vendor), tags, nil
}

func encodeVorbisComment(vendor string, tags []string) []byte {
	if vendor == "" {
		vendor = "txp"
	}
	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, uint32(len(vendor)))
	_, _ = buf.WriteString(vendor)
	_ = binary.Write(buf, binary.LittleEndian, uint32(len(tags)))
	for _, tag := range tags {
		data := []byte(tag)
		_ = binary.Write(buf, binary.LittleEndian, uint32(len(data)))
		_, _ = buf.Write(data)
	}
	return buf.Bytes()
}
