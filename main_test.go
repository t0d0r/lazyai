package main

import (
	"strings"
	"testing"
	"time"

	"lazyai/internal/agent"
	"lazyai/internal/watch"
)

func TestResolve(t *testing.T) {
	agents := []agent.Agent{
		{Proc: agent.Proc{PID: 10}, Role: agent.RoleInteractive, TmuxSession: "lazyai-claude-x", Name: "x-1"},
		{Proc: agent.Proc{PID: 11}, Role: agent.RoleHelper, TmuxSession: "lazyai-claude-x"},
		{Proc: agent.Proc{PID: 20}, Role: agent.RoleInteractive, Name: "dup"},
		{Proc: agent.Proc{PID: 21}, Role: agent.RoleInteractive, Name: "dup"},
	}
	for ref, want := range map[string]int{"10": 10, "11": 11, "lazyai-claude-x": 10, "claude-x": 10, "x-1": 10} {
		a, err := resolve(agents, ref)
		if err != nil || a.PID != want {
			t.Errorf("resolve(%q) = %d, %v; want %d", ref, a.PID, err, want)
		}
	}
	for _, ref := range []string{"99", "nope", "dup"} {
		if _, err := resolve(agents, ref); err == nil {
			t.Errorf("resolve(%q) succeeded", ref)
		}
	}
}

func TestFormatEvent(t *testing.T) {
	at := time.Date(2026, 10, 18, 1, 2, 3, 0, time.Local)
	a := agent.Agent{Proc: agent.Proc{PID: 42}, Kind: agent.KindCodex, Role: agent.RoleInteractive, CWD: "/w"}
	got := formatEvent(watch.Event{Time: at, Type: watch.EventFlags, To: "high-mem", Agent: a})
	for _, part := range []string{"01:02:03", "flags", "42", "codex", "- -> high-mem", "/w"} {
		if !strings.Contains(got, part) {
			t.Errorf("%q lacks %q", got, part)
		}
	}
}
