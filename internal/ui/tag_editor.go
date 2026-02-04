package ui

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"

	"txp/internal/model"
)

type TagEditorDialog struct {
	Root      *tview.Form
	Active    bool
	Title     *tview.InputField
	Artist    *tview.InputField
	Album     *tview.InputField
	Genre     *tview.InputField
	Year      *tview.InputField
	TrackNum  *tview.InputField
	BPM       *tview.InputField
	Key       *tview.InputField
	Supported *tview.InputField
	Path      string
	onSave    func(TagUpdate) bool
	onCancel  func()
}

type TagUpdate struct {
	Path     string
	Title    string
	Artist   string
	Album    string
	Genre    string
	Year     string
	TrackNum string
}

func NewTagEditorDialog() *TagEditorDialog {
	form := tview.NewForm()
	form.SetBorder(true).SetTitle("[ Tag Editor ]")

	title := tview.NewInputField().SetLabel("Title: ")
	artist := tview.NewInputField().SetLabel("Artist: ")
	album := tview.NewInputField().SetLabel("Album: ")
	genre := tview.NewInputField().SetLabel("Genre: ")
	year := tview.NewInputField().SetLabel("Year: ")
	trackNum := tview.NewInputField().SetLabel("Track #: ")

	form.AddFormItem(title)
	form.AddFormItem(artist)
	form.AddFormItem(album)
	form.AddFormItem(genre)
	form.AddFormItem(year)
	form.AddFormItem(trackNum)
	bpmView := tview.NewInputField().SetLabel("BPM: ").SetText("-")
	bpmView.SetDisabled(true)
	keyView := tview.NewInputField().SetLabel("Key: ").SetText("-")
	keyView.SetDisabled(true)
	supportedView := tview.NewInputField().SetLabel("Write: ").SetText("")
	supportedView.SetDisabled(true)

	form.AddFormItem(bpmView)
	form.AddFormItem(keyView)
	form.AddFormItem(supportedView)

	dialog := &TagEditorDialog{
		Root:      form,
		Title:     title,
		Artist:    artist,
		Album:     album,
		Genre:     genre,
		Year:      year,
		TrackNum:  trackNum,
		BPM:       bpmView,
		Key:       keyView,
		Supported: supportedView,
	}

	form.AddButton("Save", func() {
		if dialog.onSave == nil {
			return
		}
		update := dialog.currentUpdate()
		if dialog.onSave(update) {
			dialog.Active = false
		}
	})
	form.AddButton("Cancel", func() {
		if dialog.onCancel != nil {
			dialog.onCancel()
		}
	})
	form.SetCancelFunc(func() {
		if dialog.onCancel != nil {
			dialog.onCancel()
		}
	})

	return dialog
}

func (d *TagEditorDialog) Configure(track model.Track, onSave func(TagUpdate) bool, onCancel func()) {
	d.Path = track.Path
	d.onSave = onSave
	d.onCancel = onCancel
	d.Active = true

	d.Title.SetText(strings.TrimSpace(track.Title))
	d.Artist.SetText(strings.TrimSpace(track.Artist))
	d.Album.SetText(strings.TrimSpace(track.Album))
	d.Genre.SetText(strings.TrimSpace(track.Genre))
	if track.Year > 0 {
		d.Year.SetText(fmt.Sprintf("%d", track.Year))
	} else {
		d.Year.SetText("")
	}
	if track.TrackNum > 0 {
		d.TrackNum.SetText(fmt.Sprintf("%d", track.TrackNum))
	} else {
		d.TrackNum.SetText("")
	}

	bpm := "-"
	if track.BPM > 0 {
		bpm = fmt.Sprintf("%.1f", track.BPM)
	}
	key := "-"
	if strings.TrimSpace(track.Key) != "" {
		key = strings.TrimSpace(track.Key)
	}

	d.BPM.SetText(bpm)
	d.Key.SetText(key)
}

func (d *TagEditorDialog) SetSupportedText(text string) {
	if d.Supported == nil {
		return
	}
	d.Supported.SetText(text)
}

func (d *TagEditorDialog) applyTextColors(theme Theme) {
	form := d.Root
	if form == nil {
		return
	}
	for i := 0; i < form.GetFormItemCount(); i++ {
		item := form.GetFormItem(i)
		switch view := item.(type) {
		case *tview.TextView:
			view.SetTextColor(theme.Fg).SetBackgroundColor(theme.Bg)
		case *tview.InputField:
			view.SetFieldTextColor(theme.Fg).SetFieldBackgroundColor(theme.Bg)
			view.SetLabelColor(theme.Muted)
		}
	}
}

func (d *TagEditorDialog) currentUpdate() TagUpdate {
	return TagUpdate{
		Path:     d.Path,
		Title:    strings.TrimSpace(d.Title.GetText()),
		Artist:   strings.TrimSpace(d.Artist.GetText()),
		Album:    strings.TrimSpace(d.Album.GetText()),
		Genre:    strings.TrimSpace(d.Genre.GetText()),
		Year:     strings.TrimSpace(d.Year.GetText()),
		TrackNum: strings.TrimSpace(d.TrackNum.GetText()),
	}
}
