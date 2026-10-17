package watch

import (
	"strings"
	"testing"
	"time"

	"lazyai/internal/agent"
)

func types(events []Event) string {
	var s []string
	for _, e := range events {
		s = append(s, e.Type+":"+e.From+">"+e.To)
	}
	return strings.Join(s, " ")
}

func TestTrackerAndFlags(t *testing.T) {
	t0 := time.Now()
	a := agent.Agent{Proc: agent.Proc{PID: 1, Start: t0.Add(-time.Hour), CPU: 95, RSS: 3 << 30}, Status: "busy"}
	w := New()
	w.Update([]agent.Agent{a}, t0)
	tr, _ := w.Tracker(1)
	if got := strings.Join(Flags(a, tr, t0), ","); got != "high-mem" || tr.Known {
		t.Errorf("first look: %v known=%v", got, tr.Known)
	}
	later := t0.Add(31 * time.Minute)
	w.Update([]agent.Agent{a}, later)
	tr, _ = w.Tracker(1)
	if got := strings.Join(Flags(a, tr, later), ","); got != "high-cpu,high-mem,long-busy" {
		t.Errorf("after 31m: %v", got)
	}
	a.Status, a.CPU = "idle", 0
	w.Update([]agent.Agent{a}, later.Add(time.Minute))
	tr, _ = w.Tracker(1)
	if !tr.HighCPUSince.IsZero() || !tr.Since.Equal(later.Add(time.Minute)) || !tr.Known {
		t.Errorf("status change not tracked: %+v", tr)
	}
}

func TestClaudeStatusSinceIsKnown(t *testing.T) {
	since := time.Now().Add(-10 * time.Minute)
	a := agent.Agent{Proc: agent.Proc{PID: 1}, Status: "idle", StatusSince: since}
	w := New()
	w.Update([]agent.Agent{a}, time.Now())
	if tr, _ := w.Tracker(1); !tr.Known || !tr.Since.Equal(since) {
		t.Errorf("tracker: %+v", tr)
	}
}

func TestEvents(t *testing.T) {
	t0 := time.Now()
	start := t0.Add(-time.Hour)
	a := agent.Agent{Proc: agent.Proc{PID: 1, Start: start}, Status: "idle"}
	b := agent.Agent{Proc: agent.Proc{PID: 2, Start: start}, Status: "idle"}
	w := New()
	if got := types(w.Update([]agent.Agent{a, b}, t0)); got != "started:>idle started:>idle" {
		t.Errorf("first: %s", got)
	}
	if got := types(w.Update([]agent.Agent{a, b}, t0.Add(time.Second))); got != "" {
		t.Errorf("no change: %s", got)
	}
	a.Status, a.RSS = "busy", 3<<30
	if got := types(w.Update([]agent.Agent{a}, t0.Add(2*time.Second))); got != "exited:> status:idle>busy flags:>high-mem" {
		t.Errorf("change: %s", got)
	}
	reused := agent.Agent{Proc: agent.Proc{PID: 1, Start: t0}, Status: "idle"}
	got := w.Update([]agent.Agent{reused}, t0.Add(3*time.Second))
	if types(got) != "exited:> started:>idle" || got[0].Agent.Start != start {
		t.Errorf("reuse: %s", types(got))
	}
}
