package tags

import (
	"strings"

	"github.com/bogem/id3v2"
)

func writeMP3Tags(path string, fields TagFields) error {
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return err
	}
	defer func() {
		_ = tag.Close()
	}()

	setTextFrame(tag, tag.CommonID("Title"), fields.Title)
	setTextFrame(tag, tag.CommonID("Artist"), fields.Artist)
	setTextFrame(tag, tag.CommonID("Album/Movie/Show title"), fields.Album)
	setTextFrame(tag, tag.CommonID("Content type"), fields.Genre)
	setTextFrame(tag, tag.CommonID("Year"), fields.Year)
	setTextFrame(tag, tag.CommonID("Track number/Position in set"), fields.TrackNum)

	return tag.Save()
}

func setTextFrame(tag *id3v2.Tag, id string, value string) {
	value = strings.TrimSpace(value)
	tag.DeleteFrames(id)
	if value == "" {
		return
	}
	tag.AddTextFrame(id, tag.DefaultEncoding(), value)
}
