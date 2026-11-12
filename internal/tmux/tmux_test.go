package tmux

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSessionName(t *testing.T) {
	cases := map[string]string{
		"claude-ai":          "lazyai-claude-ai",
		"lazyai-x":           "lazyai-x",
		"my agent; rm -rf /": "lazyai-my-agent--rm--rf",
		"a.b:c":              "lazyai-a-b-c",
	}
	for in, want := range cases {
		if got := SessionName(in); got != want {
			t.Errorf("SessionName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestKeyName(t *testing.T) {
	cases := map[string]string{"C-b": "ctrl+b", "C-a": "ctrl+a", "M-Space": "alt+Space", "C-M-x": "ctrl+alt+x", "`": "`"}
	for in, want := range cases {
		if got := KeyName(in); got != want {
			t.Errorf("KeyName(%q) = %q, want %q", in, got, want)
		}
	}
	if StatusHint("ctrl+b", true) != "ctrl+b L: back to lazyai" {
		t.Error("nested hint")
	}
}

// TestLaunchSendPanes runs /bin/cat in a real tmux session on the default
// server, types into it and checks the pane shows the text unexpanded.
func TestLaunchSendPanes(t *testing.T) {
	if !Available() {
		t.Skip("tmux not installed")
	}
	session := FreeSessionName("test-" + time.Now().Format("150405.000"))
	if err := Launch(session, t.TempDir(), []string{"cat"}); err != nil {
		t.Fatal(err)
	}
	defer exec.Command("tmux", "kill-session", "-t", "="+session).Run()

	if err := Launch(session, t.TempDir(), []string{"cat"}); err == nil {
		t.Error("second launch with the same name succeeded")
	}
	right, _ := exec.Command("tmux", "show-options", "-t", "="+session+":", "-v", "status-right").Output()
	if !strings.Contains(string(right), " d: detach, back to lazyai") {
		t.Errorf("status-right: %q", right)
	}
	global, _ := exec.Command("tmux", "show-options", "-gv", "status-right").Output()
	if strings.Contains(string(global), "lazyai") {
		t.Error("hint leaked into the global status-right")
	}
	var pane Pane
	for _, p := range Panes() {
		if p.Session == session {
			pane = p
		}
	}
	if pane.ID == "" {
		t.Fatalf("pane not found: %+v", pane)
	}
	// Discovery matches agents to panes by tty, so the name must be ps's.
	psTTY, _ := exec.Command("ps", "-o", "tty=", "-p", strconv.Itoa(pane.PID)).Output()
	if got := strings.TrimSpace(string(psTTY)); pane.TTY != got {
		t.Errorf("pane tty %q, ps says %q", pane.TTY, got)
	}
	// No shell in between: the pane runs cat itself.
	psArgs, _ := exec.Command("ps", "-o", "args=", "-p", strconv.Itoa(pane.PID)).Output()
	if got := strings.TrimSpace(string(psArgs)); filepath.Base(got) != "cat" {
		t.Errorf("pane runs %q, want cat", got)
	}
	text := `hello $HOME "q" $(id)`
	if err := Send(pane.ID, text); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	out, err := exec.Command("tmux", "capture-pane", "-p", "-t", pane.ID).Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(out), text) != 2 { // typed echo plus cat's output
		t.Errorf("pane content:\n%s", out)
	}
	screen, err := Capture(pane.ID, 100, false)
	if err != nil || strings.Count(screen, text) != 2 || strings.HasSuffix(screen, "\n") {
		t.Errorf("Capture: %v\n%q", err, screen)
	}

	// Rename: the session and its pane go on under the new name.
	renamed := FreeSessionName(strings.TrimPrefix(session, Prefix) + "-renamed")
	defer exec.Command("tmux", "kill-session", "-t", "="+renamed).Run()
	if err := RenameSession(session, renamed); err != nil {
		t.Fatal(err)
	}
	if HasSession(session) || !HasSession(renamed) {
		t.Errorf("after rename: old %v new %v", HasSession(session), HasSession(renamed))
	}
	if err := RenameSession(renamed, renamed); err != nil {
		t.Errorf("rename to the same name: %v", err)
	}
	other := FreeSessionName("test-other")
	if err := Launch(other, t.TempDir(), []string{"cat"}); err != nil {
		t.Fatal(err)
	}
	defer exec.Command("tmux", "kill-session", "-t", "="+other).Run()
	if err := RenameSession(renamed, other); err == nil || !strings.Contains(err.Error(), "exists already") {
		t.Errorf("rename onto a taken name: %v", err)
	}
	if err := RenameSession("lazyai-no-such-session-x", "lazyai-no-such-y"); err == nil {
		t.Error("renaming a missing session succeeded")
	}
}

func TestCheckSessionName(t *testing.T) {
	for _, ok := range []string{"web", "lazyai-web_2", "A-b"} {
		if err := CheckSessionName(ok); err != nil {
			t.Errorf("%q refused: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "a b", "a:b", "a.b", "web/x", "\u00e9", strings.Repeat("a", 101)} {
		if CheckSessionName(bad) == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestParsePanes(t *testing.T) {
	out := "work\t%1\t/dev/ttys001\t100\t2\t1762151400\n" +
		"lazyai-x\t%2\t/dev/pts/3\t200\t0\t\n" +
		"short\t%3\n" +
		"last\t%4\t/dev/pts/4\t300\t0\t\n" // tmux 3.4: empty last field on the last line
	got := parsePanes(out)
	if len(got) != 3 || got[2].Session != "last" {
		t.Fatalf("panes: %+v", got)
	}
	if p := got[0]; p.Session != "work" || p.TTY != "ttys001" || p.PID != 100 || p.Clients != 2 ||
		!p.LastAttached.Equal(time.Unix(1762151400, 0)) {
		t.Errorf("attached pane: %+v", p)
	}
	if p := got[1]; p.TTY != "pts/3" || p.Clients != 0 || !p.LastAttached.IsZero() {
		t.Errorf("never attached pane: %+v", p)
	}
}

// TestLaunchEarlyExit launches commands that die at once and one that
// keeps running, in real tmux sessions on the default server.
func TestLaunchEarlyExit(t *testing.T) {
	if !Available() {
		t.Skip("tmux not installed")
	}
	defer func(d time.Duration) { LaunchWatch = d }(LaunchWatch)
	LaunchWatch = 500 * time.Millisecond
	name := func(s string) string { return FreeSessionName("test-" + s + "-" + time.Now().Format("150405.000")) }

	session := name("exit")
	err := Launch(session, t.TempDir(), []string{"sh", "-c", "echo boom; echo why >&2; exit 3"})
	var e *ExitError
	if !errors.As(err, &e) || e.Command != "sh" || e.Status != 3 || strings.Join(e.Output, "|") != "boom|why" {
		t.Fatalf("early exit: %#v (%v)", e, err)
	}
	if !strings.Contains(err.Error(), "sh exited right after start (status 3): why") {
		t.Errorf("message: %v", err)
	}
	if HasSession(session) {
		exec.Command("tmux", "kill-session", "-t", "="+session).Run()
		t.Error("session of a dead command left behind")
	}

	// A single word runs through env, whose own error is kept.
	if err := Launch(name("env"), t.TempDir(), []string{"false"}); !errors.As(err, &e) || e.Status != 1 {
		t.Errorf("false: %v", err)
	}

	session = name("run")
	if err := Launch(session, t.TempDir(), []string{"cat"}); err != nil {
		t.Fatal(err)
	}
	defer exec.Command("tmux", "kill-session", "-t", "="+session).Run()
	opt, _ := exec.Command("tmux", "show-options", "-w", "-v", "-t", "="+session+":", "remain-on-exit").Output()
	if strings.TrimSpace(string(opt)) != "off" {
		t.Errorf("running command: remain-on-exit %q, want off", opt)
	}
}

func TestDeadOutput(t *testing.T) {
	text := "a\n\nb  \n" + strings.Repeat("x\n", 30) + "\nPane is dead (status 1, Fri Nov  6 23:35:28 2026)\n"
	got := deadOutput(text)
	if len(got) != launchLines || got[0] != "x" {
		t.Errorf("deadOutput: %d lines, first %q", len(got), got[0])
	}
	if got := deadOutput("only\n\nPane is dead (status 2)\n"); strings.Join(got, "|") != "only" {
		t.Errorf("deadOutput: %q", got)
	}
}
