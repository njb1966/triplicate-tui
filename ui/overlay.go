package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

// overlayInnerHeight returns the number of item rows visible in an overlay
// given terminal height h.
func overlayInnerHeight(h int) int {
	// Content area minus 2 for top/bottom border, minus 2 for margin rows.
	n := ContentHeight(h) - 4
	if n < 1 {
		return 1
	}
	return n
}

// DrawOverlay renders a bordered, scrollable list over the content area.
//
//	title:    text shown in the top border
//	labels:   display strings for each row (may differ from URLs)
//	selected: index of the highlighted row
//	scroll:   index of the first visible row
func DrawOverlay(s tcell.Screen, title string, labels []string, selected, scroll int) {
	w, h := s.Size()

	// Box bounds: 2-col margin left/right, 1-row margin top/bottom inside content.
	x0, x1 := 2, w-3
	y0 := ContentTop() + 1
	y1 := ContentTop() + ContentHeight(h) - 2

	if x1-x0 < 6 || y1-y0 < 3 {
		return // terminal too small for a useful overlay
	}

	innerW := x1 - x0 - 2 // usable text columns (inside borders)
	innerH := y1 - y0 - 1 // usable item rows  (inside borders)

	borderSt := StyleHeader // bold/bright for borders
	selSt := StyleNormal.Reverse(true)
	bgSt := StyleNormal

	// Fill the whole box background.
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			s.SetContent(x, y, ' ', nil, bgSt)
		}
	}

	// Corners.
	s.SetContent(x0, y0, '┌', nil, borderSt)
	s.SetContent(x1, y0, '┐', nil, borderSt)
	s.SetContent(x0, y1, '└', nil, borderSt)
	s.SetContent(x1, y1, '┘', nil, borderSt)

	// Horizontal edges.
	for x := x0 + 1; x < x1; x++ {
		s.SetContent(x, y0, '─', nil, borderSt)
		s.SetContent(x, y1, '─', nil, borderSt)
	}
	// Vertical edges.
	for y := y0 + 1; y < y1; y++ {
		s.SetContent(x0, y, '│', nil, borderSt)
		s.SetContent(x1, y, '│', nil, borderSt)
	}

	// Title in top border: "─ Title ─".
	titleStr := " " + title + " "
	tx := x0 + 2
	for _, ch := range titleStr {
		if tx >= x1 {
			break
		}
		s.SetContent(tx, y0, ch, nil, borderSt)
		tx++
	}

	// Item count in bottom border.
	if len(labels) > 0 {
		count := fmt.Sprintf(" %d/%d ", selected+1, len(labels))
		cx := x1 - len([]rune(count)) - 1
		for i, ch := range count {
			if cx+i < x1 {
				s.SetContent(cx+i, y1, ch, nil, borderSt)
			}
		}
	}

	// Items.
	for row := 0; row < innerH; row++ {
		idx := scroll + row
		y := y0 + 1 + row
		if idx >= len(labels) {
			break
		}

		st := bgSt
		if idx == selected {
			st = selSt
		}

		label := labels[idx]
		runes := []rune(label)
		if len(runes) > innerW {
			runes = append(runes[:innerW-1], '…')
		}
		col := x0 + 1
		for _, ch := range runes {
			if col >= x1 {
				break
			}
			s.SetContent(col, y, ch, nil, st)
			col++
		}
		for ; col < x1; col++ {
			s.SetContent(col, y, ' ', nil, st)
		}
	}
}
