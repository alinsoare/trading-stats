package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Colour tokens ported from desktop/.../theme.py.
var (
	colBGBase    = hexColor("#12121f")
	colBGSurface = hexColor("#1c1c2e")
	colBGSidebar = hexColor("#161625")
	colBGInput   = hexColor("#23233a")
	colBGHover   = hexColor("#2d2d50")
	colBGSelect  = hexColor("#3d5a80")

	colAccent      = hexColor("#4f9cf9")
	colTextPrimary = hexColor("#e8eaf6")
	colTextSecond  = hexColor("#9e9ebf")
	colTextMuted   = hexColor("#5c5c7a")
	colBorder      = hexColor("#2e2e48")

	colPositive = hexColor("#4caf50")
	colNegative = hexColor("#ef5350")
)

// Theme returns the application's dark Fyne theme.
func Theme() fyne.Theme { return darkTheme{} }

// darkTheme is a Fyne theme implementing the trading dark palette.
type darkTheme struct{}

var _ fyne.Theme = (*darkTheme)(nil)

func (darkTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return colBGBase
	case theme.ColorNameForeground:
		return colTextPrimary
	case theme.ColorNamePrimary:
		return colAccent
	case theme.ColorNameButton:
		return colBGInput
	case theme.ColorNameInputBackground:
		return colBGInput
	case theme.ColorNameInputBorder:
		return colBorder
	case theme.ColorNameHover:
		return colBGHover
	case theme.ColorNameSelection:
		return colBGSelect
	case theme.ColorNameDisabled:
		return colTextMuted
	case theme.ColorNameDisabledButton:
		return colBGSurface
	case theme.ColorNamePlaceHolder:
		return colTextSecond
	case theme.ColorNameMenuBackground:
		return colBGSurface
	case theme.ColorNameOverlayBackground:
		return colBGSurface
	case theme.ColorNameScrollBar:
		return colBGHover
	case theme.ColorNameSeparator:
		return colBorder
	case theme.ColorNameShadow:
		return color.NRGBA{R: 0, G: 0, B: 0, A: 100}
	default:
		return theme.DefaultTheme().Color(name, theme.VariantDark)
	}
}

func (darkTheme) Font(s fyne.TextStyle) fyne.Resource { return theme.DefaultTheme().Font(s) }

func (darkTheme) Icon(n fyne.ThemeIconName) fyne.Resource { return theme.DefaultTheme().Icon(n) }

func (darkTheme) Size(n fyne.ThemeSizeName) float32 { return theme.DefaultTheme().Size(n) }

func hexColor(s string) color.NRGBA {
	if len(s) > 0 && s[0] == '#' {
		s = s[1:]
	}
	r := hexByte(s[0:2])
	g := hexByte(s[2:4])
	b := hexByte(s[4:6])
	return color.NRGBA{R: r, G: g, B: b, A: 255}
}

func hexByte(s string) uint8 {
	v := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		var d int
		switch {
		case c >= '0' && c <= '9':
			d = int(c - '0')
		case c >= 'a' && c <= 'f':
			d = int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = int(c-'A') + 10
		}
		v = v*16 + d
	}
	return uint8(v)
}
