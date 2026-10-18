package agent

import (
	"path/filepath"
	"strings"
)

// Client modes: what a docker or podman command line asks for.
const (
	modeRun         = "run"          // docker run, docker container run
	modeExec        = "exec"         // docker exec, docker container exec
	modeComposeRun  = "compose-run"  // docker compose run, docker-compose run
	modeComposeExec = "compose-exec" // docker compose exec
	modeSandbox     = "sandbox"      // docker sandbox run AGENT
)

// clientCmd is a docker or podman command line that starts something in a
// container: "docker run [OPTIONS] IMAGE [COMMAND]" and its relatives.
type clientCmd struct {
	mode     string
	image    string   // run: the image as given
	name     string   // run --name, exec's container, compose service, sandbox agent
	command  []string // what runs in the container, when given
	detached bool
}

// Options that take a separate value, so the value is not mistaken for the
// image, container or service.
var (
	globalValueFlags = flagSet("--context", "-c", "--host", "-H", "--config", "--log-level", "-l",
		"--connection", "--url", "--identity")
	runValueFlags = flagSet("-v", "--volume", "-e", "--env", "--env-file",
		"-w", "--workdir", "-p", "--publish", "--name", "-u", "--user", "-l", "--label", "--label-file",
		"--mount", "--network", "--net", "-h", "--hostname", "--add-host", "--entrypoint", "--platform",
		"--cpus", "-m", "--memory", "--device", "--pull", "--restart", "--log-driver", "--log-opt",
		"--cap-add", "--cap-drop", "--security-opt", "--ulimit", "--tmpfs", "--shm-size", "--gpus",
		"--runtime", "--dns", "--expose", "--cidfile", "--detach-keys", "--ipc", "--pid", "--stop-signal")
	execValueFlags        = flagSet("-e", "--env", "--env-file", "-u", "--user", "-w", "--workdir", "--detach-keys")
	composeValueFlags     = flagSet("-f", "--file", "-p", "--project-name", "--project-directory", "--profile", "--env-file", "--ansi", "--progress", "--parallel")
	composeRunValueFlags  = flagSet("-e", "--env", "-l", "--label", "--name", "-p", "--publish", "-u", "--user", "-v", "--volume", "-w", "--workdir", "--entrypoint", "--cap-add", "--cap-drop", "--pull")
	composeExecValueFlags = flagSet("-e", "--env", "-u", "--user", "-w", "--workdir", "--index")
	sandboxValueFlags     = flagSet("--credentials", "-e", "--env", "--name", "-t", "--template", "-v", "--volume", "-w", "--workspace")
)

func flagSet(flags ...string) map[string]bool {
	m := make(map[string]bool, len(flags))
	for _, f := range flags {
		m[f] = true
	}
	return m
}

// parseClient reads a whole command line whose program is docker, podman or
// docker-compose. It returns false for everything else, docker ps included.
func parseClient(tokens []string) (clientCmd, bool) {
	switch filepath.Base(tokens[0]) {
	case "docker", "podman":
	case "docker-compose", "podman-compose":
		return parseCompose(tokens[1:])
	default:
		return clientCmd{}, false
	}
	rest, _ := skipFlags(tokens[1:], globalValueFlags, nil)
	if len(rest) > 0 && rest[0] == "container" {
		rest = rest[1:]
	}
	if len(rest) == 0 {
		return clientCmd{}, false
	}
	switch rest[0] {
	case "run":
		return parseRun(modeRun, rest[1:], runValueFlags)
	case "exec":
		return parseRun(modeExec, rest[1:], execValueFlags)
	case "compose":
		return parseCompose(rest[1:])
	case "sandbox":
		if len(rest) > 1 && rest[1] == "run" {
			return parseRun(modeSandbox, rest[2:], sandboxValueFlags)
		}
	}
	return clientCmd{}, false
}

// parseCompose reads what follows "docker compose".
func parseCompose(args []string) (clientCmd, bool) {
	rest, _ := skipFlags(args, composeValueFlags, nil)
	if len(rest) == 0 {
		return clientCmd{}, false
	}
	switch rest[0] {
	case "run":
		return parseRun(modeComposeRun, rest[1:], composeRunValueFlags)
	case "exec":
		return parseRun(modeComposeExec, rest[1:], composeExecValueFlags)
	}
	return clientCmd{}, false
}

// parseRun reads "[OPTIONS] TARGET [COMMAND...]", where TARGET is the
// image, container, service or sandbox agent.
func parseRun(mode string, args []string, valueFlags map[string]bool) (clientCmd, bool) {
	cmd := clientCmd{mode: mode}
	rest, flags := skipFlags(args, valueFlags, func(flag, value string) {
		if flag == "--name" && mode == modeRun {
			cmd.name = value
		}
	})
	for _, f := range flags {
		// --detach, -d, or d among combined short flags such as -dit
		if f == "--detach" || f == "--detached" || !strings.HasPrefix(f, "--") && strings.Contains(f, "d") {
			cmd.detached = true
		}
	}
	if len(rest) == 0 {
		return clientCmd{}, false
	}
	if mode == modeRun {
		cmd.image = rest[0]
	} else {
		cmd.name = rest[0]
	}
	cmd.command = rest[1:]
	return cmd, true
}

// skipFlags returns args from the first one that is not an option, and the
// options it skipped. Options in valueFlags take the next word as their
// value; set, when not nil, is told each option's value.
func skipFlags(args []string, valueFlags map[string]bool, set func(flag, value string)) ([]string, []string) {
	var flags []string
	for i := 0; i < len(args); i++ {
		t := args[i]
		if !strings.HasPrefix(t, "-") || t == "-" {
			return args[i:], flags
		}
		if t == "--" {
			return args[i+1:], flags
		}
		flag, value, inline := strings.Cut(t, "=")
		flags = append(flags, flag)
		if !inline && valueFlags[flag] && i+1 < len(args) {
			i++
			value = args[i]
		}
		if set != nil {
			set(flag, value)
		}
	}
	return nil, flags
}

// classify decides the kind from the command run in the container, and
// otherwise from the image (run), the service (compose run) or the sandbox
// agent. A detached container counts as a server.
func (c clientCmd) classify() (Kind, Role, bool) {
	kind, role, ok := classifyCommand(c.command)
	if !ok {
		switch c.mode {
		case modeRun:
			kind = Kind(imageRepo(c.image))
		case modeComposeRun, modeSandbox:
			kind = Kind(c.name)
		}
		role, ok = RoleInteractive, kindRank(kind) < len(Kinds)
	}
	if !ok {
		return "", "", false
	}
	if c.detached && role == RoleInteractive {
		role = RoleServer
	}
	return kind, role, true
}

// classifyCommand classifies a command given as words, looking through a
// shell's command string: "bash -lc claude --resume x" is claude.
func classifyCommand(tokens []string) (Kind, Role, bool) {
	if len(tokens) == 0 {
		return "", "", false
	}
	if k, r, ok := Classify(strings.Join(tokens, " ")); ok {
		return k, r, true
	}
	base := filepath.Base(tokens[0])
	if isInterpreter(base) {
		for i, t := range tokens[1:] {
			if runsCommandString(base, t) {
				return classifyCommand(tokens[i+2:])
			}
			if !strings.HasPrefix(t, "-") {
				break
			}
		}
	}
	return "", "", false
}

// imageRepo returns the last path segment of an image reference without tag
// or digest: ghcr.io/anomalyco/opencode:latest -> opencode.
func imageRepo(image string) string {
	image, _, _ = strings.Cut(image, "@")
	if j := strings.LastIndex(image, ":"); j > strings.LastIndex(image, "/") {
		image = image[:j] // drop the tag, not a registry port
	}
	return filepath.Base(image)
}

// sameImage compares two image references the way docker resolves them:
// "node" is "docker.io/library/node:latest".
func sameImage(a, b string) bool {
	return normImage(a) == normImage(b)
}

func normImage(image string) string {
	image, _, _ = strings.Cut(image, "@")
	if strings.LastIndex(image, ":") <= strings.LastIndex(image, "/") {
		image += ":latest"
	}
	image = strings.TrimPrefix(image, "docker.io/")
	return strings.TrimPrefix(image, "library/")
}
