# 2026-11-02 - launch from PATH, failed lines in history, list filter

## Done

- Launch history keeps every submitted line, also one that does not
  parse; the error in the message line says up in `n` brings it back.
  Lines whose launch fails later were already kept (specs/tui.md).
- KIND may be any bare command name on PATH, for example a wrapper
  script `claude-bst` that runs claude with another profile. Names with
  `/` or a leading `-` are refused. Applies to the TUI and `lazyai
  launch` alike, through agent.LaunchCommand / agent.IsCommand
  (specs/control.md).
- Launch completion: the first word also completes to executables on
  PATH once a letter is typed (`claude-` -> `claude-bst`).
- `/` filters the list in use: live while typing, enter keeps, esc
  clears; words match kind, role, label, directory, sessions, container,
  status or PID, case-insensitive, in any order. The title shows
  `1 Agents /bst`, an empty result says "no match". Help popup, footer
  and README list the key (specs/tui.md, Filter).
- Checked in tmux at 120x30 against the live agents: filter narrows the
  Agents list, completion offers PATH wrappers.
- Tests: TestLaunchCommand (PATH command accepted, paths and unknown
  names refused), TestLaunchHistory (bad line kept and recalled),
  TestCompleteLaunch (PATH commands, non-executables left out),
  TestFilter, TestParseLaunch.

## Decisions

- Any command on PATH instead of a fixed list: agents are often started
  through wrapper scripts; the line is typed by the local user with the
  rights of a shell, so the kind list gave no protection. Bare names
  only, so the command always comes from PATH.
- Filter per list, not global; status panel counts stay unfiltered.

## Open

- The launch popup hint line (kinds list) is cut at 80 columns.
- Not released yet (todo: release v0.5.0).

## Files

internal/agent/{launch.go,agent_test.go},
internal/tui/{tui.go,input.go,view.go,keys.go,tui_test.go},
specs/{control.md,tui.md,cli.md}, README.md, todo/todo.md.

## Run and verify

`make check`; `make run`, then `/` in a list and `n` with `claude-` tab.

## Next step

Release v0.5.0 (specs/release.md).
