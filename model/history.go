package model

// History tracks the back/forward navigation stack.
type History struct {
	entries []string
	idx     int // current position; -1 when empty
}

func NewHistory() *History { return &History{idx: -1} }

// Push adds url after the current position, discarding any forward history.
// A duplicate of the current entry is silently ignored.
func (h *History) Push(url string) {
	if url == "" {
		return
	}
	if h.idx >= 0 && h.entries[h.idx] == url {
		return
	}
	// Truncate forward history.
	if h.idx < len(h.entries)-1 {
		h.entries = h.entries[:h.idx+1]
	}
	h.entries = append(h.entries, url)
	h.idx = len(h.entries) - 1
}

// CanBack reports whether there is a previous entry.
func (h *History) CanBack() bool { return h.idx > 0 }

// CanForward reports whether there is a next entry.
func (h *History) CanForward() bool { return h.idx < len(h.entries)-1 }

// PeekBack returns the previous URL without moving (empty string if none).
func (h *History) PeekBack() string {
	if !h.CanBack() {
		return ""
	}
	return h.entries[h.idx-1]
}

// PeekForward returns the next URL without moving (empty string if none).
func (h *History) PeekForward() string {
	if !h.CanForward() {
		return ""
	}
	return h.entries[h.idx+1]
}

// GoBack moves the index back by one. Call only after confirming CanBack.
func (h *History) GoBack() {
	if h.idx > 0 {
		h.idx--
	}
}

// GoForward moves the index forward by one. Call only after confirming CanForward.
func (h *History) GoForward() {
	if h.idx < len(h.entries)-1 {
		h.idx++
	}
}

// Current returns the URL at the current index, or "".
func (h *History) Current() string {
	if h.idx < 0 || h.idx >= len(h.entries) {
		return ""
	}
	return h.entries[h.idx]
}

// Entries returns all history URLs oldest-first.
func (h *History) Entries() []string { return h.entries }

// Index returns the current position in the history.
func (h *History) Index() int { return h.idx }
