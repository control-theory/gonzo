package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AppState holds persistent application state (separate from user config).
type AppState struct {
	LastSeenVersion string `json:"last_seen_version"`
}

// stateDir returns ~/.config/gonzo (same base as config).
func stateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "gonzo"), nil
}

// statePath returns the full path to the state file.
func statePath() (string, error) {
	dir, err := stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}

// Load reads the app state from disk. Returns zero value if file is missing or unreadable.
func Load() AppState {
	p, err := statePath()
	if err != nil {
		return AppState{}
	}

	data, err := os.ReadFile(p)
	if err != nil {
		return AppState{}
	}

	var s AppState
	if err := json.Unmarshal(data, &s); err != nil {
		return AppState{}
	}
	return s
}

// Save writes the app state to disk, creating the directory if needed.
func Save(s AppState) error {
	p, err := statePath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(p, data, 0o644)
}
