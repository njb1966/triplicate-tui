package ui

import "github.com/gdamore/tcell/v2"

const hints = " q:quit  o:go  b:back  /:search  ?:help"

const (
	tofuHints         = " [t] trust once   [a] always trust   [r] reject"
	tofuMismatchHints = " [c] update trust   [r] reject"
)

// DrawCmdbar renders the bottom command/prompt bar.
//
//   - ModeNormal:   key hint strip
//   - ModePrompt:   labelled input field with cursor
//   - ModeTOFU:     certificate decision prompts
//   - ModeOverlay:  overlay navigation hints
func DrawCmdbar(s tcell.Screen, mode Mode, tofuMismatch bool, promptLabel, promptText string) {
	w, h := s.Size()
	row := CmdbarRow(h)

	switch mode {
	case ModeNormal:
		s.HideCursor()
		DrawText(s, 0, row, w, hints, StyleCmdbar)

	case ModeTOFU:
		s.HideCursor()
		bar := tofuHints
		if tofuMismatch {
			bar = tofuMismatchHints
		}
		DrawText(s, 0, row, w, bar, StylePrompt)

	case ModeOverlay:
		s.HideCursor()
		DrawText(s, 0, row, w, " j/k: move   Enter: go   q: close", StyleCmdbar)

	case ModePrompt:
		line := " " + promptLabel + promptText
		DrawText(s, 0, row, w, line, StylePrompt)
		// Place cursor right after typed text.
		cursorX := 1 + len([]rune(promptLabel)) + len([]rune(promptText))
		if cursorX >= w {
			cursorX = w - 1
		}
		s.ShowCursor(cursorX, row)
	}
}
