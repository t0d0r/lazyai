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
- Fixed `lazyai -h` (it went to the tui flag parser and never printed
  the usage). The CLI is now Unix-style: commands live in a table in
  commands.go that drives parsing and generated help; flags.go wraps
  flag.FlagSet with long/short options and argument permutation;
  `--version` is stamped by `make build`; exit status 2 for usage errors.
  `stop` now reads confirmations through one buffered reader, so piped
  answers for several agents are no longer lost.
- Operator reported Colibri running Qwen (driven by Hermes Agent) was not
  shown. Causes: the `.app/Contents/` desktop-app exclusion also matched
  Homebrew's framework Python, hiding Hermes; and Hermes, Colibri and
  Qwen Code were not known kinds. Fixed the exclusion, added the three
  kinds and roles `job` and `engine` (ollama runners moved from helper to
  engine). Verified live: hermes interactive, `coli tune` job, `qwen36`
  engine flagged high-mem.

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

- Renamed the project from agentctl to lazyai (directory ../lazyai, Go
  module, binary, tmux prefix `lazyai-`). Older lines in this note were
  rewritten with the new name.
- Rebuilt the TUI after lazydocker: Agents / Services / Models list panels
  plus a status panel on the left; Info, Screen (live tmux pane), Processes
  (agent's process tree) and Stats (ASCII CPU and memory charts) tabs on
  the right; x action menu, ? keybindings, confirm and input popups
  overlaid in the middle; mouse select and scroll. Discovery now records
  each agent's descendants and tree totals; the watcher keeps 150 samples
  of history. Verified in a 160x40 tmux window against the live system
  and a throwaway Claude session (Screen showed its trust prompt).
- Operator screenshot in iTerm: Agents took all spare rows and Services /
  Models were squeezed. Lists now split the column evenly; only very
  short terminals give the list in use the room.
- Operator asked how to leave an attached session; lazyai sessions now
  show `ctrl+b d: detach, back to lazyai` (or `ctrl+b L` when lazyai runs
  inside tmux) in their own tmux status-right. Verified by attaching in a
  pseudo terminal. tmux option commands need `=name:` as the target.

## Files

main.go, commands.go, flags.go, main_test.go, cli_test.go,
internal/agent/*, internal/tmux/*, internal/tui/{tui,view,keys,chart}.go,
internal/watch/*, internal/display/display.go, Makefile, README.md,
AGENTS.md, specs/*, todo/todo.md.

## Run and verify

```sh
make check
make run
```

`make install` was not run; it copies the binary to ~/.local/bin.

The tmux test creates and removes a session named `lazyai-test-*` on the
default tmux server.

## Next step

Run `bin/lazyai` in a normal terminal, launch a Claude session with `n`,
watch it in the Screen tab (`]`), attach with `a`, detach with ctrl+b d,
send a prompt with `p`, and try the mouse.
