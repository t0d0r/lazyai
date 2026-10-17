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
bin/agentctl show claude-x         # details of one agent; -json too
bin/agentctl watch                 # agents, then changes as they happen
bin/agentctl watch -json | jq .    # one JSON event per line
bin/agentctl launch claude         # new Claude Code session in tmux, here
bin/agentctl launch -dir ~/work/AI/x -name x codex
bin/agentctl launch ollama run llama3.2
bin/agentctl attach claude-x
bin/agentctl send claude-x "run the tests"
bin/agentctl stop claude-x         # asks first; -yes skips, -force kills
bin/agentctl kill 12345            # same as stop -force
```

AGENT arguments accept a PID, a tmux session name (the `agentctl-` prefix
is optional) or a Claude Code session name.

Every dashboard action has a command:

| key       | action                                            | command             |
|-----------|---------------------------------------------------|---------------------|
| j / k     | move                                              | -                   |
| enter / d | toggle details                                    | `show AGENT`        |
| a         | attach to the agent's tmux session                | `attach AGENT`      |
| p         | type a prompt into the agent and press Enter      | `send AGENT TEXT`   |
| n         | launch: `KIND [DIR] [NAME] [-- ARGS]`             | `launch`            |
| s / K     | stop (SIGTERM) / force kill (SIGKILL), with y/N   | `stop` / `kill`     |
| h         | show or hide helper processes                     | `list -all`         |
| live view | status durations and flags                        | `watch`             |
| r / q     | refresh / quit                                    | -                   |

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
