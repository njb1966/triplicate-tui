package gopher

import (
	"fmt"
	"strings"

	"triplicate-tui/model"
)

// ParseMenu converts a Gopher menu (type 1) response into RenderLines.
func ParseMenu(body []byte) []model.RenderLine {
	var lines []model.RenderLine
	for _, raw := range strings.Split(strings.ToValidUTF8(string(body), "\uFFFD"), "\n") {
		line := strings.TrimRight(raw, "\r")

		// '.' alone terminates the menu.
		if line == "." || line == "" {
			continue
		}

		if len(line) < 1 {
			continue
		}

		itemType := line[0]
		rest := line[1:]

		// Split into up to 4 fields: display, selector, host, port.
		parts := strings.SplitN(rest, "\t", 4)
		display := parts[0]
		selector := ""
		host := ""
		port := ""
		if len(parts) > 1 {
			selector = parts[1]
		}
		if len(parts) > 2 {
			host = parts[2]
		}
		if len(parts) > 3 {
			port = strings.TrimSpace(parts[3])
		}

		switch itemType {
		case 'i':
			// Informational — not selectable.
			lines = append(lines, model.RenderLine{Type: model.LineText, Text: display})

		case '3':
			// Error item.
			lines = append(lines, model.RenderLine{
				Type: model.LineText,
				Text: "[error] " + display,
			})

		default:
			// Navigable item — build URL and emit as GopherItem.
			linkURL := buildURL(host, port, itemType, selector)
			if linkURL == "" {
				// Can't navigate (null host), show as plain text.
				lines = append(lines, model.RenderLine{Type: model.LineText, Text: display})
				continue
			}
			label := typePrefix(itemType) + display
			lines = append(lines, model.RenderLine{
				Type: model.LineGopherItem,
				Text: label,
				URL:  linkURL,
			})
		}
	}
	return lines
}

// ParseText converts a Gopher text file (type 0) response into RenderLines.
func ParseText(body []byte) []model.RenderLine {
	var lines []model.RenderLine
	for _, raw := range strings.Split(strings.ToValidUTF8(string(body), "\uFFFD"), "\n") {
		line := strings.TrimRight(raw, "\r")
		if line == "." {
			break
		}
		lines = append(lines, model.RenderLine{Type: model.LineText, Text: line})
	}
	return lines
}

// buildURL constructs a gopher:// URL for a menu item.
// Returns "" if the host is a placeholder (null, error, etc.).
func buildURL(host, port string, itemType byte, selector string) string {
	if host == "" || host == "(null)" || host == "error.host" || host == "null" {
		return ""
	}

	// Type 'h' with URL: selector — the selector is an external URL.
	if itemType == 'h' && strings.HasPrefix(selector, "URL:") {
		return strings.TrimPrefix(selector, "URL:")
	}

	addr := host
	if port != "" && port != "70" && port != "0" {
		addr = fmt.Sprintf("%s:%s", host, port)
	}
	return fmt.Sprintf("gopher://%s/%c%s", addr, itemType, selector)
}

// typePrefix returns a short display prefix for non-info item types.
func typePrefix(t byte) string {
	switch t {
	case '1':
		return ""
	case '0':
		return ""
	case '7':
		return "[?] "
	case '4', '5', '6', '9', 'g', 'I', 's':
		return "[bin] "
	case 'h':
		return "[www] "
	default:
		return ""
	}
}
