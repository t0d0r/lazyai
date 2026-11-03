package display

import (
	"strings"
	"testing"
	"time"

	"lazyai/internal/agent"
)

func TestAttached(t *testing.T) {
	last := time.Date(2026, 11, 3, 14, 2, 0, 0, time.Local)
	cases := []struct {
		clients int
		last    time.Time
		want    string
	}{
		{1, last, "attached (1 client)"},
		{2, last, "attached (2 clients)"},
		{0, last, "detached, last attached 2026-11-03 14:02"},
		{0, time.Time{}, "detached, never attached"},
	}
	for _, c := range cases {
		a := agent.Agent{TmuxSession: "s", TmuxPane: "%1", TmuxClients: c.clients, TmuxLastAttached: c.last}
		if got := Attached(a); got != c.want {
			t.Errorf("Attached(%d clients) = %q, want %q", c.clients, got, c.want)
		}
	}
	for _, f := range Details(agent.Agent{TmuxSession: "s", TmuxPane: "%1", TmuxClients: 2}) {
		if f.Name == "tmux" && !strings.Contains(f.Value, "s pane %1, attached (2 clients)") {
			t.Errorf("tmux detail: %q", f.Value)
		}
	}
}
