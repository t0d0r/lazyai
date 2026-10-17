package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestObserveAndFlags(t *testing.T) {
	t0 := time.Now()
	a := agent.Agent{Proc: agent.Proc{PID: 1, Start: t0.Add(-time.Hour), CPU: 95, RSS: 3 << 30}, Status: "busy"}
	track := observe(nil, []agent.Agent{a}, t0)
	if got := flags(a, track[1], t0); strings.Join(got, ",") != "high-mem" {
		t.Errorf("first look: %v", got)
	}
	track = observe(track, []agent.Agent{a}, t0.Add(31*time.Minute))
	got := strings.Join(flags(a, track[1], t0.Add(31*time.Minute)), ",")
	if got != "high-cpu,high-mem,long-busy" {
		t.Errorf("after 31m: %v", got)
	}
	a.Status, a.CPU = "idle", 0
	track = observe(track, []agent.Agent{a}, t0.Add(32*time.Minute))
	if tr := track[1]; !tr.highCPUSince.IsZero() || !tr.since.Equal(t0.Add(32*time.Minute)) || !tr.known {
		t.Errorf("status change not tracked: %+v", tr)
	}
	reused := a
	reused.Start = t0
	track = observe(track, []agent.Agent{reused}, t0.Add(33*time.Minute))
	if !track[1].since.Equal(t0.Add(33 * time.Minute)) {
		t.Errorf("PID reuse not detected: %+v", track[1])
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
