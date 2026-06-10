package ui

import (
	"encoding/json"

	"fyne.io/fyne/v2"
)

const (
	prefKeyPaths      = "data_folders"
	prefKeyThresholds = "be_thresholds"
)

func loadPaths(p fyne.Preferences) []string {
	return p.StringList(prefKeyPaths)
}

func savePaths(p fyne.Preferences, paths []string) {
	p.SetStringList(prefKeyPaths, paths)
}

func loadThresholds(p fyne.Preferences) map[string]float64 {
	raw := p.StringWithFallback(prefKeyThresholds, "{}")
	out := map[string]float64{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]float64{}
	}
	return out
}

func saveThresholds(p fyne.Preferences, m map[string]float64) {
	b, err := json.Marshal(m)
	if err != nil {
		return
	}
	p.SetString(prefKeyThresholds, string(b))
}
