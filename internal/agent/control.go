package agent

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

var (
	ErrProtected = errors.New("refusing to stop agentctl's own parent process")
	ErrNotOwner  = errors.New("process belongs to another user")
	ErrChanged   = errors.New("process exited or its PID was reused since it was listed")
)

// StopTimeout is how long Stop waits for an agent to exit after SIGTERM.
const StopTimeout = 5 * time.Second

// Stop ends an agent found by Discover. Without force it sends SIGTERM and
// waits up to timeout; with force it sends SIGKILL. Before signalling it
// checks that the PID still runs the same command started at the same time,
// so a reused PID is never hit.
func Stop(a Agent, force bool, timeout time.Duration) error {
	if a.Protected {
		return ErrProtected
	}
	if a.UID != os.Getuid() {
		return ErrNotOwner
	}
	cur, err := Lookup(a.PID)
	if err != nil || !cur.Start.Equal(a.Start) || cur.Args != a.Args {
		return ErrChanged
	}
	sig := syscall.SIGTERM
	if force {
		sig = syscall.SIGKILL
	}
	if err := syscall.Kill(a.PID, sig); err != nil {
		return fmt.Errorf("signal %d: %w", a.PID, err)
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !Alive(a) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("PID %d still running after %s; use force to kill it", a.PID, timeout)
}

// Alive reports whether the agent's process still exists as the same process.
func Alive(a Agent) bool {
	if syscall.Kill(a.PID, 0) != nil {
		return false
	}
	cur, err := Lookup(a.PID)
	return err == nil && cur.Start.Equal(a.Start) && cur.Stat[0] != 'Z'
}
