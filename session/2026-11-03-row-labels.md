# 2026-11-03 - tmux session names as row labels

## Done

- Row label: an agent in tmux is labelled by its full tmux session name
  (as `tmux ls` shows it); a Claude session name follows it, dimmed,
  only when it fits whole. Agents outside tmux keep the Claude name or
  the last two directories (specs/tui.md, Layout, Row).
- `lazyai list` TMUX_SESSION shows the full session name. It used to
  drop the `lazyai-` prefix, which disagreed with specs/cli.md; the spec
  wins. AGENT references still accept the name without the prefix.
- Tests: TestAttachMark (session first, Claude name after it, dropped
  in a narrow row), TestTmuxColumns expects the full name.
- Checked live: TMUX_SESSION now matches `tmux ls` for `lazyai-*`
  sessions.

## Decisions

- tmux session before the Claude name: it is what `tmux ls` shows and
  `a` attaches to, and sessions are named after the work
  (`claude-ticket-123`); generated Claude names (`hosts-various-a1`)
  carry little and looked like the name of another session.
- The Claude name is dropped whole rather than cut, so a half name is
  never read as part of the session name.

## Open

- Ideas in todo: count agents outside tmux in the status panel; a `~`
  short form for `lazyai-` sessions in narrow rows.
- Not released; the installed v0.6.0 still shows the Claude name first.

## Files

internal/tui/{keys.go,view.go,tui_test.go}, commands.go, cli_test.go,
specs/{tui.md,cli.md}, README.md, todo/todo.md.

## Run and verify

`make check`; `make run` and compare the Agents rows with `tmux ls`;
`bin/lazyai list` TMUX_SESSION column.

## Next step

Release the label change (specs/release.md), or pick one of the todo
ideas above.
