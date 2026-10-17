// Package tmux launches agents in tmux sessions and talks to them there.
// Commands run without a shell, so names, paths and prompt text are never
// expanded or interpreted.
package tmux

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Prefix marks sessions created by lazyai.
const Prefix = "lazyai-"

// Pane is one tmux pane.
type Pane struct {
	Session string
	ID      string // "%3"
	TTY     string // "ttys003"
	PID     int
}

// Available reports whether tmux is installed.
func Available() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// Panes lists every pane on the default tmux server. A missing server is
// not an error: it simply has no panes.
func Panes() []Pane {
	out, err := exec.Command("tmux", "list-panes", "-a", "-F",
		"#{session_name}\t#{pane_id}\t#{pane_tty}\t#{pane_pid}").Output()
	if err != nil {
		return nil
	}
	var panes []Pane
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 4 {
			continue
		}
		pid, _ := strconv.Atoi(f[3])
		panes = append(panes, Pane{Session: f[0], ID: f[1], TTY: filepath.Base(f[2]), PID: pid})
	}
	return panes
}

// SessionName turns a user-supplied name into an lazyai session name.
func SessionName(name string) string {
	name = strings.TrimPrefix(name, Prefix)
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return Prefix + strings.Trim(b.String(), "-")
}

// HasSession reports whether a session with exactly this name exists.
func HasSession(session string) bool {
	return exec.Command("tmux", "has-session", "-t", "="+session).Run() == nil
}

// FreeSessionName returns SessionName(name), adding -2, -3 ... if taken.
func FreeSessionName(name string) string {
	base := SessionName(name)
	session := base
	for i := 2; HasSession(session); i++ {
		session = base + "-" + strconv.Itoa(i)
	}
	return session
}

// Launch starts argv in a new detached session rooted at dir.
func Launch(session, dir string, argv []string) error {
	if len(argv) == 0 {
		return errors.New("nothing to launch")
	}
	if !Available() {
		return errors.New("tmux is not installed")
	}
	path, err := exec.LookPath(argv[0])
	if err != nil {
		return fmt.Errorf("%s not found in PATH", argv[0])
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}
	if HasSession(session) {
		return fmt.Errorf("tmux session %s already exists", session)
	}
	args := append([]string{"new-session", "-d", "-s", session, "-c", dir, "--", path}, argv[1:]...)
	if out, err := exec.Command("tmux", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("tmux new-session: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// AttachCmd returns the command that puts the terminal on session. Inside
// tmux it switches the client; outside it attaches and returns on detach.
func AttachCmd(session string) *exec.Cmd {
	if os.Getenv("TMUX") != "" {
		return exec.Command("tmux", "switch-client", "-t", "="+session)
	}
	return exec.Command("tmux", "attach-session", "-t", "="+session)
}

// Capture returns a pane's visible screen plus up to history lines above
// it, as plain text without trailing blank lines.
func Capture(paneID string, history int) (string, error) {
	out, err := exec.Command("tmux", "capture-pane", "-p", "-t", paneID, "-S", "-"+strconv.Itoa(history)).Output()
	if err != nil {
		return "", fmt.Errorf("tmux capture-pane: %w", err)
	}
	return strings.TrimRight(string(out), " \n"), nil
}

// Send types text into a pane and presses Enter. The pause lets agents that
// treat fast input as a paste finish reading it before the Enter arrives.
func Send(paneID, text string) error {
	if out, err := exec.Command("tmux", "send-keys", "-t", paneID, "-l", "--", text).CombinedOutput(); err != nil {
		return fmt.Errorf("tmux send-keys: %s", strings.TrimSpace(string(out)))
	}
	time.Sleep(150 * time.Millisecond)
	if out, err := exec.Command("tmux", "send-keys", "-t", paneID, "Enter").CombinedOutput(); err != nil {
		return fmt.Errorf("tmux send-keys: %s", strings.TrimSpace(string(out)))
	}
	return nil
}
