package agent

import (
	"path/filepath"
	"strings"
)

// Kind is the agent product a process belongs to.
type Kind string

const (
	KindClaude   Kind = "claude"
	KindCodex    Kind = "codex"
	KindOpencode Kind = "opencode"
	KindGemini   Kind = "gemini"
	KindAider    Kind = "aider"
	KindCursor   Kind = "cursor-agent"
	KindOllama   Kind = "ollama"
)

// Kinds lists every supported kind in display order.
var Kinds = []Kind{KindClaude, KindCodex, KindOpencode, KindGemini, KindAider, KindCursor, KindOllama}

// Role says what the process does for its agent.
type Role string

const (
	RoleInteractive Role = "interactive" // a session a person talks to
	RoleServer      Role = "server"      // a long-running API or app server
	RoleDaemon      Role = "daemon"      // supervisor or background manager
	RoleHelper      Role = "helper"      // spare or pty host processes, model runners
)

// Classify decides whether a command line belongs to a supported agent.
func Classify(args string) (Kind, Role, bool) {
	tokens := strings.Fields(args)
	if len(tokens) == 0 {
		return "", "", false
	}
	argv0 := tokens[0]
	// Desktop apps (Claude.app, Cursor.app and their helpers) are out of scope.
	if strings.Contains(argv0, ".app/Contents/") {
		return "", "", false
	}
	candidates := []string{argv0}
	rest := tokens[1:]
	if isInterpreter(filepath.Base(argv0)) {
		for i, t := range rest {
			if !strings.HasPrefix(t, "-") {
				candidates = append(candidates, t)
				rest = rest[i+1:]
				break
			}
		}
	}
	for _, c := range candidates {
		if k := kindOf(c); k != "" {
			return k, roleOf(k, rest), true
		}
	}
	return "", "", false
}

func isInterpreter(base string) bool {
	switch base {
	case "node", "bun", "deno", "bash", "sh", "zsh":
		return true
	}
	return strings.HasPrefix(base, "python")
}

func kindOf(path string) Kind {
	base := filepath.Base(path)
	for _, ext := range []string{".js", ".mjs", ".cjs", ".py"} {
		base = strings.TrimSuffix(base, ext)
	}
	for _, k := range Kinds {
		if base == string(k) {
			return k
		}
	}
	switch {
	case strings.Contains(path, "/.local/share/claude/versions/"),
		strings.Contains(path, "/@anthropic-ai/claude-code/"):
		return KindClaude
	case strings.Contains(path, "/@openai/codex/"):
		return KindCodex
	case strings.Contains(path, "/gemini-cli/"):
		return KindGemini
	case strings.Contains(path, "/cursor-cli/"):
		return KindCursor
	}
	return ""
}

func roleOf(k Kind, rest []string) Role {
	has := func(word string) bool {
		for _, t := range rest {
			if t == word {
				return true
			}
		}
		return false
	}
	first := ""
	if len(rest) > 0 {
		first = rest[0]
	}
	switch k {
	case KindClaude:
		switch {
		case has("bg-pty-host"), has("bg-spare"):
			return RoleHelper
		case first == "daemon":
			return RoleDaemon
		}
	case KindCodex:
		switch {
		case has("app-server") && has("daemon"):
			return RoleDaemon
		case has("app-server"), first == "mcp-server":
			return RoleServer
		}
	case KindOpencode:
		if first == "serve" || first == "web" {
			return RoleServer
		}
	case KindOllama:
		switch first {
		case "serve":
			return RoleServer
		case "runner":
			return RoleHelper
		}
	}
	return RoleInteractive
}
