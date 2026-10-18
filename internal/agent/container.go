package agent

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"lazyai/internal/tmux"
)

// Container is one running container and the processes in it, as docker
// reports them. On macOS the PIDs are those of Docker's Linux VM.
type Container struct {
	ID, Name, Image string
	Service         string // compose service, from its label
	Created         time.Time
	Procs           []Proc
}

// dockerTimeout bounds each docker call, so a hung daemon cannot stall a
// refresh.
const dockerTimeout = 3 * time.Second

// topFormat lists the ps columns docker top is asked for: 8 words, then
// the command line. times is CPU seconds used, etimes seconds since start.
const topFormat = "pid,ppid,uid,pcpu,rss,times,etimes,stat,args"

// Containers asks docker for its running containers and the processes in
// them. No docker, no daemon, or no permission means no containers.
func Containers() []Container {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), dockerTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "docker", "ps", "--no-trunc", "--format",
		`{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.CreatedAt}}\t{{.Label "com.docker.compose.service"}}`).Output()
	if err != nil {
		return nil
	}
	cs := parseDockerPS(out)
	now := time.Now()
	var wg sync.WaitGroup
	for i := range cs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			top, err := exec.CommandContext(ctx, "docker", "top", cs[i].ID, "-o", topFormat).Output()
			if err == nil {
				cs[i].Procs = parseTop(top, cs[i].ID, now)
			}
		}()
	}
	wg.Wait()
	ctrCPU.forget(cs)
	return cs
}

// parseDockerPS reads the docker ps --format line of Containers.
func parseDockerPS(out []byte) []Container {
	var cs []Container
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		f := strings.Split(sc.Text(), "\t")
		if len(f) != 5 || f[0] == "" {
			continue
		}
		created, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", f[3])
		cs = append(cs, Container{ID: f[0], Name: f[1], Image: f[2], Created: created, Service: f[4]})
	}
	return cs
}

// parseTop reads docker top output in topFormat. The header and malformed
// lines are skipped.
func parseTop(out []byte, id string, now time.Time) []Proc {
	var procs []Proc
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		f, args := splitFields(sc.Text(), 8)
		if len(f) < 8 || args == "" {
			continue
		}
		var n [7]int64
		ok := true
		for i, s := range []string{f[0], f[1], f[2], f[4], f[5], f[6]} {
			v, err := strconv.ParseInt(s, 10, 64)
			n[i], ok = v, ok && err == nil
		}
		cpu, err := strconv.ParseFloat(f[3], 64)
		if !ok || err != nil {
			continue // the header
		}
		pid := int(n[0])
		elapsed := time.Duration(n[5]) * time.Second
		start, recent := ctrCPU.sample(id+"/"+f[0], now, now.Add(-elapsed), n[4], cpu)
		procs = append(procs, Proc{PID: pid, PPID: int(n[1]), UID: int(n[2]), CPU: recent,
			RSS: n[3] * 1024, Stat: f[7], Start: start, Args: args})
	}
	return procs
}

// cpuWindow is how far back container CPU is measured. docker top reports
// CPU time in whole seconds, so a short window would only ever show 0% or
// a multiple of 50%.
const (
	cpuWindow    = 30 * time.Second
	cpuWindowMin = 4 * time.Second
)

type ctrSample struct {
	at   time.Time
	used int64 // CPU seconds
}

type ctrTrack struct {
	start   time.Time
	samples []ctrSample // oldest first, within cpuWindow
}

// ctrTracks keeps recent samples per container process ("ID/PID") across
// refreshes, for CPU over the last cpuWindow and a steady start time.
type ctrTracks struct {
	sync.Mutex
	m map[string]*ctrTrack
}

var ctrCPU = &ctrTracks{m: map[string]*ctrTrack{}}

// sample records CPU seconds used for key and returns the process start
// and its CPU use over the window. start is now minus the elapsed seconds,
// so it moves by a second between refreshes; the first value is kept while
// it stays within 2 s, so the watcher does not take it for a new process.
// Until there is a window of cpuWindowMin, ps's own pcpu (the lifetime
// average) is returned.
func (t *ctrTracks) sample(key string, now, start time.Time, used int64, pcpu float64) (time.Time, float64) {
	t.Lock()
	defer t.Unlock()
	tr := t.m[key]
	if tr == nil || absDuration(tr.start.Sub(start)) > 2*time.Second {
		tr = &ctrTrack{start: start}
		t.m[key] = tr
	}
	for len(tr.samples) > 1 && now.Sub(tr.samples[0].at) > cpuWindow {
		tr.samples = tr.samples[1:]
	}
	cpu := pcpu
	if len(tr.samples) > 0 {
		first := tr.samples[0]
		if age := now.Sub(first.at); age >= cpuWindowMin && used >= first.used {
			cpu = float64(used-first.used) / age.Seconds() * 100
		}
	}
	tr.samples = append(tr.samples, ctrSample{at: now, used: used})
	return tr.start, cpu
}

// forget drops tracks of containers that are no longer running.
func (t *ctrTracks) forget(running []Container) {
	ids := map[string]bool{}
	for _, c := range running {
		ids[c.ID] = true
	}
	t.Lock()
	defer t.Unlock()
	for key := range t.m {
		id, _, _ := strings.Cut(key, "/")
		if !ids[id] {
			delete(t.m, key)
		}
	}
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// client is a docker or podman process on this machine.
type client struct {
	proc Proc
	cmd  clientCmd
	used bool // a run client has found its container
}

// MergeContainers adds the agents running inside containers to agents,
// which Build made from procs.
//
// An agent inside a container is folded into the docker client row that
// started it (docker run IMAGE claude, docker exec CTR claude, docker
// compose run/exec SERVICE claude): the row keeps the client's PID, which
// can be stopped and has the tmux pane, and takes the agent's CPU, memory
// and status, as the client itself does nothing. Any other agent in a
// container, for example one typed into a shell there, gets its own row
// with ForeignPID set. When a single interactive agent sits in a container
// reached by an interactive client in a tmux pane (docker exec -it CTR
// bash), it gets that pane, so attach and send work.
//
// On Linux docker reports host PIDs; an agent already in agents (same PID
// and command) is marked with its container instead of added twice.
func MergeContainers(agents []Agent, procs []Proc, containers []Container, panes []tmux.Pane) []Agent {
	if len(containers) == 0 {
		return agents
	}
	hostArgs := map[int]string{}
	for _, p := range procs {
		hostArgs[p.PID] = p.Args
	}
	index := map[int]int{}
	for i, a := range agents {
		index[a.PID] = i
	}
	byTTY := map[string]tmux.Pane{}
	for _, p := range panes {
		byTTY[p.TTY] = p
	}
	var clients []*client
	for _, p := range procs {
		if tokens := strings.Fields(p.Args); len(tokens) > 0 {
			if cmd, ok := parseClient(tokens); ok {
				clients = append(clients, &client{proc: p, cmd: cmd})
			}
		}
	}
	moved := map[int]bool{} // host agents that are now container agents
	var added []Agent
	for _, c := range containers {
		inside := containerAgents(c, agents, index, hostArgs, moved)
		ours := clientsOf(clients, c)
		for _, cl := range ours {
			i, isAgent := index[cl.proc.PID]
			if !isAgent {
				continue
			}
			for j := range inside {
				if inside[j].Kind == agents[i].Kind && !inside[j].folded && topmost(inside, j) {
					fold(&agents[i], inside[j].Agent, c.Name)
					inside[j].folded = true
					break
				}
			}
		}
		for _, cl := range ours {
			if _, isAgent := index[cl.proc.PID]; !isAgent {
				givePane(inside, cl, byTTY)
			}
		}
		for _, in := range inside {
			if !in.folded {
				added = append(added, in.Agent)
			}
		}
	}
	out := make([]Agent, 0, len(agents)+len(added))
	for i, a := range agents {
		if !moved[i] {
			out = append(out, a)
		}
	}
	out = append(out, added...)
	sortAgents(out)
	return out
}

type insideAgent struct {
	Agent
	folded bool
}

// containerAgents classifies the processes of c. Agents that are already
// host agents (Linux) are taken out of agents through moved.
func containerAgents(c Container, agents []Agent, index map[int]int, hostArgs map[int]string, moved map[int]bool) []insideAgent {
	kids := map[int][]Proc{}
	for _, p := range c.Procs {
		kids[p.PPID] = append(kids[p.PPID], p)
	}
	var inside []insideAgent
	for _, p := range c.Procs {
		kind, role, ok := Classify(p.Args)
		if !ok {
			continue
		}
		if args, on := hostArgs[p.PID]; on && args == p.Args {
			if i, isAgent := index[p.PID]; isAgent {
				a := agents[i]
				a.Container, a.ContainerID = c.Name, c.ID
				inside = append(inside, insideAgent{Agent: a})
				moved[i] = true
			}
			continue
		}
		a := Agent{Proc: p, Kind: kind, Role: role, Command: RedactArgs(p.Args),
			Status: cpuStatus(p.CPU), StatusSource: SourceCPU,
			Container: c.Name, ContainerID: c.ID, ForeignPID: true}
		a.Children = descendants(kids, p.PID)
		a.TreeCPU, a.TreeRSS = p.CPU, p.RSS
		for _, ch := range a.Children {
			a.TreeCPU += ch.CPU
			a.TreeRSS += ch.RSS
		}
		inside = append(inside, insideAgent{Agent: a})
	}
	return inside
}

// clientsOf returns the clients that started or entered container c.
func clientsOf(clients []*client, c Container) []*client {
	var out []*client
	var run *client // the run client started closest before c was created
	for _, cl := range clients {
		switch cl.cmd.mode {
		case modeExec:
			if cl.cmd.name == c.Name || len(cl.cmd.name) >= 4 && strings.HasPrefix(c.ID, cl.cmd.name) {
				out = append(out, cl)
			}
		case modeComposeExec:
			if c.Service != "" && cl.cmd.name == c.Service {
				out = append(out, cl)
			}
		case modeRun, modeComposeRun:
			if cl.used {
				continue
			}
			// Compose containers are only started by compose clients.
			match := cl.cmd.mode == modeRun && c.Service == "" && (cl.cmd.name == c.Name ||
				cl.cmd.name == "" && sameImage(cl.cmd.image, c.Image)) ||
				cl.cmd.mode == modeComposeRun && c.Service != "" && cl.cmd.name == c.Service
			// ps start times are whole seconds; the image may be pulled first.
			if match && !c.Created.Before(cl.proc.Start.Add(-2*time.Second)) &&
				(run == nil || cl.proc.Start.After(run.proc.Start)) {
				run = cl
			}
		}
	}
	if run != nil {
		run.used = true
		out = append([]*client{run}, out...)
	}
	return out
}

// topmost reports whether no other agent of the same kind in the container
// is an ancestor of inside[j], so a client takes the agent, not its helper.
func topmost(inside []insideAgent, j int) bool {
	for i := range inside {
		if i == j || inside[i].Kind != inside[j].Kind {
			continue
		}
		for _, ch := range inside[i].Children {
			if ch.PID == inside[j].PID {
				return false
			}
		}
	}
	return true
}

// fold puts the container agent in under its docker client row.
func fold(cl *Agent, in Agent, container string) {
	cl.Container, cl.ContainerID = container, in.ContainerID
	cl.CPU, cl.RSS = in.CPU, in.RSS
	cl.Status, cl.StatusSince, cl.StatusSource = in.Status, in.StatusSince, in.StatusSource
	if in.CWD != "" {
		cl.CWD = in.CWD
	}
	cl.Children = append(cl.Children, ChildProc{PID: in.PID, PPID: in.PPID, Depth: 1, CPU: in.Proc.CPU,
		RSS: in.Proc.RSS, Command: "[" + container + "] " + in.Command})
	for _, ch := range in.Children {
		ch.Depth++
		cl.Children = append(cl.Children, ch)
	}
	cl.TreeCPU += in.TreeCPU
	cl.TreeRSS += in.TreeRSS
}

// givePane hands an interactive client's tmux pane to the one interactive
// agent in its container.
func givePane(inside []insideAgent, cl *client, byTTY map[string]tmux.Pane) {
	pane, ok := byTTY[cl.proc.TTY]
	if !ok || cl.proc.TTY == "" || cl.cmd.detached {
		return
	}
	only := -1
	for i, in := range inside {
		if in.Role == RoleInteractive && !in.folded {
			if only >= 0 {
				return
			}
			only = i
		}
	}
	if only >= 0 && inside[only].TmuxPane == "" {
		inside[only].TmuxSession, inside[only].TmuxPane = pane.Session, pane.ID
	}
}
