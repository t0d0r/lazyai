package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"agentctl/internal/agent"
)

func TestParseLaunch(t *testing.T) {
	home, _ := os.UserHomeDir()
	kind, dir, name, extra, err := parseLaunch("claude ~/work/p p1 -- --model x", "/base")
	if err != nil || kind != agent.KindClaude || dir != filepath.Join(home, "work/p") || name != "p1" ||
		strings.Join(extra, " ") != "--model x" {
		t.Errorf("got %q %q %q %q %v", kind, dir, name, extra, err)
	}
	_, dir, name, extra, err = parseLaunch("codex", "/base")
	if err != nil || dir != "/base" || name != "" || len(extra) != 0 {
		t.Errorf("defaults: %q %q %q %v", dir, name, extra, err)
	}
	if _, dir, _, _, _ = parseLaunch("codex sub", "/base"); dir != "/base/sub" {
		t.Errorf("relative dir: %q", dir)
	}
	for _, bad := range []string{"", "rm -rf /", "claude a b c d"} {
		if _, _, _, _, err := parseLaunch(bad, "/base"); err == nil {
			t.Errorf("parseLaunch(%q) accepted", bad)
		}
	}
}

func TestProtectedCannotBeStopped(t *testing.T) {
	m := newModel(Options{})
	m.all = []agent.Agent{{Proc: agent.Proc{PID: 7}, Kind: agent.KindClaude, Role: agent.RoleInteractive, Protected: true}}
	m.applyFilter()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("K")})
	if nm := next.(model); nm.mode != modeList || !nm.msgErr {
		t.Errorf("protected agent reached confirm: mode=%v msg=%q", nm.mode, nm.msg)
	}
}

func TestHelpersHiddenByDefault(t *testing.T) {
	m := newModel(Options{})
	m.all = []agent.Agent{
		{Proc: agent.Proc{PID: 1}, Role: agent.RoleInteractive},
		{Proc: agent.Proc{PID: 2}, Role: agent.RoleHelper},
	}
	m.applyFilter()
	if len(m.rows) != 1 {
		t.Fatalf("rows %d, want 1", len(m.rows))
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if len(next.(model).rows) != 2 {
		t.Error("h did not show helpers")
	}
}
