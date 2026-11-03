# 2026-11-03 - focus keys

## Done

- Left / right arrows and h / l no longer move the focus between panels.
  The focus moves with tab / shift+tab, 1 2 3 (Agents, Services, Models)
  and enter / esc (main panel and back). The `?` popup, README and
  specs/tui.md (Keys) list only those.
- TestFocusAndTabs checks that right, left, l and h keep the focus.

## Decisions

- Only explicit keys move the focus, so a stray arrow key does not take
  it away from the list in use. h / l went with the arrows, since they
  were the same binding.

## Open

- Not released; the installed v0.7.0 still moves the focus with arrows.

## Files

internal/tui/{tui.go,keys.go,tui_test.go}, specs/tui.md, README.md,
todo/todo.md.

## Run and verify

`make check`; `make run`, press left / right and h / l: the focused
panel stays; tab and 1 2 3 move it.

## Next step

Release with the next batch of changes (specs/release.md).
