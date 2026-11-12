package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"lazyai/internal/agent"
	"lazyai/internal/tmux"
)

func key(s string) tea.KeyMsg {
	switch s {
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
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
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
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
	return loadedSize(Options{Version: "test"}, w, h)
}

func loadedWith(opts Options) model {
	return loadedSize(opts, 120, 30)
}

func loadedSize(opts Options, w, h int) model {
	m := newModel(opts)
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
	if len(rows) != 2 || !strings.HasPrefix(rows[0], "@claude lazyai-c c-1") || !strings.HasPrefix(rows[1], " hermes") {
		t.Errorf("Agents rows: %q; want @ only before the agent in tmux", rows)
	}
	// The tmux session is the label; the Claude name follows only whole.
	narrow := loaded(105, 30)
	if p := ansi.Strip(narrow.listPanel(listAgents, narrow.layout())); !strings.Contains(p, "@claude lazyai-") || strings.Contains(p, "c-1") {
		t.Errorf("narrow row keeps the session and drops the Claude name:\n%s", p)
	}
	if help := ansi.Strip(m.helpPopup()); !strings.Contains(help, "@ before a row") {
		t.Errorf("help lacks the @ legend:\n%s", help)
	}
}

func TestPathLabel(t *testing.T) {
	home, _ := os.UserHomeDir()
	for p, want := range map[string]string{
		home + "/work/alpha/web-frontend": "alpha/web-frontend",
		home + "/x":                       "~/x",
		home:                              "~",
		"/w/p":                            "w/p",
		"/srv":                            "/srv",
		"/":                               "/",
		"":                                "-",
	} {
		if got := lastDirs(p); got != want {
			t.Errorf("lastDirs(%q) = %q, want %q", p, got, want)
		}
	}
	for _, c := range []struct {
		s    string
		w    int
		want string
	}{
		{"alpha/web-frontend", 20, "alpha/web-frontend"},
		{"alpha/web-frontend", 10, "alpha/..nd"},
		{"alpha/web-frontend", 8, "..ontend"},
		{"web-frontend", 8, "..ontend"},
		{"alpha/web-frontend", 2, "alpha/web-frontend"}, // no room: spread cuts it
	} {
		if got := shortenPath(c.s, c.w); got != c.want {
			t.Errorf("shortenPath(%q, %d) = %q, want %q", c.s, c.w, got, c.want)
		}
	}

	// In a row, the label gives way before status, CPU and memory. At 135
	// columns a Services row has 11 cells left for it.
	m := loaded(135, 30)
	m.all[2].CWD = home + "/work/alpha/web-frontend"
	m.lists[listServices].rows[0].CWD = m.all[2].CWD
	panel := ansi.Strip(m.listPanel(listServices, m.layout()))
	if !strings.Contains(panel, "codex server alpha/..end ") || !strings.Contains(panel, "idle") {
		t.Errorf("narrow Services row:\n%s", panel)
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

// inLayout is m switched to the given layout with v.
func inLayout(m model, mode layoutMode) model {
	for i := 0; i < int(numLayouts) && m.mode != mode; i++ {
		m = press(m, "v")
	}
	return m
}

func TestViewFitsScreen(t *testing.T) {
	for _, size := range [][2]int{{40, 10}, {48, 16}, {60, 20}, {80, 24}, {120, 30}, {200, 60}} {
		for mode := range numLayouts {
			m := inLayout(loaded(size[0], size[1]), mode)
			for _, keys := range [][]string{nil, {"]"}, {"]", "]"}, {"]", "]", "]"}, {"?"}, {"x"}, {"n"}, {"/"}, {"enter"}} {
				mm := press(m, keys...)
				lines := strings.Split(mm.View(), "\n")
				if len(lines) != size[1] {
					t.Errorf("%v %v %v: %d lines, want %d", size, mode, keys, len(lines), size[1])
				}
				// Every line fills the width exactly, so a layout switch
				// leaves nothing of the previous arrangement on the screen.
				for i, l := range lines {
					if w := ansi.StringWidth(l); w != size[0] {
						t.Errorf("%v %v %v: line %d is %d wide", size, mode, keys, i, w)
					}
				}
			}
		}
	}
}

func TestLayoutCycle(t *testing.T) {
	m := loaded(120, 30)
	if m.mode != layoutSplit {
		t.Fatalf("start layout %v, want split", m.mode)
	}
	m = press(m, "j", "]", "]") // agent 2, Processes tab
	for _, want := range []layoutMode{layoutDetail, layoutLists, layoutScreen, layoutSplit} {
		m = press(m, "v")
		if m.mode != want {
			t.Fatalf("v: layout %v, want %v", m.mode, want)
		}
		name := want.String() + " layout"
		if !strings.Contains(m.msg, name) {
			t.Errorf("message %q does not name the %s", m.msg, name)
		}
		// The screen layout has no status panel; its list panel shows "lazyai".
		if s := ansi.Strip(m.statusPanel(m.layout())); want != layoutScreen && !strings.Contains(s, name) {
			t.Errorf("status panel does not show the %s:\n%s", name, s)
		}
		if a, _ := m.selected(); a.PID != 2 || m.tab != tabProcesses || m.lastList != listAgents {
			t.Errorf("%v: selected %d tab %d list %d", want, a.PID, m.tab, m.lastList)
		}
	}
	// Lists hides the main panel: the focus goes back to the list.
	if mm := press(m, "enter", "v", "v"); mm.mode != layoutLists || mm.focus != listAgents {
		t.Errorf("lists from the main panel: layout %v focus %d", mm.mode, mm.focus)
	}
	// The x menu offers the next layout and runs it with its key.
	mm := press(m, "x")
	var item menuItem
	for _, it := range mm.menu {
		if it.key == "v" {
			item = it
		}
	}
	if !strings.Contains(item.label, "detail") {
		t.Errorf("menu item v: %+v", item)
	}
	if mm = press(mm, "v"); mm.mode != layoutDetail || mm.popup != popupNone {
		t.Errorf("v in the menu: layout %v popup %v", mm.mode, mm.popup)
	}
	// v typed into the filter or a popup is text.
	if mm := press(m, "/", "v"); mm.mode != layoutSplit || mm.filters[listAgents] != "v" {
		t.Errorf("v in the filter: layout %v filter %q", mm.mode, mm.filters[listAgents])
	}
	if mm := press(m, "n", "v"); mm.mode != layoutSplit || string(mm.buf) != "v" {
		t.Errorf("v in the launch popup: layout %v input %q", mm.mode, string(mm.buf))
	}
}

// panelRow is the first row of list i that contains s, without styles.
func panelRow(m model, i int, s string) string {
	for _, line := range strings.Split(ansi.Strip(m.listPanel(i, m.layout())), "\n") {
		if strings.Contains(line, s) {
			return line
		}
	}
	return ""
}

func TestLayoutStructure(t *testing.T) {
	split := loaded(120, 30)
	detail := inLayout(split, layoutDetail)
	lists := inLayout(split, layoutLists)
	ls, ld, ll := split.layout(), detail.layout(), lists.layout()

	if ls.leftW != 40 || ls.rightW != 80 {
		t.Errorf("split widths %d + %d, want 40 + 80", ls.leftW, ls.rightW)
	}
	if ld.leftW >= ls.leftW || ld.rightW < ls.rightW+15 || ld.leftW+ld.rightW != 120 {
		t.Errorf("detail widths %d + %d: the main panel must gain at least 15 columns", ld.leftW, ld.rightW)
	}
	if ll.leftW != 120 || ll.rightW != 0 {
		t.Errorf("lists widths %d + %d, want 120 + 0", ll.leftW, ll.rightW)
	}
	if ld.listH != ls.listH || ll.listH != ls.listH || ld.listY != ls.listY || ll.listY != ls.listY {
		t.Errorf("heights differ: split %v detail %v lists %v", ls.listH, ld.listH, ll.listH)
	}

	// Side by side: the main panel's corner sits where the column ends.
	for _, c := range []struct {
		name string
		m    model
		l    layout
	}{{"split", split, ls}, {"detail", detail, ld}} {
		top := []rune(ansi.Strip(strings.Split(c.m.View(), "\n")[0]))
		if string(top[c.l.leftW]) != "╭" || !strings.Contains(string(top), "Info - Screen - Processes - Stats") {
			t.Errorf("%s top line: %s", c.name, string(top))
		}
	}
	if v := ansi.Strip(lists.View()); strings.Count(strings.Split(v, "\n")[0], "╭") != 1 || strings.Contains(v, "Info - Screen") {
		t.Errorf("lists layout shows a main panel:\n%s", v)
	}

	// Rows: full in split and lists, compact in detail (no CPU or memory;
	// status only when busy).
	if r := panelRow(split, listServices, "codex"); !strings.Contains(r, "codex server w/p") || !strings.Contains(r, "idle") || !strings.Contains(r, "%") {
		t.Errorf("split Services row: %q", r)
	}
	if r := panelRow(detail, listServices, "codex"); !strings.Contains(r, " codex server w/p ") || strings.Contains(r, "idle") || strings.Contains(r, "%") {
		t.Errorf("detail Services row: %q", r)
	}
	busy := sample()
	busy[2].Status = "busy"
	next, _ := detail.Update(agentsMsg{agents: busy})
	if r := panelRow(next.(model), listServices, "codex"); !strings.HasSuffix(strings.TrimRight(r, "│"), "busy") {
		t.Errorf("detail busy row: %q", r)
	}
	if r := panelRow(detail, listAgents, "claude"); !strings.Contains(r, "@claude lazyai-c") {
		t.Errorf("detail Agents row: %q", r)
	}
	if r := panelRow(lists, listAgents, "claude"); !strings.Contains(r, "@claude lazyai-c c-1") || !strings.Contains(r, "%") {
		t.Errorf("lists Agents row: %q", r)
	}

	// Mouse: the same column is a list in split and the main panel in detail.
	x := ld.leftW + 2
	click := tea.MouseMsg{X: x, Y: ls.listY[listModels] + 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	if next, _ := split.Update(click); next.(model).focus != listModels {
		t.Errorf("split click at x=%d: focus %d", x, next.(model).focus)
	}
	if next, _ := detail.Update(click); next.(model).focus != focusMain {
		t.Errorf("detail click at x=%d: focus %d", x, next.(model).focus)
	}
	click.X = 100
	next, _ = lists.Update(click)
	if a, _ := next.(model).selected(); next.(model).focus != listModels || a.PID != 6 {
		t.Errorf("lists click at x=100: focus %d selected %d", next.(model).focus, a.PID)
	}
}

func TestLayoutListsHidesMain(t *testing.T) {
	m := press(loaded(120, 30), "]") // Screen tab
	if !m.capturing {
		t.Fatal("split does not capture the Screen tab")
	}
	m.capturing = false
	m = inLayout(m, layoutLists)
	if m.capture() != nil {
		t.Error("lists layout captures a hidden Screen tab")
	}
	// A capture that arrives while the panel is hidden is dropped.
	next, _ := m.Update(screenMsg{pane: "%1", text: "old"})
	if m = next.(model); m.screen.text != "" {
		t.Errorf("hidden capture kept: %q", m.screen.text)
	}
	// tab cycles the lists only.
	for i, want := range []int{listServices, listModels, listAgents} {
		if m = press(m, "tab"); m.focus != want {
			t.Errorf("tab %d: focus %d, want %d", i+1, m.focus, want)
		}
	}
	if mm := press(m, "shift+tab"); mm.focus == focusMain {
		t.Error("shift+tab focused the hidden main panel")
	}
	// Main panel keys say why they do nothing.
	for _, k := range []string{"enter", "]", "["} {
		mm := press(m, k)
		if mm.focus == focusMain || mm.tab != tabScreen || !strings.Contains(mm.msg, "v") {
			t.Errorf("%s in lists: focus %d tab %d msg %q", k, mm.focus, mm.tab, mm.msg)
		}
	}
	// Back to a layout with the main panel: the Screen tab captures again.
	if _, cmd := m.Update(key("v")); cmd == nil {
		t.Error("leaving lists does not capture the Screen tab")
	}
}

func TestLayoutNarrow(t *testing.T) {
	for w := 30; w <= 250; w++ {
		m := loaded(w, 30)
		s, d := m.layout(), inLayout(m, layoutDetail).layout()
		if d.leftW > w/2 || d.leftW > s.leftW || d.rightW < s.rightW || d.leftW+d.rightW != w {
			t.Errorf("width %d: detail %d + %d, split %d + %d", w, d.leftW, d.rightW, s.leftW, s.rightW)
		}
	}
	// Under 50 columns detail keeps split's widths.
	if s, d := loaded(48, 20).layout(), inLayout(loaded(48, 20), layoutDetail).layout(); s.leftW != d.leftW {
		t.Errorf("48 columns: split %d, detail %d", s.leftW, d.leftW)
	}
	// The counts are never cut in the narrow navigator.
	for _, w := range []int{80, 120, 200} {
		m := inLayout(attachedModel(w, 30), layoutDetail)
		if s := ansi.Strip(m.statusPanel(m.layout())); !strings.Contains(s, "5 run, 0 busy") {
			t.Errorf("detail status panel at %d:\n%s", w, s)
		}
	}
}

func TestLayoutHints(t *testing.T) {
	m := loaded(120, 30)
	if help := ansi.Strip(press(m, "?").View()); !strings.Contains(help, "layout: split, detail, lists, screen (now split)") {
		t.Errorf("help lacks the layout key:\n%s", help)
	}
	if help := ansi.Strip(inLayout(m, layoutDetail).helpPopup()); !strings.Contains(help, "(now detail)") {
		t.Errorf("help does not name the detail layout:\n%s", help)
	}
	if f := ansi.Strip(m.footer()); !strings.Contains(f, "v layout") || !strings.Contains(f, "[ ] tabs") {
		t.Errorf("split footer: %s", f)
	}
	if f := ansi.Strip(press(m, "enter").footer()); !strings.Contains(f, "v layout") {
		t.Errorf("main panel footer: %s", f)
	}
	lists := inLayout(m, layoutLists)
	lists.msg = "" // the message would cut the hints
	if f := ansi.Strip(lists.footer()); !strings.Contains(f, "v layout") || strings.Contains(f, "[ ] tabs") {
		t.Errorf("lists footer: %s", f)
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
	for _, k := range []string{"right", "left", "l", "h"} {
		if mm := press(m, k); mm.focus != listAgents {
			t.Errorf("%s moved the focus to %d", k, mm.focus)
		}
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

// attachedModel is loaded with the tmux agent attached by two clients.
func attachedModel(w, h int) model {
	m := newModel(Options{Version: "test"})
	next, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	agents := sample()
	agents[0].TmuxClients = 2
	next, _ = next.(model).Update(agentsMsg{agents: agents})
	return next.(model)
}

func TestAttachedMark(t *testing.T) {
	m := attachedModel(120, 30)
	l := m.layout()
	if !strings.Contains(ansi.Strip(m.listPanel(listAgents, l)), ">claude lazyai-c c-1") {
		t.Errorf("attached agent lacks >:\n%s", ansi.Strip(m.listPanel(listAgents, l)))
	}
	if help := ansi.Strip(m.helpPopup()); !strings.Contains(help, "> before a row") {
		t.Errorf("help lacks the > legend:\n%s", help)
	}
	if s := ansi.Strip(m.statusPanel(l)); !strings.Contains(s, "5 run, 0 busy, 1 att") {
		t.Errorf("status panel at 120: %s", s)
	}
	wide := attachedModel(170, 30)
	if s := ansi.Strip(wide.statusPanel(wide.layout())); !strings.Contains(s, "5 running, 0 busy, 1 attached") {
		t.Errorf("status panel at 170: %s", s)
	}
	narrow := attachedModel(64, 30)
	if s := ansi.Strip(narrow.statusPanel(narrow.layout())); !strings.Contains(s, "5 run, 0 busy ") {
		t.Errorf("status panel at 64: %s", s)
	}
	if s := ansi.Strip(loaded(170, 30).statusPanel(wide.layout())); !strings.Contains(s, "0 attached") {
		t.Errorf("detached tmux agent not counted as 0 attached: %s", s)
	}
	if got := pids(press(m, "/", "a", "t", "t", "a", "c", "h", "e", "d").lists[listAgents]); len(got) != 1 || got[0] != 1 {
		t.Errorf("/attached: %v", got)
	}
	if got := pids(press(loaded(120, 30), "/", "d", "e", "t", "a", "c", "h").lists[listAgents]); len(got) != 1 || got[0] != 1 {
		t.Errorf("/detach: %v", got)
	}
}

func TestPromptAttachedConfirm(t *testing.T) {
	m := attachedModel(120, 30)
	mm := press(m, "p")
	if mm.popup != popupConfirm || !mm.confirmSend || !strings.Contains(ansi.Strip(mm.View()), "someone may be typing") {
		t.Fatalf("p on attached: popup %v confirmSend %v", mm.popup, mm.confirmSend)
	}
	if y := press(mm, "y"); y.popup != popupInput || y.input != inputSend || y.confirmSend {
		t.Errorf("y: popup %v input %v", y.popup, y.input)
	}
	if n := press(mm, "n"); n.popup != popupNone || n.msg != "cancelled" || n.confirmSend {
		t.Errorf("n: popup %v %q", n.popup, n.msg)
	}
	if s := press(mm, "n", "s"); s.popup != popupConfirm || s.confirmSend {
		t.Error("s after a cancelled prompt confirm does not ask to stop")
	}
}

func TestProcessesParents(t *testing.T) {
	a := agent.Agent{Proc: agent.Proc{PID: 70, PPID: 60}, Command: "claude",
		Parents:  []agent.ChildProc{{PID: 1, Command: "launchd"}, {PID: 60, PPID: 1, Command: "fish"}},
		Children: []agent.ChildProc{{PID: 80, PPID: 70, Depth: 1, Command: "node"}}}
	got := ansi.Strip(strings.Join(processLines(a), "\n"))
	for _, want := range []string{"  launchd", "    fish", "      claude", "        node", "in 2 processes; the 2 above it are not counted"} {
		if !strings.Contains(got, want) {
			t.Errorf("processes lack %q:\n%s", want, got)
		}
	}
	if strings.Index(got, "launchd") > strings.Index(got, "claude") {
		t.Errorf("parents not above the agent:\n%s", got)
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

	// A line that does not parse is kept too, so it can be fixed.
	m = press(m, "n", "c", "l", "u", "d", "e", " ", "a", " ", "b", " ", "c", "enter")
	if !m.msgErr || m.hist[len(m.hist)-1] != "clude a b c" {
		t.Errorf("bad line: err %v history %q", m.msgErr, m.hist)
	}
	if m = press(m, "n", "up"); string(m.buf) != "clude a b c" {
		t.Errorf("bad line not recalled: %q", string(m.buf))
	}
	if got := loadHistory(path); got[len(got)-1] != "clude a b c" {
		t.Errorf("bad line not saved: %q", got)
	}
}

func TestFilter(t *testing.T) {
	m := press(loaded(120, 30), "/", "c", "l")
	if !m.filtering || pids(m.lists[listAgents])[0] != 1 || len(m.lists[listAgents].rows) != 1 {
		t.Fatalf("filter cl: filtering %v rows %v", m.filtering, pids(m.lists[listAgents]))
	}
	if v := ansi.Strip(m.View()); !strings.Contains(v, "1 Agents /cl") || !strings.Contains(v, "filter /cl_") {
		t.Error("filter not shown")
	}
	// Keys are typed into the filter, not run as commands.
	if m = press(m, "q", "x"); m.filters[listAgents] != "clqx" || m.popup != popupNone {
		t.Errorf("filter %q popup %v", m.filters[listAgents], m.popup)
	}
	if m = press(m, "backspace", "backspace"); len(m.lists[listAgents].rows) != 1 {
		t.Error("backspace")
	}
	m = press(m, "enter")
	if m.filtering || m.filters[listAgents] != "cl" || len(m.lists[listAgents].rows) != 1 {
		t.Errorf("enter: filtering %v filter %q", m.filtering, m.filters[listAgents])
	}
	// The filter holds across refreshes and leaves other lists alone.
	next, _ := m.Update(agentsMsg{agents: sample()})
	if m = next.(model); len(m.lists[listAgents].rows) != 1 || len(m.lists[listServices].rows) != 1 {
		t.Errorf("after refresh: %v %v", pids(m.lists[listAgents]), pids(m.lists[listServices]))
	}
	if m = press(m, "esc"); m.filters[listAgents] != "" || len(m.lists[listAgents].rows) != 2 {
		t.Errorf("esc did not clear: %q", m.filters[listAgents])
	}
	m = press(m, "/", "z", "z")
	if !strings.Contains(ansi.Strip(m.View()), "no match") {
		t.Error("no match not shown")
	}
	if m = press(m, "esc"); m.filtering || len(m.lists[listAgents].rows) != 2 {
		t.Error("esc while typing did not clear")
	}
	// Words match anywhere, in any order: kind and session.
	if m = press(m, "/", "l", "a", "z", "y", " ", "c", "l", "a", "u"); len(m.lists[listAgents].rows) != 1 {
		t.Errorf("two words: %v", pids(m.lists[listAgents]))
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
	bin := t.TempDir()
	for name, mode := range map[string]os.FileMode{"claude-bst": 0o755, "claude-notes": 0o644} {
		if err := os.WriteFile(filepath.Join(bin, name), nil, mode); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)
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
		{"cl", "claude", "claude claude-bst"},
		{"claude-", "claude-bst ", ""}, // a command on PATH; not executable files are left out
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
	m = press(m, "n", "c", "l", "a", "u", "d", "e", " ", "p", "tab")
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
	s, err := parseLaunch("claude ~/work/p p1 -- --model x", "/base")
	if err != nil || s.Kind != agent.KindClaude || s.Dir != filepath.Join(home, "work/p") || s.Name != "p1" ||
		strings.Join(s.Args, " ") != "--model x" {
		t.Errorf("got %+v %v", s, err)
	}
	s, err = parseLaunch("codex", "/base")
	if err != nil || s.Dir != "/base" || s.Name != "" || len(s.Args) != 0 {
		t.Errorf("defaults: %+v %v", s, err)
	}
	if s, _ = parseLaunch("codex sub", "/base"); s.Dir != "/base/sub" {
		t.Errorf("relative dir: %q", s.Dir)
	}
	for _, bad := range []string{"", "no-such-command-x -rf /", "claude a b c d", "colibri", "../bin/claude",
		"claude --model x", "claude dir --model"} {
		if _, err := parseLaunch(bad, "/base"); err == nil {
			t.Errorf("parseLaunch(%q) accepted", bad)
		}
	}
}

func TestLayoutScreen(t *testing.T) {
	m := inLayout(loaded(120, 30), layoutScreen)
	l := m.layout()
	if !l.tabbed || l.screenH < screenMinH || l.mainH+l.screenH != l.bodyH || l.leftW+l.rightW != 120 {
		t.Fatalf("screen layout sizes: %+v", l)
	}
	lines := strings.Split(ansi.Strip(m.View()), "\n")
	if l.statusH != 0 || !strings.HasPrefix(lines[0], "╭ lazyai - 1 Agents - 2 Services - 3 Models") {
		t.Errorf("list panel title: %q", lines[0])
	}
	// The status line sits in the list panel's bottom border, before the position.
	if bottom := lines[l.mainH-1]; !strings.Contains(bottom, " running, ") || !strings.Contains(bottom, " - mem ") || !strings.Contains(bottom, " - 1 of 2 ╯╰") {
		t.Errorf("list panel bottom border: %q", bottom)
	}
	if !strings.Contains(lines[0], "Info - Processes - Stats") || strings.Contains(lines[0], "Screen") {
		t.Errorf("main panel tabs: %q", lines[0])
	}
	if !strings.HasPrefix(lines[l.screenY], "╭ Screen lazyai-c - w/p - c-1 ") {
		t.Errorf("Screen panel top border: %q", lines[l.screenY])
	}

	// The Screen panel is captured whatever the tab, and shows the pane.
	if m.tab != tabInfo || !m.capturing {
		t.Errorf("tab %d: entering the screen layout did not capture", m.tab)
	}
	next, _ := m.Update(screenMsg{pane: "%1", text: "hello from the pane"})
	m = next.(model)
	if !strings.Contains(ansi.Strip(m.View()), "hello from the pane") {
		t.Error("Screen panel does not show the capture")
	}

	// ] and [ skip Screen, which has its own panel.
	if mm := press(m, "]"); mm.tab != tabProcesses {
		t.Errorf("]: tab %d, want Processes", mm.tab)
	}
	if mm := press(m, "["); mm.tab != tabStats {
		t.Errorf("[: tab %d, want Stats", mm.tab)
	}
	// Coming from the Screen tab, the tabs show Info.
	if mm := inLayout(press(loaded(120, 30), "]"), layoutScreen); mm.tab != tabInfo {
		t.Errorf("screen layout from the Screen tab: tab %d", mm.tab)
	}

	// tab cycles the list, the main panel and the Screen panel.
	var order []int
	mm := m
	for range 3 {
		mm = press(mm, "tab")
		order = append(order, mm.focus)
	}
	if fmt.Sprint(order) != fmt.Sprint([]int{focusMain, focusScreen, listAgents}) {
		t.Errorf("tab order: %v", order)
	}
	// 2 shows Services in the one list panel.
	if mm := press(m, "2"); mm.focus != listServices || !strings.Contains(ansi.Strip(mm.View()), "codex server") {
		t.Errorf("2: focus %d", mm.focus)
	}
	// Keys scroll the focused Screen panel, not the list.
	m.screen.text = strings.Repeat("line\n", 100)
	mm = press(m, "tab", "tab", "k", "k")
	if mm.focus != focusScreen || mm.screenFollow || mm.lists[listAgents].cursor != 0 {
		t.Errorf("k in Screen: focus %d follow %v", mm.focus, mm.screenFollow)
	}
	if mm = press(mm, "G"); !mm.screenFollow {
		t.Error("G does not follow the Screen panel again")
	}
	if mm = press(mm, "esc"); mm.focus != listAgents {
		t.Errorf("esc from Screen: focus %d", mm.focus)
	}
	// Mouse: a click in the Screen panel focuses it.
	next, _ = m.Update(tea.MouseMsg{X: 10, Y: l.screenY + 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if next.(model).focus != focusScreen {
		t.Error("click did not focus the Screen panel")
	}
	// Leaving the layout from the Screen panel focuses the main panel.
	if mm := press(m, "tab", "tab", "v"); mm.mode != layoutSplit || mm.focus != focusMain {
		t.Errorf("v from Screen: layout %v focus %d", mm.mode, mm.focus)
	}
}

func TestScreenResize(t *testing.T) {
	m := inLayout(loaded(120, 60), layoutScreen)
	l := m.layout()
	if want := l.bodyH * screenTopDefault / 100; l.mainH != want {
		t.Fatalf("top row %d rows, want %d (%d%% of %d)", l.mainH, want, screenTopDefault, l.bodyH)
	}
	// + grows the Screen panel, - shrinks it, within the limits.
	if mm := press(m, "-"); mm.screenTopPct != screenTopDefault+screenTopStep || mm.layout().screenH >= l.screenH {
		t.Errorf("-: top %d%%, Screen %d rows (was %d)", mm.screenTopPct, mm.layout().screenH, l.screenH)
	}
	if mm := press(m, "+"); mm.screenTopPct != screenTopLow || mm.layout().screenH <= l.screenH {
		t.Errorf("+: top %d%%, Screen %d rows (was %d)", mm.screenTopPct, mm.layout().screenH, l.screenH)
	}
	if mm := press(m, "+", "=", "+"); mm.screenTopPct != screenTopLow {
		t.Errorf("+ past the limit: top %d%%", mm.screenTopPct)
	}
	mm := m
	for range 10 {
		mm = press(mm, "-")
	}
	if mm.screenTopPct != screenTopHigh || mm.layout().screenH < screenMinH {
		t.Errorf("- past the limit: top %d%%", mm.screenTopPct)
	}
	// Other layouts leave the size alone and say where it works.
	if mm := press(loaded(120, 60), "-"); mm.screenTopPct != screenTopDefault || !strings.Contains(mm.msg, "screen layout") {
		t.Errorf("- in split: top %d%%, msg %q", mm.screenTopPct, mm.msg)
	}
}

func TestLaunchFeedback(t *testing.T) {
	// A launch that dies at once shows its output in a popup.
	m := loaded(120, 30)
	next, _ := m.Update(actionMsg{err: &tmux.ExitError{Command: "claude", Status: 1, Output: []string{"error: unknown option '--bogus'"}}})
	m = next.(model)
	if m.popup != popupLaunchErr || !m.msgErr || !strings.Contains(m.msg, "claude exited right after start (status 1): error: unknown option") {
		t.Fatalf("popup %d msg %q", m.popup, m.msg)
	}
	if v := ansi.Strip(m.View()); !strings.Contains(v, "Launch failed") || !strings.Contains(v, "unknown option '--bogus'") {
		t.Errorf("launch error popup not shown:\n%s", v)
	}
	if m = press(m, "x"); m.popup != popupNone {
		t.Errorf("a key does not close the popup: %d", m.popup)
	}

	// A launched session is selected, focused and yellow once it shows up.
	m = press(loaded(120, 30), "2")
	next, _ = m.Update(actionMsg{text: "launched lazyai-c", launched: "lazyai-c"})
	m = next.(model)
	next, _ = m.Update(agentsMsg{agents: sample()})
	m = next.(model)
	a, _ := m.selected()
	if m.focus != listAgents || a.TmuxSession != "lazyai-c" || m.pending != "" {
		t.Fatalf("after launch: focus %d selected %q pending %q", m.focus, a.TmuxSession, m.pending)
	}
	now := time.Now()
	if !m.flashing(a, now) || m.flashing(a, now.Add(flashTime+time.Second)) {
		t.Error("launched row does not flash for flashTime")
	}
	// A session that never appears is given up after pendingTime.
	m.pending, m.pendingUntil = "lazyai-none", now.Add(-time.Second)
	if m.selectPending(now); m.pending != "" {
		t.Error("pending launch not given up")
	}
}

func TestScreenText(t *testing.T) {
	const esc = "\x1b"
	in := esc + "[38;5;153mblue" + esc + "[39m\tx\n" +
		esc + "[48;5;68mbg starts\n" + // background left on
		"still bg" + esc + "[0m plain\n" +
		"evil" + esc + "]52;c;aGVsbG8=" + "\a" + esc + "[2J" + esc + "[H" + esc + "]0;title" + esc + "\\" + "\r\bok" + esc + "c!"
	got := screenText(in)
	want := []string{
		esc + "[38;5;153mblue" + esc + "[39m    x",
		esc + "[38;5;153m" + esc + "[39m" + esc + "[48;5;68mbg starts",
		esc + "[38;5;153m" + esc + "[39m" + esc + "[48;5;68mstill bg" + esc + "[0m plain",
		"evilok!",
	}
	if len(got) != len(want) {
		t.Fatalf("lines: %q", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d:\n got %q\nwant %q", i, got[i], want[i])
		}
	}
	// "0;..." resets before setting: only it carries to the next line.
	if got := screenText(esc + "[31ma" + esc + "[0;1mb\nc"); got[1] != esc+"[0;1mc" {
		t.Errorf("0; reset carried as %q", got[1])
	}
	// Shown text is the same as without colors.
	if s := ansi.Strip(strings.Join(got, "\n")); s != "blue    x\nbg starts\nstill bg plain\nevilok!" {
		t.Errorf("stripped: %q", s)
	}
}

func TestImportantMark(t *testing.T) {
	m := loaded(120, 30)
	a, _ := m.selected()
	if m = press(m, "!"); !m.isImportant(a) || !strings.Contains(m.msg, "marked important") {
		t.Fatalf("! does not mark: msg %q", m.msg)
	}
	row := ""
	for _, l := range strings.Split(ansi.Strip(m.View()), "\n") {
		if strings.Contains(l, label(a)) {
			row = l
			break
		}
	}
	if !strings.Contains(row, label(a)+" "+importantMark) {
		t.Errorf("row has no important mark: %q", row)
	}
	// The mark stays on the agent when the selection moves.
	if m = press(m, "j"); !m.isImportant(a) {
		t.Error("mark lost when the selection moved")
	}
	if m = press(m, "k", "!"); m.isImportant(a) || !strings.Contains(m.msg, "no longer important") {
		t.Errorf("second ! does not unmark: msg %q", m.msg)
	}
	// The x menu offers it too.
	m = press(m, "x", "!")
	if !m.isImportant(a) || m.popup != popupNone {
		t.Errorf("menu ! does not mark: popup %d", m.popup)
	}
}

func TestStatusColors(t *testing.T) {
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.ANSI)
	busy, idle := statusText("busy"), statusText("idle")
	if busy == idle || ansi.Strip(busy) != "busy" || ansi.Strip(idle) != "idle" || busy == "busy" {
		t.Errorf("busy %q idle %q", busy, idle)
	}
	if got := statusText("other"); got != "other" {
		t.Errorf("unknown status styled: %q", got)
	}
	// The colored status does not change the row width.
	m := loaded(120, 30)
	for _, l := range strings.Split(m.View(), "\n") {
		if w := ansi.StringWidth(l); w > 120 {
			t.Errorf("line %d wide: %q", w, ansi.Strip(l))
		}
	}
	if !strings.Contains(m.View(), idle) {
		t.Error("idle status not colored in the list")
	}
}
