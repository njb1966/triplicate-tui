package model

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Bookmark is a saved URL with an optional title.
type Bookmark struct {
	URL   string
	Title string
}

// Bookmarks manages the bookmarks file.
type Bookmarks struct {
	path  string
	items []Bookmark
}

// NewBookmarks loads (or creates) the bookmarks file at path.
func NewBookmarks(path string) (*Bookmarks, error) {
	b := &Bookmarks{path: path}
	return b, b.load()
}

func (b *Bookmarks) load() error {
	f, err := os.Open(b.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		bm := Bookmark{URL: parts[0]}
		if len(parts) == 2 {
			bm.Title = strings.TrimSpace(parts[1])
		}
		b.items = append(b.items, bm)
	}
	return sc.Err()
}

// Add appends a bookmark if the URL is not already present, then saves.
func (b *Bookmarks) Add(rawURL, title string) error {
	if rawURL == "" {
		return nil
	}
	for _, bm := range b.items {
		if bm.URL == rawURL {
			return nil // already bookmarked
		}
	}
	b.items = append(b.items, Bookmark{URL: rawURL, Title: title})
	return b.save()
}

func (b *Bookmarks) save() error {
	f, err := os.Create(b.path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for _, bm := range b.items {
		if bm.Title != "" {
			fmt.Fprintf(w, "%s %s\n", bm.URL, bm.Title)
		} else {
			fmt.Fprintf(w, "%s\n", bm.URL)
		}
	}
	return w.Flush()
}

// Items returns all bookmarks in order.
func (b *Bookmarks) Items() []Bookmark { return b.items }
