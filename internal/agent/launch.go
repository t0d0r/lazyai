package agent

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"agentctl/internal/tmux"
)

// LaunchCommand returns the argv that starts kind with extra arguments.
func LaunchCommand(kind Kind, extra []string) ([]string, error) {
	if kindRank(kind) == len(Kinds) {
		names := make([]string, len(Kinds))
		for i, k := range Kinds {
			names[i] = string(k)
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
