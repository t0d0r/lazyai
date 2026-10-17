# TUI

## Layout

- Header: agent count, busy count, helpers shown or hidden, last refresh.
- Table: PID, KIND, ROLE, STATUS, CPU%, MEM, UP, TMUX, FLAGS, WHERE.
  - STATUS shows a duration only when it is known: from the Claude session
    file, or after agentctl has seen the status change.
  - TMUX drops the `agentctl-` prefix.
  - WHERE is the Claude session name (if any) and the cwd with `~`.
  - Selected row reversed; flagged rows red; busy rows green.
- Optional detail pane: PID, parent, uid, tty, start, cwd, name, session id,
  tmux session and pane, status source, redacted command.
- Message line for results and errors, then the key help or the active
  prompt.

## Keys

j/k or arrows move, g/G first/last, enter or d details, a attach, p prompt,
n launch, s stop, K force kill, h helpers, r refresh, q or ctrl+c quit.
In prompts: enter submits, esc cancels, backspace, ctrl+u clears.

Launch prompt syntax: `KIND [DIR] [NAME] [-- ARGS...]`; DIR may use `~` and
is relative to the directory agentctl was started in.

## Refresh

Every 2 seconds (`-interval`), never more than one discovery at a time.
Selection follows the PID across refreshes.

## Watch flags

Computed by `internal/watch`, the same code the `list`, `show` and `watch`
commands use.

| flag      | condition                                            |
|-----------|------------------------------------------------------|
| self      | agentctl or a parent of it; cannot be stopped here   |
| high-cpu  | CPU >= 80% continuously for 30 seconds               |
| high-mem  | RSS >= 2 GiB                                         |
| long-busy | busy for 30 minutes or more                          |

Trackers are keyed by PID and reset when the process start time changes.
