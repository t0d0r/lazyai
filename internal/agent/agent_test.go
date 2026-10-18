package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"lazyai/internal/tmux"
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

const brewPython = "/opt/homebrew/Cellar/python@3.14/3.14.8_2/Frameworks/Python.framework/Versions/3.14/Resources/Python.app/Contents/MacOS/Python"

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
		{"/opt/homebrew/bin/ollama runner --model m", KindOllama, RoleEngine, true},
		// Homebrew framework Python runs from Python.app; it must not be taken for a desktop app.
		{brewPython + " /opt/homebrew/Cellar/hermes-agent/0.22.0/libexec/bin/hermes", KindHermes, RoleInteractive, true},
		{brewPython + " /opt/homebrew/Cellar/hermes-agent/0.22.0/libexec/bin/hermes gateway run", KindHermes, RoleServer, true},
		{"/opt/homebrew/bin/python3 /x/hermes-agent/bin/hermes chat", KindHermes, RoleInteractive, true},
		{brewPython + " src/c/coli tune --model /m --ram 14", KindColibri, RoleJob, true},
		{"/opt/homebrew/bin/python3 src/c/coli serve --port 8000", KindColibri, RoleServer, true},
		{"/opt/homebrew/bin/python3 src/c/coli start --background", KindColibri, RoleServer, true},
		{"python3 /w/colibri/src/c/coli chat", KindColibri, RoleInteractive, true},
		{"/Users/u/work/AI/colibri/src/c/qwen36 69", KindColibri, RoleEngine, true},
		{"/Users/u/work/AI/colibri/src/c/colibri --model m", KindColibri, RoleEngine, true},
		{"/Users/u/bin/qwen36 69", "", "", false},
		{"/opt/homebrew/opt/node/bin/node /opt/homebrew/Cellar/qwen-code/0.25.2/libexec/lib/node_modules/@qwen-code/qwen-code/cli-entry.js", KindQwen, RoleInteractive, true},
		{"node /opt/homebrew/bin/qwen", KindQwen, RoleInteractive, true},
		{"/Applications/Xcode.app/Contents/Developer/usr/bin/make tune", "", "", false},
		// grok, copilot, mlx-lm
		{"grok", KindGrok, RoleInteractive, true},
		{"/Users/u/.grok/downloads/grok-1.0.62-macos-aarch64 --continue", KindGrok, RoleInteractive, true},
		{"/opt/homebrew/Caskroom/copilot-cli/1.0.48/copilot", KindCopilot, RoleInteractive, true},
		{"copilot --acp", KindCopilot, RoleServer, true},
		{"copilot -p fix the tests", KindCopilot, RoleJob, true},
		{"node /x/copilot.vim/dist/agent.js --stdio", "", "", false},
		{"/opt/homebrew/Cellar/mlx-lm/0.32.2/libexec/bin/python /opt/homebrew/bin/mlx_lm.server --model m", KindMLX, RoleServer, true},
		{"python3 -m mlx_lm.chat --model m", KindMLX, RoleInteractive, true},
		{"python3 -m mlx_lm generate --prompt hi", KindMLX, RoleJob, true},
		// desktop apps whose bundle names contain spaces
		{"/Applications/Grok Bot.app/Contents/MacOS/Grok Bot", "", "", false},
		{"/Applications/Grok Bot.app/Contents/Frameworks/Grok Bot Helper.app/Contents/MacOS/Grok Bot Helper --type=gpu-process", "", "", false},
		// containers
		{"docker run --rm -it -v /Users/u/.local/share/opencode:/root/.local/share/opencode -v /w:/workspace -w /workspace -e HOST_PWD=/w ghcr.io/anomalyco/opencode:latest", KindOpencode, RoleInteractive, true},
		{"/usr/local/bin/docker run -dit --name o -p 11434:11434 localhost:5000/ollama/ollama:0.4 serve", KindOllama, RoleServer, true},
		{"podman run --rm -it docker.io/library/aider@sha256:abc", KindAider, RoleInteractive, true},
		{"docker run --rm -it ubuntu bash", "", "", false},
		{"docker ps", "", "", false},
		{"docker run --rm -v /x/claude:/y alpine", "", "", false},
		{"/opt/homebrew/bin/bash -c source /tmp/hermes-snap.sh || true", "", "", false},
		{"/Applications/Claude.app/Contents/MacOS/Claude", "", "", false},
		{"/System/Library/PrivateFrameworks/X.framework/XPCServices/CursorUIViewService.xpc/Contents/MacOS/CursorUIViewService", "", "", false},
		{"grep claude", "", "", false},
		{"vim claude.md", "", "", false},
		{"tmux new-session -d -s lazyai-x -- /usr/local/bin/claude", "", "", false},
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
	panes := []tmux.Pane{{Session: "lazyai-p", ID: "%1", TTY: "ttys001"}}
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
	// PID 300 (codex, 50% CPU) runs under claude PID 200.
	if a.TreeCPU != 50 || len(a.Children) != 1 || a.Children[0].PID != 300 || a.Children[0].Depth != 1 {
		t.Errorf("claude tree: cpu %v children %+v", a.TreeCPU, a.Children)
	}
}

func TestDescendants(t *testing.T) {
	procs := []Proc{
		{PID: 10, PPID: 1, Args: "hermes"},
		{PID: 11, PPID: 10, CPU: 1, RSS: 100, Args: "make tune"},
		{PID: 12, PPID: 11, CPU: 2, RSS: 200, Args: "coli tune --api-key sk-1"},
		{PID: 13, PPID: 12, CPU: 90, RSS: 5000, Args: "qwen36 69"},
		{PID: 14, PPID: 10, Args: "bash"},
		{PID: 20, PPID: 1, Args: "other"},
	}
	kids := map[int][]Proc{}
	for _, p := range procs {
		kids[p.PPID] = append(kids[p.PPID], p)
	}
	got := descendants(kids, 10)
	var order []int
	for _, c := range got {
		order = append(order, c.PID*10+c.Depth)
	}
	if fmt.Sprint(order) != "[111 122 133 141]" {
		t.Errorf("order (pid*10+depth): %v", order)
	}
	if got[1].Command != "coli tune --api-key [redacted]" {
		t.Errorf("child command not redacted: %q", got[1].Command)
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
	if _, err := LaunchCommand(KindColibri, nil); err == nil {
		t.Error("colibri accepted")
	}
	if got, err := LaunchCommand(KindHermes, nil); err != nil || got[0] != "hermes" {
		t.Errorf("hermes: %v %v", got, err)
	}
	got, err := LaunchCommand(KindClaude, []string{"--model", "x"})
	if err != nil || len(got) != 3 || got[0] != "claude" {
		t.Errorf("got %v, %v", got, err)
	}
}
