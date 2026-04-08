package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"triplicate-tui/ui"
)

func main() {
	// Optional: first argument is a URL to navigate to on startup.
	startURL := ""
	if len(os.Args) > 1 {
		startURL = os.Args[1]
	}

	app, err := ui.NewApp(startURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "triplicate-tui: %v\n", err)
		os.Exit(1)
	}

	// Restore terminal on SIGINT / SIGTERM.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		app.Shutdown()
	}()

	app.Run()
}
