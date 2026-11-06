package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lazyai", "settings.json")
	m := inLayout(loaded(120, 30), layoutScreen)
	m.opts.SettingsFile = path
	m = press(m, "]", "+") // Processes tab, Screen 90%
	m = press(m, "S")
	if !strings.Contains(m.msg, "saved for the next start: screen layout, Processes tab") {
		t.Fatalf("S: message %q", m.msg)
	}
	if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("settings file: %v %v", fi, err)
	}

	n := newModel(Options{SettingsFile: path})
	if n.mode != layoutScreen || n.tab != tabProcesses || n.screenTopPct != screenTopLow {
		t.Errorf("restored: layout %v tab %d top %d%%", n.mode, n.tab, n.screenTopPct)
	}
	// The x menu offers it too.
	found := false
	for _, it := range m.menuItems() {
		found = found || it.key == saveKey
	}
	if !found {
		t.Error("x menu has no save item")
	}
}

func TestLoadSettingsTolerant(t *testing.T) {
	dir := t.TempDir()
	write := func(s string) string {
		p := filepath.Join(dir, "settings.json")
		if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	// Missing file: defaults, no message.
	if m := newModel(Options{SettingsFile: filepath.Join(dir, "none.json")}); m.mode != layoutSplit || m.msg != "" {
		t.Errorf("missing file: layout %v msg %q", m.mode, m.msg)
	}
	// Unknown names and out-of-range numbers are ignored.
	m := newModel(Options{SettingsFile: write(`{"layout":"nope","tab":"Nope","screen_percent":5}`)})
	if m.mode != layoutSplit || m.tab != tabInfo || m.screenTopPct != screenTopDefault {
		t.Errorf("bad values: layout %v tab %d top %d%%", m.mode, m.tab, m.screenTopPct)
	}
	// The screen layout never starts on the Screen tab.
	if m := newModel(Options{SettingsFile: write(`{"layout":"screen","tab":"Screen"}`)}); m.tab != tabInfo {
		t.Errorf("screen layout with Screen tab: tab %d", m.tab)
	}
	// A broken file starts with the defaults and says so.
	if m := newModel(Options{SettingsFile: write(`{`)}); m.mode != layoutSplit || !strings.Contains(m.msg, "settings ignored") {
		t.Errorf("broken file: layout %v msg %q", m.mode, m.msg)
	}
}
