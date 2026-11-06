# 2026-11-06 - launch feedback

## Done

- tmux.Launch creates the session with remain-on-exit on (same tmux
  call, targeted at the new session) and watches the pane for 1.5 s
  (tmux.LaunchWatch). A dead pane becomes tmux.ExitError with the
  command name, exit status and last 20 output lines; the session is
  killed. A running pane gets remain-on-exit off.
- TUI: an ExitError opens a `Launch failed` popup with the output; the
  message line has the one-line form. A successful launch sets a
  pending session; when discovery lists an agent in it (within 15 s)
  it is selected, its list focused, and its row yellow for 3 s
  (flashStyle / flashSelStyle).
- actionMsg gained a launched field; its literals use field names.
- Tests: TestLaunchEarlyExit and TestDeadOutput (real tmux on the
  default server), TestLaunchFeedback (popup, select, flash, give up).
- Checked live inside tmux: `ls . lzfail -- /nonexistent-dir` shows the
  popup with ls's error and leaves no session; a stand-in claude
  process launched from Services was selected in Agents with a yellow
  background.

## Decisions

- The untargeted set-option first tried changed lazyai's own window
  when lazyai runs inside tmux; the target `=SESSION:` fixes it. Not
  covered by a unit test (needs a tmux client), checked by hand.
- Launches cost 1.5 s more before the "launched" message; acceptable
  for catching the error.

## Open

- Not released. A command failing after 1.5 s is still silent.

## Files

internal/tmux/{tmux.go,tmux_test.go}, internal/tui/{tui.go,view.go,
tui_test.go}, specs/{control.md,tui.md}, todo/todo.md.

## Run and verify

`make check`. By hand: `make run`, n, `ls . x -- /nonexistent`, Enter:
the popup shows ls's error.

## Next step

Release v0.10.1 with the Homebrew update check, the Screen title and
launch feedback (specs/release.md).
