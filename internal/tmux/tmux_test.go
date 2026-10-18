package tmux

import (
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
	screen, err := Capture(pane.ID, 100)
	if err != nil || strings.Count(screen, text) != 2 || strings.HasSuffix(screen, "\n") {
		t.Errorf("Capture: %v\n%q", err, screen)
	}
}
