# 2026-11-12 status colors and important mark

## Done

- Status colors: statusText (internal/tui/view.go) renders `busy` green
  and `idle` cyan; other statuses stay plain. Used for the row's right
  side (full and compact rows) and the detail panel's `state` line.
  The row's label room is now measured with ansi.StringWidth, since the
  right side carries color codes.
- Important mark: `!` (importantKey, internal/tui/keys.go) toggles the
  selected agent's mark; model.important holds agent keys while lazyai
  runs. Marked rows end the label with ` !` (importantMark) and render
  the left part bold magenta (impStyle) unless flashing, selected or
  flagged. The x menu has `mark important` / `unmark important`; the `?`
  popup lists the key and the mark.
- Row style order: flash, selected, flagged, important, busy.
- Tests: TestImportantMark (toggle, row mark, survives selection move,
  menu), TestStatusColors (colors differ, widths unchanged, idle colored
  in the list).
- go.mod: termenv is a direct dependency now (the color test sets the
  ANSI profile); go mod tidy also listed three indirect modules that
  were missing.
- The launch flash (select, focus and yellow row for 3 s after `n`) is
  on main since 0d113be but not in v0.10.0; it ships with v0.10.1.

## Decisions

- Idle colors only the status word, not the row, so idle rows stay
  quiet; busy keeps its green label.
- Important marks are in memory only: the PID-and-start key does not
  survive an agent restart, so saving them needs a stable identity first.

## Observed

- spread() resets SGR after the left part, so row styles (selected
  reverse, flagged red, busy green) cover only the left part of a row;
  the right side (status, CPU, memory) is plain. Unchanged here; the
  status color relies on it.

## Open

- Keeping marks across restarts (todo).
- Not released; with the earlier commits it waits for v0.10.1.

## Files

internal/tui/{view.go,tui.go,keys.go,tui_test.go}, go.mod, go.sum,
specs/tui.md, todo/todo.md.

## Run and verify

`make check`; `make run`: idle rows show a cyan `idle`, busy rows a green
`busy`; `!` on a row adds ` !` and turns it magenta, `!` again clears it.

## Next step

Release v0.10.1 (specs/release.md), or the next open todo item.
