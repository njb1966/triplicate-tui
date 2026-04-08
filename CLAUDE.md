# CLAUDE.md — triplicate-tui

## Project Overview

**triplicate-tui** is a keyboard-driven TUI browser for Gemini, Gopher, and Finger protocols.

- **Design model:** Lynx-inspired; minimal, 80×24-safe, three-region layout
- **Language:** Go
- **TUI library:** `github.com/gdamore/tcell/v2`
- **TLS:** Go stdlib `crypto/tls` (for Gemini)
- **Cert policy:** TOFU (trust-on-first-use) — fingerprint store at `~/.config/triplicate/known_hosts`
- **Status:** Fully implemented (all 8 phases complete)

---

## Directory Structure

```
triplicate-tui/
├── main.go                   # Entry point; optional URL argument
├── go.mod
├── go.sum
├── Makefile
├── ui/
│   ├── app.go                # App struct, main event loop, screen init
│   ├── layout.go             # 3-region layout helpers: statusbar, content, cmdbar
│   ├── content.go            # Scrollable content pane: wrapping, scroll, search
│   ├── statusbar.go          # Top bar: protocol, host, TLS/TOFU indicator
│   ├── cmdbar.go             # Bottom bar: hints + prompt input mode
│   ├── keys.go               # Keymap definitions and dispatch (Action enum)
│   ├── theme.go              # Color styles: Normal, Statusbar, Link, Header, Quote, Cmdbar
│   ├── events.go             # Custom tcell events for async goroutine→UI communication
│   ├── navigation.go         # Page builders: pageFromResponse, errorPage, tofuChallengePage
│   └── overlay.go            # DrawOverlay: bordered scrollable list over content area
├── protocols/
│   ├── gemini/
│   │   ├── client.go         # TLS dial, send request, read response, redirect loop
│   │   ├── parser.go         # text/gemini → []RenderLine
│   │   └── tofu.go           # TOFU: known_hosts read/write/verify/prompt
│   ├── gopher/
│   │   ├── client.go         # TCP dial, send selector, read response
│   │   └── parser.go         # Gopher menu/text parsing → []RenderLine
│   └── finger/
│       └── client.go         # TCP port 79, raw text → []byte
├── model/
│   ├── page.go               # Page, RenderLine, LineType
│   ├── history.go            # History stack (back/forward)
│   └── bookmark.go           # Bookmarks file (~/.config/triplicate/bookmarks)
└── config/
    └── config.go             # Config struct, Load/SaveConfig, path helpers
```

---

## Core Data Types

```go
// model/page.go
type LineType int
const (
    LineText LineType = iota
    LineH1
    LineH2
    LineH3
    LineLink
    LineQuote
    LinePre
    LineGopherItem
    LineEmpty
)

type RenderLine struct {
    Type LineType
    Text string
    URL  string  // non-empty for links and gopher items
}

type Page struct {
    URL   string
    Title string
    Lines []RenderLine
}
```

---

## UI Layout (3 regions)

```
┌──────────────────────────────────────────────────────────────────┐
│ GEMINI  gemini.circumlunar.space                    [TOFU OK]     │  ← statusbar (1 line)
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Content area — scrollable, soft-wrapped                         │  ← content (rows-2 lines)
│  Links highlighted, headers bold, quotes marked with ▌           │
│                                                                  │
├──────────────────────────────────────────────────────────────────┤
│ q:quit  o:go  b:back  /:search  ?:help                           │  ← cmdbar (1 line)
└──────────────────────────────────────────────────────────────────┘
```

- **Normal mode:** cmdbar shows hints
- **Prompt mode:** cmdbar shows `Go to: ` or `Search: ` with cursor input; status bar shows `PROMPT (go)`
- **TOFU mode:** cmdbar shows `[t] trust once   [a] always trust   [r] reject`
- **Overlay mode:** cmdbar shows `j/k: move   Enter: go   q: close`

---

## Color Pairs

| Name          | Foreground | Background  | Attrs     |
|---------------|------------|-------------|-----------|
| StyleNormal   | Default    | Default     | —         |
| StyleStatusbar| White      | Navy        | —         |
| StyleLink     | Teal       | Default     | Underline |
| StyleHeader   | White      | Default     | Bold      |
| StyleH3       | Purple     | Default     | —         |
| StyleQuote    | Olive      | Default     | Dim       |
| StyleCmdbar   | Black      | Silver      | —         |
| StylePrompt   | Black      | Silver      | Bold      |

> **Note:** Silver (ANSI 7 / light gray) is used for cmdbar/prompt backgrounds.
> `tcell.ColorGray` (ANSI 8 / dark gray) must be avoided — it renders near-black on dark
> terminals, making text invisible.

Low-color fallback (`low_color = true` in config, or < 8 terminal colors):
`Bold` for headers, `Underline` for links, `Reverse` for statusbar/cmdbar/prompt.

---

## Async Architecture

Protocol fetches run in goroutines. Communication back to the UI uses custom `tcell.Event`
types posted via `screen.PostEvent()`. The main event loop handles them like any other event.

Custom events (defined in `ui/events.go`):

| Event | Purpose |
|-------|---------|
| `EventFetchResult` | Gemini fetch complete (Response) |
| `EventGopherResult` | Gopher fetch complete (Type + Body) |
| `EventFingerResult` | Finger fetch complete (Body) |
| `EventLoadError` | Any protocol fetch error |
| `EventTOFUChallenge` | TOFU decision needed (contains `chan<- TOFUDecision`) |
| `EventQuit` | Posted by signal handler to cleanly exit the event loop |

The TOFU challenge uses a blocking channel: the fetch goroutine posts the challenge and
blocks on `select { case d := <-ch: ... case <-app.done: ... }` until the user responds
or the app shuts down.

---

## Gemini Protocol

- URL scheme: `gemini://host[:port]/path[?query]`
- TLS via `crypto/tls` with `InsecureSkipVerify: true` — TOFU handles trust manually
- Auto-follows redirects up to 5 hops

### TOFU Flow

1. Compute SHA-256 fingerprint of leaf cert on connect
2. Look up `host` in `~/.config/triplicate/known_hosts`
3. **Unknown host:** show CN, expiry, fingerprint → prompt `[t]rust once / [a]lways / [r]eject`
4. **Known + match:** connect silently, show `[TOFU OK]` in statusbar
5. **Known + MISMATCH:** warn loudly, require explicit `[c]` to update trust or `[r]` to reject

`known_hosts` format: one entry per line: `hostname sha256:FINGERPRINT`

### Status Codes

| Code | Handling |
|------|----------|
| `10/11` | Input — open cmdbar prompt with server's prompt text, resend with query |
| `20` | Render body (default MIME: `text/gemini`) |
| `30/31` | Redirect — auto-followed up to 5 hops |
| `40-49` | Client error — display `Gemini 4x: <message>` in content pane |
| `50-59` | Server error — display `Gemini 5x: <message>` in content pane |

---

## Gopher Protocol

- URL scheme: `gopher://host[:port]/[type][selector]`
- TCP only (no TLS), default port 70
- Item types handled: `0` (text file), `1` (menu), `7` (search), `i` (info), `3` (error), `h` (HTTP link)
- Unsupported types: shown as `[bin]` prefix, navigable but not renderable
- Search (type `7`): open cmdbar prompt, send `selector\tquery\r\n`
- Type-7 query encoding: `%09` in URL path; decoded by Go's `url.Parse` → tab

---

## Finger Protocol

- Syntax: `finger://user@host` or `finger://host`
- TCP port 79, send `user\r\n` (or `\r\n` for host-only)
- Raw text output only — render as plain `LineText` lines

---

## Navigation Keymap

| Key              | Action                      |
|------------------|-----------------------------|
| `j` / `↓`        | Scroll down 1 line          |
| `k` / `↑`        | Scroll up 1 line            |
| `Ctrl+D` / PgDn  | Half-page down              |
| `Ctrl+U` / PgUp  | Half-page up                |
| `g`              | Top of document             |
| `G`              | Bottom of document          |
| `Tab`            | Next link                   |
| `Shift+Tab`      | Previous link               |
| `Enter`          | Follow selected link        |
| `o`              | Open URL prompt             |
| `b`              | Back                        |
| `f`              | Forward                     |
| `H`              | Show history list           |
| `m`              | Bookmark current page       |
| `B`              | Show bookmarks              |
| `/`              | Search in page              |
| `n` / `N`        | Next / previous match       |
| `q`              | Quit                        |
| `?`              | Help overlay                |

---

## Configuration

File: `~/.config/triplicate/config`

```ini
homepage  = gemini://gemini.circumlunar.space/
theme     = dark
low_color = false
```

Persistent state files:

| File | Purpose |
|------|---------|
| `~/.config/triplicate/known_hosts` | TOFU fingerprints (`hostname sha256:FP`) |
| `~/.config/triplicate/bookmarks` | One `url [title]` per line |

---

## Build & Run

```bash
make build    # produces ./triplicate-tui binary
make run      # build + run with default homepage
make clean    # remove binary
```

Manual:
```bash
go build -o triplicate-tui .
./triplicate-tui                                  # navigates to configured homepage
./triplicate-tui gemini://gemini.circumlunar.space/  # opens URL directly
```

---

## Implementation Phases

### Phase 1 — Scaffold
- [x] `go mod init triplicate-tui`
- [x] `go get github.com/gdamore/tcell/v2`
- [x] Create directory structure
- [x] `main.go`: init tcell screen, run app loop, handle `q` to quit
- [x] Write `Makefile` (`build`, `run`, `clean`)

### Phase 2 — Core UI
- [x] `ui/layout.go`: 3-region layout, terminal resize handling
- [x] `ui/statusbar.go`: protocol, host, TOFU status
- [x] `ui/cmdbar.go`: hint bar + prompt input mode
- [x] `ui/theme.go`: color pairs + low-color fallback
- [x] `ui/keys.go`: global keymap dispatch
- [x] `ui/app.go`: App struct, event loop

### Phase 3 — Content Model & Rendering
- [x] `model/page.go`: Page, RenderLine, LineType types
- [x] `ui/content.go`: scroll model (`scrollOffset`, `cursorIdx`, viewport)
- [x] Soft-wrap on page load; re-wrap on resize
- [x] Link cursor highlight (Tab navigation through RenderLines where URL != "")
- [x] Page search (`/`, `n`, `N`)

### Phase 4 — Gemini
- [x] `protocols/gemini/client.go`: TLS dial, send request, read response header + body
- [x] `protocols/gemini/tofu.go`: fingerprint compute, known_hosts lookup, interactive prompt
- [x] `protocols/gemini/parser.go`: text/gemini → `[]RenderLine`
- [x] Handle all status code families (10/11/20/3x/4x/5x)
- [x] Redirect auto-follow (up to 5 hops)

### Phase 5 — Gopher
- [x] `protocols/gopher/client.go`: TCP dial, send selector, read response
- [x] `protocols/gopher/parser.go`: menu → `[]RenderLine` (type 0, 1, 7, i, 3, h)
- [x] Search prompt for type 7 items

### Phase 6 — Finger
- [x] `protocols/finger/client.go`: TCP port 79, raw text → `[]RenderLine`

### Phase 7 — History & Bookmarks
- [x] `model/history.go`: back/forward stack
- [x] `model/bookmark.go`: load/save bookmarks file
- [x] History overlay (`H`), bookmarks overlay (`B`)

### Phase 8 — Config & Polish
- [x] `config/config.go`: load/save INI-style config (homepage, theme, low_color)
- [x] Help overlay (`?`) with full keymap reference
- [x] 80×24 minimum size guard (warning message, graceful resize)
- [x] UTF-8 safety (`strings.ToValidUTF8` on all raw protocol input)
- [x] Command-line URL argument (`./triplicate-tui <url>`)

---

## Test Targets

- Gemini capsule with self-signed cert → TOFU prompt appears
- Gemini cert fingerprint change → mismatch warning fires
- Gemini redirect (30/31) → auto-followed silently
- Gemini input request (status 10) → cmdbar prompt with server text
- Gopher menu depth 2+ → nested navigation works
- Gopher search (type 7) → prompt + re-fetch
- Finger lookup on a public test server
- Terminal resize mid-browse → re-wrap + redraw without crash
- 80×24 terminal → no layout breakage
- Sub-80×24 terminal → size warning shown
