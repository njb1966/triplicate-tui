# CLAUDE.md — triplicate-tui

## Project Overview

**triplicate-tui** is a keyboard-driven TUI browser for Gemini, Gopher, and Finger protocols.

- **Design model:** Lynx-inspired; minimal, 80×24-safe, three-region layout
- **Language:** Go
- **TUI library:** `github.com/gdamore/tcell/v2`
- **TLS:** Go stdlib `crypto/tls` (for Gemini)
- **Cert policy:** TOFU (trust-on-first-use) — fingerprint store at `~/.config/triplicate/known_hosts`

---

## Directory Structure

```
triplicate-tui/
├── main.go                   # Entry point
├── go.mod
├── go.sum
├── Makefile
├── ui/
│   ├── app.go                # App struct, main event loop, screen init
│   ├── layout.go             # 3-region layout: statusbar, content, cmdbar
│   ├── content.go            # Scrollable content pane rendering
│   ├── statusbar.go          # Top bar: protocol, host, TLS/TOFU indicator
│   ├── cmdbar.go             # Bottom bar: hints + prompt input mode
│   ├── keys.go               # Keymap definitions and dispatch
│   └── theme.go              # Color pairs: NORMAL, STATUSBAR, LINK, HEADER, QUOTE
├── protocols/
│   ├── gemini/
│   │   ├── client.go         # TLS dial, send request, read response
│   │   ├── parser.go         # text/gemini → AST (RenderLine slice)
│   │   └── tofu.go           # TOFU: known_hosts read/write/verify/prompt
│   ├── gopher/
│   │   ├── client.go         # TCP dial, send selector, read response
│   │   └── parser.go         # Gopher menu parsing → RenderLine slice
│   └── finger/
│       └── client.go         # TCP port 79, raw text → RenderLine slice
├── model/
│   ├── page.go               # Page, RenderLine, LineType
│   ├── history.go            # History stack (back/forward)
│   └── bookmark.go           # Bookmarks file (~/.config/triplicate/bookmarks)
└── config/
    └── config.go             # Load/save ~/.config/triplicate/config
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
)

type RenderLine struct {
    Type  LineType
    Text  string
    URL   string        // non-empty for links and gopher items
    Attrs tcell.AttrMask
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

---

## Color Pairs

| Name          | Foreground | Background | Attrs     |
|---------------|------------|------------|-----------|
| PairNormal    | Default    | Default    | —         |
| PairStatusbar | Black      | Blue       | —         |
| PairLink      | Cyan       | Default    | Underline |
| PairHeader    | White      | Default    | Bold      |
| PairQuote     | Yellow     | Default    | Dim       |
| PairCmdbar    | Black      | DarkGray   | —         |

Low-color fallback: `Bold` for headers, `Underline` for links, `Reverse` for statusbar/cmdbar.

---

## Gemini Protocol

- URL scheme: `gemini://host[:port]/path[?query]`
- TLS via `crypto/tls` with `InsecureSkipVerify: true` — TOFU handles trust manually

### TOFU Flow

1. Compute SHA-256 fingerprint of leaf cert on connect
2. Look up `host` in `~/.config/triplicate/known_hosts`
3. **Unknown host:** show CN, expiry, fingerprint → prompt `[t]rust once / [a]lways / [r]eject`
4. **Known + match:** connect silently, show `[TOFU OK]` in statusbar
5. **Known + MISMATCH:** warn loudly, require explicit `[o]verride` to continue

`known_hosts` format: one entry per line: `hostname sha256:FINGERPRINT`

### Status Codes

| Code | Handling |
|------|----------|
| `10/11` | Input — open cmdbar prompt with server's prompt text, resend with query |
| `20` | Render body (default MIME: `text/gemini`) |
| `30/31` | Redirect — prompt `Follow redirect to <URL>? [y/N]` |
| `40-49` | Client error — display `Gemini 4x: <message>` in content pane |
| `50-59` | Server error — display `Gemini 5x: <message>` in content pane |

---

## Gopher Protocol

- URL scheme: `gopher://host[:port]/[type][selector]`
- TCP only (no TLS), default port 70
- Item types to handle: `0` (text file), `1` (menu), `7` (search)
- Unsupported types: show inline warning, don't crash
- Search (type `7`): open cmdbar prompt, send `selector\tquery\r\n`
- Render menus as selectable link list (same Tab/Enter navigation as Gemini)

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
./triplicate-tui gemini://gemini.circumlunar.space/
```

---

## Implementation Phases

### Phase 1 — Scaffold
- [ ] `go mod init triplicate-tui`
- [ ] `go get github.com/gdamore/tcell/v2`
- [ ] Create directory structure
- [ ] `main.go`: init tcell screen, run app loop, handle `q` to quit
- [ ] Write `Makefile` (`build`, `run`, `clean`)

### Phase 2 — Core UI
- [ ] `ui/layout.go`: 3-region layout, terminal resize handling
- [ ] `ui/statusbar.go`: protocol, host, TOFU status
- [ ] `ui/cmdbar.go`: hint bar + prompt input mode
- [ ] `ui/theme.go`: color pairs + low-color fallback
- [ ] `ui/keys.go`: global keymap dispatch
- [ ] `ui/app.go`: App struct, event loop

### Phase 3 — Content Model & Rendering
- [ ] `model/page.go`: Page, RenderLine, LineType types
- [ ] `ui/content.go`: scroll model (`scroll_offset`, `cursor_index`, `viewport_height`)
- [ ] Soft-wrap on page load; re-wrap on resize
- [ ] Link cursor highlight (Tab navigation through RenderLines where URL != "")
- [ ] Page search (`/`, `n`, `N`)

### Phase 4 — Gemini
- [ ] `protocols/gemini/client.go`: TLS dial, send request, read response header + body
- [ ] `protocols/gemini/tofu.go`: fingerprint compute, known_hosts lookup, interactive prompt
- [ ] `protocols/gemini/parser.go`: text/gemini → `[]RenderLine`
- [ ] Handle all status code families (10/11/20/3x/4x/5x)
- [ ] Redirect prompt (non-auto-follow)

### Phase 5 — Gopher
- [ ] `protocols/gopher/client.go`: TCP dial, send selector, read response
- [ ] `protocols/gopher/parser.go`: menu → `[]RenderLine` (type 0, 1, 7)
- [ ] Search prompt for type 7 items

### Phase 6 — Finger
- [ ] `protocols/finger/client.go`: TCP port 79, raw text → `[]RenderLine`

### Phase 7 — History & Bookmarks
- [ ] `model/history.go`: back/forward stack
- [ ] `model/bookmark.go`: load/save bookmarks file
- [ ] History overlay (`H`), bookmarks overlay (`B`)

### Phase 8 — Config & Polish
- [ ] `config/config.go`: load/save INI-style config
- [ ] Help overlay (`?`)
- [ ] 80×24 minimum size guard (warn + graceful resize)
- [ ] UTF-8 safety (skip/replace invalid bytes, no panics)

---

## Test Targets

- Gemini capsule with self-signed cert → TOFU prompt appears
- Gemini cert fingerprint change → mismatch warning fires
- Gemini redirect (30/31) → prompt before following
- Gemini input request (status 10) → cmdbar prompt with server text
- Gopher menu depth 2+ → nested navigation works
- Gopher search (type 7) → prompt + re-fetch
- Finger lookup on a public test server
- Terminal resize mid-browse → re-wrap + redraw without crash
- 80×24 terminal → no layout breakage
