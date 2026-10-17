# CLI

Every TUI action has a subcommand, so anything done by hand can be
scripted. Both use the same packages: `agent` for discovery and control,
`watch` for status durations, flags and change events, `display` for
formatting.

## Commands

| command                                  | TUI      | does                                   |
|------------------------------------------|----------|----------------------------------------|
| `tui [-interval D]` (default)            | -        | the dashboard                          |
| `list [-json] [-all]`                    | table, h | one snapshot; `-all` adds helpers      |
| `show [-json] AGENT`                     | enter, d | detail fields, resources, flags        |
| `watch [-json] [-all] [-interval D]`     | refresh  | stream of change events until ctrl+c   |
| `launch [-dir D] [-name N] KIND [ARGS]`  | n        | see control.md                         |
| `attach AGENT`                           | a        | see control.md                         |
| `send AGENT TEXT...`                     | p        | words are joined with single spaces    |
| `stop [-force] [-yes] AGENT...`          | s        | asks `[y/N]` per agent unless `-yes`   |
| `kill [-yes] AGENT...`                   | K        | `stop -force`                          |

## AGENT references

- A number is a PID and matches any agent, helpers included.
- Otherwise the reference matches interactive agents by tmux session name,
  by tmux session name without the `agentctl-` prefix, or by Claude Code
  session name. It must match exactly one agent; otherwise the command
  fails and asks for a PID.
- `stop` and `kill` resolve all references against one discovery.

## list

Columns: PID, KIND, ROLE, STATUS, CPU%, MEM, UP, TMUX, FLAGS, WHERE.
A single snapshot only knows durations from Claude session files, and
flags that need no history (`self`, `high-mem`, `long-busy` from a Claude
status time). `high-cpu` needs `watch` or the TUI.

## watch events

The first refresh prints `started` for every agent present. Afterwards:

| event   | from -> to                      |
|---------|---------------------------------|
| started | status                          |
| exited  | -                               |
| status  | old status -> new status        |
| flags   | old flags -> new flags          |

Exited events come first in each refresh, sorted by PID. A PID whose start
time changed is reported as exited and started. Text lines are
`HH:MM:SS  event  pid  kind  role  change  where`; `-json` prints one
object per line with `time`, `event`, `from`, `to` and the full `agent`
(redacted command, never the raw one).

## Exit status

0 on success, 1 on any error, including when some agents in a `stop` or
`kill` were not stopped.
