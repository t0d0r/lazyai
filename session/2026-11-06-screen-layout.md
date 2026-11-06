# 2026-11-06 - screen layout

## Done

- Fourth layout on `v`: split -> detail -> lists -> screen -> split
  (specs/tui.md, Layouts, Screen).
  - Top row (two fifths of the body, at least 8 rows): the status panel
    and one list panel left (three fifths of the width, 32 to 90
    columns, full rows), with the lists as tabs in its title
    `1 Agents - 2 Services - 3 Models`; the main panel right with Info,
    Processes and Stats (] / [ skip Screen).
  - Screen panel below across the whole width, at least 5 rows. It is
    captured on every refresh whatever tab is shown, follows the bottom
    until scrolled, and names the tmux session in its title.
  - tab / shift+tab: list -> main panel -> Screen panel. j / k, g / G
    and page keys scroll the focused panel; esc returns to the list;
    mouse click focuses, the wheel scrolls the panel under it.
  - Short screens: the status panel is dropped before the list.
- Model: focusScreen, screenScroll / screenFollow, screenShown() for
  capture; layout gains mainH, screenY, screenH and tabbed.
- Help popup text shortened to `layout: split, detail, lists, screen`
  so `(now detail)` still fits 76 columns.
- Tests: TestLayoutScreen (sizes, titles, capture, tab skipping, focus
  cycle, list tabs, Screen scroll, mouse, leaving the layout);
  TestLayoutCycle and TestViewFitsScreen cover the fourth layout.
- Checked live at 120x30 in a temporary tmux session: the status
  border reads `screen layout`, the top row shows the tabbed list and
  Info - Processes - Stats, and a live Claude pane fills the bottom.

## Decisions

- Screen last in the cycle so the order of the existing layouts stays.
- The status panel stays on top left (not in the sketch) since every
  layout shows it; it gives way on short screens.
- One Screen scroll position separate from the main panel's, so
  scrolling Processes does not move the pane.

## Open

- Not released; the installed v0.9.0 has three layouts.
- Not tried by hand: resize while in the screen layout.

## Files

internal/tui/{tui.go,view.go,keys.go,tui_test.go}, specs/{tui.md,
README.md}, README.md, todo/todo.md.

## Run and verify

`make check`; `make run`, press `v` three times: the status border
reads screen layout and the selected agent's pane fills the bottom.

## Next step

Release the screen layout together with the Homebrew v0.9.0 follow-up
(specs/release.md).
