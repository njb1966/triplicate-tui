# triplicate-tui

A keyboard-driven TUI browser for **Gemini**, **Gopher**, and **Finger** protocols.

Designed after Lynx. Minimal, fast, 80×24-safe.

---

## Features

- **Gemini** — full text/gemini rendering with TOFU certificate trust
- **Gopher** — menu navigation, text files, type-7 search
- **Finger** — user lookups over TCP port 79
- Scrollable, soft-wrapped content pane
- Tab/Shift-Tab link navigation, Enter to follow
- In-page search (`/`, `n`, `N`)
- History stack (`b`/`f`, `H` overlay)
- Bookmarks (`m` to save, `B` overlay)
- Help overlay (`?`)
- User config at `~/.config/triplicate/config`

---

## Requirements

- Go 1.21+
- A terminal emulator (80×24 minimum)

---

## Build

```bash
git clone https://github.com/njb1966/triplicate-tui
cd triplicate-tui
make build
```

Or manually:

```bash
go build -o triplicate-tui .
```

---

## Usage

```bash
# Start at the configured homepage
./triplicate-tui

# Navigate directly to a URL
./triplicate-tui gemini://gemini.circumlunar.space/
./triplicate-tui gopher://gopher.floodgap.com/
./triplicate-tui finger://sdf.org
```

---

## Key Bindings

| Key | Action |
|-----|--------|
| `j` / `↓` | Scroll down |
| `k` / `↑` | Scroll up |
| `Ctrl+D` / `PgDn` | Half page down |
| `Ctrl+U` / `PgUp` | Half page up |
| `g` / `G` | Top / bottom |
| `Tab` / `Shift+Tab` | Next / previous link |
| `Enter` | Follow selected link |
| `o` | Open URL prompt |
| `/` | Search in page |
| `n` / `N` | Next / previous match |
| `b` / `f` | Back / forward |
| `H` | History overlay |
| `m` | Bookmark current page |
| `B` | Bookmarks overlay |
| `?` | Help |
| `q` / `Esc` | Quit |

---

## Configuration

Create `~/.config/triplicate/config`:

```ini
homepage  = gemini://gemini.circumlunar.space/
theme     = dark
low_color = false
```

| Key | Default | Description |
|-----|---------|-------------|
| `homepage` | `gemini://gemini.circumlunar.space/` | URL loaded on startup |
| `theme` | `dark` | `dark` or `light` (reserved for future use) |
| `low_color` | `false` | Force attribute-only rendering (bold/underline/reverse instead of colors) |

---

## Gemini & TOFU

Gemini uses TLS. On first visit to a host, triplicate-tui presents the certificate details and asks:

```
[t] trust once   [a] always trust   [r] reject
```

Trusted fingerprints are stored in `~/.config/triplicate/known_hosts`.  
If a previously-trusted certificate changes, you will be warned before connecting.

---

## Persistent State

| File | Purpose |
|------|---------|
| `~/.config/triplicate/config` | User settings |
| `~/.config/triplicate/known_hosts` | TOFU certificate fingerprints |
| `~/.config/triplicate/bookmarks` | Saved URLs (`url [title]` per line) |

---

## Troubleshooting

**Text in the command bar is hard to read or invisible**

Add `low_color = true` to your config file. This switches the UI to
attribute-only rendering (reverse video for the command bar) which works
reliably on any terminal color scheme.

```ini
low_color = true
```

**Terminal too small message on startup**

triplicate-tui requires at least 80×24. Resize your terminal window or
reduce the font size.

**Gemini certificate warning on every visit**

This means the server's certificate changed since you last visited. Review
the new fingerprint shown on screen and press `[c]` to update your trust
store, or `[r]` to cancel.

---

## License

MIT
