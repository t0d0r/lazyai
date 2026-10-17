// Package watch follows agents across refreshes: how long each has been in
// its status, which warning flags apply, and what changed since last time.
// The TUI and the watch and list commands share it.
package watch

import (
	"sort"
	"strings"
	"time"

	"lazyai/internal/agent"
)

// Thresholds for flags.
const (
	HighCPU     = 80.0             // percent
	HighCPUFor  = 30 * time.Second // sustained this long before flagging
	HighMem     = 2 << 30          // bytes
	LongBusyFor = 30 * time.Minute
)

// Tracker is what earlier refreshes saw for one process.
type Tracker struct {
	Start        time.Time // process start, to notice PID reuse
	Status       string
	Since        time.Time // when Status was first seen
	Known        bool      // Since is a real transition, not just first sight
	HighCPUSince time.Time // zero while CPU is below HighCPU
}

// Event types.
const (
	EventStarted = "started" // new agent, or present on the first refresh
	EventExited  = "exited"
	EventStatus  = "status" // From and To are statuses
	EventFlags   = "flags"  // From and To are comma-separated flags
)

// Event is one change between two refreshes.
type Event struct {
	Time  time.Time   `json:"time"`
	Type  string      `json:"event"`
	From  string      `json:"from,omitempty"`
	To    string      `json:"to,omitempty"`
	Agent agent.Agent `json:"agent"`
}

// Watcher holds the state between refreshes. The zero value is not usable;
// call New.
type Watcher struct {
	track  map[int]Tracker
	agents map[int]agent.Agent
	flags  map[int]string
}

// New returns an empty Watcher.
func New() *Watcher {
	return &Watcher{track: map[int]Tracker{}, agents: map[int]agent.Agent{}, flags: map[int]string{}}
}

// Tracker returns the state for pid after the last Update.
func (w *Watcher) Tracker(pid int) (Tracker, bool) {
	t, ok := w.track[pid]
	return t, ok
}

// Update folds a fresh discovery into the state and returns what changed:
// exited agents first (by PID), then the rest in the order given.
func (w *Watcher) Update(agents []agent.Agent, now time.Time) []Event {
	var exited, events []Event
	track := make(map[int]Tracker, len(agents))
	seen := make(map[int]agent.Agent, len(agents))
	flags := make(map[int]string, len(agents))
	for _, a := range agents {
		t, existed := w.track[a.PID]
		if existed && !t.Start.Equal(a.Start) {
			exited = append(exited, Event{Time: now, Type: EventExited, Agent: w.agents[a.PID]})
			existed = false
		}
		switch {
		case !existed:
			t = Tracker{Start: a.Start, Status: a.Status, Since: now}
			events = append(events, Event{Time: now, Type: EventStarted, To: a.Status, Agent: a})
		case t.Status != a.Status:
			events = append(events, Event{Time: now, Type: EventStatus, From: t.Status, To: a.Status, Agent: a})
			t.Status, t.Since, t.Known = a.Status, now, true
		}
		if !a.StatusSince.IsZero() {
			t.Since, t.Known = a.StatusSince, true
		}
		switch {
		case a.CPU < HighCPU:
			t.HighCPUSince = time.Time{}
		case t.HighCPUSince.IsZero():
			t.HighCPUSince = now
		}
		f := strings.Join(Flags(a, t, now), ",")
		if existed && w.flags[a.PID] != f {
			events = append(events, Event{Time: now, Type: EventFlags, From: w.flags[a.PID], To: f, Agent: a})
		}
		track[a.PID], seen[a.PID], flags[a.PID] = t, a, f
	}
	for pid, a := range w.agents {
		if _, ok := seen[pid]; !ok {
			exited = append(exited, Event{Time: now, Type: EventExited, Agent: a})
		}
	}
	sort.Slice(exited, func(i, j int) bool { return exited[i].Agent.PID < exited[j].Agent.PID })
	w.track, w.agents, w.flags = track, seen, flags
	return append(exited, events...)
}

// Flags lists the warnings that apply to an agent.
func Flags(a agent.Agent, t Tracker, now time.Time) []string {
	var f []string
	if a.Protected {
		f = append(f, "self")
	}
	if !t.HighCPUSince.IsZero() && now.Sub(t.HighCPUSince) >= HighCPUFor {
		f = append(f, "high-cpu")
	}
	if a.RSS >= HighMem {
		f = append(f, "high-mem")
	}
	// Busy for the whole time watched counts even without a known transition.
	if a.Status == "busy" && !t.Since.IsZero() && now.Sub(t.Since) >= LongBusyFor {
		f = append(f, "long-busy")
	}
	return f
}
