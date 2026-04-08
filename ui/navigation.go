package ui

import (
	"fmt"
	"strings"

	"triplicate-tui/model"
	"triplicate-tui/protocols/gemini"
)

// pageFromResponse converts a Gemini Response (status 2x) into a Page.
func pageFromResponse(rawURL string, resp *gemini.Response) *model.Page {
	mimeType := "text/gemini"
	if resp.Meta != "" {
		parts := strings.SplitN(resp.Meta, ";", 2)
		mimeType = strings.ToLower(strings.TrimSpace(parts[0]))
	}

	var lines []model.RenderLine
	switch {
	case mimeType == "text/gemini":
		lines = gemini.ParseGemtext(resp.Body)
	case strings.HasPrefix(mimeType, "text/"):
		for _, raw := range strings.Split(string(resp.Body), "\n") {
			lines = append(lines, model.RenderLine{
				Type: model.LineText,
				Text: strings.TrimRight(raw, "\r"),
			})
		}
	default:
		lines = []model.RenderLine{
			{Type: model.LineH2, Text: "Cannot display"},
			{Type: model.LineEmpty},
			{Type: model.LineText, Text: "MIME type: " + mimeType},
			{Type: model.LineText, Text: "URL: " + rawURL},
		}
	}

	return &model.Page{URL: rawURL, Lines: lines}
}

// errorPage builds a content page from a Gemini error status.
func errorPage(status int, meta, rawURL string) *model.Page {
	heading := fmt.Sprintf("Gemini %d", status)
	return &model.Page{
		URL: rawURL,
		Lines: []model.RenderLine{
			{Type: model.LineH2, Text: heading},
			{Type: model.LineEmpty},
			{Type: model.LineText, Text: meta},
			{Type: model.LineEmpty},
			{Type: model.LineText, Text: rawURL},
		},
	}
}

// msgPage builds a simple informational page.
func msgPage(heading, body, rawURL string) *model.Page {
	return &model.Page{
		URL: rawURL,
		Lines: []model.RenderLine{
			{Type: model.LineH2, Text: heading},
			{Type: model.LineEmpty},
			{Type: model.LineText, Text: body},
		},
	}
}

// tofuChallengePage shows cert details for an unknown certificate.
func tofuChallengePage(info *gemini.CertInfo) *model.Page {
	expiry := info.NotAfter.Format("2006-01-02")
	return &model.Page{
		URL: "tofu://" + info.Host,
		Lines: []model.RenderLine{
			{Type: model.LineH2, Text: "Unknown Certificate"},
			{Type: model.LineEmpty},
			{Type: model.LineText, Text: "Host:     " + info.Host},
			{Type: model.LineText, Text: "Subject:  " + info.Subject},
			{Type: model.LineText, Text: "Expires:  " + expiry},
			{Type: model.LineEmpty},
			{Type: model.LinePre, Text: info.Fingerprint},
			{Type: model.LineEmpty},
			{Type: model.LineText, Text: "[t] trust once   [a] always trust   [r] reject"},
		},
	}
}

// tofuMismatchPage shows a warning when a previously trusted cert has changed.
func tofuMismatchPage(info *gemini.CertInfo) *model.Page {
	expiry := info.NotAfter.Format("2006-01-02")
	return &model.Page{
		URL: "tofu://" + info.Host,
		Lines: []model.RenderLine{
			{Type: model.LineH2, Text: "WARNING: Certificate Changed"},
			{Type: model.LineEmpty},
			{Type: model.LineText, Text: "The certificate for " + info.Host + " does not match"},
			{Type: model.LineText, Text: "the previously trusted fingerprint."},
			{Type: model.LineEmpty},
			{Type: model.LineText, Text: "This may indicate a server upgrade or a MITM attack."},
			{Type: model.LineEmpty},
			{Type: model.LineText, Text: "New fingerprint (expires " + expiry + "):"},
			{Type: model.LinePre, Text: info.Fingerprint},
			{Type: model.LineEmpty},
			{Type: model.LineText, Text: "[c] continue and update trust   [r] reject"},
		},
	}
}
