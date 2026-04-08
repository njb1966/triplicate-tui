package model

// LineType classifies the semantic type of a rendered line.
type LineType int

const (
	LineText       LineType = iota // plain paragraph text
	LineH1                         // # heading
	LineH2                         // ## heading
	LineH3                         // ### heading
	LineLink                       // => URL [description]
	LineQuote                      // > blockquote
	LinePre                        // preformatted (no wrap)
	LineGopherItem                 // gopher menu entry
	LineEmpty                      // blank separator
)

// RenderLine is a single logical line parsed from a document.
// The rendering layer (ui/content.go) converts these to wrapped display lines.
type RenderLine struct {
	Type LineType
	Text string // display text; for links, the description (or URL if no description)
	URL  string // non-empty for LineLink and LineGopherItem
}

// Page holds a fully parsed document ready for display.
type Page struct {
	URL   string
	Title string
	Lines []RenderLine
}
