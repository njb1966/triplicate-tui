package gemini

import (
	"bufio"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// TOFUDecision is the user's response to a certificate challenge.
type TOFUDecision int

const (
	TOFUTrustOnce   TOFUDecision = iota // trust this session only
	TOFUTrustAlways                     // trust and persist to known_hosts
	TOFUReject                          // abort the connection
)

// TOFUResult is the outcome of checking a cert against known_hosts.
type TOFUResult int

const (
	TOFUKnownMatch TOFUResult = iota // cert matches stored fingerprint
	TOFUUnknown                      // host not yet seen
	TOFUMismatch                     // host known but fingerprint differs
)

// CertInfo carries the details shown to the user during a TOFU challenge.
type CertInfo struct {
	Host        string
	Fingerprint string    // "sha256:HEXHEX..."
	Subject     string    // cert CN
	NotAfter    time.Time // cert expiry
	IsMismatch  bool      // true when a previous fingerprint exists but differs
}

// TOFU manages the known_hosts store in memory with optional persistence.
type TOFU struct {
	mu    sync.RWMutex
	path  string
	hosts map[string]string // hostname → fingerprint
}

// NewTOFU loads (or creates) the known_hosts file at path.
func NewTOFU(path string) (*TOFU, error) {
	t := &TOFU{path: path, hosts: make(map[string]string)}
	if err := t.load(); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *TOFU) load() error {
	f, err := os.Open(t.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		parts := strings.Fields(sc.Text())
		if len(parts) == 2 {
			t.hosts[parts[0]] = parts[1]
		}
	}
	return sc.Err()
}

// Check compares the cert against the stored fingerprint for host.
func (t *TOFU) Check(host string, cert *x509.Certificate) (TOFUResult, *CertInfo) {
	fp := certFingerprint(cert)
	info := &CertInfo{
		Host:        host,
		Fingerprint: fp,
		Subject:     cert.Subject.CommonName,
		NotAfter:    cert.NotAfter,
	}
	t.mu.RLock()
	known, ok := t.hosts[host]
	t.mu.RUnlock()
	if !ok {
		return TOFUUnknown, info
	}
	if known == fp {
		return TOFUKnownMatch, info
	}
	info.IsMismatch = true
	return TOFUMismatch, info
}

// TrustSession stores the fingerprint in memory only (this session).
func (t *TOFU) TrustSession(host, fp string) {
	t.mu.Lock()
	t.hosts[host] = fp
	t.mu.Unlock()
}

// TrustPersist stores the fingerprint in memory and writes it to disk.
func (t *TOFU) TrustPersist(host, fp string) error {
	t.TrustSession(host, fp)
	if err := os.MkdirAll(filepath.Dir(t.path), 0700); err != nil {
		return err
	}
	return t.save()
}

func (t *TOFU) save() error {
	f, err := os.Create(t.path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	t.mu.RLock()
	defer t.mu.RUnlock()
	for host, fp := range t.hosts {
		fmt.Fprintf(w, "%s %s\n", host, fp)
	}
	return w.Flush()
}

func certFingerprint(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.Raw)
	return fmt.Sprintf("sha256:%x", sum)
}
