package ui

import "github.com/gdamore/tcell/v2"

var (
	StyleNormal    = tcell.StyleDefault
	StyleStatusbar = tcell.StyleDefault
	StyleLink      = tcell.StyleDefault
	StyleHeader    = tcell.StyleDefault
	StyleH3        = tcell.StyleDefault
	StyleQuote     = tcell.StyleDefault
	StyleCmdbar    = tcell.StyleDefault
	StylePrompt    = tcell.StyleDefault
)

// InitTheme sets up color styles based on what the terminal supports.
// Pass forceLowColor=true to override and use attribute-only styles even on
// color-capable terminals (respects the user's low_color config setting).
func InitTheme(s tcell.Screen, forceLowColor bool) {
	if !forceLowColor && s.Colors() >= 8 {
		StyleStatusbar = tcell.StyleDefault.
			Background(tcell.ColorNavy).
			Foreground(tcell.ColorWhite)
		StyleLink = tcell.StyleDefault.
			Foreground(tcell.ColorTeal).
			Underline(true)
		StyleHeader = tcell.StyleDefault.
			Foreground(tcell.ColorWhite).
			Bold(true)
		StyleH3 = tcell.StyleDefault.
			Foreground(tcell.ColorPurple)
		StyleQuote = tcell.StyleDefault.
			Foreground(tcell.ColorOlive).
			Dim(true)
		StyleCmdbar = tcell.StyleDefault.
			Background(tcell.ColorGray).
			Foreground(tcell.ColorBlack)
		StylePrompt = tcell.StyleDefault.
			Background(tcell.ColorGray).
			Foreground(tcell.ColorBlack).
			Bold(true)
	} else {
		// Low-color fallback: attributes only
		StyleStatusbar = tcell.StyleDefault.Reverse(true)
		StyleLink = tcell.StyleDefault.Underline(true)
		StyleHeader = tcell.StyleDefault.Bold(true)
		StyleH3 = tcell.StyleDefault.Bold(true)
		StyleQuote = tcell.StyleDefault.Dim(true)
		StyleCmdbar = tcell.StyleDefault.Reverse(true)
		StylePrompt = tcell.StyleDefault.Reverse(true).Bold(true)
	}
}
