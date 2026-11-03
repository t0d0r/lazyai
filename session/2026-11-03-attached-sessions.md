# 2026-11-03 - show attached tmux sessions

## Done

- Discovery reads `#{session_attached}` and `#{session_last_attached}`
  in the same `tmux list-panes -a` call (no extra process per refresh).
  tmux.Pane gains Clients and LastAttached; parsing moved to parsePanes
  so it is unit-tested. agent.Agent gains `tmux_clients` and
  `tmux_last_attached` (JSON, left out when 0 / never), set through
  Agent.setPane for host and container agents (specs/discovery.md).
- Row gutter: `>` for an agent in tmux with a client attached, `@` for
  detached, blank outside tmux. The `?` popup explains both.
- Status panel: `N running, B busy, A attached` when any shown agent is
  in tmux; narrow forms keep `mem` before dropping the long words, then
  drop `att` last (specs/tui.md, Layout).
- Info tab and `lazyai show`: the tmux line ends with
  `attached (N clients)`, `detached, last attached YYYY-MM-DD HH:MM` or
  `detached, never attached` (display.Attached).
- Filter: `attached` / `detached` match the tmux state.
- `p` on an attached agent asks for `y` before the prompt popup; the CLI
  `send` does not ask (specs/control.md, Send).
- Tests: TestParsePanes, TestAttached (display), TestAttachedMark,
  TestPromptAttachedConfirm.

## Decisions

- A separate character, not only a color, so the state reads without
  colors and in the reversed selected row.
- Only the client count is read, not `list-clients` (user, tty, host):
  enough to know whether someone looks, without exposing who.
- No warning on `a` for an already attached session: the message line is
  replaced when attach returns, and `>` already shows the state.
- CLI `list` table columns unchanged, so scripts reading them keep
  working; the state is in `-json` and `show`.

## Open

- Not released yet (todo: release v0.5.0).
- Checked at 140x30 against live agents: `>` and `@` rows and
  `28 run, 2 busy, 6 att  mem ...` in the status panel; the attached
  count equals the `(attached)` sessions in `tmux ls`. The `p` confirm
  is only covered by tests so far.

## Files

internal/tmux/{tmux.go,tmux_test.go}, internal/agent/{discover.go,
container.go}, internal/display/{display.go,display_test.go},
internal/tui/{tui.go,view.go,keys.go,tui_test.go},
specs/{tui.md,discovery.md,control.md}, README.md, todo/todo.md.

## Run and verify

`make check`; `make run`, look for `>` rows, `/detached`, and `p` on an
attached agent.

## Next step

Check the marks in a live tmux, then release v0.5.0 (specs/release.md).
