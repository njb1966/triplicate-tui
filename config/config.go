package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Dir returns ~/.config/triplicate, creating it if absent.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "triplicate")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// KnownHostsPath returns the path to the TOFU known_hosts file.
func KnownHostsPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "known_hosts"), nil
}

// BookmarksPath returns the path to the bookmarks file.
func BookmarksPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "bookmarks"), nil
}

// ConfigPath returns the path to the user config file.
func ConfigPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config"), nil
}

// Config holds user-configurable settings.
type Config struct {
	Homepage string
	Theme    string // "dark" or "light"
	LowColor bool
}

// DefaultConfig returns built-in sane defaults.
func DefaultConfig() Config {
	return Config{
		Homepage: "gemini://gemini.circumlunar.space/",
		Theme:    "dark",
		LowColor: false,
	}
}

// LoadConfig reads ~/.config/triplicate/config and returns a Config.
// Missing keys are filled from DefaultConfig. A missing file is not an error.
func LoadConfig() (Config, error) {
	cfg := DefaultConfig()
	path, err := ConfigPath()
	if err != nil {
		return cfg, err
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "homepage":
			if val != "" {
				cfg.Homepage = val
			}
		case "theme":
			if val == "dark" || val == "light" {
				cfg.Theme = val
			}
		case "low_color":
			cfg.LowColor = val == "true" || val == "1" || val == "yes"
		}
	}
	return cfg, sc.Err()
}

// SaveConfig writes the Config to disk.
func SaveConfig(cfg Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	fmt.Fprintf(w, "homepage = %s\n", cfg.Homepage)
	fmt.Fprintf(w, "theme = %s\n", cfg.Theme)
	lowColor := "false"
	if cfg.LowColor {
		lowColor = "true"
	}
	fmt.Fprintf(w, "low_color = %s\n", lowColor)
	return w.Flush()
}
