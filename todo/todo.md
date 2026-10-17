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

## Next

- [ ] verify attach by hand from a real terminal, outside and inside tmux
- [ ] verify send into a real Claude Code session in tmux (paste and Enter
      timing)
- [ ] check classification against real gemini, aider, cursor-agent and
      ollama processes (only claude, codex, opencode were running)
- [ ] decide opencode launch: the fish alias runs it in docker, agentctl
      runs the Homebrew binary
- [ ] Codex status and session id from ~/.codex metadata (needs a spec)
- [ ] run `make install` and confirm ~/.local/bin is on PATH in fish
- [ ] add a gitleaks pre-commit hook
