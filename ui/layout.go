package ui

import "github.com/gdamore/tcell/v2"

// Row/region helpers for the 3-region layout.

func StatusbarRow() int      { return 0 }
func ContentTop() int        { return 1 }
func CmdbarRow(h int) int    { return h - 1 }
func ContentHeight(h int) int { return h - 2 }

// DrawText writes text at (x, y) using style, then pads with spaces from the
// end of text to column w-1 so the style's background fills the row.
func DrawText(s tcell.Screen, x, y, w int, text string, style tcell.Style) {
	col := x
	for _, ch := range text {
		if col >= w {
			break
		}
		s.SetContent(col, y, ch, nil, style)
		col++
	}
	for ; col < w; col++ {
		s.SetContent(col, y, ' ', nil, style)
	}
}

// DrawTextAt writes text at (x, y) with no padding — for content body lines.
func DrawTextAt(s tcell.Screen, x, y int, text string, style tcell.Style) int {
	col := x
	w, _ := s.Size()
	for _, ch := range text {
		if col >= w {
			break
		}
		s.SetContent(col, y, ch, nil, style)
		col++
	}
	return col
}
