package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"lazyai/internal/agent"
	"lazyai/internal/display"
	"lazyai/internal/tmux"
	"lazyai/internal/tui"
	"lazyai/internal/watch"
)

var commands = []command{
	{
		name:     "tui",
		synopsis: "[OPTIONS]",
		summary:  "interactive dashboard (the default)",
		about: `
Show running agents in a live table.

Keys: j/k move, enter details, a attach, p send a prompt, n launch,
s stop, K kill, h show helpers, r refresh, q quit.`,
		setup: func(f *flagDefs) action {
			interval := f.Duration("i", "interval", "DURATION", 2*time.Second, "refresh period")
			return func(e env, args []string) error {
				if err := noArgs(args); err != nil {
					return err
				}
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				return tui.Run(tui.Options{Interval: *interval, LaunchDir: cwd})
			}
		},
	},
	{
		name:     "list",
		aliases:  []string{"ls"},
		synopsis: "[OPTIONS]",
		summary:  "list running agents",
		about: `
Print one snapshot of the running agents. Helper processes (Claude pty
hosts and spares, ollama runners) are hidden unless --all is given.`,
		permute: true,
		setup: func(f *flagDefs) action {
			all := f.Bool("a", "all", "include helper processes")
			asJSON := f.Bool("j", "json", "print JSON")
			return func(e env, args []string) error {
				if err := noArgs(args); err != nil {
					return err
				}
				return list(e, *all, *asJSON)
			}
		},
	},
	{
		name:     "show",
		synopsis: "[OPTIONS] AGENT",
		summary:  "show details of one agent",
		about:    `Print everything known about AGENT.`,
		permute:  true,
		setup: func(f *flagDefs) action {
			asJSON := f.Bool("j", "json", "print JSON")
			return func(e env, args []string) error {
				if len(args) != 1 {
					return usagef("expected one AGENT, got %d arguments", len(args))
				}
				return show(e, args[0], *asJSON)
			}
		},
	},
	{
		name:     "watch",
		synopsis: "[OPTIONS]",
		summary:  "print agents, then changes as they happen",
		about: `
Print a "started" line for every running agent, then one line per change:
started, exited, status (busy/idle) and flags. Runs until interrupted.`,
		permute: true,
		setup: func(f *flagDefs) action {
			all := f.Bool("a", "all", "include helper processes")
			asJSON := f.Bool("j", "json", "print one JSON object per event")
			interval := f.Duration("i", "interval", "DURATION", 2*time.Second, "refresh period")
			return func(e env, args []string) error {
				if err := noArgs(args); err != nil {
					return err
				}
				return watchAgents(e, *all, *asJSON, *interval)
			}
		},
	},
	{
		name:     "launch",
		synopsis: "[OPTIONS] KIND [AGENT-ARGS...]",
		summary:  "start an agent in a new tmux session",
		about: `
Start KIND in a new detached tmux session named lazyai-NAME. Options must
come before KIND; everything after KIND is passed to the agent unchanged.`,
		setup: func(f *flagDefs) action {
			dir := f.String("d", "dir", "DIR", ".", "working directory")
			name := f.String("n", "name", "NAME", "", "session name (default KIND-DIRNAME)")
			return func(e env, args []string) error {
				if len(args) == 0 {
					return usagef("missing KIND")
				}
				session, err := agent.Launch(agent.Kind(args[0]), *dir, *name, args[1:])
				if err != nil {
					return err
				}
				fmt.Fprintf(e.out, "launched %s; attach with: lazyai attach %s\n", session, session)
				return nil
			}
		},
	},
	{
		name:     "attach",
		synopsis: "AGENT",
		summary:  "attach to an agent's tmux session",
		about: `
Attach the terminal to AGENT's tmux session; detach with the tmux prefix
key and d. Inside tmux, switch the client to that session instead.`,
		permute: true,
		setup: func(f *flagDefs) action {
			return func(e env, args []string) error {
				if len(args) != 1 {
					return usagef("expected one AGENT, got %d arguments", len(args))
				}
				return attach(args[0])
			}
		},
	},
	{
		name:     "send",
		synopsis: "AGENT TEXT...",
		summary:  "type a prompt into an agent and press Enter",
		about: `
Type TEXT into AGENT's tmux pane and press Enter. Everything after AGENT
is text, joined with spaces, including words that start with "-".`,
		setup: func(f *flagDefs) action {
			return func(e env, args []string) error {
				if len(args) < 2 {
					return usagef("expected AGENT and TEXT")
				}
				return send(args[0], strings.Join(args[1:], " "))
			}
		},
	},
	{
		name:     "stop",
		synopsis: "[OPTIONS] AGENT...",
		summary:  "stop agents (SIGTERM)",
		about: `
Send SIGTERM to each AGENT and wait up to 5 seconds for it to exit. Asks
for confirmation per agent unless --yes is given.`,
		permute: true,
		setup: func(f *flagDefs) action {
			force := f.Bool("f", "force", "send SIGKILL instead of SIGTERM")
			yes := f.Bool("y", "yes", "do not ask for confirmation")
			return func(e env, args []string) error {
				return stop(e, args, *force, *yes)
			}
		},
	},
	{
		name:     "kill",
		synopsis: "[OPTIONS] AGENT...",
		summary:  "kill agents (SIGKILL); same as stop --force",
		about: `
Send SIGKILL to each AGENT. The agent cannot clean up. Asks for
confirmation per agent unless --yes is given.`,
		permute: true,
		setup: func(f *flagDefs) action {
			yes := f.Bool("y", "yes", "do not ask for confirmation")
			return func(e env, args []string) error {
				return stop(e, args, true, *yes)
			}
		},
	},
}

func noArgs(args []string) error {
	if len(args) > 0 {
		return usagef("unexpected argument %q", args[0])
	}
	return nil
}

func list(e env, all, asJSON bool) error {
	agents, err := agent.Discover()
	if err != nil {
		return err
	}
	agents = withoutHelpers(agents, all)
	if asJSON {
		return printJSON(e.out, agents)
	}
	now := time.Now()
	w := watch.New()
	w.Update(agents, now)
	tw := tabwriter.NewWriter(e.out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "PID\tKIND\tROLE\tSTATUS\tCPU%\tMEM\tUP\tTMUX\tFLAGS\tWHERE")
	for _, a := range agents {
		t, _ := w.Tracker(a.PID)
		tm := strings.TrimPrefix(a.TmuxSession, tmux.Prefix)
		if tm == "" {
			tm = "-"
		}
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%.1f\t%s\t%s\t%s\t%s\t%s\n", a.PID, a.Kind, a.Role,
			display.Status(a, t, now), a.CPU, display.Bytes(a.RSS), display.Duration(now.Sub(a.Start)),
			tm, display.Flags(a, t, now), display.Where(a))
	}
	return tw.Flush()
}

func show(e env, ref string, asJSON bool) error {
	a, err := find(ref)
	if err != nil {
		return err
	}
	if asJSON {
		return printJSON(e.out, a)
	}
	now := time.Now()
	w := watch.New()
	w.Update([]agent.Agent{a}, now)
	t, _ := w.Tracker(a.PID)
	fields := append(display.Details(a),
		display.Field{Name: "resources", Value: fmt.Sprintf("%.1f%% CPU, %s, up %s",
			a.CPU, display.Bytes(a.RSS), display.Duration(now.Sub(a.Start)))},
		display.Field{Name: "flags", Value: display.Flags(a, t, now)})
	for _, f := range fields {
		fmt.Fprintf(e.out, "%-10s %s\n", f.Name, f.Value)
	}
	return nil
}

func watchAgents(e env, all, asJSON bool, interval time.Duration) error {
	w := watch.New()
	enc := json.NewEncoder(e.out)
	for {
		agents, err := agent.Discover()
		if err != nil {
			fmt.Fprintln(e.err, "lazyai:", err)
		}
		for _, ev := range w.Update(withoutHelpers(agents, all), time.Now()) {
			if asJSON {
				if err := enc.Encode(ev); err != nil {
					return err
				}
				continue
			}
			fmt.Fprintln(e.out, formatEvent(ev))
		}
		time.Sleep(interval)
	}
}

func formatEvent(ev watch.Event) string {
	change := ev.To
	if ev.Type == watch.EventStatus || ev.Type == watch.EventFlags {
		change = orDash(ev.From) + " -> " + orDash(ev.To)
	}
	a := ev.Agent
	return fmt.Sprintf("%s  %-7s  %-6d  %-12s  %-11s  %-20s  %s",
		ev.Time.Format("15:04:05"), ev.Type, a.PID, a.Kind, a.Role, change, display.Where(a))
}

func attach(ref string) error {
	a, err := find(ref)
	if err != nil {
		return err
	}
	if a.TmuxSession == "" {
		return fmt.Errorf("PID %d is not running in tmux", a.PID)
	}
	cmd := tmux.AttachCmd(a.TmuxSession)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func send(ref, text string) error {
	a, err := find(ref)
	if err != nil {
		return err
	}
	if a.TmuxPane == "" {
		return fmt.Errorf("PID %d is not running in tmux", a.PID)
	}
	return tmux.Send(a.TmuxPane, text)
}

func stop(e env, refs []string, force, yes bool) error {
	if len(refs) == 0 {
		return usagef("missing AGENT")
	}
	agents, err := agent.Discover()
	if err != nil {
		return err
	}
	verb, signal := "Stop", "SIGTERM"
	if force {
		verb, signal = "Kill", "SIGKILL"
	}
	answers := bufio.NewReader(e.in) // one reader, so piped answers are not lost
	failed := 0
	for _, ref := range refs {
		a, err := resolve(agents, ref)
		if err != nil {
			fmt.Fprintln(e.err, "lazyai:", err)
			failed++
			continue
		}
		q := fmt.Sprintf("%s (%s) %s %s %d in %s?", verb, signal, a.Kind, a.Role, a.PID, display.Path(a.CWD))
		if !yes && !confirm(e.out, answers, q) {
			fmt.Fprintln(e.out, "skipped", a.PID)
			continue
		}
		if err := agent.Stop(a, force, agent.StopTimeout); err != nil {
			fmt.Fprintf(e.err, "lazyai: PID %d: %v\n", a.PID, err)
			failed++
			continue
		}
		fmt.Fprintln(e.out, "stopped", a.PID)
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d not stopped", failed, len(refs))
	}
	return nil
}

func find(ref string) (agent.Agent, error) {
	agents, err := agent.Discover()
	if err != nil {
		return agent.Agent{}, err
	}
	return resolve(agents, ref)
}

// resolve turns a PID into any agent, or a tmux session name (with or
// without the lazyai- prefix) or a Claude session name into exactly one
// interactive agent.
func resolve(agents []agent.Agent, ref string) (agent.Agent, error) {
	if pid, err := strconv.Atoi(ref); err == nil {
		for _, a := range agents {
			if a.PID == pid {
				return a, nil
			}
		}
		return agent.Agent{}, fmt.Errorf("PID %d is not a running agent", pid)
	}
	var matches []agent.Agent
	for _, a := range agents {
		if a.Role == agent.RoleInteractive &&
			(a.TmuxSession == ref || a.TmuxSession == tmux.Prefix+ref || a.Name == ref) {
			matches = append(matches, a)
		}
	}
	switch len(matches) {
	case 0:
		return agent.Agent{}, fmt.Errorf("no interactive agent matches %q", ref)
	case 1:
		return matches[0], nil
	}
	return agent.Agent{}, fmt.Errorf("%q matches %d agents; use a PID", ref, len(matches))
}

func withoutHelpers(agents []agent.Agent, all bool) []agent.Agent {
	if all {
		return agents
	}
	var kept []agent.Agent
	for _, a := range agents {
		if a.Role != agent.RoleHelper {
			kept = append(kept, a)
		}
	}
	return kept
}

func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func confirm(w io.Writer, answers *bufio.Reader, question string) bool {
	fmt.Fprint(w, question+" [y/N] ")
	line, _ := answers.ReadString('\n')
	return strings.TrimSpace(strings.ToLower(line)) == "y"
}
