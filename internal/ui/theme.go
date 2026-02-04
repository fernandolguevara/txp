package ui

import "github.com/gdamore/tcell/v2"

type Theme struct {
	Name    string
	Bg      tcell.Color
	Fg      tcell.Color
	Accent  tcell.Color
	Border  tcell.Color
	Muted   tcell.Color
	Warning tcell.Color
}

var Themes = map[string]Theme{
	"Default": {
		Name:    "Default",
		Bg:      tcell.ColorBlack,
		Fg:      tcell.ColorWhite,
		Accent:  tcell.ColorLightBlue,
		Border:  tcell.ColorGray,
		Muted:   tcell.ColorSilver,
		Warning: tcell.ColorRed,
	},
	"Ocean": {
		Name:    "Ocean",
		Bg:      tcell.ColorBlack,
		Fg:      tcell.ColorLightCyan,
		Accent:  tcell.ColorTeal,
		Border:  tcell.ColorDarkCyan,
		Muted:   tcell.ColorDarkSlateGray,
		Warning: tcell.ColorLightCoral,
	},
	"Amber": {
		Name:    "Amber",
		Bg:      tcell.ColorBlack,
		Fg:      tcell.ColorLightGoldenrodYellow,
		Accent:  tcell.ColorOrange,
		Border:  tcell.ColorDarkGoldenrod,
		Muted:   tcell.ColorTan,
		Warning: tcell.ColorIndianRed,
	},
	"Forest": {
		Name:    "Forest",
		Bg:      tcell.ColorBlack,
		Fg:      tcell.ColorHoneydew,
		Accent:  tcell.ColorLimeGreen,
		Border:  tcell.ColorDarkOliveGreen,
		Muted:   tcell.ColorDarkSeaGreen,
		Warning: tcell.ColorSalmon,
	},
	"Rose": {
		Name:    "Rose",
		Bg:      tcell.ColorBlack,
		Fg:      tcell.ColorMistyRose,
		Accent:  tcell.ColorHotPink,
		Border:  tcell.ColorIndianRed,
		Muted:   tcell.ColorRosyBrown,
		Warning: tcell.ColorRed,
	},
	"Solar": {
		Name:    "Solar",
		Bg:      tcell.ColorBlack,
		Fg:      tcell.ColorLightYellow,
		Accent:  tcell.ColorGold,
		Border:  tcell.ColorDarkOrange,
		Muted:   tcell.ColorKhaki,
		Warning: tcell.ColorOrangeRed,
	},
	"Cyberpunk": {
		Name:    "Cyberpunk",
		Bg:      tcell.NewHexColor(0x030d22),
		Fg:      tcell.NewHexColor(0xfdfeff),
		Accent:  tcell.NewHexColor(0x0ef3ff),
		Border:  tcell.NewHexColor(0xee0077),
		Muted:   tcell.NewHexColor(0x47a1fa),
		Warning: tcell.NewHexColor(0xff2e97),
	},
	"Mono": {
		Name:    "Mono",
		Bg:      tcell.ColorBlack,
		Fg:      tcell.ColorWhite,
		Accent:  tcell.ColorGray,
		Border:  tcell.ColorGray,
		Muted:   tcell.ColorDarkGray,
		Warning: tcell.ColorWhite,
	},
}

func ThemeByName(name string) Theme {
	if theme, ok := Themes[name]; ok {
		return theme
	}
	return Themes["Default"]
}
