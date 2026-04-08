package ui

import "github.com/gdamore/tcell/v2"

// DrawStatusbar renders the top status bar.
//
// Layout:  " PROTOCOL  host                   [TLS STATUS]  MODE "
func DrawStatusbar(s tcell.Screen, protocol, host, tlsStatus, modeName string) {
	w, _ := s.Size()

	left := " " + protocol
	if host != "" {
		left += "  " + host
	}

	right := ""
	if tlsStatus != "" {
		right += "[" + tlsStatus + "]  "
	}
	right += modeName + " "

	// Fill between left and right with spaces.
	pad := w - len(left) - len(right)
	line := left
	for i := 0; i < pad; i++ {
		line += " "
	}
	if pad >= 0 {
		line += right
	}

	DrawText(s, 0, StatusbarRow(), w, line, StyleStatusbar)
}
