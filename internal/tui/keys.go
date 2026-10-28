package tui

import "lazyai/internal/agent"

type binding struct {
	keys, desc string
}

type bindingGroup struct {
	title    string
	bindings []binding
}

// keyHelp is what the ? popup shows.
var keyHelp = []bindingGroup{
	{"Navigation", []binding{
		{"tab / shift+tab", "next / previous panel"},
		{"l / h, right / left", "next / previous panel"},
		{"1 2 3", "Agents, Services, Models panel"},
		{"enter / esc", "focus the main panel / back to the list"},
		{"j / k, down / up", "move the selection, or scroll the main panel"},
		{"g / G", "first / last; in the main panel top / bottom"},
		{"pgdn / pgup", "scroll the main panel a page"},
		{"] / [", "next / previous tab"},
		{"mouse", "click to select, wheel to move or scroll"},
	}},
	{"Actions", []binding{
		{"x", "menu of actions for the selected agent"},
		{"a", "attach to the agent's tmux session"},
		{"p", "type a prompt into the agent and press Enter"},
		{"n", "launch a new agent in tmux"},
		{"s", "stop (SIGTERM)"},
		{"K", "kill (SIGKILL)"},
		{"H", "show / hide helper processes"},
		{"r", "refresh now"},
		{"?", "this help"},
		{"q, ctrl+c", "quit"},
	}},
	{"Marks", []binding{
		{attachMark + " before a row", "runs in tmux: a attach and p prompt work"},
		{"* after a row", "lazyai's own parent: not stopped from here"},
	}},
}

// attachMark starts the rows of agents that can be attached to.
const attachMark = "@"

// footerKeys are the short hints in the bottom line.
var (
	footerList = "x menu  a attach  p prompt  s stop  K kill  n new  [ ] tabs  ? help  q quit"
	footerMain = "j/k scroll  g/G top/bottom  [ ] tabs  esc back  x menu  ? help  q quit"
)

type menuItem struct {
	key, label string
	disabled   string // why the item cannot run, or ""
}

func (m model) menuItems() []menuItem {
	var items []menuItem
	if a, ok := m.selected(); ok {
		for _, it := range []struct{ key, label string }{
			{"a", "attach to tmux session"},
			{"p", "send a prompt"},
			{"s", "stop (SIGTERM)"},
			{"K", "kill (SIGKILL)"},
		} {
			reason := ""
			switch disabledReason(it.key, a) {
			case "":
			case agent.ErrProtected.Error():
				reason = "lazyai's parent"
			default:
				reason = "not in tmux"
			}
			items = append(items, menuItem{it.key, it.label, reason})
		}
	}
	helpers := "show helper processes"
	if m.showHelpers {
		helpers = "hide helper processes"
	}
	return append(items,
		menuItem{"n", "launch a new agent", ""},
		menuItem{"H", helpers, ""},
		menuItem{"r", "refresh", ""},
	)
}

// label is the name a list row shows for an agent.
func label(a agent.Agent) string {
	l, _ := rowLabel(a)
	return l
}

// rowLabel is label, and whether it is a path that shortenPath may cut.
func rowLabel(a agent.Agent) (string, bool) {
	switch {
	case a.Name != "":
		return a.Name, false
	case a.TmuxSession != "":
		return a.TmuxSession, false
	case a.Container != "" && a.CWD == "":
		return "docker:" + a.Container, false
	}
	return lastDirs(a.CWD), true
}
