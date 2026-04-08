package finger

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"time"
)

const (
	defaultPort = "79"
	dialTimeout = 30 * time.Second
	ioTimeout   = 30 * time.Second
)

// Result holds the raw output from a Finger server.
type Result struct {
	Body []byte
}

// Fetch connects to a finger:// URL and returns the server's response.
//
// URL forms:
//
//	finger://user@host  — query a specific user
//	finger://host       — query the host (typically lists users)
func Fetch(rawURL string) (*Result, error) {
	user, host, err := parseFingerURL(rawURL)
	if err != nil {
		return nil, err
	}

	addr := net.JoinHostPort(host, defaultPort)
	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return nil, fmt.Errorf("connect %s: %w", addr, err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(ioTimeout))

	// Send query: "user\r\n" or "\r\n" for host-only.
	if _, err := fmt.Fprintf(conn, "%s\r\n", user); err != nil {
		return nil, fmt.Errorf("send query: %w", err)
	}

	body, err := io.ReadAll(conn)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	return &Result{Body: body}, nil
}

// parseFingerURL extracts user and host from a finger:// URL.
func parseFingerURL(rawURL string) (user, host string, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("parse URL: %w", err)
	}
	if u.Scheme != "finger" {
		return "", "", fmt.Errorf("not a finger URL: %s", rawURL)
	}
	host = u.Hostname()
	if host == "" {
		return "", "", fmt.Errorf("missing host in finger URL: %s", rawURL)
	}
	if u.User != nil {
		user = u.User.Username()
	}
	return user, host, nil
}
