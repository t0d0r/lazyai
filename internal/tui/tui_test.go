package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"lazyai/internal/agent"
)

func key(s string) tea.KeyMsg {
	switch s {
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "ctrl+h":
		return tea.KeyMsg{Type: tea.KeyCtrlH}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func press(m model, keys ...string) model {
	for _, k := range keys {
		next, _ := m.Update(key(k))
		m = next.(model)
	}
	return m
}

func sample() []agent.Agent {
	start := time.Now().Add(-time.Hour)
	mk := func(pid int, kind agent.Kind, role agent.Role) agent.Agent {
		return agent.Agent{Proc: agent.Proc{PID: pid, Start: start}, Kind: kind, Role: role, Status: "idle", CWD: "/w/p"}
	}
	claude := mk(1, agent.KindClaude, agent.RoleInteractive)
	claude.TmuxSession, claude.TmuxPane, claude.Name = "lazyai-c", "%1", "c-1"
	self := mk(2, agent.KindHermes, agent.RoleInteractive)
	self.Protected = true
	return []agent.Agent{
		claude, self,
		mk(3, agent.KindCodex, agent.RoleServer),
		mk(4, agent.KindClaude, agent.RoleHelper),
		mk(5, agent.KindColibri, agent.RoleEngine),
		mk(6, agent.KindColibri, agent.RoleJob),
	}
}

func loaded(w, h int) model {
	m := newModel(Options{Version: "test"})
	next, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	next, _ = next.(model).Update(agentsMsg{agents: sample()})
	return next.(model)
}

func pids(l list) []int {
	var out []int
	for _, a := range l.rows {
		out = append(out, a.PID)
	}
	return out
}

func TestGrouping(t *testing.T) {
	m := loaded(120, 30)
	got := [numLists][]int{pids(m.lists[0]), pids(m.lists[1]), pids(m.lists[2])}
	if len(got[0]) != 2 || len(got[1]) != 1 || len(got[2]) != 2 {
		t.Errorf("groups: %v (helper must be hidden)", got)
	}
	m = press(m, "H")
	if len(m.lists[listServices].rows) != 2 {
		t.Errorf("H did not show the helper: %v", pids(m.lists[listServices]))
	}
}

func TestAttachMark(t *testing.T) {
	m := loaded(120, 30)
	l := m.layout()
	var rows []string
	for _, line := range strings.Split(ansi.Strip(m.listPanel(listAgents, l)), "\n") {
		if strings.Contains(line, "claude") || strings.Contains(line, "hermes") {
			rows = append(rows, strings.TrimLeft(line, "│|"))
		}
	}
	if len(rows) != 2 || !strings.HasPrefix(rows[0], "@ claude c-1") || !strings.HasPrefix(rows[1], "  hermes") {
		t.Errorf("Agents rows: %q; want @ only before the agent in tmux", rows)
	}
	if help := ansi.Strip(m.helpPopup()); !strings.Contains(help, "@ before a row") {
		t.Errorf("help lacks the @ legend:\n%s", help)
	}
}

func TestTotalRSS(t *testing.T) {
	const m = 1 << 20
	parent := agent.Agent{Proc: agent.Proc{PID: 1, RSS: 100 * m}, TreeRSS: 300 * m,
		Children: []agent.ChildProc{{PID: 2, RSS: 150 * m}, {PID: 3, RSS: 50 * m}}}
	nested := agent.Agent{Proc: agent.Proc{PID: 2, RSS: 150 * m}, TreeRSS: 150 * m}
	other := agent.Agent{Proc: agent.Proc{PID: 9, RSS: 10 * m}, TreeRSS: 40 * m}
	// PID 2 inside a container is another process than PID 2 on the host.
	foreign := agent.Agent{Proc: agent.Proc{PID: 2, RSS: 5 * m}, TreeRSS: 5 * m,
		ForeignPID: true, ContainerID: "abc"}
	if got := totalRSS([]agent.Agent{parent, nested, other, foreign}); got != 345*m {
		t.Errorf("totalRSS = %dM, want 345M", got/m)
	}

	mm := loaded(120, 30)
	mm.all[0].TreeRSS = 2 << 30
	if view := mm.View(); !strings.Contains(view, "mem 2.0G "+mm.updated.Format("15:04:05")) {
		t.Errorf("status panel lacks the memory total next to the clock:\n%s", view)
	}
}

func TestViewFitsScreen(t *testing.T) {
	for _, size := range [][2]int{{40, 10}, {80, 24}, {120, 30}, {200, 60}} {
		m := loaded(size[0], size[1])
		for _, keys := range [][]string{nil, {"]"}, {"]", "]"}, {"]", "]", "]"}, {"?"}, {"x"}, {"n"}} {
			mm := press(m, keys...)
			lines := strings.Split(mm.View(), "\n")
			if len(lines) != size[1] {
				t.Errorf("%v %v: %d lines, want %d", size, keys, len(lines), size[1])
			}
			for i, l := range lines {
				if w := ansi.StringWidth(l); w > size[0] {
					t.Errorf("%v %v: line %d is %d wide", size, keys, i, w)
				}
			}
		}
	}
}

func TestLayoutHeights(t *testing.T) {
	m := loaded(120, 30)
	l := m.layout()
	sum := l.statusH
	for _, h := range l.listH {
		sum += h
	}
	if sum != l.bodyH {
		t.Errorf("panels use %d rows, body has %d", sum, l.bodyH)
	}
	// Even split however many rows each list has.
	var many []agent.Agent
	for i := range 40 {
		many = append(many, agent.Agent{Proc: agent.Proc{PID: 100 + i}, Role: agent.RoleServer})
	}
	next, _ := m.Update(agentsMsg{agents: append(sample(), many...)})
	l = next.(model).layout()
	lo, hi := l.listH[0], l.listH[0]
	for _, h := range l.listH {
		lo, hi = min(lo, h), max(hi, h)
	}
	if hi-lo > 1 {
		t.Errorf("uneven split: %v", l.listH)
	}
	// Too short to split: the list in use grows.
	m = press(loaded(80, 14), "2")
	l = m.layout()
	if l.listH[listServices] <= l.listH[listAgents] || l.listY[listModels]+l.listH[listModels] != l.bodyH {
		t.Errorf("short layout: %+v", l)
	}
}

func TestEmptyListMessage(t *testing.T) {
	m := loaded(120, 30)
	next, _ := m.Update(agentsMsg{agents: sample()[:2]})
	m = press(next.(model), "3")
	if got := ansi.Strip(m.mainLines(80)[0]); got != "no models running" {
		t.Errorf("empty list: %q", got)
	}
}

func TestFocusAndTabs(t *testing.T) {
	m := loaded(120, 30)
	if m.focus != listAgents {
		t.Fatalf("initial focus %d", m.focus)
	}
	m = press(m, "tab")
	if m.focus != listServices || m.lastList != listServices {
		t.Errorf("tab: focus %d last %d", m.focus, m.lastList)
	}
	m = press(m, "tab", "tab")
	if m.focus != focusMain || m.lastList != listModels {
		t.Errorf("tab to main: focus %d last %d", m.focus, m.lastList)
	}
	m = press(m, "esc", "]", "]")
	if m.focus != listModels || m.tab != tabProcesses {
		t.Errorf("esc and ]: focus %d tab %d", m.focus, m.tab)
	}
	m = press(m, "[", "[", "[")
	if m.tab != tabStats {
		t.Errorf("[ wraps: tab %d", m.tab)
	}
}

func TestSelectionFollowsPID(t *testing.T) {
	m := press(loaded(120, 30), "j")
	if a, _ := m.selected(); a.PID != 2 {
		t.Fatalf("selected %d", a.PID)
	}
	agents := sample()
	agents[0], agents[1] = agents[1], agents[0] // reorder
	next, _ := m.Update(agentsMsg{agents: agents})
	if a, _ := next.(model).selected(); a.PID != 2 {
		t.Errorf("selection jumped to %d", a.PID)
	}
}

func TestActions(t *testing.T) {
	m := loaded(120, 30)
	if mm := press(m, "s"); mm.popup != popupConfirm || mm.target.PID != 1 || mm.force {
		t.Errorf("s: popup %v target %d", mm.popup, mm.target.PID)
	}
	if mm := press(m, "K"); !mm.force {
		t.Error("K is not forced")
	}
	if mm := press(m, "j", "K"); mm.popup != popupNone || !mm.msgErr {
		t.Errorf("protected agent reached confirm: %v %q", mm.popup, mm.msg)
	}
	if mm := press(m, "j", "p"); mm.popup != popupNone || !strings.Contains(mm.msg, "tmux") {
		t.Errorf("prompt without tmux: %v %q", mm.popup, mm.msg)
	}
	if mm := press(m, "p", "h", "i", "enter"); mm.popup != popupNone {
		t.Errorf("prompt popup not closed: %v", mm.popup)
	}
	if mm := press(m, "s", "n"); mm.popup != popupNone || mm.msg != "cancelled" {
		t.Errorf("confirm no: %v %q", mm.popup, mm.msg)
	}
}

func TestInputBackspaceVariants(t *testing.T) {
	for _, k := range []string{"backspace", "ctrl+h"} {
		t.Run(k, func(t *testing.T) {
			m := press(loaded(120, 30), "n", "c", "l", "a", k)
			if got := string(m.buf); got != "cl" {
				t.Errorf("%s left input %q, want %q", k, got, "cl")
			}
		})
	}
}

func TestLaunchHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "launch_history")
	m := loaded(120, 30)
	m.opts.HistoryFile = path
	m = press(m, "n", "c", "o", "d", "e", "x", "enter", "n", "c", "l", "a", "u", "d", "e", "enter")
	if strings.Join(m.hist, ",") != "codex,claude" {
		t.Fatalf("history %q", m.hist)
	}
	if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("history file: %v %v", fi, err)
	}
	m = press(m, "n", "x", "up")
	if got := string(m.buf); got != "claude" {
		t.Errorf("up: %q", got)
	}
	for _, step := range []struct{ key, want string }{{"up", "codex"}, {"up", "codex"}, {"down", "claude"}, {"down", "x"}, {"down", "x"}} {
		if m = press(m, step.key); string(m.buf) != step.want {
			t.Errorf("%s: %q, want %q", step.key, string(m.buf), step.want)
		}
	}

	m = press(m, "esc", "n", "up", "up", "enter") // codex again moves to the end
	if got := loadHistory(path); strings.Join(got, ",") != "claude,codex" {
		t.Errorf("reloaded history %q", got)
	}
	if m := newModel(Options{HistoryFile: path}); len(m.hist) != 2 {
		t.Errorf("new model history %q", m.hist)
	}
}

func TestAddHistoryLimit(t *testing.T) {
	var h []string
	for i := range historyMax + 5 {
		h = addHistory(h, fmt.Sprint("claude d", i))
	}
	if len(h) != historyMax || h[0] != "claude d5" {
		t.Errorf("len %d first %q", len(h), h[0])
	}
	if h = addHistory(h, "  claude   d5 "); h[len(h)-1] != "claude d5" || h[0] != "claude d6" {
		t.Errorf("duplicate not moved: %q ... %q", h[0], h[len(h)-1])
	}
}

func TestCompleteLaunch(t *testing.T) {
	base := t.TempDir()
	for _, d := range []string{"proj-a", "proj-b", "other", ".hidden", "proj-a/sub"} {
		if err := os.Mkdir(filepath.Join(base, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(base, "proj-file"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		in, want, choices string
	}{
		{"cl", "claude ", ""},
		{"co", "co", "codex copilot"},
		{"zz", "zz", ""},
		{"claude ", "claude ", "other/ proj-a/ proj-b/"},
		{"claude p", "claude proj-", "proj-a/ proj-b/"},
		{"claude proj-a", "claude proj-a/", ""},
		{"claude proj-a/", "claude proj-a/sub/", ""},
		{"claude .h", "claude .hidden/", ""},
		{"./", "./", ""}, // the first word is a kind, never a directory
		{"claude " + base + "/o", "claude " + base + "/other/", ""},
		{"claude ~", "claude ~/", ""},
		{"claude proj-a n", "claude proj-a n", ""},
		{"claude -- p", "claude -- p", ""},
	} {
		got, choices := completeLaunch(c.in, base)
		if got != c.want || strings.Join(choices, " ") != c.choices {
			t.Errorf("completeLaunch(%q) = %q %q, want %q %q", c.in, got, choices, c.want, c.choices)
		}
	}

	m := loaded(120, 30)
	m.opts.LaunchDir = base
	m = press(m, "n", "c", "l", "tab", "p", "tab")
	if got := string(m.buf); got != "claude proj-" || len(m.choices) != 2 {
		t.Errorf("tab in popup: %q %q", got, m.choices)
	}
	if !strings.Contains(ansi.Strip(m.View()), "proj-a/  proj-b/") {
		t.Error("choices not shown")
	}
	if m = press(m, "a"); m.choices != nil {
		t.Error("choices kept after typing")
	}
}

func TestMenu(t *testing.T) {
	m := press(loaded(120, 30), "j", "x")
	if m.popup != popupMenu {
		t.Fatal("x did not open the menu")
	}
	var stop menuItem
	for _, it := range m.menu {
		if it.key == "s" {
			stop = it
		}
	}
	if stop.disabled == "" {
		t.Error("stop on a protected agent is not disabled")
	}
	m = press(m, "esc", "k", "x", "j", "j", "enter") // third item on agent 1: stop
	if m.popup != popupConfirm || m.target.PID != 1 {
		t.Errorf("menu enter: popup %v target %d", m.popup, m.target.PID)
	}
}

func TestMouseSelect(t *testing.T) {
	m := loaded(120, 30)
	l := m.layout()
	click := tea.MouseMsg{X: 2, Y: l.listY[listModels] + 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	next, _ := m.Update(click)
	m = next.(model)
	if a, _ := m.selected(); m.focus != listModels || a.PID != 6 {
		t.Errorf("click: focus %d selected %d", m.focus, a.PID)
	}
	next, _ = m.Update(tea.MouseMsg{X: l.leftW + 5, Y: 5, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if next.(model).focus != focusMain {
		t.Error("click on the main panel did not focus it")
	}
}

func TestOverlay(t *testing.T) {
	if got := ansi.Strip(overlay("aaaa\nbbbb\ncccc", "XY", 1, 1)); got != "aaaa\nbXYb\ncccc" {
		t.Errorf("overlay: %q", got)
	}
}

func TestBarChart(t *testing.T) {
	lines := barChart([]float64{0, 50, 100}, 20, 2, 100, func(v float64) string { return "" })
	got := []string{ansi.Strip(lines[0]), ansi.Strip(lines[1])}
	if !strings.HasSuffix(got[0], "|  #") || !strings.HasSuffix(got[1], "| ##") {
		t.Errorf("chart:\n%s\n%s", got[0], got[1])
	}
	for peak, want := range map[float64]float64{0: 1, 3: 5, 7.6: 10, 10: 10, 21: 25, 99: 100, 250: 250, 260: 500} {
		if got := scaleCPU(peak); got != want {
			t.Errorf("scaleCPU(%v) = %v, want %v", peak, got, want)
		}
	}
}

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
	for _, bad := range []string{"", "rm -rf /", "claude a b c d", "colibri"} {
		if _, _, _, _, err := parseLaunch(bad, "/base"); err == nil {
			t.Errorf("parseLaunch(%q) accepted", bad)
		}
	}
}
