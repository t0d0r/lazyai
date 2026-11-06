package tui

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// settings is what the S key saves for the next start (specs/tui.md,
// Saved settings). Names, not numbers, so the file survives reordering.
type settings struct {
	Layout        string `json:"layout,omitempty"`
	Tab           string `json:"tab,omitempty"`
	ScreenPercent int    `json:"screen_percent,omitempty"`
}

// SettingsPath is $XDG_CONFIG_HOME/lazyai/settings.json, by default
// ~/.config/lazyai/settings.json, or "" without a home directory.
func SettingsPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "lazyai", "settings.json")
}

// loadSettings reads the file. A missing file is no settings; a broken
// one is no settings and an error to show.
func loadSettings(path string) (settings, error) {
	var s settings
	if path == "" {
		return s, nil
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return settings{}, err
	}
	return s, nil
}

func saveSettings(path string, s settings) error {
	if path == "" {
		return errors.New("no settings file: home directory unknown")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// currentSettings is the layout as it is now.
func (m model) currentSettings() settings {
	return settings{Layout: m.mode.String(), Tab: tabTitles[m.tab], ScreenPercent: 100 - m.screenTopPct}
}

// applySettings starts the model in saved settings. Unknown names and
// out-of-range numbers are ignored, so an old or hand-edited file still
// starts lazyai.
func (m *model) applySettings(s settings) {
	for i, n := range layoutNames {
		if n == s.Layout {
			m.mode = layoutMode(i)
		}
	}
	for i, n := range tabTitles {
		if n == s.Tab {
			m.tab = i
		}
	}
	if p := 100 - s.ScreenPercent; s.ScreenPercent != 0 && p >= screenTopLow && p <= screenTopHigh {
		m.screenTopPct = p
	}
	if m.mode == layoutScreen && m.tab == tabScreen {
		m.tab = tabInfo // the Screen panel shows it
	}
}

// saveLayout writes the current settings and says where.
func (m *model) saveLayout() {
	if err := saveSettings(m.opts.SettingsFile, m.currentSettings()); err != nil {
		m.setMsg("save settings: "+err.Error(), true)
		return
	}
	s := m.currentSettings()
	m.setMsg("saved for the next start: "+s.Layout+" layout, "+s.Tab+" tab", false)
}
