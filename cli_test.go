package main

import (
	"bytes"
	"strings"
	"testing"
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
