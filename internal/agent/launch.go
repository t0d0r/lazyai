package agent

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"lazyai/internal/tmux"
)

// LaunchKinds lists the kinds launch can start. Colibri is left out: it is
// not on PATH and is started with its own tooling (make run in its project).
func LaunchKinds() []Kind {
	var kinds []Kind
	for _, k := range Kinds {
		if k != KindColibri {
			kinds = append(kinds, k)
		}
	}
	return kinds
}

// LaunchCommand returns the argv that starts kind with extra arguments.
func LaunchCommand(kind Kind, extra []string) ([]string, error) {
	if kind == KindColibri {
		return nil, errors.New("colibri is not launched by lazyai; start it with its own tooling, for example make run in its project")
	}
	if kindRank(kind) == len(Kinds) {
		var names []string
		for _, k := range LaunchKinds() {
			names = append(names, string(k))
		}
		return nil, fmt.Errorf("unknown agent kind %q (known: %s)", kind, strings.Join(names, ", "))
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
