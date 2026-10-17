package tui

import (
	"time"

	"agentctl/internal/agent"
)

// Watch thresholds.
const (
	HighCPU     = 80.0             // percent
	HighCPUFor  = 30 * time.Second // sustained this long before flagging
	HighMem     = 2 << 30          // bytes
	LongBusyFor = 30 * time.Minute
)

// tracker remembers what earlier refreshes saw for one process.
type tracker struct {
	start        time.Time // process start, to notice PID reuse
	status       string
	since        time.Time // when status was first seen
	known        bool      // since is a real transition, not just first sight
	highCPUSince time.Time // zero while CPU is below HighCPU
}

// observe folds a fresh snapshot into the trackers and drops gone PIDs.
func observe(track map[int]tracker, agents []agent.Agent, now time.Time) map[int]tracker {
	next := make(map[int]tracker, len(agents))
	for _, a := range agents {
		t, ok := track[a.PID]
		if !ok || !t.start.Equal(a.Start) {
			t = tracker{start: a.Start, status: a.Status, since: now}
		}
		if t.status != a.Status {
			t.status, t.since, t.known = a.Status, now, true
		}
		if !a.StatusSince.IsZero() {
			t.since, t.known = a.StatusSince, true
		}
		switch {
		case a.CPU < HighCPU:
			t.highCPUSince = time.Time{}
		case t.highCPUSince.IsZero():
			t.highCPUSince = now
		}
		next[a.PID] = t
	}
	return next
}

// flags lists the warnings shown for an agent.
func flags(a agent.Agent, t tracker, now time.Time) []string {
	var f []string
	if a.Protected {
		f = append(f, "self")
	}
	if !t.highCPUSince.IsZero() && now.Sub(t.highCPUSince) >= HighCPUFor {
		f = append(f, "high-cpu")
	}
	if a.RSS >= HighMem {
		f = append(f, "high-mem")
	}
	// Busy for the whole time watched counts even without a known transition.
	if a.Status == "busy" && !t.since.IsZero() && now.Sub(t.since) >= LongBusyFor {
		f = append(f, "long-busy")
	}
	return f
}
