package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"lazyai/internal/agent"
	"lazyai/internal/update"
)

func runCLI(args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := run(args, env{in: strings.NewReader(""), out: &out, err: &errOut})
	return code, out.String(), errOut.String()
}

func TestTopLevelHelp(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"--help"}, {"-help"}, {"help"}} {
		code, out, errOut := runCLI(args...)
		if code != 0 || errOut != "" || !strings.HasPrefix(out, "Usage: lazyai [COMMAND]") {
			t.Errorf("%v: code %d, stderr %q, stdout %.40q", args, code, errOut, out)
		}
		for _, c := range commands {
			if !strings.Contains(out, "  "+c.name+" ") {
				t.Errorf("%v: command list lacks %s", args, c.name)
			}
		}
	}
}

func TestCommandHelp(t *testing.T) {
	for _, c := range commands {
		for _, args := range [][]string{{"help", c.name}, {c.name, "-h"}, {c.name, "--help"}} {
			code, out, errOut := runCLI(args...)
			if code != 0 || errOut != "" || !strings.HasPrefix(out, "Usage: lazyai ") ||
				!strings.Contains(out, "-h, --help") {
				t.Errorf("%v: code %d, stderr %q, stdout:\n%s", args, code, errOut, out)
			}
		}
	}
	_, out, _ := runCLI("list", "--help")
	for _, want := range []string{"-a, --all", "-j, --json"} {
		if !strings.Contains(out, want) {
			t.Errorf("list help lacks %q:\n%s", want, out)
		}
	}
	_, out, _ = runCLI("watch", "-h")
	if !strings.Contains(out, "-i, --interval DURATION") || !strings.Contains(out, "(default 2s)") {
		t.Errorf("watch help:\n%s", out)
	}
	if _, out, _ = runCLI("ls", "-h"); !strings.HasPrefix(out, "Usage: lazyai list") {
		t.Errorf("alias help:\n%s", out)
	}
}

func TestExamples(t *testing.T) {
	for _, c := range commands {
		if len(examples[c.name]) == 0 {
			t.Errorf("no examples for %s", c.name)
		}
		for _, ex := range examples[c.name] {
			if !strings.HasPrefix(ex.cmd, "lazyai") || ex.note == "" {
				t.Errorf("%s: bad example %+v", c.name, ex)
			}
		}
		if _, out, _ := runCLI("help", c.name); !strings.Contains(out, "Examples:\n  "+examples[c.name][0].cmd) {
			t.Errorf("help %s lacks examples:\n%s", c.name, out)
		}
	}
	code, out, _ := runCLI("examples")
	if code != 0 || !strings.Contains(out, "launch - start an agent") ||
		!strings.Contains(out, "lazyai launch claude-bst ~/work/alpha/web-frontend web") {
		t.Errorf("examples: %d\n%s", code, out)
	}
	if code, out, _ = runCLI("examples", "ls"); code != 0 || !strings.HasPrefix(out, "  lazyai list ") {
		t.Errorf("examples ls: %d %q", code, out)
	}
	for _, args := range [][]string{{"help", "examples"}, {"examples", "-h"}} {
		if code, out, _ = runCLI(args...); code != 0 || !strings.HasPrefix(out, "Usage: lazyai examples") {
			t.Errorf("%v: %d %q", args, code, out)
		}
	}
	for _, args := range [][]string{{"examples", "nope"}, {"examples", "a", "b"}} {
		if code, _, _ = runCLI(args...); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
	// Every launch example parses as a launch line.
	for _, ex := range examples["launch"] {
		words := strings.Fields(ex.cmd)[2:]
		for len(words) > 0 && strings.HasPrefix(words[0], "--dir") {
			words = words[2:]
		}
		if _, err := agent.ParseLaunch(words, "/base"); errors.Is(err, agent.ErrLaunchSyntax) {
			t.Errorf("%q: %v", ex.cmd, err)
		}
	}
}

func TestLaunchLine(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"launch", "claude", "--model", "x"}, "agent arguments such as --model go after --"},
		{[]string{"launch", "claude", "a", "b", "c"}, "expected KIND [DIR] [NAME] [-- ARGS...]"},
		{[]string{"launch", "--name", "x", "claude", ".", "y"}, "NAME given twice"},
	} {
		if code, _, errOut := runCLI(c.args...); code != 2 || !strings.Contains(errOut, c.want) {
			t.Errorf("%v: %d %q", c.args, code, errOut)
		}
	}
	if code, _, errOut := runCLI("launch", "no-such-command-x"); code != 1 || !strings.Contains(errOut, "command on PATH") {
		t.Errorf("unknown kind: %d %q", code, errOut)
	}
}

func TestVersion(t *testing.T) {
	for _, args := range [][]string{{"-V"}, {"--version"}, {"version"}} {
		if code, out, _ := runCLI(args...); code != 0 || out != "lazyai dev\n" {
			t.Errorf("%v: %d %q", args, code, out)
		}
	}
}

func TestUsageErrors(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"nope"}, `lazyai: unknown command "nope"`},
		{[]string{"help", "nope"}, `lazyai: unknown command "nope"`},
		{[]string{"list", "--bogus"}, "lazyai list: flag provided but not defined: -bogus\nRun 'lazyai list --help' for usage."},
		{[]string{"list", "extra"}, `lazyai list: unexpected argument "extra"`},
		{[]string{"show"}, "lazyai show: expected one AGENT, got 0 arguments"},
		{[]string{"stop"}, "lazyai stop: missing AGENT"},
		{[]string{"launch"}, "lazyai launch: missing KIND"},
		{[]string{"send", "x"}, "lazyai send: expected AGENT and TEXT"},
		{[]string{"rename", "x"}, "lazyai rename: expected AGENT and NAME"},
		{[]string{"watch", "-i"}, "lazyai watch: flag needs an argument: -i"},
		{[]string{"--bogus"}, "lazyai: flag provided but not defined: -bogus\nRun 'lazyai --help' for usage."},
	}
	for _, c := range cases {
		code, out, errOut := runCLI(c.args...)
		if code != 2 || out != "" || !strings.Contains(errOut, c.want) {
			t.Errorf("%v: code %d, stdout %q, stderr %q; want %q", c.args, code, out, errOut, c.want)
		}
	}
}

func TestTmuxColumns(t *testing.T) {
	cases := []struct {
		name            string
		agent           agent.Agent
		wantIn, wantSID string
	}{
		{"outside", agent.Agent{}, "no", "-"},
		{"in tmux", agent.Agent{TmuxSession: "lazyai-work", TmuxPane: "%3"}, "yes", "lazyai-work"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in, session := tmuxColumns(tc.agent)
			if in != tc.wantIn || session != tc.wantSID {
				t.Errorf("tmuxColumns() = %q, %q; want %q, %q", in, session, tc.wantIn, tc.wantSID)
			}
		})
	}
}

func TestParsePermute(t *testing.T) {
	f := newFlagDefs("stop")
	yes := f.Bool("y", "yes", "")
	force := f.Bool("f", "force", "")
	pos, err := f.parse([]string{"123", "-y", "abc", "--force", "--", "-7"}, true)
	if err != nil || strings.Join(pos, " ") != "123 abc -7" || !*yes || !*force {
		t.Errorf("permute: %v %v yes=%v force=%v", pos, err, *yes, *force)
	}

	f = newFlagDefs("send")
	pos, err = f.parse([]string{"agent", "-h", "--now"}, false)
	if err != nil || strings.Join(pos, " ") != "agent -h --now" {
		t.Errorf("no permute: %v %v", pos, err)
	}

	f = newFlagDefs("launch")
	dir := f.String("d", "dir", "DIR", ".", "")
	pos, err = f.parse([]string{"--dir", "/x", "claude", "--model", "m"}, false)
	if err != nil || *dir != "/x" || strings.Join(pos, " ") != "claude --model m" {
		t.Errorf("launch: %v %v dir=%q", pos, err, *dir)
	}
}

func TestBrewTarget(t *testing.T) {
	v := func(s string) update.Version { x, _ := update.Parse(s); return x }
	cur, latest := v("v0.9.0"), v("v0.10.0")
	for _, tc := range []struct {
		tap    string
		err    error
		want   string
		inNote string
	}{
		{"v0.10.0", nil, "v0.10.0", ""},
		{"v0.9.0", nil, "v0.9.0", "the Homebrew tap still has v0.9.0"},
		{"v0.8.0", nil, "v0.9.0", "run lazyai update again later"},
		{"v0.9.5", nil, "v0.9.5", "brew upgrade installs v0.9.5"},
		{"", errors.New("404"), "v0.10.0", "could not read the Homebrew formula (404)"},
	} {
		got, note := brewTarget(cur, latest, v(tc.tap), tc.err)
		if got.String() != tc.want || (tc.inNote == "") != (note == "") || !strings.Contains(note, tc.inNote) {
			t.Errorf("tap %s: target %s, note %q", tc.tap, got, note)
		}
	}
}
