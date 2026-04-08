package ui

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gdamore/tcell/v2"
	"triplicate-tui/config"
	"triplicate-tui/model"
	"triplicate-tui/protocols/finger"
	"triplicate-tui/protocols/gemini"
	"triplicate-tui/protocols/gopher"
)

// minWidth and minHeight define the smallest terminal this app will render in.
const (
	minWidth  = 80
	minHeight = 24
)

// Mode represents the current input mode of the application.
type Mode int

const (
	ModeNormal  Mode = iota
	ModePrompt       // text input prompt at bottom
	ModeTOFU         // TOFU certificate challenge
	ModeOverlay      // history or bookmarks overlay
)

// PromptType distinguishes what a prompt is collecting.
type PromptType int

const (
	PromptGo     PromptType = iota // navigate to a URL
	PromptSearch                   // search within current page
	PromptInput                    // Gemini status 10/11 server-driven input
)

// OverlayType distinguishes which overlay is shown.
type OverlayType int

const (
	OverlayHistory   OverlayType = iota
	OverlayBookmarks
	OverlayHelp
)

// App is the top-level application state.
type App struct {
	screen tcell.Screen
	mode   Mode
	done   chan struct{}

	// Prompt state
	promptType  PromptType
	promptText  string
	promptLabel string
	inputURL    string // base URL for PromptInput re-navigation

	// Content
	content *ContentPane
	page    *model.Page

	// Page metadata
	protocol  string
	host      string
	tlsStatus string

	// Navigation state
	loading        bool
	loadingURL     string
	noHistoryPush  bool // suppress history push for back/forward
	historyDelta   int  // -1 = went back, +1 = went forward (for rollback on error)
	pendingTOFU    *EventTOFUChallenge
	tofuMismatch   bool

	// Overlay state (ModeOverlay)
	overlayType   OverlayType
	overlayItems  []string // navigation URLs
	overlayLabels []string // display strings
	overlayIdx    int
	overlayScroll int

	// Protocol clients & persistence
	tofu      *gemini.TOFU
	history   *model.History
	bookmarks *model.Bookmarks
	cfg       config.Config

	// pendingURL is navigated on the first iteration of Run().
	pendingURL string
}

// NewApp initialises the terminal screen and returns a ready App.
// startURL is navigated immediately on Run(); if empty, the configured
// homepage is used instead.
func NewApp(startURL string) (*App, error) {
	// Load user config first so theme and homepage are available.
	cfg, _ := config.LoadConfig() // ignore error; defaults are safe

	s, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("create screen: %w", err)
	}
	if err = s.Init(); err != nil {
		return nil, fmt.Errorf("init screen: %w", err)
	}
	s.SetStyle(StyleNormal)
	InitTheme(s, cfg.LowColor)
	s.HideCursor()

	khPath, err := config.KnownHostsPath()
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	tofu, err := gemini.NewTOFU(khPath)
	if err != nil {
		return nil, fmt.Errorf("tofu store: %w", err)
	}

	bmPath, err := config.BookmarksPath()
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	bookmarks, err := model.NewBookmarks(bmPath)
	if err != nil {
		return nil, fmt.Errorf("bookmarks: %w", err)
	}

	pending := startURL
	if pending == "" {
		pending = cfg.Homepage
	}

	w, _ := s.Size()
	app := &App{
		screen:     s,
		content:    NewContentPane(),
		tofu:       tofu,
		history:    model.NewHistory(),
		bookmarks:  bookmarks,
		cfg:        cfg,
		done:       make(chan struct{}),
		pendingURL: pending,
	}
	app.page = demoPage()
	app.content.SetPage(app.page, w)
	return app, nil
}

// Run starts the main event loop and blocks until the user quits.
func (app *App) Run() {
	defer app.screen.Fini()
	defer app.closeDone()
	app.draw()
	// Kick off navigation to the startup URL (homepage or command-line arg).
	if app.pendingURL != "" {
		u := app.pendingURL
		app.pendingURL = ""
		app.navigate(u)
	}
	for {
		ev := app.screen.PollEvent()
		if ev == nil {
			return // screen Fini'd externally
		}
		switch ev := ev.(type) {
		case *tcell.EventResize:
			app.screen.Sync()
			w, _ := app.screen.Size()
			app.content.Resize(w)
			app.draw()

		case *tcell.EventKey:
			if app.handleKey(ev) {
				return
			}
			app.draw()

		case *EventFetchResult:
			app.loading = false
			app.handleFetchResult(ev)
			app.draw()

		case *EventGopherResult:
			app.loading = false
			app.handleGopherResult(ev)
			app.draw()

		case *EventFingerResult:
			app.loading = false
			app.handleFingerResult(ev)
			app.draw()

		case *EventLoadError:
			app.loading = false
			// Roll back optimistic history movement on error.
			if app.noHistoryPush {
				if app.historyDelta < 0 {
					app.history.GoForward()
				} else if app.historyDelta > 0 {
					app.history.GoBack()
				}
				app.noHistoryPush = false
				app.historyDelta = 0
			}
			app.setPage(msgPage("Error", ev.Msg, ev.URL), "ERROR", "", "")
			app.draw()

		case *EventTOFUChallenge:
			app.loading = false
			app.pendingTOFU = ev
			app.tofuMismatch = ev.Info.IsMismatch
			app.mode = ModeTOFU
			if ev.Info.IsMismatch {
				app.setPage(tofuMismatchPage(ev.Info), "TOFU", ev.Info.Host, "MISMATCH")
			} else {
				app.setPage(tofuChallengePage(ev.Info), "TOFU", ev.Info.Host, "UNKNOWN")
			}
			app.draw()

		case *EventQuit:
			return
		}
	}
}

// handleKey processes a key event and returns true if the app should quit.
func (app *App) handleKey(ev *tcell.EventKey) (quit bool) {
	_, h := app.screen.Size()
	height := ContentHeight(h)

	// TOFU challenge mode — only cert decision keys.
	if app.mode == ModeTOFU && app.pendingTOFU != nil {
		switch ev.Rune() {
		case 't':
			if !app.tofuMismatch {
				app.sendTOFUDecision(gemini.TOFUTrustOnce)
			}
		case 'a':
			app.sendTOFUDecision(gemini.TOFUTrustAlways)
		case 'c':
			if app.tofuMismatch {
				app.sendTOFUDecision(gemini.TOFUTrustAlways)
			}
		case 'r':
			app.sendTOFUDecision(gemini.TOFUReject)
		case 'q', 'Q':
			app.sendTOFUDecision(gemini.TOFUReject)
			return true
		}
		return false
	}

	// Overlay mode — j/k/Enter/q.
	if app.mode == ModeOverlay {
		return app.handleOverlayKey(ev, h)
	}

	action := KeyToAction(ev, app.mode)

	switch action {
	case ActionQuit:
		return true

	// --- Scrolling ---
	case ActionScrollDown:
		app.content.ScrollDown(1, height)
	case ActionScrollUp:
		app.content.ScrollUp(1)
	case ActionHalfPageDown:
		app.content.HalfPageDown(height)
	case ActionHalfPageUp:
		app.content.HalfPageUp(height)
	case ActionTop:
		app.content.GoTop()
	case ActionBottom:
		app.content.GoBottom(height)

	// --- Link navigation ---
	case ActionNextLink:
		app.content.NextLink(height)
	case ActionPrevLink:
		app.content.PrevLink(height)
	case ActionFollow:
		if u := app.content.SelectedURL(); u != "" {
			app.navigate(u)
		}

	// --- Search ---
	case ActionSearchNext:
		app.content.SearchNext(height)
	case ActionSearchPrev:
		app.content.SearchPrev(height)

	// --- History ---
	case ActionBack:
		if app.history.CanBack() {
			app.history.GoBack()
			app.noHistoryPush = true
			app.historyDelta = -1
			app.navigate(app.history.Current())
		}
	case ActionForward:
		if app.history.CanForward() {
			app.history.GoForward()
			app.noHistoryPush = true
			app.historyDelta = +1
			app.navigate(app.history.Current())
		}
	case ActionHistory:
		app.openHistoryOverlay()

	// --- Bookmarks ---
	case ActionBookmark:
		app.addBookmark()
	case ActionBookmarkList:
		app.openBookmarksOverlay()

	// --- Prompts ---
	case ActionOpenPrompt:
		app.mode = ModePrompt
		app.promptType = PromptGo
		app.promptText = ""
		app.promptLabel = "Go to: "

	case ActionSearchPrompt:
		app.mode = ModePrompt
		app.promptType = PromptSearch
		app.promptText = ""
		app.promptLabel = "Search: "

	case ActionPromptCancel:
		app.mode = ModeNormal
		app.promptText = ""
		app.promptLabel = ""

	case ActionPromptConfirm:
		input := app.promptText
		ptype := app.promptType
		app.mode = ModeNormal
		app.promptText = ""
		app.promptLabel = ""
		switch ptype {
		case PromptGo:
			app.navigate(input)
		case PromptSearch:
			app.content.Search(input, height)
		case PromptInput:
			if strings.HasPrefix(app.inputURL, "gopher://") {
				app.navigate(gopher.AppendQuery(app.inputURL, input))
			} else {
				app.navigate(app.inputURL + "?" + url.QueryEscape(input))
			}
		}

	case ActionPromptBackspace:
		if len(app.promptText) > 0 {
			r := []rune(app.promptText)
			app.promptText = string(r[:len(r)-1])
		}

	case ActionNone:
		if app.mode == ModePrompt && ev.Key() == tcell.KeyRune {
			app.promptText += string(ev.Rune())
		}

	case ActionHelp:
		app.openHelpOverlay()
	}

	return false
}

// handleOverlayKey handles keys when the history or bookmarks overlay is open.
func (app *App) handleOverlayKey(ev *tcell.EventKey, h int) (quit bool) {
	innerH := overlayInnerHeight(h)

	switch ev.Key() {
	case tcell.KeyEscape:
		app.closeOverlay()
		return false
	case tcell.KeyEnter:
		if app.overlayType == OverlayHelp {
			app.closeOverlay()
		} else if app.overlayIdx < len(app.overlayItems) {
			target := app.overlayItems[app.overlayIdx]
			app.closeOverlay()
			app.navigate(target)
		}
		return false
	case tcell.KeyDown:
		app.overlayDown(innerH)
		return false
	case tcell.KeyUp:
		app.overlayUp()
		return false
	}

	switch ev.Rune() {
	case 'q', 'Q':
		app.closeOverlay()
	case 'j':
		app.overlayDown(innerH)
	case 'k':
		app.overlayUp()
	case 'g':
		app.overlayIdx = 0
		app.overlayScroll = 0
	case 'G':
		app.overlayIdx = max(0, app.overlayLen()-1)
		if app.overlayIdx >= app.overlayScroll+innerH {
			app.overlayScroll = app.overlayIdx - innerH + 1
		}
	}
	return false
}

// overlayLen returns the number of rows in the current overlay.
// For navigation overlays items and labels have equal length; for help the
// labels slice is authoritative and items is nil.
func (app *App) overlayLen() int {
	if len(app.overlayLabels) > len(app.overlayItems) {
		return len(app.overlayLabels)
	}
	return len(app.overlayItems)
}

func (app *App) overlayDown(innerH int) {
	if app.overlayIdx < app.overlayLen()-1 {
		app.overlayIdx++
		if app.overlayIdx >= app.overlayScroll+innerH {
			app.overlayScroll++
		}
	}
}

func (app *App) overlayUp() {
	if app.overlayIdx > 0 {
		app.overlayIdx--
		if app.overlayIdx < app.overlayScroll {
			app.overlayScroll = app.overlayIdx
		}
	}
}

func (app *App) closeOverlay() {
	app.mode = ModeNormal
	app.overlayItems = nil
	app.overlayLabels = nil
}

// openHistoryOverlay shows the history list overlay, most recent first.
func (app *App) openHistoryOverlay() {
	entries := app.history.Entries()
	if len(entries) == 0 {
		return
	}
	n := len(entries)
	items := make([]string, n)
	for i, e := range entries {
		items[n-1-i] = e // reverse: most recent at top
	}
	app.mode = ModeOverlay
	app.overlayType = OverlayHistory
	app.overlayItems = items
	app.overlayLabels = items
	app.overlayIdx = 0
	app.overlayScroll = 0
}

// openBookmarksOverlay shows the bookmarks list overlay.
func (app *App) openBookmarksOverlay() {
	bms := app.bookmarks.Items()
	if len(bms) == 0 {
		return
	}
	items := make([]string, len(bms))
	labels := make([]string, len(bms))
	for i, bm := range bms {
		items[i] = bm.URL
		if bm.Title != "" {
			labels[i] = bm.Title + " — " + bm.URL
		} else {
			labels[i] = bm.URL
		}
	}
	app.mode = ModeOverlay
	app.overlayType = OverlayBookmarks
	app.overlayItems = items
	app.overlayLabels = labels
	app.overlayIdx = 0
	app.overlayScroll = 0
}

// openHelpOverlay shows the key-binding reference overlay.
func (app *App) openHelpOverlay() {
	labels := []string{
		"  j / ↓          Scroll down one line",
		"  k / ↑          Scroll up one line",
		"  Ctrl+D / PgDn  Half page down",
		"  Ctrl+U / PgUp  Half page up",
		"  g              Top of document",
		"  G              Bottom of document",
		"  Tab            Next link",
		"  Shift+Tab      Previous link",
		"  Enter          Follow selected link",
		"  o              Open URL prompt",
		"  /              Search in page",
		"  n / N          Next / previous search match",
		"  b              Back",
		"  f              Forward",
		"  H              History overlay",
		"  m              Bookmark current page",
		"  B              Bookmarks overlay",
		"  q / Esc        Quit",
		"  ?              This help (q or Esc to close)",
	}
	app.mode = ModeOverlay
	app.overlayType = OverlayHelp
	app.overlayItems = nil // no navigation; Enter just closes
	app.overlayLabels = labels
	app.overlayIdx = 0
	app.overlayScroll = 0
}

// addBookmark saves the current page URL to the bookmarks file.
func (app *App) addBookmark() {
	if app.page == nil {
		return
	}
	u := app.page.URL
	if !strings.Contains(u, "://") || strings.HasPrefix(u, "about:") {
		return
	}
	title := app.page.Title
	if title == "" {
		title = app.host
	}
	app.bookmarks.Add(u, title) // ignore error silently
}

// pushHistory adds url to history unless suppressed (back/forward navigation).
func (app *App) pushHistory(rawURL string) {
	if !app.noHistoryPush {
		app.history.Push(rawURL)
	}
	app.noHistoryPush = false
	app.historyDelta = 0
}

// sendTOFUDecision sends a decision on the pending TOFU channel and resets state.
func (app *App) sendTOFUDecision(d gemini.TOFUDecision) {
	if app.pendingTOFU == nil {
		return
	}
	app.pendingTOFU.Decision <- d
	app.pendingTOFU = nil
	app.mode = ModeNormal
	app.loading = (d != gemini.TOFUReject)
}

// navigate resolves a URL and dispatches to the appropriate protocol handler.
func (app *App) navigate(rawURL string) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return
	}
	if !strings.Contains(rawURL, "://") {
		rawURL = "gemini://" + rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		app.setPage(msgPage("Invalid URL", err.Error(), rawURL), "ERROR", "", "")
		return
	}

	switch u.Scheme {
	case "gemini":
		app.loading = true
		app.loadingURL = rawURL
		go app.fetchGemini(rawURL)

	case "gopher":
		if gopher.NeedsQuery(rawURL) {
			app.mode = ModePrompt
			app.promptType = PromptInput
			app.promptLabel = "Search: "
			app.promptText = ""
			app.inputURL = rawURL
			return
		}
		app.loading = true
		app.loadingURL = rawURL
		go app.fetchGopher(rawURL)

	case "finger":
		app.loading = true
		app.loadingURL = rawURL
		go app.fetchFinger(rawURL)

	default:
		app.setPage(msgPage("Unsupported Protocol",
			"Protocol not yet supported: "+u.Scheme, rawURL), "ERROR", "", "")
	}
}

// --- Protocol fetch goroutines ---

func (app *App) fetchGemini(rawURL string) {
	defer func() { recover() }()
	resp, err := gemini.Fetch(rawURL, gemini.FetchOptions{
		TOFU:       app.tofu,
		TOFUPrompt: app.tofuPromptFn,
	})
	if err != nil {
		app.screen.PostEvent(&EventLoadError{eventTime: now(), URL: rawURL, Msg: err.Error()})
		return
	}
	app.screen.PostEvent(&EventFetchResult{eventTime: now(), URL: rawURL, Resp: resp})
}

func (app *App) tofuPromptFn(info *gemini.CertInfo) gemini.TOFUDecision {
	ch := make(chan gemini.TOFUDecision, 1)
	ev := &EventTOFUChallenge{eventTime: now(), Info: info, Decision: ch}
	if err := app.screen.PostEvent(ev); err != nil {
		return gemini.TOFUReject
	}
	select {
	case d := <-ch:
		return d
	case <-app.done:
		return gemini.TOFUReject
	}
}

func (app *App) fetchGopher(rawURL string) {
	defer func() { recover() }()
	result, err := gopher.Fetch(rawURL)
	if err != nil {
		app.screen.PostEvent(&EventLoadError{eventTime: now(), URL: rawURL, Msg: err.Error()})
		return
	}
	app.screen.PostEvent(&EventGopherResult{
		eventTime: now(), URL: rawURL, Type: result.Type, Body: result.Body,
	})
}

func (app *App) fetchFinger(rawURL string) {
	defer func() { recover() }()
	result, err := finger.Fetch(rawURL)
	if err != nil {
		app.screen.PostEvent(&EventLoadError{eventTime: now(), URL: rawURL, Msg: err.Error()})
		return
	}
	app.screen.PostEvent(&EventFingerResult{eventTime: now(), URL: rawURL, Body: result.Body})
}

// --- Result handlers ---

func (app *App) handleFetchResult(ev *EventFetchResult) {
	u, _ := url.Parse(ev.URL)
	host := ""
	if u != nil {
		host = u.Hostname()
	}
	switch ev.Resp.Status / 10 {
	case 2:
		page := pageFromResponse(ev.URL, ev.Resp)
		app.setPage(page, "GEMINI", host, "TOFU OK")
		app.pushHistory(ev.URL)
	case 1:
		app.protocol = "GEMINI"
		app.host = host
		app.tlsStatus = "TOFU OK"
		app.mode = ModePrompt
		app.promptType = PromptInput
		app.promptText = ""
		app.promptLabel = ev.Resp.Meta + ": "
		app.inputURL = ev.URL
		// No history push for input prompts.
		app.noHistoryPush = false
	case 4, 5:
		page := errorPage(ev.Resp.Status, ev.Resp.Meta, ev.URL)
		app.setPage(page, "GEMINI", host, "ERROR")
		app.noHistoryPush = false
	default:
		app.setPage(msgPage(
			fmt.Sprintf("Unexpected status %d", ev.Resp.Status),
			ev.Resp.Meta, ev.URL,
		), "GEMINI", host, "")
		app.noHistoryPush = false
	}
}

func (app *App) handleGopherResult(ev *EventGopherResult) {
	u, _ := url.Parse(ev.URL)
	host := ""
	if u != nil {
		host = u.Hostname()
	}
	var lines []model.RenderLine
	switch ev.Type {
	case '0':
		lines = gopher.ParseText(ev.Body)
	case '1', '7':
		lines = gopher.ParseMenu(ev.Body)
	default:
		lines = []model.RenderLine{
			{Type: model.LineH2, Text: "Cannot display"},
			{Type: model.LineEmpty},
			{Type: model.LineText, Text: fmt.Sprintf("Gopher item type: %q", ev.Type)},
			{Type: model.LineText, Text: ev.URL},
		}
	}
	page := &model.Page{URL: ev.URL, Lines: lines}
	app.setPage(page, "GOPHER", host, "NO TLS")
	app.pushHistory(ev.URL)
}

func (app *App) handleFingerResult(ev *EventFingerResult) {
	u, _ := url.Parse(ev.URL)
	host := ""
	if u != nil {
		host = u.Hostname()
	}
	var lines []model.RenderLine
	text := strings.ToValidUTF8(string(ev.Body), "\uFFFD")
	for _, raw := range strings.Split(text, "\n") {
		lines = append(lines, model.RenderLine{
			Type: model.LineText,
			Text: strings.TrimRight(raw, "\r"),
		})
	}
	page := &model.Page{URL: ev.URL, Lines: lines}
	app.setPage(page, "FINGER", host, "NO TLS")
	app.pushHistory(ev.URL)
}

// setPage loads a page into the content pane and updates statusbar metadata.
func (app *App) setPage(page *model.Page, protocol, host, tlsStatus string) {
	app.page = page
	app.protocol = protocol
	app.host = host
	app.tlsStatus = tlsStatus
	w, _ := app.screen.Size()
	app.content.SetPage(page, w)
}

// draw clears the screen and redraws all UI regions.
func (app *App) draw() {
	app.screen.Clear()
	w, h := app.screen.Size()

	// Guard: show a plain message if the terminal is too small to render.
	if w < minWidth || h < minHeight {
		msg := fmt.Sprintf("Terminal too small (%dx%d) — minimum %dx%d", w, h, minWidth, minHeight)
		runes := []rune(msg)
		if len(runes) > w {
			msg = "Too small — resize terminal"
			runes = []rune(msg)
		}
		x := (w - len(runes)) / 2
		if x < 0 {
			x = 0
		}
		y := h / 2
		if y < 0 {
			y = 0
		}
		DrawTextAt(app.screen, x, y, msg, StyleNormal)
		app.screen.Show()
		return
	}

	protocol := app.protocol
	host := app.host
	if protocol == "" {
		protocol = "---"
	}
	if app.loading {
		protocol = "LOADING"
		if app.loadingURL != "" {
			if u, err := url.Parse(app.loadingURL); err == nil {
				host = u.Host
			}
		}
	}

	DrawStatusbar(app.screen, protocol, host, app.tlsStatus, modeLabel(app.mode, app.promptType))

	if app.page != nil {
		app.content.Draw(app.screen, ContentTop(), ContentHeight(h))
	} else {
		app.drawPlaceholder(h)
	}

	// Draw overlay on top of content when active.
	if app.mode == ModeOverlay {
		title := "History"
		switch app.overlayType {
		case OverlayBookmarks:
			title = "Bookmarks"
		case OverlayHelp:
			title = "Key Bindings"
		}
		DrawOverlay(app.screen, title, app.overlayLabels, app.overlayIdx, app.overlayScroll)
	}

	DrawCmdbar(app.screen, app.mode, app.tofuMismatch, app.promptLabel, app.promptText)
	app.screen.Show()
}

func (app *App) drawPlaceholder(h int) {
	w, _ := app.screen.Size()
	msg := "[ no page loaded — press o to open a URL ]"
	x := (w - len(msg)) / 2
	y := ContentTop() + ContentHeight(h)/2
	if x < 0 {
		x = 0
	}
	DrawTextAt(app.screen, x, y, msg, StyleNormal)
}

func modeLabel(m Mode, pt PromptType) string {
	switch m {
	case ModePrompt:
		switch pt {
		case PromptGo:
			return "PROMPT (go)"
		case PromptSearch:
			return "PROMPT (search)"
		case PromptInput:
			return "PROMPT (input)"
		}
	case ModeTOFU:
		return "TOFU"
	case ModeOverlay:
		return "OVERLAY"
	}
	return "NORMAL"
}

// Shutdown is called by the OS signal handler to cleanly restore the terminal.
func (app *App) Shutdown() {
	defer func() { recover() }()
	app.screen.PostEvent(&EventQuit{eventTime: now()})
}

func (app *App) closeDone() {
	select {
	case <-app.done:
	default:
		close(app.done)
	}
}

// demoPage is shown on startup before any navigation.
func demoPage() *model.Page {
	return &model.Page{
		URL: "about:welcome",
		Lines: []model.RenderLine{
			{Type: model.LineH1, Text: "Welcome to triplicate-tui"},
			{Type: model.LineH2, Text: "A multi-protocol TUI browser"},
			{Type: model.LineEmpty},
			{Type: model.LineText, Text: "Press 'o' to navigate to a URL. Tab/Shift-Tab move between links, Enter follows, '/' searches the page."},
			{Type: model.LineEmpty},
			{Type: model.LineH3, Text: "Quick links"},
			{Type: model.LineLink, Text: "Project Gemini", URL: "gemini://gemini.circumlunar.space/"},
			{Type: model.LineLink, Text: "Floodgap Gopher", URL: "gopher://gopher.floodgap.com/"},
			{Type: model.LineLink, Text: "SDF Public Access Unix", URL: "gemini://sdf.org/"},
			{Type: model.LineLink, Text: "A long link description to test soft-wrapping of link lines in the content pane", URL: "gemini://example.com/long"},
			{Type: model.LineEmpty},
			{Type: model.LineH3, Text: "Keyboard shortcuts"},
			{Type: model.LineText, Text: "j/k or arrows: scroll   Ctrl+D/U or PgDn/Up: half page   g/G: top/bottom"},
			{Type: model.LineText, Text: "Tab/Shift-Tab: next/prev link   Enter: follow link   b/f: back/forward"},
			{Type: model.LineText, Text: "o: open URL   /: search   n/N: next/prev match   H: history   B: bookmarks   q: quit"},
			{Type: model.LineEmpty},
			{Type: model.LineQuote, Text: "Gemini is a new internet protocol which is heavier than gopher but lighter than the web."},
			{Type: model.LineEmpty},
			{Type: model.LineH3, Text: "Protocol ports"},
			{Type: model.LinePre, Text: "  Gemini   port 1965   TLS required"},
			{Type: model.LinePre, Text: "  Gopher   port 70    plain TCP"},
			{Type: model.LinePre, Text: "  Finger   port 79    plain TCP"},
		},
	}
}
