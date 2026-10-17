package agent

import (
	"os"
	"sort"
	"time"

	"lazyai/internal/tmux"
)

// BusyCPU is the CPU percentage at or above which an agent without its own
// status reporting counts as busy.
const BusyCPU = 2.0

// Status source values.
const (
	SourceClaude = "claude-session" // ~/.claude/sessions/<pid>.json
	SourceCPU    = "cpu"            // BusyCPU heuristic
)

// Agent is a running agent process with everything known about it.
type Agent struct {
	Proc
	Kind         Kind      `json:"kind"`
	Role         Role      `json:"role"`
	Command      string    `json:"command"` // redacted command line
	CWD          string    `json:"cwd,omitempty"`
	Name         string    `json:"name,omitempty"`
	SessionID    string    `json:"session_id,omitempty"`
	Status       string    `json:"status"` // "busy" or "idle"
	StatusSince  time.Time `json:"status_since,omitzero"`
	StatusSource string    `json:"status_source"`
	TmuxSession  string    `json:"tmux_session,omitempty"`
	TmuxPane     string    `json:"tmux_pane,omitempty"`
	// Protected is set for lazyai's own ancestors: stopping one of them
	// would kill the terminal lazyai runs in.
	Protected bool `json:"protected,omitempty"`
	// Children are the processes the agent started, directly or not, in
	// tree order. TreeCPU and TreeRSS add them to the agent's own usage.
	Children []ChildProc `json:"children,omitempty"`
	TreeCPU  float64     `json:"tree_cpu"`
	TreeRSS  int64       `json:"tree_rss_bytes"`
}

// ChildProc is one process below an agent in the process tree.
type ChildProc struct {
	PID     int     `json:"pid"`
	PPID    int     `json:"ppid"`
	Depth   int     `json:"depth"` // 1 for direct children
	CPU     float64 `json:"cpu"`
	RSS     int64   `json:"rss_bytes"`
	Command string  `json:"command"` // redacted
}

// Discover lists running agents.
func Discover() ([]Agent, error) {
	procs, err := Snapshot()
	if err != nil {
		return nil, err
	}
	agents := Build(procs, os.Getpid(), ClaudeSessionsDir(), tmux.Panes())
	missing := []int{}
	for _, a := range agents {
		if a.CWD == "" {
			missing = append(missing, a.PID)
		}
	}
	dirs := workingDirs(missing)
	for i := range agents {
		if agents[i].CWD == "" {
			agents[i].CWD = dirs[agents[i].PID]
		}
	}
	return agents, nil
}

// Build turns a process snapshot into agents. It does no process I/O beyond
// reading Claude session files, so it can be tested with fixed input.
func Build(procs []Proc, self int, claudeDir string, panes []tmux.Pane) []Agent {
	protected := Ancestors(procs, self)
	kids := map[int][]Proc{}
	for _, p := range procs {
		kids[p.PPID] = append(kids[p.PPID], p)
	}
	byTTY := map[string]tmux.Pane{}
	for _, p := range panes {
		byTTY[p.TTY] = p
	}
	var agents []Agent
	for _, p := range procs {
		kind, role, ok := Classify(p.Args)
		if !ok {
			continue
		}
		a := Agent{
			Proc:         p,
			Kind:         kind,
			Role:         role,
			Command:      RedactArgs(p.Args),
			Status:       cpuStatus(p.CPU),
			StatusSource: SourceCPU,
			Protected:    protected[p.PID],
		}
		if kind == KindClaude && role == RoleInteractive {
			a.SessionID = resumeID(p.Args)
			if s, ok := loadClaudeSession(claudeDir, p); ok {
				a.SessionID, a.CWD, a.Name = s.SessionID, s.CWD, s.Name
				if s.Status != "" {
					a.Status, a.StatusSince, a.StatusSource = s.Status, s.statusSince(), SourceClaude
				}
			}
		}
		if pane, ok := byTTY[p.TTY]; ok && p.TTY != "" {
			a.TmuxSession, a.TmuxPane = pane.Session, pane.ID
		}
		a.Children = descendants(kids, p.PID)
		a.TreeCPU, a.TreeRSS = p.CPU, p.RSS
		for _, c := range a.Children {
			a.TreeCPU += c.CPU
			a.TreeRSS += c.RSS
		}
		agents = append(agents, a)
	}
	sort.SliceStable(agents, func(i, j int) bool {
		if agents[i].Kind != agents[j].Kind {
			return kindRank(agents[i].Kind) < kindRank(agents[j].Kind)
		}
		if agents[i].Role != agents[j].Role {
			return agents[i].Role == RoleInteractive
		}
		return agents[i].Start.Before(agents[j].Start)
	})
	return agents
}

// descendants walks the process tree below pid depth first.
func descendants(kids map[int][]Proc, pid int) []ChildProc {
	var out []ChildProc
	seen := map[int]bool{pid: true}
	var walk func(pid, depth int)
	walk = func(pid, depth int) {
		for _, c := range kids[pid] {
			if seen[c.PID] {
				continue
			}
			seen[c.PID] = true
			out = append(out, ChildProc{PID: c.PID, PPID: c.PPID, Depth: depth, CPU: c.CPU, RSS: c.RSS,
				Command: RedactArgs(c.Args)})
			walk(c.PID, depth+1)
		}
	}
	walk(pid, 1)
	return out
}

// Ancestors returns pid and every parent of it found in procs.
func Ancestors(procs []Proc, pid int) map[int]bool {
	parent := map[int]int{}
	for _, p := range procs {
		parent[p.PID] = p.PPID
	}
	seen := map[int]bool{}
	for pid > 1 && !seen[pid] {
		seen[pid] = true
		pid = parent[pid]
	}
	return seen
}

func cpuStatus(cpu float64) string {
	if cpu >= BusyCPU {
		return "busy"
	}
	return "idle"
}

func kindRank(k Kind) int {
	for i, kk := range Kinds {
		if kk == k {
			return i
		}
	}
	return len(Kinds)
}
