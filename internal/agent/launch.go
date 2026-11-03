package agent

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"lazyai/internal/tmux"
)

// LaunchKinds lists the kinds launch can start. Colibri is left out: it is
// not on PATH and is started with its own tooling (make run in its project).
func LaunchKinds() []Kind {
	var kinds []Kind
	for _, k := range Kinds {
		if notLaunched[k] == "" {
			kinds = append(kinds, k)
		}
	}
	return kinds
}

// notLaunched names the kinds launch refuses and says how to start them.
var notLaunched = map[Kind]string{
	KindColibri: "colibri is not launched by lazyai; start it with its own tooling, for example make run in its project",
	KindMLX:     "mlx-lm is not launched by lazyai; start it with mlx_lm.server or mlx_lm.chat and a --model",
}

// LaunchCommand returns the argv that starts kind with extra arguments.
// Besides the known kinds, any command name found on PATH is accepted, so
// wrapper scripts such as claude-bst (claude with another profile) launch
// like the agent they wrap.
func LaunchCommand(kind Kind, extra []string) ([]string, error) {
	if reason := notLaunched[kind]; reason != "" {
		return nil, errors.New(reason)
	}
	if kindRank(kind) == len(Kinds) && !IsCommand(string(kind)) {
		var names []string
		for _, k := range LaunchKinds() {
			names = append(names, string(k))
		}
		return nil, fmt.Errorf("%q is not a known agent kind (%s) or a command on PATH",
			kind, strings.Join(names, ", "))
	}
	if kind == KindOllama && len(extra) == 0 {
		return nil, errors.New("ollama needs arguments, for example: ollama -- run llama3.2")
	}
	return append([]string{string(kind)}, extra...), nil
}

// Launch starts an agent in a new tmux session and returns the session name.
// An empty name defaults to "<kind>-<dir basename>".
func Launch(kind Kind, dir, name string, extra []string) (string, error) {
	argv, err := LaunchCommand(kind, extra)
	if err != nil {
		return "", err
	}
	if dir, err = filepath.Abs(dir); err != nil {
		return "", err
	}
	if name == "" {
		name = string(kind) + "-" + filepath.Base(dir)
	}
	session := tmux.FreeSessionName(name)
	return session, tmux.Launch(session, dir, argv)
}

// IsCommand reports whether name is a bare command name (no path, no
// leading "-") that resolves to an executable on PATH.
func IsCommand(name string) bool {
	if name == "" || strings.ContainsRune(name, '/') || strings.HasPrefix(name, "-") {
		return false
	}
	_, err := exec.LookPath(name)
	return err == nil
}

// LaunchUsage is the launch line that both the TUI (n) and the CLI read.
const LaunchUsage = "KIND [DIR] [NAME] [-- ARGS...]"

// ErrLaunchSyntax marks a launch line that does not have the form of
// LaunchUsage, as opposed to one naming an unknown kind.
var ErrLaunchSyntax = errors.New("usage: " + LaunchUsage)

// LaunchSpec is a parsed launch line.
type LaunchSpec struct {
	Kind Kind
	Dir  string   // absolute working directory
	Name string   // session name, "" for the default
	Args []string // agent arguments, everything after "--"
}

// ParseLaunch reads "KIND [DIR] [NAME] [-- ARGS...]" from words. DIR may
// use "~" and is relative to baseDir, which is also the default. Words
// before "--" that start with "-" are refused, since agent arguments go
// after "--".
func ParseLaunch(words []string, baseDir string) (LaunchSpec, error) {
	head, tail := words, []string(nil)
	if i := slices.Index(words, "--"); i >= 0 {
		head, tail = words[:i], words[i+1:]
	}
	if len(head) == 0 || len(head) > 3 {
		return LaunchSpec{}, ErrLaunchSyntax
	}
	for _, w := range head[1:] {
		if strings.HasPrefix(w, "-") {
			return LaunchSpec{}, fmt.Errorf("%w (agent arguments such as %s go after --)", ErrLaunchSyntax, w)
		}
	}
	spec := LaunchSpec{Kind: Kind(head[0]), Dir: ExpandDir(".", baseDir), Args: tail}
	if _, err := LaunchCommand(spec.Kind, []string{"check"}); err != nil {
		return LaunchSpec{}, err
	}
	if len(head) > 1 {
		spec.Dir = ExpandDir(head[1], baseDir)
	}
	if len(head) > 2 {
		spec.Name = head[2]
	}
	return spec, nil
}

// ExpandDir resolves p with "~" for the home directory and relative to base.
func ExpandDir(p, base string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(base, p)
	}
	return filepath.Clean(p)
}
