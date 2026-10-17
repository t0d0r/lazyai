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
)

const usage = `agentctl - control local AI agents

Usage:
  agentctl [tui] [-interval 2s]        interactive dashboard (default)
  agentctl list [-json] [-all]         list running agents
  agentctl stop [-force] [-yes] PID... stop agents (SIGTERM, or SIGKILL with -force)
  agentctl launch [-dir D] [-name N] KIND [ARGS...]
                                       start an agent in a new tmux session
  agentctl attach PID|SESSION          attach to an agent's tmux session
  agentctl send PID|SESSION TEXT...    type a prompt into an agent and press Enter

Kinds: claude, codex, opencode, gemini, aider, cursor-agent, ollama
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
	case "stop":
		err = runStop(args)
	case "launch":
		err = runLaunch(args)
	case "attach":
		err = runAttach(args)
	case "send":
		err = runSend(args)
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
	if !*all {
		kept := agents[:0]
		for _, a := range agents {
			if a.Role != agent.RoleHelper {
				kept = append(kept, a)
			}
		}
		agents = kept
	}
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(agents)
	}
	now := time.Now()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PID\tKIND\tROLE\tSTATUS\tCPU%\tMEM\tUP\tTMUX\tWHERE")
	for _, a := range agents {
		tm := a.TmuxSession
		if tm == "" {
			tm = "-"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%.1f\t%s\t%s\t%s\t%s\n", a.PID, a.Kind, a.Role, a.Status,
			a.CPU, display.Bytes(a.RSS), display.Duration(now.Sub(a.Start)), tm, display.Where(a))
	}
	return w.Flush()
}

func runStop(args []string) error {
	fs := flag.NewFlagSet("stop", flag.ExitOnError)
	force := fs.Bool("force", false, "send SIGKILL instead of SIGTERM")
	yes := fs.Bool("yes", false, "do not ask for confirmation")
	fs.Parse(args)
	if fs.NArg() == 0 {
		return errors.New("usage: agentctl stop [-force] [-yes] PID...")
	}
	agents, err := agent.Discover()
	if err != nil {
		return err
	}
	var failed int
	for _, ref := range fs.Args() {
		pid, err := strconv.Atoi(ref)
		if err != nil {
			return fmt.Errorf("%q is not a PID", ref)
		}
		a, ok := byPID(agents, pid)
		if !ok {
			fmt.Fprintf(os.Stderr, "PID %d is not a running agent\n", pid)
			failed++
			continue
		}
		if !*yes && !confirm(fmt.Sprintf("Stop %s %s %d in %s?", a.Kind, a.Role, a.PID, display.Path(a.CWD))) {
			fmt.Println("skipped", pid)
			continue
		}
		if err := agent.Stop(a, *force, agent.StopTimeout); err != nil {
			fmt.Fprintf(os.Stderr, "PID %d: %v\n", pid, err)
			failed++
			continue
		}
		fmt.Println("stopped", pid)
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d not stopped", failed, fs.NArg())
	}
	return nil
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
		return errors.New("usage: agentctl attach PID|SESSION")
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
		return errors.New("usage: agentctl send PID|SESSION TEXT...")
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

// find resolves a PID, a tmux session name (with or without the agentctl-
// prefix) or a Claude session name to one interactive agent.
func find(ref string) (agent.Agent, error) {
	agents, err := agent.Discover()
	if err != nil {
		return agent.Agent{}, err
	}
	if pid, err := strconv.Atoi(ref); err == nil {
		if a, ok := byPID(agents, pid); ok {
			return a, nil
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

func byPID(agents []agent.Agent, pid int) (agent.Agent, bool) {
	for _, a := range agents {
		if a.PID == pid {
			return a, true
		}
	}
	return agent.Agent{}, false
}

func confirm(question string) bool {
	fmt.Print(question + " [y/N] ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(strings.ToLower(line)) == "y"
}
