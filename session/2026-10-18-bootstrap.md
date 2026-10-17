# 2026-10-18 bootstrap

## Done

- Created the project: Go TUI plus CLI to list, watch, launch, attach to,
  send prompts to, and stop local AI agents.
- Operator choices: all four capability groups, TUI, Go; agents Claude
  Code, Codex and the others (opencode, gemini, aider, cursor-agent,
  ollama); Claude Desktop excluded.
- Found that Claude Code writes `~/.claude/sessions/<pid>.json` with session
  id, cwd, name and busy/idle status; `procStart` there is UTC.
- Verified on the live system: discovery of claude, codex and opencode
  processes; launch of a Claude session in tmux; TUI render and detail
  view in a 160x34 tmux window; stop through the TUI confirm flow.
- Added a Makefile (help, build, test, lint, fmt, check, run, install,
  uninstall, tidy, clean). The parent ../AGENTS.md now requires a Makefile
  with standard targets in every project.
- CLI now covers every TUI action: `show`, `watch` (change events, text or
  JSON lines), `kill`, FLAGS column in `list`, and `stop`/`kill` by PID or
  name. Tracking and flags moved from internal/tui to internal/watch so
  both interfaces share them. Verified live: watch reported started,
  busy -> idle and exited for a test Claude session launched and killed by
  name.
- Fixed `agentctl -h` (it went to the tui flag parser and never printed
  the usage). The CLI is now Unix-style: commands live in a table in
  commands.go that drives parsing and generated help; flags.go wraps
  flag.FlagSet with long/short options and argument permutation;
  `--version` is stamped by `make build`; exit status 2 for usage errors.
  `stop` now reads confirmations through one buffered reader, so piped
  answers for several agents are no longer lost.

## Decisions

- tmux is the only control channel for attach and send. Agents started in
  plain terminal tabs can be listed and stopped only.
- Status for non-Claude agents is a CPU heuristic (>= 2%).
- Durations for CPU-based status appear only after a change is observed.
- Helper processes (Claude pty hosts and spares, ollama runners) are hidden
  by default.

## Open

- Attach was not exercised with a real terminal client (no client in the
  agent's shell). Send was tested against `cat`, not a real agent.
- gemini, aider, cursor-agent and ollama were not running; their
  classification is tested with sample command lines only.

## Files

main.go, commands.go, flags.go, main_test.go, cli_test.go, internal/agent/*, internal/tmux/*, internal/tui/*,
internal/watch/*, internal/display/display.go, Makefile, README.md,
AGENTS.md, specs/*, todo/todo.md.

## Run and verify

```sh
make check
make run
```

`make install` was not run; it copies the binary to ~/.local/bin.

The tmux test creates and removes a session named `agentctl-test-*` on the
default tmux server.

## Next step

Run `bin/agentctl` in a normal terminal, launch a Claude session with `n`,
attach with `a`, detach with ctrl+b d, and send a prompt with `p`.
