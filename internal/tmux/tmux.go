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
	TTY     string // as ps names it: "ttys003" on macOS, "pts/3" on Linux
	PID     int
	// Clients is how many tmux clients show the pane's session now; 0 is
	// detached. LastAttached is when a client last attached, zero if never.
	Clients      int
	LastAttached time.Time
}

// Available reports whether tmux is installed.
func Available() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// Panes lists every pane on the default tmux server. A missing server is
// not an error: it simply has no panes. -u keeps the tabs: without a UTF-8
// locale (common on Linux in containers, cron or ssh) tmux prints them, and
// any non-ASCII session name, as "_".
func Panes() []Pane {
	out, err := exec.Command("tmux", "-u", "list-panes", "-a", "-F",
		"#{session_name}\t#{pane_id}\t#{pane_tty}\t#{pane_pid}\t#{session_attached}\t#{session_last_attached}").Output()
	if err != nil {
		return nil
	}
	return parsePanes(string(out))
}

// parsePanes reads the list-panes lines of Panes. Lines with another
// number of fields are skipped. Only line ends are trimmed: a session
// never attached has an empty last field, so its line ends in a tab.
func parsePanes(out string) []Pane {
	var panes []Pane
	for line := range strings.SplitSeq(strings.TrimRight(out, "\r\n"), "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 6 {
			continue
		}
		pid, _ := strconv.Atoi(f[3])
		clients, _ := strconv.Atoi(f[4])
		p := Pane{Session: f[0], ID: f[1], TTY: strings.TrimPrefix(f[2], "/dev/"), PID: pid, Clients: clients}
		if sec, err := strconv.ParseInt(f[5], 10, 64); err == nil && sec > 0 {
			p.LastAttached = time.Unix(sec, 0)
		}
		panes = append(panes, p)
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

// CheckSessionName refuses a name for RenameSession: empty, longer than
// 100 characters, or with characters other than letters, digits, - and _.
// tmux itself would change ":" and "." and allows spaces, which make a
// session hard to name on a command line.
func CheckSessionName(name string) error {
	if name == "" {
		return errors.New("empty session name")
	}
	if len(name) > 100 {
		return errors.New("session name longer than 100 characters")
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return fmt.Errorf("session name %q: use letters, digits, - and _", name)
		}
	}
	return nil
}

// RenameSession renames the tmux session from to to. The new name is
// taken as given, with no lazyai- prefix added.
func RenameSession(from, to string) error {
	if err := CheckSessionName(to); err != nil {
		return err
	}
	if from == to {
		return nil
	}
	if HasSession(to) {
		return fmt.Errorf("a tmux session named %s exists already", to)
	}
	out, err := exec.Command("tmux", "rename-session", "-t", "="+from, "--", to).CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("tmux rename-session: %s", msg)
		}
		return fmt.Errorf("tmux rename-session: %w", err)
	}
	return nil
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
	cmd := append([]string{path}, argv[1:]...)
	if len(cmd) == 1 {
		// tmux hands a single word to the shell. As two words it runs
		// them directly, and env then execs the agent in its own place.
		env, err := exec.LookPath("env")
		if err != nil {
			return errors.New("env not found in PATH")
		}
		if strings.Contains(path, "=") {
			return fmt.Errorf("%s: env would take a path with '=' for a variable", path)
		}
		cmd = []string{env, path}
	}
	// remain-on-exit, set in the same tmux call so no exit can come
	// first, keeps the pane and its output if the command dies at once.
	// The explicit target matters inside tmux, where an untargeted
	// set-option would change the window lazyai runs in.
	args := append([]string{"new-session", "-d", "-P", "-F", "#{pane_id}", "-s", session, "-c", dir, "--"}, cmd...)
	args = append(args, ";", "set-option", "-w", "-t", "="+session+":", "remain-on-exit", "on")
	out, err := exec.Command("tmux", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux new-session: %s", strings.TrimSpace(string(out)))
	}
	pane := strings.TrimSpace(string(out))
	if err := watchStart(session, pane, filepath.Base(argv[0]), LaunchWatch); err != nil {
		return err
	}
	SetStatusHint(session, false)
	return nil
}

// LaunchWatch is how long Launch waits for a new command to fail.
var LaunchWatch = 1500 * time.Millisecond

// launchLines is how many of the last output lines an ExitError keeps.
const launchLines = 20

// ExitError is a launched command that ended within LaunchWatch. Its
// session is gone; Output is what it printed, the last launchLines lines.
type ExitError struct {
	Command string
	Status  int
	Output  []string
}

func (e *ExitError) Error() string {
	msg := fmt.Sprintf("%s exited right after start (status %d)", e.Command, e.Status)
	if n := len(e.Output); n > 0 {
		msg += ": " + e.Output[n-1]
	}
	return msg
}

// watchStart waits up to d for the pane to die. A dead pane's output and
// exit status become an ExitError and its session is killed; a pane
// still running gets remain-on-exit off, so the session ends with the
// command as before.
func watchStart(session, pane, command string, d time.Duration) error {
	for end := time.Now().Add(d); ; {
		out, err := exec.Command("tmux", "display-message", "-p", "-t", pane,
			"#{pane_dead} #{pane_dead_status}").Output()
		if err != nil {
			return nil // gone already; nothing left to read
		}
		var dead, status int
		fmt.Sscan(string(out), &dead, &status)
		if dead == 1 {
			text, _ := exec.Command("tmux", "capture-pane", "-p", "-J", "-S", "-", "-t", pane).Output()
			exec.Command("tmux", "kill-session", "-t", "="+session).Run()
			return &ExitError{Command: command, Status: status, Output: deadOutput(string(text))}
		}
		if time.Now().After(end) {
			exec.Command("tmux", "set-option", "-w", "-t", pane, "remain-on-exit", "off").Run()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// deadOutput is the text of a dead pane without tmux's "Pane is dead"
// line and blank lines, the last launchLines lines.
func deadOutput(text string) []string {
	var lines []string
	for _, l := range strings.Split(text, "\n") {
		l = strings.TrimRight(l, " ")
		if l == "" || strings.HasPrefix(l, "Pane is dead") {
			continue
		}
		lines = append(lines, l)
	}
	if len(lines) > launchLines {
		lines = lines[len(lines)-launchLines:]
	}
	return lines
}

// AttachCmd returns the command that puts the terminal on session. Inside
// tmux it switches the client; outside it attaches and returns on detach.
// lazyai sessions get a status line hint for the way back.
func AttachCmd(session string) *exec.Cmd {
	nested := os.Getenv("TMUX") != ""
	if strings.HasPrefix(session, Prefix) {
		SetStatusHint(session, nested)
	}
	if nested {
		return exec.Command("tmux", "switch-client", "-t", "="+session)
	}
	return exec.Command("tmux", "attach-session", "-t", "="+session)
}

// StatusHint is the text on the right of a lazyai session's status line.
// When lazyai runs inside tmux the agent is reached with switch-client, so
// the way back is the last-session key (L), not detach (d).
func StatusHint(prefix string, nested bool) string {
	if nested {
		return prefix + " L: back to lazyai"
	}
	return prefix + " d: detach, back to lazyai"
}

// SetStatusHint puts StatusHint and the time on the right of session's
// status line. Only that session's options change. Errors are ignored: the
// hint is a convenience.
func SetStatusHint(session string, nested bool) {
	right := " " + StatusHint(prefixKey(), nested) + " | %H:%M "
	target := "=" + session + ":" // exact session; set-option takes a pane target
	exec.Command("tmux", "set-option", "-t", target, "status-right-length", "80").Run()
	exec.Command("tmux", "set-option", "-t", target, "status-right", right).Run()
}

// prefixKey returns the tmux prefix in readable form, such as "ctrl+b".
func prefixKey() string {
	out, err := exec.Command("tmux", "show-options", "-gv", "prefix").Output()
	if err != nil {
		return "ctrl+b"
	}
	return KeyName(strings.TrimSpace(string(out)))
}

// KeyName turns tmux key notation into readable form: "C-b" -> "ctrl+b",
// "M-a" -> "alt+a".
func KeyName(key string) string {
	var mods []string
	for {
		switch {
		case strings.HasPrefix(key, "C-"):
			mods, key = append(mods, "ctrl"), key[2:]
			continue
		case strings.HasPrefix(key, "M-"):
			mods, key = append(mods, "alt"), key[2:]
			continue
		case strings.HasPrefix(key, "S-"):
			mods, key = append(mods, "shift"), key[2:]
			continue
		}
		break
	}
	return strings.Join(append(mods, key), "+")
}

// Capture returns a pane's visible screen plus up to history lines above
// it, as plain text without trailing blank lines.
//
// With color, the text keeps the pane's colors and attributes as SGR
// escape sequences (capture-pane -e); the caller must not pass anything
// else from it to a terminal (see the TUI's sgrOnly).
func Capture(paneID string, history int, color bool) (string, error) {
	args := []string{"capture-pane", "-p", "-t", paneID, "-S", "-" + strconv.Itoa(history)}
	if color {
		args = append(args, "-e")
	}
	out, err := exec.Command("tmux", args...).Output()
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
