// Command agentctl lists, watches, launches and stops local AI coding agents.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"agentctl/internal/agent"
	"agentctl/internal/display"
	"agentctl/internal/tmux"
	"agentctl/internal/tui"
	"agentctl/internal/watch"
)

const usage = `agentctl - control local AI agents

Usage:
  agentctl [tui] [-interval 2s]           interactive dashboard (default)
  agentctl list [-json] [-all]            list running agents
  agentctl show [-json] AGENT             details of one agent
  agentctl watch [-json] [-all] [-interval 2s]
                                          print agents, then changes as they happen
  agentctl launch [-dir D] [-name N] KIND [ARGS...]
                                          start an agent in a new tmux session
  agentctl attach AGENT                   attach to an agent's tmux session
  agentctl send AGENT TEXT...             type a prompt into an agent, press Enter
  agentctl stop [-force] [-yes] AGENT...  stop agents (SIGTERM, SIGKILL with -force)
  agentctl kill [-yes] AGENT...           same as stop -force

AGENT is a PID, a tmux session name (the agentctl- prefix is optional) or
a Claude Code session name.

Kinds: claude, codex, opencode, gemini, aider, cursor-agent, ollama

TUI keys and the matching commands:
  enter show   a attach   p send   n launch   s stop   K kill   h list -all
`

func main() {
	args := os.Args[1:]
	cmd := "tui"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	var err error
	switch cmd {
	case "tui":
		err = runTUI(args)
	case "list", "ls":
		err = runList(args)
	case "show":
		err = runShow(args)
	case "watch":
		err = runWatch(args)
	case "launch":
		err = runLaunch(args)
	case "attach":
		err = runAttach(args)
	case "send":
		err = runSend(args)
	case "stop":
		err = runStop("stop", args, false)
	case "kill":
		err = runStop("kill", args, true)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		err = fmt.Errorf("unknown command %q\n\n%s", cmd, usage)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "agentctl:", err)
		os.Exit(1)
	}
}

func runTUI(args []string) error {
	fs := flag.NewFlagSet("tui", flag.ExitOnError)
	interval := fs.Duration("interval", 2*time.Second, "refresh period")
	fs.Parse(args)
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return tui.Run(tui.Options{Interval: *interval, LaunchDir: cwd})
}

func runList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "print JSON")
	all := fs.Bool("all", false, "include helper processes")
	fs.Parse(args)
	agents, err := agent.Discover()
	if err != nil {
		return err
	}
	agents = withoutHelpers(agents, *all)
	if *asJSON {
		return printJSON(agents)
	}
	now := time.Now()
	w := watch.New()
	w.Update(agents, now)
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
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

func runShow(args []string) error {
	fs := flag.NewFlagSet("show", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "print JSON")
	fs.Parse(args)
	if fs.NArg() != 1 {
		return errors.New("usage: agentctl show [-json] AGENT")
	}
	a, err := find(fs.Arg(0))
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(a)
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
		fmt.Printf("%-10s %s\n", f.Name, f.Value)
	}
	return nil
}

func runWatch(args []string) error {
	fs := flag.NewFlagSet("watch", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "print one JSON object per event")
	all := fs.Bool("all", false, "include helper processes")
	interval := fs.Duration("interval", 2*time.Second, "refresh period")
	fs.Parse(args)
	w := watch.New()
	enc := json.NewEncoder(os.Stdout)
	for {
		agents, err := agent.Discover()
		if err != nil {
			fmt.Fprintln(os.Stderr, "agentctl:", err)
		}
		for _, e := range w.Update(withoutHelpers(agents, *all), time.Now()) {
			if *asJSON {
				if err := enc.Encode(e); err != nil {
					return err
				}
				continue
			}
			fmt.Println(formatEvent(e))
		}
		time.Sleep(*interval)
	}
}

func formatEvent(e watch.Event) string {
	change := e.To
	if e.Type == watch.EventStatus || e.Type == watch.EventFlags {
		change = orDash(e.From) + " -> " + orDash(e.To)
	}
	a := e.Agent
	return fmt.Sprintf("%s  %-7s  %-6d  %-12s  %-11s  %-20s  %s",
		e.Time.Format("15:04:05"), e.Type, a.PID, a.Kind, a.Role, change, display.Where(a))
}

func runLaunch(args []string) error {
	fs := flag.NewFlagSet("launch", flag.ExitOnError)
	dir := fs.String("dir", ".", "working directory")
	name := fs.String("name", "", "session name (default KIND-DIRNAME)")
	fs.Parse(args)
	if fs.NArg() == 0 {
		return errors.New("usage: agentctl launch [-dir D] [-name N] KIND [ARGS...]")
	}
	session, err := agent.Launch(agent.Kind(fs.Arg(0)), *dir, *name, fs.Args()[1:])
	if err != nil {
		return err
	}
	fmt.Printf("launched %s; attach with: agentctl attach %s\n", session, session)
	return nil
}

func runAttach(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: agentctl attach AGENT")
	}
	a, err := find(args[0])
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

func runSend(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: agentctl send AGENT TEXT...")
	}
	a, err := find(args[0])
	if err != nil {
		return err
	}
	if a.TmuxPane == "" {
		return fmt.Errorf("PID %d is not running in tmux", a.PID)
	}
	return tmux.Send(a.TmuxPane, strings.Join(args[1:], " "))
}

// runStop serves both stop and kill; kill always forces.
func runStop(name string, args []string, kill bool) error {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	force := &kill
	if !kill {
		force = fs.Bool("force", false, "send SIGKILL instead of SIGTERM")
	}
	yes := fs.Bool("yes", false, "do not ask for confirmation")
	fs.Parse(args)
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: agentctl %s [-yes] AGENT...", name)
	}
	agents, err := agent.Discover()
	if err != nil {
		return err
	}
	verb, signal := "Stop", "SIGTERM"
	if *force {
		verb, signal = "Kill", "SIGKILL"
	}
	failed := 0
	for _, ref := range fs.Args() {
		a, err := resolve(agents, ref)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			failed++
			continue
		}
		if !*yes && !confirm(fmt.Sprintf("%s (%s) %s %s %d in %s?", verb, signal, a.Kind, a.Role, a.PID, display.Path(a.CWD))) {
			fmt.Println("skipped", a.PID)
			continue
		}
		if err := agent.Stop(a, *force, agent.StopTimeout); err != nil {
			fmt.Fprintf(os.Stderr, "PID %d: %v\n", a.PID, err)
			failed++
			continue
		}
		fmt.Println("stopped", a.PID)
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d not stopped", failed, fs.NArg())
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
// without the agentctl- prefix) or a Claude session name into exactly one
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

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func confirm(question string) bool {
	fmt.Print(question + " [y/N] ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(strings.ToLower(line)) == "y"
}
