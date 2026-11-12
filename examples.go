package main

import (
	"fmt"
	"io"
	"strings"
)

// example is one command line with what it does.
type example struct {
	cmd, note string
}

// examples per command, shown under "Examples:" in a command's help and
// by 'lazyai examples'. Directories and names are invented.
var examples = map[string][]example{
	"tui": {
		{"lazyai", "open the dashboard"},
		{"lazyai --interval 5s", "refresh every 5 seconds instead of 2"},
	},
	"list": {
		{"lazyai list", "running agents, services and models"},
		{"lazyai ls --all", "also helper processes"},
		{"lazyai list --json", "the same as JSON, for scripts"},
	},
	"show": {
		{"lazyai show 12345", "by PID"},
		{"lazyai show web", "by tmux session (lazyai-web) or Claude session name"},
		{"lazyai show web --json", "everything known, as JSON"},
	},
	"watch": {
		{"lazyai watch", "started, exited, busy/idle and flag changes"},
		{"lazyai watch --json --interval 10s", "one JSON object per event"},
	},
	"launch": {
		{"lazyai launch claude", "Claude Code in the current directory"},
		{"lazyai launch claude ~/work/alpha/web-frontend", "in another directory"},
		{"lazyai launch claude-bst ~/work/alpha/web-frontend web", "a wrapper script on PATH, session lazyai-web"},
		{"lazyai launch claude . review -- --model sonnet", "arguments for the agent go after --"},
		{"lazyai launch --dir ~/work codex alpha/api", "DIR relative to --dir"},
		{"lazyai launch ollama -- run llama3.2", "ollama needs its arguments"},
	},
	"attach": {
		{"lazyai attach web", "attach to lazyai-web; leave with the tmux prefix and d"},
		{"lazyai attach 12345", "by PID"},
	},
	"send": {
		{`lazyai send web "run the tests and fix what fails"`, "type a prompt and press Enter"},
	},
	"rename": {
		{"lazyai rename web api-review", "rename lazyai-web to api-review"},
		{"lazyai rename 12345 lazyai-docs", "by PID; keep the lazyai- prefix by typing it"},
	},
	"stop": {
		{"lazyai stop web", "SIGTERM, after a y/N question"},
		{"lazyai stop 12345 12346 --yes", "several agents, without asking"},
	},
	"kill": {
		{"lazyai kill 12345", "SIGKILL, after a y/N question"},
	},
	"update": {
		{"lazyai update --check", "only say whether a newer release exists"},
		{"lazyai update --yes", "install it without asking"},
	},
}

// writeExamples prints the examples of one command.
func writeExamples(w io.Writer, name string) {
	ex := examples[name]
	width := 0
	for _, e := range ex {
		width = max(width, len(e.cmd))
	}
	for _, e := range ex {
		fmt.Fprintf(w, "  %-*s  # %s\n", width, e.cmd, e.note)
	}
}

// runExamples is 'lazyai examples [COMMAND]': the examples of every
// command, or of one.
func runExamples(args []string, e env) int {
	switch {
	case len(args) > 1:
		fmt.Fprintf(e.err, "lazyai examples: too many arguments\nRun 'lazyai help examples' for usage.\n")
		return 2
	case len(args) == 1 && (args[0] == "-h" || args[0] == "--help" || args[0] == "-help"):
		writeExamplesHelp(e.out)
		return 0
	case len(args) == 1:
		c, ok := lookup(args[0])
		if !ok {
			fmt.Fprintf(e.err, "lazyai: unknown command %q\nRun 'lazyai --help' for usage.\n", args[0])
			return 2
		}
		writeExamples(e.out, c.name)
		return 0
	}
	for i, c := range commands {
		if i > 0 {
			fmt.Fprintln(e.out)
		}
		fmt.Fprintf(e.out, "%s - %s\n", c.name, c.summary)
		writeExamples(e.out, c.name)
	}
	return 0
}

func writeExamplesHelp(w io.Writer) {
	fmt.Fprint(w, strings.TrimLeft(`
Usage: lazyai examples [COMMAND]

Print example command lines for every command, or for COMMAND only. Each
command's help ('lazyai help COMMAND') ends with the same examples.
`, "\n"))
}
