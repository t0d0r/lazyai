# 2026-11-12 - Screen colors

## Done

- The Screen tab and the screen layout's Screen panel show the pane in
  color: tmux.Capture takes a color flag (capture-pane -e) and the TUI
  passes true.
- screenText (internal/tui/view.go) keeps only SGR sequences; other
  escape sequences (CSI, OSC including clipboard and title, DCS) and
  control characters are dropped, tabs become four spaces. The SGR state
  at a line's end is repeated at the next line's start (reset by ESC[0m,
  ESC[m, restarted by ESC[0;...m, at most 16 carried).
- Tests: TestScreenText (kept colors, carry, reset, dropped sequences,
  same text as without colors).
- Checked live at 120x30 in the screen layout with a Claude Code pane:
  20 distinct SGR sequences shown, every Screen row ends at the border.
- todo: the Screen colors item, the layout release (v0.10.0) and the
  saved layout item (S) are marked done.

## Decisions

- Colors always on, no switch, until a pane clashes with a theme.
- Filter to SGR in lazyai rather than trust capture-pane -e: tmux emits
  only SGR today, but the filter keeps a pane from reaching the
  terminal if that changes.

## Open

- Not released; five earlier commits (update check, Screen title,
  launch feedback) also wait for v0.10.1.

## Files

internal/tmux/{tmux.go,tmux_test.go}, internal/tui/{tui.go,view.go,
tui_test.go}, specs/tui.md, todo/todo.md.

## Run and verify

`make check`; `make run`, `v` to the screen layout: the pane shows in
its colors.

## Next step

Next open todo item, or release v0.10.1 (specs/release.md).
