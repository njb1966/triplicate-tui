package ui

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"triplicate-tui/protocols/gemini"
)

// eventTime is a helper that satisfies tcell.Event by embedding a time.
type eventTime struct{ t time.Time }

func (e *eventTime) When() time.Time { return e.t }
func now() eventTime                 { return eventTime{t: time.Now()} }

// EventTOFUChallenge is posted by the navigation goroutine when a certificate
// needs user approval. The goroutine blocks on Decision until the App responds.
type EventTOFUChallenge struct {
	eventTime
	Info     *gemini.CertInfo
	Decision chan<- gemini.TOFUDecision
}

// EventFetchResult is posted when a Gemini fetch completes (any status code).
type EventFetchResult struct {
	eventTime
	URL  string
	Resp *gemini.Response
}

// EventLoadError is posted when a network or protocol error prevents a fetch.
type EventLoadError struct {
	eventTime
	URL string
	Msg string
}

// EventGopherResult is posted when a Gopher fetch completes.
type EventGopherResult struct {
	eventTime
	URL  string
	Type byte
	Body []byte
}

// EventFingerResult is posted when a Finger fetch completes.
type EventFingerResult struct {
	eventTime
	URL  string
	Body []byte
}

// EventQuit is posted by Shutdown() to break out of PollEvent cleanly.
type EventQuit struct{ eventTime }

// Ensure our custom types satisfy tcell.Event at compile time.
var _ tcell.Event = (*EventTOFUChallenge)(nil)
var _ tcell.Event = (*EventFetchResult)(nil)
var _ tcell.Event = (*EventLoadError)(nil)
var _ tcell.Event = (*EventGopherResult)(nil)
var _ tcell.Event = (*EventFingerResult)(nil)
var _ tcell.Event = (*EventQuit)(nil)
