package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"triplicate-tui/model"
)

// WrappedLine is a single display line after soft-wrapping.
type WrappedLine struct {
	Style tcell.Style
	Text  string
	URL   string // non-empty only on the first wrapped piece of a link/gopher item
}

// ContentPane manages the scrollable content area: wrapping, scrolling,
// link cursor navigation, and text search.
type ContentPane struct {
	page         *model.Page
	lines        []WrappedLine
	wrapWidth    int
	scrollOffset int
	cursorIdx    int   // index into lines of selected link; -1 = none
	searchHits   []int // line indices matching current search
	searchIdx    int   // current hit; -1 = no active search
}

func NewContentPane() *ContentPane {
	return &ContentPane{cursorIdx: -1, searchIdx: -1}
}

// SetPage loads a new page and wraps it to the given width.
func (c *ContentPane) SetPage(page *model.Page, width int) {
	c.page = page
	c.wrapWidth = width
	c.lines = wrapPage(page, width)
	c.scrollOffset = 0
	c.cursorIdx = -1
	c.searchHits = nil
	c.searchIdx = -1
}

// Resize re-wraps the current page if the terminal width changed.
func (c *ContentPane) Resize(width int) {
	if c.page == nil || width == c.wrapWidth {
		return
	}
	c.lines = wrapPage(c.page, width)
	c.wrapWidth = width
	c.scrollOffset = min(c.scrollOffset, max(0, len(c.lines)-1))
	c.cursorIdx = -1
	c.searchHits = nil
	c.searchIdx = -1
}

// LineCount returns the number of wrapped display lines.
func (c *ContentPane) LineCount() int { return len(c.lines) }

// SelectedURL returns the URL of the currently highlighted link, or "".
func (c *ContentPane) SelectedURL() string {
	if c.cursorIdx < 0 || c.cursorIdx >= len(c.lines) {
		return ""
	}
	return c.lines[c.cursorIdx].URL
}

// --- Scroll actions ---

func (c *ContentPane) ScrollDown(n, height int) {
	maxOff := max(0, len(c.lines)-height)
	c.scrollOffset = min(c.scrollOffset+n, maxOff)
}

func (c *ContentPane) ScrollUp(n int) {
	c.scrollOffset = max(0, c.scrollOffset-n)
}

func (c *ContentPane) HalfPageDown(height int) { c.ScrollDown(max(1, height/2), height) }
func (c *ContentPane) HalfPageUp(height int)   { c.ScrollUp(max(1, height/2)) }

func (c *ContentPane) GoTop() {
	c.scrollOffset = 0
	c.cursorIdx = -1
}

func (c *ContentPane) GoBottom(height int) {
	c.scrollOffset = max(0, len(c.lines)-height)
	c.cursorIdx = -1
}

// --- Link navigation ---

// NextLink moves the cursor to the next selectable link (Tab).
func (c *ContentPane) NextLink(height int) {
	start := c.cursorIdx + 1
	if start < 0 {
		start = 0
	}
	for i := start; i < len(c.lines); i++ {
		if c.lines[i].URL != "" {
			c.cursorIdx = i
			c.ensureVisible(i, height)
			return
		}
	}
	// Wrap around to beginning.
	for i := 0; i < start && i < len(c.lines); i++ {
		if c.lines[i].URL != "" {
			c.cursorIdx = i
			c.ensureVisible(i, height)
			return
		}
	}
}

// PrevLink moves the cursor to the previous selectable link (Shift+Tab).
func (c *ContentPane) PrevLink(height int) {
	start := c.cursorIdx - 1
	if start < 0 {
		start = len(c.lines) - 1
	}
	for i := start; i >= 0; i-- {
		if c.lines[i].URL != "" {
			c.cursorIdx = i
			c.ensureVisible(i, height)
			return
		}
	}
	// Wrap around to end.
	for i := len(c.lines) - 1; i > start; i-- {
		if c.lines[i].URL != "" {
			c.cursorIdx = i
			c.ensureVisible(i, height)
			return
		}
	}
}

// ensureVisible scrolls so that lineIdx is within the visible window.
func (c *ContentPane) ensureVisible(lineIdx, height int) {
	if lineIdx < c.scrollOffset {
		c.scrollOffset = lineIdx
	} else if lineIdx >= c.scrollOffset+height {
		c.scrollOffset = lineIdx - height + 1
	}
}

// --- Search ---

// Search finds all case-insensitive occurrences of query and jumps to the
// first match at or after the current scroll position. Returns false if no
// matches found.
func (c *ContentPane) Search(query string, height int) bool {
	c.searchHits = nil
	c.searchIdx = -1
	if query == "" {
		return false
	}
	lower := strings.ToLower(query)
	for i, wl := range c.lines {
		if strings.Contains(strings.ToLower(wl.Text), lower) {
			c.searchHits = append(c.searchHits, i)
		}
	}
	if len(c.searchHits) == 0 {
		return false
	}
	// Start from first match at or after current scroll offset.
	c.searchIdx = 0
	for i, hit := range c.searchHits {
		if hit >= c.scrollOffset {
			c.searchIdx = i
			break
		}
	}
	c.ensureVisible(c.searchHits[c.searchIdx], height)
	return true
}

func (c *ContentPane) SearchNext(height int) {
	if len(c.searchHits) == 0 {
		return
	}
	c.searchIdx = (c.searchIdx + 1) % len(c.searchHits)
	c.ensureVisible(c.searchHits[c.searchIdx], height)
}

func (c *ContentPane) SearchPrev(height int) {
	if len(c.searchHits) == 0 {
		return
	}
	c.searchIdx = (c.searchIdx - 1 + len(c.searchHits)) % len(c.searchHits)
	c.ensureVisible(c.searchHits[c.searchIdx], height)
}

func (c *ContentPane) ClearSearch() {
	c.searchHits = nil
	c.searchIdx = -1
}

// --- Drawing ---

// Draw renders the content area starting at screen row `top` for `height` rows.
func (c *ContentPane) Draw(s tcell.Screen, top, height int) {
	w, _ := s.Size()
	for row := 0; row < height; row++ {
		y := top + row
		lineIdx := c.scrollOffset + row

		if lineIdx >= len(c.lines) {
			clearRow(s, y, w)
			continue
		}

		wl := c.lines[lineIdx]
		style := wl.Style

		// Highlight selected link cursor.
		if lineIdx == c.cursorIdx && wl.URL != "" {
			style = style.Reverse(true)
		}
		// Highlight current search match.
		if c.searchIdx >= 0 && lineIdx == c.searchHits[c.searchIdx] {
			style = tcell.StyleDefault.
				Background(tcell.ColorYellow).
				Foreground(tcell.ColorBlack)
		}

		col := 0
		for _, ch := range wl.Text {
			if col >= w {
				break
			}
			s.SetContent(col, y, ch, nil, style)
			col++
		}
		// Fill remainder of row with default background.
		for ; col < w; col++ {
			s.SetContent(col, y, ' ', nil, StyleNormal)
		}
	}
}

func clearRow(s tcell.Screen, y, w int) {
	for col := 0; col < w; col++ {
		s.SetContent(col, y, ' ', nil, StyleNormal)
	}
}

// --- Wrapping ---

func wrapPage(page *model.Page, width int) []WrappedLine {
	if page == nil {
		return nil
	}
	var out []WrappedLine
	for _, line := range page.Lines {
		out = append(out, wrapLine(line, width)...)
	}
	return out
}

func wrapLine(line model.RenderLine, width int) []WrappedLine {
	switch line.Type {
	case model.LineEmpty:
		return []WrappedLine{{Style: StyleNormal, Text: ""}}

	case model.LineH1:
		// Extra blank line above H1 for visual separation.
		pre := []WrappedLine{{Style: StyleNormal, Text: ""}}
		style := StyleHeader.Underline(true)
		return append(pre, wrapTextToLines(line.Text, width, style, "")...)

	case model.LineH2:
		return wrapTextToLines(line.Text, width, StyleHeader, "")

	case model.LineH3:
		return wrapTextToLines(line.Text, width, StyleH3, "")

	case model.LineText:
		if line.Text == "" {
			return []WrappedLine{{Style: StyleNormal, Text: ""}}
		}
		return wrapTextToLines(line.Text, width, StyleNormal, "")

	case model.LineLink:
		display := line.Text
		if display == "" {
			display = line.URL
		}
		wrapped := wrapTextToLines("=> "+display, width, StyleLink, line.URL)
		// Only the first wrapped piece gets the URL (it's the Tab target).
		for i := 1; i < len(wrapped); i++ {
			wrapped[i].URL = ""
		}
		return wrapped

	case model.LineQuote:
		return wrapTextToLines("▌ "+line.Text, width, StyleQuote, "")

	case model.LinePre:
		// No wrap. Truncate to width if needed.
		text := line.Text
		runes := []rune(text)
		if len(runes) > width {
			text = string(runes[:width])
		}
		return []WrappedLine{{Style: StyleNormal, Text: text}}

	case model.LineGopherItem:
		display := line.Text
		if display == "" {
			display = line.URL
		}
		wrapped := wrapTextToLines("  "+display, width, StyleLink, line.URL)
		for i := 1; i < len(wrapped); i++ {
			wrapped[i].URL = ""
		}
		return wrapped
	}

	return []WrappedLine{{Style: StyleNormal, Text: line.Text}}
}

// wrapTextToLines splits text into lines of at most width runes, breaking at
// word boundaries. url is attached only to the first line.
func wrapTextToLines(text string, width int, style tcell.Style, url string) []WrappedLine {
	if width < 4 {
		width = 4
	}
	if text == "" {
		return []WrappedLine{{Style: style, Text: "", URL: url}}
	}

	var result []WrappedLine
	firstURL := url

	for text != "" {
		runes := []rune(text)
		if len(runes) <= width {
			result = append(result, WrappedLine{Style: style, Text: text, URL: firstURL})
			break
		}
		// Find a word-boundary break point at or before width.
		bp := width
		for bp > 0 && runes[bp-1] != ' ' {
			bp--
		}
		if bp == 0 {
			bp = width // no space found: hard break
		}
		chunk := strings.TrimRight(string(runes[:bp]), " ")
		result = append(result, WrappedLine{Style: style, Text: chunk, URL: firstURL})
		firstURL = ""
		text = strings.TrimLeft(string(runes[bp:]), " ")
	}

	return result
}
