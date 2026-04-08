package gemini

import (
	"strings"

	"triplicate-tui/model"
)

// ParseGemtext converts a text/gemini body into a slice of RenderLines.
func ParseGemtext(body []byte) []model.RenderLine {
	var lines []model.RenderLine
	inPre := false

	// Ensure valid UTF-8; replace invalid sequences with U+FFFD.
	text := strings.ToValidUTF8(string(body), "\uFFFD")
	for _, rawLine := range strings.Split(text, "\n") {
		line := strings.TrimRight(rawLine, "\r")

		// Toggle preformatted block on ``` lines.
		if strings.HasPrefix(line, "```") {
			inPre = !inPre
			continue
		}

		if inPre {
			lines = append(lines, model.RenderLine{Type: model.LinePre, Text: line})
			continue
		}

		switch {
		case strings.HasPrefix(line, "=>"):
			lines = append(lines, parseLink(line))
		case strings.HasPrefix(line, "### "):
			lines = append(lines, model.RenderLine{Type: model.LineH3, Text: line[4:]})
		case strings.HasPrefix(line, "##"):
			text := strings.TrimPrefix(line, "##")
			text = strings.TrimPrefix(text, " ")
			lines = append(lines, model.RenderLine{Type: model.LineH2, Text: text})
		case strings.HasPrefix(line, "#"):
			text := strings.TrimPrefix(line, "#")
			text = strings.TrimPrefix(text, " ")
			lines = append(lines, model.RenderLine{Type: model.LineH1, Text: text})
		case strings.HasPrefix(line, ">"):
			text := strings.TrimPrefix(line, ">")
			text = strings.TrimPrefix(text, " ")
			lines = append(lines, model.RenderLine{Type: model.LineQuote, Text: text})
		case line == "":
			lines = append(lines, model.RenderLine{Type: model.LineEmpty})
		default:
			lines = append(lines, model.RenderLine{Type: model.LineText, Text: line})
		}
	}
	return lines
}

// parseLink parses a Gemini link line: "=> URL [description]"
func parseLink(line string) model.RenderLine {
	rest := strings.TrimSpace(strings.TrimPrefix(line, "=>"))
	if rest == "" {
		return model.RenderLine{Type: model.LineText, Text: line}
	}
	// Split on whitespace: first token is URL, rest is description.
	fields := strings.Fields(rest)
	linkURL := fields[0]
	desc := ""
	if len(fields) > 1 {
		desc = strings.Join(fields[1:], " ")
	}
	return model.RenderLine{Type: model.LineLink, Text: desc, URL: linkURL}
}
