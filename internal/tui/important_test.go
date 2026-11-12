package tui

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"lazyai/internal/agent"
)

func TestMarkID(t *testing.T) {
	start := time.Unix(1700000000, 0)
	a := agent.Agent{Proc: agent.Proc{PID: 42, Start: start}, Kind: agent.KindClaude, Role: agent.RoleInteractive}
	if got := markID(a); got != "proc:42:1700000000" {
		t.Errorf("process: %q", got)
	}
	a.SessionID = "0b1c"
	if got := markID(a); got != "claude:0b1c" {
		t.Errorf("claude: %q", got)
	}
	a.TmuxSession = "lazyai-web"
	if got := markID(a); got != "tmux:lazyai-web:claude:interactive" {
		t.Errorf("tmux: %q", got)
	}
	// The same session after the agent restarts keeps the mark.
	b := a
	b.PID, b.Start, b.SessionID = 99, start.Add(time.Hour), "other"
	if markID(a) != markID(b) {
		t.Error("restart in the same session changes the mark")
	}
}

func TestMarksFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "important")
	if got := loadMarks(path); got != nil {
		t.Errorf("missing file: %q", got)
	}
	marks, on := toggleMark(nil, "tmux:a:claude:interactive")
	marks, _ = toggleMark(marks, "claude:x")
	if !on || len(marks) != 2 {
		t.Fatalf("toggle on: %v %q", on, marks)
	}
	if err := saveMarks(path, marks); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("file mode: %v %v", fi.Mode(), err)
	}
	if got := loadMarks(path); !slices.Equal(got, marks) {
		t.Errorf("reloaded %q, saved %q", got, marks)
	}
	marks, on = toggleMark(marks, "claude:x")
	if on || !slices.Equal(marks, []string{"tmux:a:claude:interactive"}) {
		t.Errorf("toggle off: %v %q", on, marks)
	}
	// The oldest marks go beyond importantMax.
	var many []string
	for i := range importantMax + 5 {
		many, _ = toggleMark(many, "proc:"+strconv.Itoa(i)+":0")
	}
	if len(many) != importantMax || many[0] != "proc:5:0" {
		t.Errorf("cap: %d marks, oldest %q", len(many), many[0])
	}
	// A rename moves only that session's marks.
	if err := saveMarks(path, []string{"tmux:a:claude:interactive", "tmux:ab:codex:interactive", "claude:a"}); err != nil {
		t.Fatal(err)
	}
	if err := RenameMarks(path, "a", "b"); err != nil {
		t.Fatal(err)
	}
	if got := loadMarks(path); !slices.Equal(got, []string{"tmux:b:claude:interactive", "tmux:ab:codex:interactive", "claude:a"}) {
		t.Errorf("after rename: %q", got)
	}
}

func TestImportantPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "important")
	m := loadedWith(Options{Version: "test", ImportantFile: path})
	a, _ := m.selected()
	m = press(m, "!")
	data, _ := os.ReadFile(path)
	if strings.TrimSpace(string(data)) != markID(a) {
		t.Fatalf("file %q, want %q", data, markID(a))
	}
	// A new lazyai shows the mark.
	if m2 := loadedWith(Options{Version: "test", ImportantFile: path}); !m2.isImportant(a) {
		t.Error("mark lost on restart")
	}
	// The agent restarted in the same tmux session is still marked.
	b := a
	b.PID++
	if !m.isImportant(b) {
		t.Error("mark lost when the agent restarted in its session")
	}
}

func TestRenameSessionInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "important")
	m := loadedWith(Options{Version: "test", ImportantFile: path})
	a, _ := m.selected()
	m = press(m, "!", "R")
	if m.popup != popupInput || m.input != inputRename || string(m.buf) != a.TmuxSession {
		t.Fatalf("R: popup %d input %d buf %q", m.popup, m.input, string(m.buf))
	}
	// A bad name is refused before tmux is asked.
	m.buf = []rune("bad name")
	m = press(m, "enter")
	if !m.msgErr || !strings.Contains(m.msg, "letters, digits") {
		t.Errorf("bad name: %q", m.msg)
	}
	// The same name changes nothing.
	m = press(m, "R", "enter")
	if m.msg != "name unchanged" {
		t.Errorf("same name: %q", m.msg)
	}
	// A done rename moves the mark to the new name.
	next, _ := m.Update(actionMsg{text: "renamed", renamed: [2]string{a.TmuxSession, "api"}})
	m = next.(model)
	b := a
	b.TmuxSession = "api"
	if m.isImportant(a) || !m.isImportant(b) || !slices.Equal(loadMarks(path), []string{markID(b)}) {
		t.Errorf("mark not moved: %q", m.important)
	}
	// Agents outside tmux cannot be renamed.
	if r := disabledReason(renameKey, agent.Agent{}); r == "" {
		t.Error("rename allowed outside tmux")
	}
}
