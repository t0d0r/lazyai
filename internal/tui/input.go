package tui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"lazyai/internal/agent"
)

// historyMax is how many launch lines the history keeps.
const historyMax = 200

// HistoryPath is the launch history file: $XDG_STATE_HOME/lazyai/launch_history,
// with ~/.local/state as the default state directory. "" when there is no home.
func HistoryPath() string {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "lazyai", "launch_history")
}

// loadHistory reads the history file, oldest line first. A missing or
// unreadable file is an empty history.
func loadHistory(path string) []string {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var h []string
	for _, line := range strings.Split(string(data), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			h = addHistory(h, line)
		}
	}
	return h
}

// addHistory appends line, dropping an older copy of it and the oldest
// lines beyond historyMax.
func addHistory(h []string, line string) []string {
	line = strings.Join(strings.Fields(line), " ")
	h = slices.DeleteFunc(slices.Clone(h), func(s string) bool { return s == line })
	h = append(h, line)
	return h[max(len(h)-historyMax, 0):]
}

// saveHistory writes the history readable by the user only; launch lines
// may carry agent arguments.
func saveHistory(path string, h []string) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(h, "\n")+"\n"), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// historyMove steps through the launch history: -1 older, +1 newer. Past
// the newest line it returns to the text that was being typed.
func (m *model) historyMove(d int) {
	pos := m.histPos + d
	if pos < 0 || pos > len(m.hist) {
		return
	}
	if m.histPos == len(m.hist) {
		m.draft = m.buf
	}
	m.histPos = pos
	if pos == len(m.hist) {
		m.buf = m.draft
	} else {
		m.buf = []rune(m.hist[pos])
	}
}

// completeLaunch extends the last word of a "KIND [DIR] [NAME] [-- ARGS]"
// line: a kind as the first word, a directory as the second. It returns the
// new line and, when several match, their names to show.
func completeLaunch(line, baseDir string) (string, []string) {
	if strings.Contains(" "+line+" ", " -- ") {
		return line, nil
	}
	words := strings.Fields(line)
	cur := ""
	if len(words) > 0 && !strings.HasSuffix(line, " ") {
		cur, words = words[len(words)-1], words[:len(words)-1]
	}
	var cands, names []string
	sep := ""
	switch len(words) {
	case 0:
		for _, k := range agent.LaunchKinds() {
			if strings.HasPrefix(string(k), cur) {
				cands = append(cands, string(k))
			}
		}
		names, sep = cands, " "
	case 1:
		cands, names = completeDir(cur, baseDir)
	}
	if len(cands) == 0 {
		return line, nil
	}
	out := line[:len(line)-len(cur)] + commonPrefix(cands)
	if len(cands) == 1 {
		return out + sep, nil
	}
	return out, names
}

// completeDir lists the directories that word may name, as full words
// ending in "/" and as bare names. Hidden ones need a leading "." typed.
func completeDir(word, baseDir string) (words, names []string) {
	if word == "~" {
		return []string{"~/"}, []string{"~/"}
	}
	i := strings.LastIndex(word, "/") + 1
	parent, prefix := word[:i], word[i:]
	dir := baseDir
	if parent != "" {
		dir = expandDir(parent, baseDir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, prefix) || (strings.HasPrefix(name, ".") && !strings.HasPrefix(prefix, ".")) {
			continue
		}
		if fi, err := os.Stat(filepath.Join(dir, name)); err != nil || !fi.IsDir() {
			continue
		}
		words = append(words, parent+name+"/")
		names = append(names, name+"/")
	}
	return words, names
}

func commonPrefix(ss []string) string {
	p := []rune(ss[0])
	for _, s := range ss[1:] {
		r := []rune(s)
		n := 0
		for n < len(p) && n < len(r) && p[n] == r[n] {
			n++
		}
		p = p[:n]
	}
	return string(p)
}
