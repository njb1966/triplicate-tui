package gopher

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"time"
)

const (
	defaultPort = "70"
	dialTimeout = 30 * time.Second
	ioTimeout   = 30 * time.Second
)

// Result holds the raw response from a Gopher server.
type Result struct {
	Type byte   // Gopher item type: '0', '1', '7', etc.
	Body []byte // raw response bytes
}

// Fetch retrieves a Gopher URL and returns the raw server response.
func Fetch(rawURL string) (*Result, error) {
	gopherType, host, port, selector, query, err := parseGopherURL(rawURL)
	if err != nil {
		return nil, err
	}

	addr := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return nil, fmt.Errorf("connect %s: %w", addr, err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(ioTimeout))

	// Build and send the request line.
	var req string
	if gopherType == '7' && query != "" {
		req = selector + "\t" + query + "\r\n"
	} else {
		req = selector + "\r\n"
	}
	if _, err := fmt.Fprint(conn, req); err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	body, err := io.ReadAll(conn)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	return &Result{Type: gopherType, Body: body}, nil
}

// parseGopherURL extracts the components of a gopher:// URL.
//
// URL form: gopher://host[:port]/[TYPE[SELECTOR[%09QUERY]]]
// Default type: '1' (directory). Default port: 70.
// A tab encoded as %09 in the path separates selector from search query.
func parseGopherURL(rawURL string) (gopherType byte, host, port, selector, query string, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return 0, "", "", "", "", fmt.Errorf("parse URL: %w", err)
	}
	if u.Scheme != "gopher" {
		return 0, "", "", "", "", fmt.Errorf("not a gopher URL: %s", rawURL)
	}

	host = u.Hostname()
	port = u.Port()
	if port == "" {
		port = defaultPort
	}

	// u.Path is already percent-decoded, so %09 becomes '\t'.
	path := u.Path
	gopherType = '1'
	selector = ""

	if len(path) >= 2 {
		gopherType = path[1]
		rest := path[2:]
		if idx := strings.IndexByte(rest, '\t'); idx >= 0 {
			selector = rest[:idx]
			query = rest[idx+1:]
		} else {
			selector = rest
		}
	}
	return gopherType, host, port, selector, query, nil
}

// NeedsQuery returns true if rawURL is a type-7 search item with no query yet.
func NeedsQuery(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || len(u.Path) < 2 {
		return false
	}
	if u.Path[1] != '7' {
		return false
	}
	// Decoded path: if it contains a tab, the query is already present.
	return !strings.ContainsRune(u.Path, '\t')
}

// AppendQuery appends a search query to a type-7 gopher URL.
func AppendQuery(baseURL, query string) string {
	// Use %09 (URL-encoded tab) as the separator per RFC 4266.
	return baseURL + "%09" + url.PathEscape(query)
}
