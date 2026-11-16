# 2026-11-16 - launch --attach

## Done

- `lazyai launch --attach KIND ...` launches as before and then attaches
  the terminal to the new session (inside tmux: switch-client), using the
  same code path as `attach` (attachSession in commands.go). No "launched"
  line is printed in that case. Long form only, since `-a` is `--all`
  elsewhere.
- Purpose: a wrapper script can run its agent under lazyai and land in it
  with one line, `exec lazyai launch --attach claude-bst -- "$@"`, without
  parsing the session name (which may get a -2, -3 suffix).
- Spec: specs/control.md (Launch), specs/cli.md (command table, options).
- Example added to `lazyai launch --help` / `lazyai examples launch`.
- Test: `--attach` parses before the launch line (cli_test.go).

- Fix: after `n` in the dashboard, the launched session was selected for
  one refresh only, then the cursor jumped back to the row selected
  before (specs/tui.md, Launch feedback, says it stays selected).
  selectPending set the list cursor but not its remembered row key, and
  list.set restores the cursor from that key on every refresh. It now
  calls list.sync(). TestLaunchFeedback starts with the cursor on another
  row and checks the selection after a second refresh; the old test
  passed only because the launched session was already row 1.

## Verified

- make check passes.
- End to end under a pseudo-terminal with a stub agent on PATH: a wrapper
  calling `launch --attach` created lazyai-<kind>-<dir>, the agent ran in
  tmux with its arguments, one tmux client was attached, and the session
  kept running after the client was killed.

## Open

- Not released yet; the Homebrew build (v0.10.2) has no `--attach`.
  Wrappers that must work with both can check
  `lazyai launch --help | grep -q -- --attach` and fall back to parsing
  the "launched NAME;" line.
- Attaching from a real terminal by hand is still the open "verify attach
  by hand" todo item.

- Dashboard, live: lazyai run in a 220x50 tmux session, `n` with
  `claude-bst . tuiprobe` from row 1: the new agent was selected (5 of 6,
  Info shows lazyai-tuiprobe) at about 3, 6 and 9 s after Enter. With the
  old build it stayed on row 1.

## Files

internal/tui/tui.go, internal/tui/tui_test.go, commands.go, examples.go, cli_test.go, specs/control.md, specs/cli.md,
todo/todo.md.

## Run and verify

make check; make build; bin/lazyai launch --attach claude

## Next step

Release with the next version (tag, tarballs, tap formula).
