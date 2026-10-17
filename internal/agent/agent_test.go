package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"agentctl/internal/tmux"
)

func TestParsePS(t *testing.T) {
	out := []byte(`  456     1   501   0.0  31552 ??       SNs  Tue Oct 13 09:30:00 2026     claude bg-pty-host --bg-pty-host /tmp/x.sock 200 50
47264 45511   501  22.5 463936 ttys031  S+   Sat Oct 17 21:20:54 2026     claude  --resume   abc
garbage line
`)
	procs, err := ParsePS(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(procs) != 2 {
		t.Fatalf("got %d procs, want 2", len(procs))
	}
	p := procs[1]
	if p.PID != 47264 || p.PPID != 45511 || p.UID != 501 || p.CPU != 22.5 || p.RSS != 463936*1024 {
		t.Errorf("numbers: %+v", p)
	}
	if p.TTY != "ttys031" || procs[0].TTY != "" {
		t.Errorf("tty: %q %q", p.TTY, procs[0].TTY)
	}
	if p.Args != "claude  --resume   abc" {
		t.Errorf("args spacing not preserved: %q", p.Args)
	}
	want := time.Date(2026, 10, 17, 21, 20, 54, 0, time.Local)
	if !p.Start.Equal(want) {
		t.Errorf("start %v, want %v", p.Start, want)
	}
}

func TestParseLstartSingleDigitDay(t *testing.T) {
	got, err := ParseLstart("Mon Oct  5 01:02:03 2026")
	if err != nil || got.Day() != 5 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		args string
		kind Kind
		role Role
		ok   bool
	}{
		{"claude", KindClaude, RoleInteractive, true},
		{"claude --resume ad98", KindClaude, RoleInteractive, true},
		{"/Users/u/.local/bin/claude daemon run --json-path x", KindClaude, RoleDaemon, true},
		{"claude bg-pty-host --bg-pty-host /tmp/a.sock 200 50 -- /x", KindClaude, RoleHelper, true},
		{"claude bg-spare --bg-spare /tmp/a.claim.sock", KindClaude, RoleHelper, true},
		{"/Users/u/.local/share/claude/versions/2.1.302", KindClaude, RoleInteractive, true},
		{"node /usr/lib/node_modules/@anthropic-ai/claude-code/cli.js", KindClaude, RoleInteractive, true},
		{"/Users/u/.codex/packages/x/bin/codex app-server --listen unix://", KindCodex, RoleServer, true},
		{"/Users/u/.codex/packages/x/bin/codex app-server daemon pid-update-loop", KindCodex, RoleDaemon, true},
		{"codex", KindCodex, RoleInteractive, true},
		{"/opt/homebrew/Cellar/opencode/2.0.25/bin/opencode serve --service", KindOpencode, RoleServer, true},
		{"/opt/homebrew/opt/node/bin/node --no-warnings /opt/homebrew/Cellar/gemini-cli/0.46.0/libexec/lib/node_modules/@google/gemini-cli/bundle/gemini.js", KindGemini, RoleInteractive, true},
		{"/opt/homebrew/Cellar/aider/0.86.2/libexec/bin/python /opt/homebrew/bin/aider --model x", KindAider, RoleInteractive, true},
		{"node /opt/homebrew/Caskroom/cursor-cli/2026.10.16/dist-package/index.js", KindCursor, RoleInteractive, true},
		{"/opt/homebrew/bin/ollama serve", KindOllama, RoleServer, true},
		{"/opt/homebrew/bin/ollama runner --model m", KindOllama, RoleHelper, true},
		{"/Applications/Claude.app/Contents/MacOS/Claude", "", "", false},
		{"/System/Library/PrivateFrameworks/X.framework/XPCServices/CursorUIViewService.xpc/Contents/MacOS/CursorUIViewService", "", "", false},
		{"grep claude", "", "", false},
		{"vim claude.md", "", "", false},
		{"tmux new-session -d -s agentctl-x -- /usr/local/bin/claude", "", "", false},
		{"", "", "", false},
	}
	for _, c := range cases {
		k, r, ok := Classify(c.args)
		if k != c.kind || r != c.role || ok != c.ok {
			t.Errorf("Classify(%q) = %q %q %v, want %q %q %v", c.args, k, r, ok, c.kind, c.role, c.ok)
		}
	}
}

func TestRedactArgs(t *testing.T) {
	cases := map[string]string{
		"aider --api-key sk-123 --model x":          "aider --api-key [redacted] --model x",
		"codex --openai-api-key=sk-1":               "codex --openai-api-key=[redacted]",
		"x ANTHROPIC_AUTH_TOKEN=abc y":              "x ANTHROPIC_AUTH_TOKEN=[redacted] y",
		"claude --password hunter2":                 "claude --password [redacted]",
		"codex app-server --listen unix://":         "codex app-server --listen unix://",
		"x --token --verbose":                       "x --token --verbose",
		"claude daemon run --json-path /tmp/d.json": "claude daemon run --json-path /tmp/d.json",
	}
	for in, want := range cases {
		if got := RedactArgs(in); got != want {
			t.Errorf("RedactArgs(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAncestors(t *testing.T) {
	procs := []Proc{{PID: 1, PPID: 0}, {PID: 10, PPID: 1}, {PID: 20, PPID: 10}, {PID: 30, PPID: 20}, {PID: 40, PPID: 10}}
	got := Ancestors(procs, 30)
	for _, pid := range []int{30, 20, 10} {
		if !got[pid] {
			t.Errorf("%d should be an ancestor", pid)
		}
	}
	if got[40] || got[1] {
		t.Errorf("unexpected ancestors: %v", got)
	}
}

func TestBuildClaudeSessionAndTmux(t *testing.T) {
	dir := t.TempDir()
	start := time.Date(2026, 10, 17, 21, 20, 54, 0, time.Local)
	meta := `{"pid":200,"sessionId":"s-1","cwd":"/w/p","name":"p-1","status":"busy","statusUpdatedAt":1792272489609,"procStart":"` +
		start.UTC().Format("Mon Jan 2 15:04:05 2006") + `"}`
	if err := os.WriteFile(filepath.Join(dir, "200.json"), []byte(meta), 0o600); err != nil {
		t.Fatal(err)
	}
	stale := `{"pid":201,"sessionId":"old","procStart":"Mon Jan 5 10:00:00 2026","status":"busy"}`
	if err := os.WriteFile(filepath.Join(dir, "201.json"), []byte(stale), 0o600); err != nil {
		t.Fatal(err)
	}
	procs := []Proc{
		{PID: 100, PPID: 1, Args: "zsh"},
		{PID: 200, PPID: 100, TTY: "ttys001", Start: start, Args: "claude"},
		{PID: 201, PPID: 1, TTY: "ttys002", Start: start, Args: "claude --resume r-9"},
		{PID: 300, PPID: 200, CPU: 50, Start: start, Args: "codex"},
	}
	panes := []tmux.Pane{{Session: "agentctl-p", ID: "%1", TTY: "ttys001"}}
	agents := Build(procs, 300, dir, panes)
	if len(agents) != 3 {
		t.Fatalf("got %d agents, want 3", len(agents))
	}
	a := agents[0]
	if a.PID != 200 || a.Name != "p-1" || a.SessionID != "s-1" || a.CWD != "/w/p" ||
		a.Status != "busy" || a.StatusSource != SourceClaude || a.TmuxPane != "%1" || !a.Protected {
		t.Errorf("claude agent: %+v", a)
	}
	b := agents[1]
	if b.SessionID != "r-9" || b.StatusSource != SourceCPU || b.Status != "idle" || b.TmuxSession != "" || b.Protected {
		t.Errorf("stale metadata must be ignored: %+v", b)
	}
	if c := agents[2]; c.Kind != KindCodex || c.Status != "busy" || !c.Protected {
		t.Errorf("codex agent: %+v", c)
	}
}

func TestStopProtectedAndChanged(t *testing.T) {
	if err := Stop(Agent{Proc: Proc{PID: 1}, Protected: true}, false, time.Second); err != ErrProtected {
		t.Errorf("protected: %v", err)
	}
	self, err := Lookup(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	moved := self
	moved.Start = self.Start.Add(-time.Hour)
	if err := Stop(Agent{Proc: moved}, false, time.Second); err != ErrChanged {
		t.Errorf("changed: %v", err)
	}
}

func TestStopSendsSignal(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { cmd.Wait(); close(done) }()
	var p Proc
	var err error
	for range 20 { // ps may not see the child for a moment
		if p, err = Lookup(cmd.Process.Pid); err == nil && p.Args == "sleep 30" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := Stop(Agent{Proc: p}, false, 3*time.Second); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("process still running")
	}
}

func TestLaunchCommand(t *testing.T) {
	if _, err := LaunchCommand("bash", nil); err == nil {
		t.Error("unknown kind accepted")
	}
	if _, err := LaunchCommand(KindOllama, nil); err == nil {
		t.Error("ollama without args accepted")
	}
	got, err := LaunchCommand(KindClaude, []string{"--model", "x"})
	if err != nil || len(got) != 3 || got[0] != "claude" {
		t.Errorf("got %v, %v", got, err)
	}
}
