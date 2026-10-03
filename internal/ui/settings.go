package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func (a *App) settingsPath() string { return filepath.Join(a.Dir, "settings.json") }

func (a *App) loadSettings() {
	b, err := os.ReadFile(a.settingsPath())
	if err != nil {
		return
	}
	var s Settings
	if json.Unmarshal(b, &s) == nil {
		if s.Name != "" {
			a.Set.Name = s.Name
		}
		if s.Port != "" {
			a.Set.Port = s.Port
		}
		a.Set.LastIP = s.LastIP
		if s.Volume > 0 {
			a.Set.Volume = s.Volume
		}
	}
}

// SaveSettings writes the settings file (errors are ignored: read-only dirs are fine).
func (a *App) SaveSettings() {
	b, _ := json.MarshalIndent(a.Set, "", "  ")
	_ = os.WriteFile(a.settingsPath(), b, 0o644)
}
