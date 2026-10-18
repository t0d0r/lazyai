package agent

import (
	"errors"
	"testing"
	"time"

	"lazyai/internal/tmux"
)

func TestClassifyClient(t *testing.T) {
	cases := []struct {
		args string
		kind Kind
		role Role
		ok   bool
	}{
		{"docker run -it --rm -v /w:/w -w /w node:22 claude", KindClaude, RoleInteractive, true},
		{"docker run -it --rm my-dev bash -lc claude --resume x", KindClaude, RoleInteractive, true},
		{"docker run -it --rm --name=dev node:22 npx @anthropic-ai/claude-code", "", "", false},
		{"docker run -d --name o ollama/ollama", KindOllama, RoleServer, true},
		{"docker --context colima container run -it alpine codex", KindCodex, RoleInteractive, true},
		{"docker exec -it -e TERM=xterm -u dev dev-1 claude", KindClaude, RoleInteractive, true},
		{"docker exec -it dev-1 /home/dev/.local/bin/claude --resume abc", KindClaude, RoleInteractive, true},
		{"docker exec -it dev-1 bash", "", "", false},
		{"docker compose -f dev.yml run --rm -e A=1 app claude", KindClaude, RoleInteractive, true},
		{"docker compose run --rm ollama", KindOllama, RoleInteractive, true},
		{"docker-compose exec app codex", KindCodex, RoleInteractive, true},
		{"docker compose exec app bash", "", "", false},
		{"docker compose up -d", "", "", false},
		{"docker sandbox run claude", KindClaude, RoleInteractive, true},
		{"docker sandbox run -w /w --name s gemini --yolo", KindGemini, RoleInteractive, true},
		{"docker sandbox ls", "", "", false},
		{"podman exec -it box aider --model x", KindAider, RoleInteractive, true},
	}
	for _, c := range cases {
		k, r, ok := Classify(c.args)
		if k != c.kind || r != c.role || ok != c.ok {
			t.Errorf("Classify(%q) = %q %q %v, want %q %q %v", c.args, k, r, ok, c.kind, c.role, c.ok)
		}
	}
}

func TestSameImage(t *testing.T) {
	if !sameImage("node", "docker.io/library/node:latest") || !sameImage("node:22", "node:22") ||
		sameImage("node:22", "node:20") || !sameImage("ghcr.io/a/b:1@sha256:x", "ghcr.io/a/b:1") {
		t.Error("sameImage")
	}
}

func TestParseDockerPSAndTop(t *testing.T) {
	cs := parseDockerPS([]byte("bc5222af9c3d\tdev-1\tnode:22\t2026-10-18 17:30:09 +0300 EEST\tapp\n\nbad line\n"))
	if len(cs) != 1 || cs[0].Name != "dev-1" || cs[0].Service != "app" ||
		!cs[0].Created.Equal(time.Date(2026, 10, 18, 14, 30, 9, 0, time.UTC)) {
		t.Fatalf("parseDockerPS = %+v", cs)
	}
	now := time.Now()
	top := []byte(`PID   PPID  UID   %CPU  RSS   TIME  ELAPSED  STAT  COMMAND
24498 24476 1001  0.5   1572  12    600      Ss    /bin/sh /tmp/b/claude --resume  x
24515 24498 1001  99.6  1504  3     3        R     codex
`)
	procs := parseTop(top, "test-parse", now)
	if len(procs) != 2 {
		t.Fatalf("parseTop: %+v", procs)
	}
	p := procs[0]
	if p.PID != 24498 || p.PPID != 24476 || p.UID != 1001 || p.CPU != 0.5 || p.RSS != 1572*1024 ||
		p.Stat != "Ss" || p.Args != "/bin/sh /tmp/b/claude --resume  x" || !p.Start.Equal(now.Add(-600*time.Second)) {
		t.Errorf("parseTop row: %+v", p)
	}
}

func TestContainerCPUWindow(t *testing.T) {
	tr := &ctrTracks{m: map[string]*ctrTrack{}}
	t0 := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	start := t0.Add(-time.Hour)
	// No window yet: ps's lifetime average.
	if s, cpu := tr.sample("c/1", t0, start, 100, 2.5); cpu != 2.5 || !s.Equal(start) {
		t.Errorf("first sample: %v %v", s, cpu)
	}
	// 2 CPU seconds in 10 s; the start computed from etimes moved by 1 s.
	if s, cpu := tr.sample("c/1", t0.Add(10*time.Second), start.Add(time.Second), 102, 2.5); cpu != 20 || !s.Equal(start) {
		t.Errorf("windowed: %v %v", s, cpu)
	}
	// A new process under the same PID starts over.
	if s, cpu := tr.sample("c/1", t0.Add(12*time.Second), t0, 0, 7); cpu != 7 || !s.Equal(t0) {
		t.Errorf("reused PID: %v %v", s, cpu)
	}
	tr.forget([]Container{{ID: "other"}})
	if len(tr.m) != 0 {
		t.Error("forget kept a stopped container")
	}
}

func TestMergeContainers(t *testing.T) {
	start := time.Date(2026, 10, 18, 12, 0, 0, 0, time.Local)
	procs := []Proc{
		{PID: 400, PPID: 1, TTY: "ttys004", Start: start, Args: "docker compose run --rm app claude"},
		{PID: 500, PPID: 1, TTY: "ttys001", Start: start, Args: "docker run -it --rm node:22 claude"},
		{PID: 600, PPID: 1, TTY: "ttys002", Start: start, Args: "docker exec -it box bash"},
		{PID: 700, PPID: 1, TTY: "ttys003", Start: start, Args: "docker exec -it box codex"},
	}
	panes := []tmux.Pane{{Session: "work", ID: "%2", TTY: "ttys002"}}
	agents := Build(procs, 1, "", panes)
	if len(agents) != 3 {
		t.Fatalf("Build found %d client agents, want 3", len(agents))
	}
	containers := []Container{
		// Same image and second as the docker run one: only compose may take it.
		{ID: "eeeeeeeeeeee5", Name: "p-app-run-1", Image: "node:22", Service: "app", Created: start.Add(time.Second), Procs: []Proc{
			{PID: 40, PPID: 0, CPU: 3, Start: start, Args: "claude"},
		}},
		{ID: "aaaaaaaaaaaa1", Name: "eager_turing", Image: "node:22", Created: start.Add(time.Second), Procs: []Proc{
			{PID: 10, PPID: 0, CPU: 12, RSS: 300 << 20, Start: start, Args: "claude"},
			{PID: 11, PPID: 10, CPU: 1, RSS: 10 << 20, Start: start, Args: "node mcp-server.js"},
			// lazyai's own PID 500 appears in the VM too; the command differs.
			{PID: 500, PPID: 0, Start: start, Args: "/usr/local/bin/gemini"},
		}},
		{ID: "bbbbbbbbbbbb2", Name: "box", Image: "dev", Created: start.Add(-time.Hour), Procs: []Proc{
			{PID: 1, PPID: 0, Start: start, Args: "sleep infinity"},
			{PID: 20, PPID: 0, Start: start, Args: "bash"},
			{PID: 21, PPID: 20, CPU: 5, Start: start, Args: "/usr/local/bin/claude"},
			{PID: 30, PPID: 0, CPU: 50, Start: start, Args: "codex"},
		}},
	}
	got := MergeContainers(agents, procs, containers, panes)
	byKey := map[string]Agent{}
	for _, a := range got {
		byKey[a.Key()] = a
	}
	if c := byKey["400"]; c.Container != "p-app-run-1" || c.CPU != 3 {
		t.Errorf("docker compose run row: %+v", c)
	}
	if len(got) != 5 {
		t.Fatalf("got %d agents: %v", len(got), byKey)
	}
	run := byKey["500"]
	if run.Kind != KindClaude || run.Container != "eager_turing" || run.CPU != 12 || run.Status != "busy" ||
		len(run.Children) != 2 || run.Children[0].Command != "[eager_turing] claude" || run.Children[1].Depth != 2 {
		t.Errorf("docker run row: %+v", run)
	}
	if exec := byKey["700"]; exec.Kind != KindCodex || exec.Container != "box" || exec.CPU != 50 {
		t.Errorf("docker exec row: %+v", exec)
	}
	typed := byKey["bbbbbbbbbbbb/21"]
	if typed.Kind != KindClaude || !typed.ForeignPID || typed.Container != "box" || typed.TmuxPane != "%2" {
		t.Errorf("claude typed in a shell: %+v", typed)
	}
	if g := byKey["aaaaaaaaaaaa/500"]; g.Kind != KindGemini || !g.ForeignPID || g.TmuxPane != "" {
		t.Errorf("gemini in the VM: %+v", g)
	}
	if err := Stop(typed, false, 0); !errors.Is(err, ErrForeign) {
		t.Errorf("Stop of a container agent: %v", err)
	}
}

// On Linux docker reports host PIDs, so the agent is already a host agent.
func TestMergeContainersHostPIDs(t *testing.T) {
	start := time.Date(2026, 10, 18, 12, 0, 0, 0, time.Local)
	procs := []Proc{
		{PID: 500, PPID: 1, Start: start, Args: "docker run -it --rm node:22 claude"},
		{PID: 900, PPID: 899, Start: start, CPU: 30, Args: "claude"},
		{PID: 950, PPID: 949, Start: start, Args: "codex"},
	}
	agents := Build(procs, 1, "", nil)
	containers := []Container{
		{ID: "cccccccccccc3", Name: "c1", Image: "node:22", Created: start, Procs: []Proc{{PID: 900, Args: "claude"}}},
		{ID: "dddddddddddd4", Name: "c2", Image: "x", Created: start, Procs: []Proc{{PID: 950, Args: "codex"}}},
	}
	got := MergeContainers(agents, procs, containers, nil)
	if len(got) != 2 {
		t.Fatalf("got %d agents: %+v", len(got), got)
	}
	for _, a := range got {
		switch a.PID {
		case 500:
			if a.Container != "c1" || a.CPU != 30 {
				t.Errorf("client row: %+v", a)
			}
		case 950:
			if a.Container != "c2" || a.ForeignPID {
				t.Errorf("host agent in a container: %+v", a)
			}
		default:
			t.Errorf("unexpected agent %+v", a)
		}
	}
}
