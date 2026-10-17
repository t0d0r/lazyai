package tmux

import (
	"os/exec"
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
	var pane Pane
	for _, p := range Panes() {
		if p.Session == session {
			pane = p
		}
	}
	if pane.ID == "" || !strings.HasPrefix(pane.TTY, "ttys") {
		t.Fatalf("pane not found: %+v", pane)
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
}
