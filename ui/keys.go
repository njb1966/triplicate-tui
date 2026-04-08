package ui

import "github.com/gdamore/tcell/v2"

// Action represents a logical user action, independent of which key triggered it.
type Action int

const (
	ActionNone Action = iota
	ActionQuit
	ActionScrollDown
	ActionScrollUp
	ActionHalfPageDown
	ActionHalfPageUp
	ActionTop
	ActionBottom
	ActionNextLink
	ActionPrevLink
	ActionFollow
	ActionOpenPrompt
	ActionSearchPrompt
	ActionBack
	ActionForward
	ActionHistory
	ActionBookmark
	ActionBookmarkList
	ActionHelp
	ActionPromptConfirm
	ActionPromptCancel
	ActionPromptBackspace
	ActionSearchNext
	ActionSearchPrev
)

// KeyToAction maps a tcell key event to an Action given the current mode.
func KeyToAction(ev *tcell.EventKey, mode Mode) Action {
	if mode == ModePrompt {
		switch ev.Key() {
		case tcell.KeyEnter:
			return ActionPromptConfirm
		case tcell.KeyEscape, tcell.KeyCtrlC:
			return ActionPromptCancel
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			return ActionPromptBackspace
		}
		// Printable runes are handled by the caller via ActionNone.
		return ActionNone
	}

	// Normal mode — keys first, then runes.
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyCtrlC:
		return ActionQuit
	case tcell.KeyDown:
		return ActionScrollDown
	case tcell.KeyUp:
		return ActionScrollUp
	case tcell.KeyCtrlD, tcell.KeyPgDn:
		return ActionHalfPageDown
	case tcell.KeyCtrlU, tcell.KeyPgUp:
		return ActionHalfPageUp
	case tcell.KeyHome:
		return ActionTop
	case tcell.KeyEnd:
		return ActionBottom
	case tcell.KeyTab:
		return ActionNextLink
	case tcell.KeyBacktab:
		return ActionPrevLink
	case tcell.KeyEnter:
		return ActionFollow
	}

	switch ev.Rune() {
	case 'q', 'Q':
		return ActionQuit
	case 'j':
		return ActionScrollDown
	case 'k':
		return ActionScrollUp
	case 'g':
		return ActionTop
	case 'G':
		return ActionBottom
	case 'o':
		return ActionOpenPrompt
	case '/':
		return ActionSearchPrompt
	case 'n':
		return ActionSearchNext
	case 'N':
		return ActionSearchPrev
	case 'b':
		return ActionBack
	case 'f':
		return ActionForward
	case 'H':
		return ActionHistory
	case 'm':
		return ActionBookmark
	case 'B':
		return ActionBookmarkList
	case '?':
		return ActionHelp
	}

	return ActionNone
}
