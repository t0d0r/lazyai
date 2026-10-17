# agentctl

Control the AI coding agents running on this Mac from one terminal
dashboard: see what is running and whether it is busy, start new agents in
tmux, attach to them, type prompts into them, and stop them.

Supported agents: Claude Code, Codex, opencode, Gemini CLI, aider,
cursor-agent, ollama. Desktop apps (Claude.app, Cursor.app) are ignored.

## Build

```sh
make build       # bin/agentctl
make check       # gofmt, go vet, tests
make run         # build and start the dashboard
make install     # copy to ~/.local/bin (PREFIX=... to change)
make help        # all targets
```

Requires macOS, Go 1.24 or newer, and tmux for launch, attach and send.

## Use

```sh
bin/agentctl                       # dashboard
bin/agentctl list                  # table; -all adds helper processes
bin/agentctl list -json            # machine-readable
bin/agentctl launch claude         # new Claude Code session in tmux, here
bin/agentctl launch -dir ~/work/AI/x -name x codex
bin/agentctl launch ollama run llama3.2
bin/agentctl attach claude-x       # PID, tmux session or Claude session name
bin/agentctl send claude-x "run the tests"
bin/agentctl stop 12345            # asks first; -yes skips, -force kills
```

Dashboard keys:

| key       | action                                            |
|-----------|---------------------------------------------------|
| j / k     | move                                              |
| enter / d | toggle details                                    |
| a         | attach to the agent's tmux session                |
| p         | type a prompt into the agent and press Enter      |
| n         | launch: `KIND [DIR] [NAME] [-- ARGS]`             |
| s / K     | stop (SIGTERM) / force kill (SIGKILL), with y/N   |
| h         | show or hide helper processes                     |
| r / q     | refresh / quit                                    |

Attach and prompt only work for agents running inside tmux. Agents started
in a plain terminal tab are listed and can be stopped, but not driven.

Status comes from Claude Code's own session files when they exist
(`~/.claude/sessions/<pid>.json`); otherwise an agent is "busy" when it uses
2% CPU or more. FLAGS warns about high CPU for 30s, memory over 2 GiB, and
busy for over 30 minutes. `self` marks agentctl's own parent processes,
which cannot be stopped from it.

See `specs/` for the full behavior.

## Security

- agentctl only signals processes it classified as agents, owned by your
  user, and still matching the PID, command and start time it listed.
- Command lines are shown with values of key, token, secret and password
  flags replaced by `[redacted]`; the redaction is a heuristic, not a
  guarantee. Review `list -json` output before sharing it.
- `send` types text into an agent exactly as given; the agent then acts on
  it with its own permissions. Treat it like typing into that terminal.
- Claude Code `*.key` files next to the session metadata are never read.
