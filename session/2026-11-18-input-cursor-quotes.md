# 2026-11-18 - input cursor and launch line quoting

## Done

- Input popups (launch `n`, prompt `p`, rename `R`) have a cursor. Left /
  right (ctrl+b / ctrl+f), home / end (ctrl+a / ctrl+e), delete (ctrl+d)
  and ctrl+k work; typed text and backspace act at the cursor. Before,
  text could only be added and removed at the end and the arrow keys
  left / right did nothing, so a recalled line could not be fixed in the
  middle. The popup draws `_` at the end or the reversed character under
  the cursor and scrolls a long text to keep the cursor in view
  (inputLine in internal/tui/view.go).
- Tab in the launch popup completes the word that ends at the cursor and
  keeps the text after it.
- The launch line typed in `n` is split with shell quoting
  (agent.SplitLine) instead of strings.Fields. Symptom fixed:
  `claude-bst . "home work"` was four words and failed with
  `usage: KIND [DIR] [NAME] [-- ARGS...]`, so the command was never
  started; it now launches claude-bst with the name `home work` (session
  `lazyai-home-work`). An unclosed quote is a usage error naming the
  quote.
- Launch history folds runs of spaces only outside quotes and not after
  a backslash (foldSpaces in internal/tui/input.go).

## Decisions

- Shell quoting in the dashboard line, nothing expanded. Reason: the
  line must work unchanged on the command line, where the shell does the
  same grouping.
- The cursor is one field on the model (cur, in runes) shared by the
  three input popups; the list filter (`/`) still edits at the end only.

## Open

- Tab completion does not understand quotes: a directory with spaces is
  not completed.
- The filter input (`/`) has no cursor.
- Not released; the installed v0.11.0 does not have the fix.

## Files

internal/agent/launch.go, internal/agent/agent_test.go,
internal/tui/tui.go, internal/tui/input.go, internal/tui/view.go,
internal/tui/tui_test.go, specs/tui.md, specs/control.md, todo/todo.md.

## Run and verify

make check. make run, then `n`, type `claude . "home work"`, move with
left / right and edit, enter: the message line says
`launched lazyai-home-work (a attaches)`.

## Next step

Release the fix as a patch version (specs/release.md), or the next open
todo item.

## Security note

Quoting groups words only; the line is still never passed to a shell,
so `$VAR`, globs, `;` and backticks are not expanded or run. The command
is resolved on PATH and started directly by tmux, as before.
