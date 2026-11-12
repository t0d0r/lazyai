# 2026-11-12 - persistent important marks and tmux session rename

## Done

- Important marks persist in `$XDG_STATE_HOME/lazyai/important`
  (default ~/.local/state/lazyai/important, 0600), one mark per line,
  oldest first, capped at 200. internal/tui/important.go: markID,
  loadMarks, saveMarks, toggleMark, moveMarks, RenameMarks,
  ImportantPath; stateDir is shared with HistoryPath.
- Mark identity, most stable first: `tmux:SESSION:KIND:ROLE`,
  `claude:SESSIONID`, `proc:KEY:START`. A marked agent restarted in its
  tmux session stays marked.
- The file is read at start and again before each toggle, so two lazyai
  instances do not drop each other's marks.
- Rename: tmux.CheckSessionName and tmux.RenameSession
  (`rename-session -t =OLD -- NEW`, refusing a taken name). TUI `R` and
  the x menu item open a popup prefilled with the session name; a done
  rename moves the session's marks. CLI `lazyai rename AGENT NAME` with
  help and examples; it moves the marks in the file too.
- The menu lists rename after kill, so the order of the earlier items is
  unchanged.
- Specs: tui.md (Important mark persistence, R key, Rename session),
  control.md (Rename), cli.md (rename row), README list entry.
- Tests: TestMarkID, TestMarksFile, TestImportantPersists,
  TestRenameSessionInput (internal/tui/important_test.go),
  TestCheckSessionName and a rename step in TestLaunchSendPanes against
  a real tmux, a rename usage error in TestUsageErrors.

- Window names: tmux showed lazyai-* windows as `0:2.1.322*`, because
  automatic-rename uses the pane's process name and Claude Code's is its
  version. Launch now passes `-n` with the command's base name and sets
  automatic-rename and allow-rename off in the same tmux call; attach
  (TUI `a`, CLI `attach`) applies the same to older lazyai-* windows
  whose automatic-rename is still on (tmux.NameWindow). AttachCmd takes
  the pane and window name. Tests: window state in
  TestLaunchSendPanes, TestNameWindow.

## Decisions

- Marks keyed by tmux session first: it is the name the work is known by
  and survives agent restarts; PIDs do not.
- Windows someone renamed (automatic-rename off) are left alone at
  attach.
- Rename takes the name as given, without adding `lazyai-`; only
  letters, digits, - and _ are allowed so the name works in `a`,
  `attach` and scripts.
- Correction: agent.Key is the PID (or container ID and PID), not PID
  and start time; the earlier spec text said otherwise and is replaced.

## Open

- A session renamed away from `lazyai-` no longer gets its status-line
  hint refreshed on attach (documented in control.md).
- Release v0.10.2 in progress (session/2026-11-12-release-v0.10.2.md).

## Files

internal/tui/{important.go,important_test.go,tui.go,keys.go,view.go,
input.go,tui_test.go}, internal/tmux/{tmux.go,tmux_test.go},
commands.go, examples.go, cli_test.go, specs/{tui.md,control.md,cli.md,
README.md}, todo/todo.md.

## Run and verify

`make check`; `make run`: `!` on a row, quit and start again, the row is
still marked; `R` on a tmux agent, type a new name, enter: the row shows
the new session and keeps its mark. `lazyai rename web api-review`.

## Next step

Release v0.10.2 or v0.11.0 when wanted (specs/release.md), or the next
open todo item.
