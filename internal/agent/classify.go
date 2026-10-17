package agent

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Kind is the agent product a process belongs to.
type Kind string

const (
	KindClaude   Kind = "claude"
	KindCodex    Kind = "codex"
	KindOpencode Kind = "opencode"
	KindGemini   Kind = "gemini"
	KindQwen     Kind = "qwen" // Qwen Code CLI
	KindAider    Kind = "aider"
	KindCursor   Kind = "cursor-agent"
	KindHermes   Kind = "hermes" // Hermes Agent
	KindOllama   Kind = "ollama"
	KindColibri  Kind = "colibri" // Colibri local model runtime
)

// Kinds lists every supported kind in display order.
var Kinds = []Kind{KindClaude, KindCodex, KindOpencode, KindGemini, KindQwen, KindAider,
	KindCursor, KindHermes, KindOllama, KindColibri}

// Role says what the process does for its agent.
type Role string

const (
	RoleInteractive Role = "interactive" // a session a person talks to
	RoleServer      Role = "server"      // a long-running API or app server
	RoleDaemon      Role = "daemon"      // supervisor or background manager
	RoleJob         Role = "job"         // batch work: tune, bench, build
	RoleEngine      Role = "engine"      // model inference process
	RoleHelper      Role = "helper"      // spare or pty host processes
)

// Classify decides whether a command line belongs to a supported agent.
func Classify(args string) (Kind, Role, bool) {
	tokens := strings.Fields(args)
	if len(tokens) == 0 {
		return "", "", false
	}
	argv0 := tokens[0]
	if isDesktopApp(argv0) {
		return "", "", false
	}
	candidates := []string{argv0}
	rest := tokens[1:]
	viaInterpreter := false
	if isInterpreter(filepath.Base(argv0)) {
		for i, t := range rest {
			if !strings.HasPrefix(t, "-") {
				candidates = append(candidates, t)
				rest = rest[i+1:]
				viaInterpreter = true
				break
			}
		}
	}
	for i, c := range candidates {
		if k := kindOf(c); k != "" {
			return k, roleOf(k, c, viaInterpreter && i > 0, rest), true
		}
	}
	return "", "", false
}

// isDesktopApp reports whether argv0 lives inside a macOS app bundle, such
// as Claude.app or Cursor.app and their helpers. Those are out of scope.
// Homebrew's framework Python runs from Python.app, so it is not one.
func isDesktopApp(argv0 string) bool {
	bundle, _, found := strings.Cut(argv0, ".app/Contents/")
	return found && !strings.HasSuffix(bundle, "/Python")
}

func isInterpreter(base string) bool {
	base = strings.ToLower(base)
	switch base {
	case "node", "bun", "deno", "bash", "sh", "zsh":
		return true
	}
	return strings.HasPrefix(base, "python")
}

// colibriEngine matches Colibri's native model engines (qwen36, qwen38,
// qwenimage, colibri, glm...). Only paths containing "colibri" qualify.
var colibriEngine = regexp.MustCompile(`^(qwen[0-9]+\w*|qwenimage|colibri|glm\w*)$`)

func kindOf(path string) Kind {
	base := filepath.Base(path)
	for _, ext := range []string{".js", ".mjs", ".cjs", ".py"} {
		base = strings.TrimSuffix(base, ext)
	}
	if base == "coli" || strings.Contains(path, "colibri") && colibriEngine.MatchString(base) {
		return KindColibri
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
	case strings.Contains(path, "/@qwen-code/"):
		return KindQwen
	case strings.Contains(path, "/cursor-cli/"):
		return KindCursor
	case strings.Contains(path, "/hermes-agent/"):
		return KindHermes
	}
	return ""
}

// roleOf decides the role from the matched program and the arguments after
// it. script is true when the program is a script run by an interpreter.
func roleOf(k Kind, program string, script bool, rest []string) Role {
	has := func(word string) bool { return slices.Contains(rest, word) }
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
	case KindHermes:
		switch first {
		case "gateway", "proxy", "lsp":
			return RoleServer
		}
	case KindOllama:
		switch first {
		case "serve":
			return RoleServer
		case "runner":
			return RoleEngine
		}
	case KindColibri:
		if !script && strings.TrimSuffix(filepath.Base(program), ".py") != "coli" {
			return RoleEngine
		}
		switch first {
		case "serve", "web", "cluster", "start":
			return RoleServer
		case "chat", "run", "":
			return RoleInteractive
		}
		return RoleJob
	}
	return RoleInteractive
}
