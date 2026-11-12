package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"lazyai/internal/agent"
	"lazyai/internal/display"
	"lazyai/internal/tmux"
	"lazyai/internal/tui"
	"lazyai/internal/update"
	"lazyai/internal/watch"
)

var commands = []command{
	{
		name:     "tui",
		synopsis: "[OPTIONS]",
		summary:  "interactive dashboard (the default)",
		about: `
Full-screen dashboard in the style of lazydocker: Agents, Services and
Models panels on the left; Info, Screen, Processes and Stats tabs for the
selected agent on the right.

Keys: tab/h/l panels, j/k move, [ ] tabs, x menu, a attach, p prompt,
n launch, s stop, K kill, H helpers, ? all keys, q quit. Mouse works.`,
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
				return tui.Run(tui.Options{Interval: *interval, LaunchDir: cwd,
					HistoryFile: tui.HistoryPath(), ImportantFile: tui.ImportantPath(), SettingsFile: tui.SettingsPath(), Version: version})
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
		synopsis: "[OPTIONS] KIND [DIR] [NAME] [-- AGENT-ARGS...]",
		summary:  "start an agent in a new tmux session",
		about: `
Start KIND in a new detached tmux session named lazyai-NAME, the same
line as n in the dashboard. KIND is a known kind or any command on PATH.
DIR may use ~ and is relative to --dir (default: here). Options must
come before KIND; agent arguments go after --, unchanged.`,
		setup: func(f *flagDefs) action {
			dir := f.String("d", "dir", "DIR", "", "base directory for DIR (default: the current one)")
			name := f.String("n", "name", "NAME", "", "session name (default KIND-DIRNAME)")
			return func(e env, args []string) error {
				if len(args) == 0 {
					return usagef("missing KIND")
				}
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				spec, err := agent.ParseLaunch(args, agent.ExpandDir(*dir, cwd))
				switch {
				case errors.Is(err, agent.ErrLaunchSyntax):
					return usagef("expected %s", strings.TrimPrefix(err.Error(), "usage: "))
				case err != nil:
					return err
				case *name != "" && spec.Name != "":
					return usagef("NAME given twice: --name %s and %s", *name, spec.Name)
				case *name != "":
					spec.Name = *name
				}
				session, err := agent.Launch(spec.Kind, spec.Dir, spec.Name, spec.Args)
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
		name:     "rename",
		synopsis: "AGENT NAME",
		summary:  "rename an agent's tmux session",
		about: `
Rename the tmux session AGENT runs in to NAME, taken as given: letters,
digits, - and _, with no lazyai- prefix added. Important marks set in
the dashboard move to the new name.`,
		setup: func(f *flagDefs) action {
			return func(e env, args []string) error {
				if len(args) != 2 {
					return usagef("expected AGENT and NAME")
				}
				return rename(e, args[0], args[1])
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
	{
		name:     "update",
		synopsis: "[OPTIONS]",
		summary:  "update lazyai to the latest release",
		about: `
Look up the latest release on GitHub and install it the way lazyai was
installed: brew upgrade for Homebrew, the release's .deb through apt for
the Debian package, otherwise the binary is replaced with the one from the
release tarball. Downloads are checked against the release's SHA256SUMS.
Asks for confirmation unless --yes is given. A build from source is not
replaced.`,
		permute: true,
		setup: func(f *flagDefs) action {
			check := f.Bool("c", "check", "only report whether an update is available")
			yes := f.Bool("y", "yes", "do not ask for confirmation")
			return func(e env, args []string) error {
				if err := noArgs(args); err != nil {
					return err
				}
				return selfUpdate(e, *check, *yes)
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
	fmt.Fprintln(tw, "PID	KIND	ROLE	STATUS	CPU%	MEM	UP	IN_TMUX	TMUX_SESSION	FLAGS	WHERE")
	for _, a := range agents {
		t, _ := w.Tracker(a.Key())
		inTmux, tmuxSession := tmuxColumns(a)
		fmt.Fprintf(tw, "%d	%s	%s	%s	%.1f	%s	%s	%s	%s	%s	%s\n", a.PID, a.Kind, a.Role,
			display.Status(a, t, now), a.CPU, display.Bytes(a.RSS), display.Duration(now.Sub(a.Start)),
			inTmux, tmuxSession, display.Flags(a, t, now), display.Where(a))
	}
	return tw.Flush()
}

// tmuxColumns makes tmux membership explicit while retaining the session name
// for processes matched to a pane during discovery.
func tmuxColumns(a agent.Agent) (inTmux, session string) {
	if a.TmuxPane == "" {
		return "no", "-"
	}
	session = a.TmuxSession
	if session == "" {
		session = "-"
	}
	return "yes", session
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
	t, _ := w.Tracker(a.Key())
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

func rename(e env, ref, name string) error {
	if err := tmux.CheckSessionName(name); err != nil {
		return err
	}
	a, err := find(ref)
	if err != nil {
		return err
	}
	if a.TmuxSession == "" {
		return fmt.Errorf("PID %d is not running in tmux", a.PID)
	}
	if a.TmuxSession == name {
		return nil
	}
	if err := tmux.RenameSession(a.TmuxSession, name); err != nil {
		return err
	}
	fmt.Fprintf(e.out, "renamed %s to %s\n", a.TmuxSession, name)
	if err := tui.RenameMarks(tui.ImportantPath(), a.TmuxSession, name); err != nil {
		return fmt.Errorf("renamed, but important marks: %w", err)
	}
	return nil
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

func selfUpdate(e env, check, yes bool) error {
	c := update.New()
	latest, err := c.Latest()
	if err != nil {
		return err
	}
	current, release := update.Parse(version)
	if !release {
		fmt.Fprintf(e.out, "lazyai %s is a build from source; the latest release is %s\n", version, latest)
		if check {
			return nil
		}
		return fmt.Errorf("not replacing a build from source; update it with git pull and make install")
	}
	if current.Compare(latest) >= 0 {
		fmt.Fprintf(e.out, "lazyai %s is up to date\n", version)
		return nil
	}
	fmt.Fprintf(e.out, "update available: %s -> %s\n", current, latest)
	exe, err := update.Executable()
	if err != nil {
		return err
	}
	method := update.Detect(exe, runtime.GOOS, update.DpkgOwns)
	target := latest
	if method == update.Homebrew {
		tap, tapErr := c.TapVersion()
		var note string
		target, note = brewTarget(current, latest, tap, tapErr)
		if note != "" {
			fmt.Fprintln(e.out, note)
		}
		if current.Compare(target) >= 0 {
			return nil // brew has nothing newer yet; not an error
		}
	}
	if check {
		return nil
	}
	q := fmt.Sprintf("Update lazyai %s -> %s (%s)?", current, target, method)
	if !yes && !confirm(e.out, bufio.NewReader(e.in), q) {
		fmt.Fprintln(e.out, "skipped")
		return nil
	}
	if err := c.Install(method, target, exe); err != nil {
		return err
	}
	if method == update.Binary {
		fmt.Fprintf(e.out, "updated lazyai %s -> %s (%s)\n", current, target, exe)
	}
	return nil
}

// brewTarget is the version brew upgrade can install, given the version
// the published tap formula has (or the error reading it), and a line
// to print about the tap, or "". brew builds from the formula, so a
// release the tap does not have yet cannot be installed with brew.
func brewTarget(current, latest, tap update.Version, tapErr error) (update.Version, string) {
	switch {
	case tapErr != nil:
		return latest, "could not read the Homebrew formula (" + tapErr.Error() +
			"); brew may not have " + latest.String() + " yet"
	case tap.Compare(current) <= 0:
		return current, fmt.Sprintf("%s is released but the Homebrew tap still has %s; "+
			"brew upgrade cannot install it yet, run lazyai update again later", latest, tap)
	case tap.Compare(latest) < 0:
		return tap, fmt.Sprintf("the Homebrew tap has %s, not yet %s; brew upgrade installs %s", tap, latest, tap)
	}
	return latest, ""
}

func find(ref string) (agent.Agent, error) {
	agents, err := agent.Discover()
	if err != nil {
		return agent.Agent{}, err
	}
	return resolve(agents, ref)
}

// resolve turns a PID or a container agent's key (CONTAINERID/PID) into
// any agent, or a tmux session name (with or without the lazyai- prefix),
// a Claude session name or a container name into exactly one interactive
// agent. A PID means a process of this machine before one in a container.
func resolve(agents []agent.Agent, ref string) (agent.Agent, error) {
	if pid, err := strconv.Atoi(ref); err == nil {
		var foreign []agent.Agent
		for _, a := range agents {
			switch {
			case a.PID == pid && !a.ForeignPID:
				return a, nil
			case a.PID == pid:
				foreign = append(foreign, a)
			}
		}
		if len(foreign) == 1 {
			return foreign[0], nil
		}
		return agent.Agent{}, fmt.Errorf("PID %d is not a running agent", pid)
	}
	for _, a := range agents {
		if a.ForeignPID && a.Key() == ref {
			return a, nil
		}
	}
	var matches []agent.Agent
	for _, a := range agents {
		if a.Role == agent.RoleInteractive &&
			(a.TmuxSession == ref || a.TmuxSession == tmux.Prefix+ref || a.Name == ref || a.Container == ref) {
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
