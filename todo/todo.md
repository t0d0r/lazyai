# Todo

## Done

- [x] process snapshot and parsing (specs/discovery.md)
- [x] agent classification and roles (specs/discovery.md)
- [x] Claude session metadata with PID-reuse check (specs/discovery.md)
- [x] tmux launch, pane mapping, send (specs/control.md)
- [x] safe stop with identity re-check and self protection (specs/control.md)
- [x] CLI: list, stop, launch, attach, send (specs/control.md)
- [x] TUI dashboard with watch flags (specs/tui.md)
- [x] unit tests and a tmux integration test
- [x] Makefile with the standard targets from ../AGENTS.md
- [x] CLI parity with the TUI: show, watch, kill, flags in list, stop by
      name (specs/cli.md)
- [x] Unix-style CLI: -h/--help everywhere, long and short options,
      options after arguments, --version, exit status 2 for usage errors
      (specs/cli.md)

- [x] fix: Homebrew framework Python (Python.app) was excluded as a desktop
      app, hiding Python agents such as Hermes (specs/discovery.md)
- [x] kinds hermes, qwen (Qwen Code), colibri; roles job and engine
      (specs/discovery.md)

## Next

- [ ] verify attach by hand from a real terminal, outside and inside tmux
- [ ] verify send into a real Claude Code session in tmux (paste and Enter
      timing)
- [ ] check classification against real gemini, aider, cursor-agent and
      ollama processes (only claude, codex, opencode were running)
- [ ] decide opencode launch: the fish alias runs it in docker, lazyai
      runs the Homebrew binary
- [ ] Codex status and session id from ~/.codex metadata (needs a spec)
- [ ] verify Hermes server roles (gateway, proxy, lsp) and Qwen Code
      against real processes; only interactive Hermes has been seen
- [ ] attribute child CPU and memory to the agent that started them (for
      example Hermes -> make -> coli tune -> qwen36), shown in details
- [ ] colibri server status from `coli status` or its pid file, if it has
      one
- [ ] run `make install` and confirm ~/.local/bin is on PATH in fish
- [ ] add a gitleaks pre-commit hook
