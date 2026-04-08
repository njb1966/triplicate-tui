package gemini

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPort  = "1965"
	maxRedirects = 5
	dialTimeout  = 30 * time.Second
	ioTimeout    = 30 * time.Second
)

// Response is a parsed Gemini server response.
type Response struct {
	Status int
	Meta   string
	Body   []byte // non-nil only for 2x responses
}

// FetchOptions configures a Fetch call.
type FetchOptions struct {
	TOFU       *TOFU
	TOFUPrompt func(*CertInfo) TOFUDecision // called when cert is unknown or mismatched
}

// Fetch retrieves a Gemini URL, auto-following up to maxRedirects redirects.
func Fetch(rawURL string, opts FetchOptions) (*Response, error) {
	return fetchLoop(rawURL, opts, 0)
}

func fetchLoop(rawURL string, opts FetchOptions, depth int) (*Response, error) {
	if depth > maxRedirects {
		return nil, fmt.Errorf("too many redirects (max %d)", maxRedirects)
	}
	resp, err := doFetch(rawURL, opts)
	if err != nil {
		return nil, err
	}
	// Auto-follow permanent and temporary redirects.
	if resp.Status >= 30 && resp.Status <= 39 {
		target, err := resolveURL(rawURL, resp.Meta)
		if err != nil {
			return nil, fmt.Errorf("redirect %d: bad target URL: %w", resp.Status, err)
		}
		return fetchLoop(target, opts, depth+1)
	}
	return resp, nil
}

func doFetch(rawURL string, opts FetchOptions) (*Response, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}
	if u.Scheme != "gemini" {
		return nil, fmt.Errorf("not a gemini URL: %s", rawURL)
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = defaultPort
	}
	addr := net.JoinHostPort(host, port)

	tlsCfg := &tls.Config{
		InsecureSkipVerify: true, // TOFU handles trust manually
		ServerName:         host,
	}

	dialer := &net.Dialer{Timeout: dialTimeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsCfg)
	if err != nil {
		return nil, fmt.Errorf("connect %s: %w", addr, err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(ioTimeout))

	// TOFU certificate check.
	if opts.TOFU != nil {
		certs := conn.ConnectionState().PeerCertificates
		if len(certs) == 0 {
			return nil, fmt.Errorf("server sent no TLS certificate")
		}
		result, info := opts.TOFU.Check(host, certs[0])
		switch result {
		case TOFUKnownMatch:
			// All good.
		case TOFUUnknown, TOFUMismatch:
			if opts.TOFUPrompt == nil {
				// No prompt available: trust silently for this session.
				opts.TOFU.TrustSession(host, info.Fingerprint)
			} else {
				decision := opts.TOFUPrompt(info)
				switch decision {
				case TOFUReject:
					return nil, fmt.Errorf("certificate rejected for %s", host)
				case TOFUTrustAlways:
					if err := opts.TOFU.TrustPersist(host, info.Fingerprint); err != nil {
						// Non-fatal: trust in-session even if persist fails.
						opts.TOFU.TrustSession(host, info.Fingerprint)
					}
				case TOFUTrustOnce:
					opts.TOFU.TrustSession(host, info.Fingerprint)
				}
			}
		}
	}

	// Send request: full URL + CRLF.
	if _, err := fmt.Fprintf(conn, "%s\r\n", rawURL); err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	// Read response header.
	reader := bufio.NewReader(conn)
	header, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	header = strings.TrimRight(header, "\r\n")

	if len(header) < 2 {
		return nil, fmt.Errorf("invalid response header: %q", header)
	}
	status, err := strconv.Atoi(header[:2])
	if err != nil {
		return nil, fmt.Errorf("invalid status code: %q", header[:2])
	}
	meta := ""
	if len(header) > 3 {
		meta = header[3:]
	}

	// Read body only for success responses.
	var body []byte
	if status/10 == 2 {
		body, err = io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("read body: %w", err)
		}
	}

	return &Response{Status: status, Meta: meta, Body: body}, nil
}

// resolveURL resolves ref against base, returning an absolute URL string.
func resolveURL(base, ref string) (string, error) {
	b, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	r, err := b.Parse(ref)
	if err != nil {
		return "", err
	}
	return r.String(), nil
}
