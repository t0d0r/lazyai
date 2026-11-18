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

- The `_` end mark is drawn only when the cursor is at the end of the
  text; it was also drawn with the cursor inside the text, which looked
  like two cursors.
- Live check in a 200x50 tmux session: `n`, a line typed, home, right,
  one character inserted, enter with `cat . "home work"`: the message
  line said `launched lazyai-home-work (a attaches)` and tmux listed the
  session.

## Release v0.11.1

- Checks before the tag: make check, make test-linux (arm64) and
  make test-linux LINUX_ARCH=amd64 pass on Ubuntu 24.04 and Debian 13.
- v0.11.1 tagged (annotated) on main; artifacts built with make dist and
  make deb VERSION=v0.11.1. README install examples point at v0.11.1.
- main and the tag pushed (gitleaks over the 2 unpushed commits: no
  leaks). GitHub release v0.11.1 published with the four tarballs, the
  amd64 and arm64 .deb packages and SHA256SUMS; the downloaded assets
  pass `shasum -a 256 -c SHA256SUMS`; the darwin-arm64 binary reports
  v0.11.1.
- Homebrew tap: Formula/lazyai.rb moved to the v0.11.1 source archive
  (sha256 381ca7f5b89dc6cc6945a6856506b9695bb292bb1e5585967da71b3a53edc269,
  the same on two downloads), committed and pushed. brew style and brew
  audit --strict clean; brew upgrade built 0.11.1 and brew test passes.
- Patch version: both changes are fixes to the dashboard input; no new
  CLI option.

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

## Files

internal/agent/launch.go, internal/agent/agent_test.go,
internal/tui/tui.go, internal/tui/input.go, internal/tui/view.go,
internal/tui/tui_test.go, specs/tui.md, specs/control.md, todo/todo.md.

## Run and verify

make check; brew upgrade lazyai && lazyai --version -> lazyai 0.11.1. make run, then `n`, type `claude . "home work"`, move with
left / right and edit, enter: the message line says
`launched lazyai-home-work (a attaches)`.

## Next step

Next open todo item.

## Security note

Quoting groups words only; the line is still never passed to a shell,
so `$VAR`, globs, `;` and backticks are not expanded or run. The command
is resolved on PATH and started directly by tmux, as before.
