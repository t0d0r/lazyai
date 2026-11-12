package tui

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"lazyai/internal/agent"
)

// importantMax is how many important marks the file keeps; the oldest
// go first, so marks of agents long gone do not pile up.
const importantMax = 200

// ImportantPath is the file of important marks:
// $XDG_STATE_HOME/lazyai/important, with ~/.local/state as the default
// state directory. "" when there is no home.
func ImportantPath() string {
	dir := stateDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "lazyai", "important")
}

// stateDir is $XDG_STATE_HOME, by default ~/.local/state, or "" without a
// home directory.
func stateDir() string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "state")
}

// markID is what an important mark is saved under, chosen to outlive the
// process where it can (specs/tui.md, Important mark):
//
//   - tmux:SESSION:KIND:ROLE for an agent in tmux, so the mark stays when
//     the agent is restarted in the same session;
//   - claude:SESSIONID for a Claude agent outside tmux;
//   - proc:KEY:START otherwise, which lasts as long as the process.
func markID(a agent.Agent) string {
	switch {
	case a.TmuxSession != "":
		return "tmux:" + a.TmuxSession + ":" + string(a.Kind) + ":" + string(a.Role)
	case a.SessionID != "":
		return "claude:" + a.SessionID
	}
	return "proc:" + a.Key() + ":" + strconv.FormatInt(a.Start.Unix(), 10)
}

// RenameMarks moves the important marks of tmux session from to to in
// the marks file at path, for a session renamed outside the TUI.
func RenameMarks(path, from, to string) error {
	marks := loadMarks(path)
	moved := moveMarks(marks, from, to)
	if slices.Equal(moved, marks) {
		return nil
	}
	return saveMarks(path, moved)
}

// moveMarks moves the marks of tmux session from to session to.
func moveMarks(marks []string, from, to string) []string {
	out := slices.Clone(marks)
	for i, id := range out {
		if rest, ok := strings.CutPrefix(id, "tmux:"+from+":"); ok {
			out[i] = "tmux:" + to + ":" + rest
		}
	}
	return out
}

// loadMarks reads the marks file, oldest first. A missing or unreadable
// file is no marks.
func loadMarks(path string) []string {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var marks []string
	for _, line := range strings.Split(string(data), "\n") {
		if line = strings.TrimSpace(line); line != "" && !slices.Contains(marks, line) {
			marks = append(marks, line)
		}
	}
	return marks[max(len(marks)-importantMax, 0):]
}

// saveMarks writes the marks readable by the user only: tmux session
// names and Claude session ids say what the user works on.
func saveMarks(path string, marks []string) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	text := strings.Join(marks, "\n")
	if text != "" {
		text += "\n"
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(text), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// toggleMark adds id at the newest end, or removes it, keeping at most
// importantMax marks. It reports whether id is marked now.
func toggleMark(marks []string, id string) ([]string, bool) {
	if i := slices.Index(marks, id); i >= 0 {
		return slices.Delete(slices.Clone(marks), i, i+1), false
	}
	marks = append(slices.Clone(marks), id)
	return marks[max(len(marks)-importantMax, 0):], true
}
