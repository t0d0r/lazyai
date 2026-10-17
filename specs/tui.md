# TUI

The dashboard follows lazydocker: bordered panels, lists on the left, a
tabbed main panel on the right about the selected item, popups for menus,
confirmations and input, key hints in the bottom line, mouse support.

## Layout

```
+ lazyai ------------------+ + Info - Screen - Processes - Stats ------+
| 20 running, 3 busy 01:36 | |                                          |
+--------------------------+ |  details of the selected agent           |
+ 1 Agents ----------------+ |                                          |
| claude ai-d1  busy  7.9% | |                                          |
| hermes hermes-ai idle 1% | |                                          |
+------------------ 1 of 2 + |                                          |
+ 2 Services --------------+ |                                          |
| codex server ...         | |                                          |
+------------------ 1 of 5 + |                                          |
+ 3 Models ----------------+ |                                          |
| colibri engine ... 1.1G  | |                                          |
+------------------ 1 of 1 + +------------------------------------------+
x menu  a attach  p prompt  s stop ...                      lazyai VERSION
```

- Left column: one third of the width, at least 32 columns (or half of a
  narrow screen), at most 56.
- Status panel (3 rows): number of agents shown, how many are busy, time
  of the last refresh.
- Lists by role:
  - 1 Agents: interactive
  - 2 Services: server, daemon, and helper when shown
  - 3 Models: engine, job
- Row: kind, role (not in Agents), label (Claude session name, else tmux
  session, else working directory basename), `*` for lazyai's own parent;
  right-aligned status and tree CPU%, plus tree memory in Models. Selected
  row reversed in the focused list, bold elsewhere; flagged rows red, busy
  rows green. The bottom border shows "N of M".
- Heights: each list gets its rows plus borders; spare rows go to Agents.
  When crowded, the list in use takes the room and the others shrink to
  at most a fifth of the column.
- Main panel: the rest of the width, full height. Tabs are in the top
  border, the active one green. A scroll position shows in the bottom
  border when the content is longer than the panel.
- The focused panel has a green border, the others grey.
- Footer: key hints for the focused side; on the right the last message
  (red for errors) or `lazyai VERSION`.

## Tabs

- Info: state with duration, the detail fields (display.Details),
  resources, process tree totals, flags.
- Screen: the selected agent's tmux pane, captured with
  `tmux capture-pane -p -S -1000` on every refresh while the tab is shown;
  it follows the bottom like a log until scrolled. Agents outside tmux get
  a hint instead.
- Processes: the agent and every descendant process in tree order with
  PID, PPID, CPU%, memory and redacted command, then the totals.
- Stats: ASCII column charts of CPU and memory for the agent and its
  children over the kept history (watch.HistoryLen samples, 5 minutes at
  2 seconds), with now, average and maximum. The CPU scale rounds the peak
  up to 1, 2, 2.5 or 5 times a power of ten; memory leaves 20% headroom.

## Keys

| key                  | action                                         |
|----------------------|------------------------------------------------|
| tab / shift+tab      | next / previous panel (lists, then main)       |
| l / h, right / left  | same                                           |
| 1 2 3                | Agents, Services, Models                       |
| enter / esc          | focus the main panel / back to the list        |
| j / k, down / up     | move the selection, or scroll the main panel   |
| g / G, home / end    | first / last; main panel top / bottom (follow) |
| pgdn / pgup, ctrl+d / ctrl+u | scroll the main panel a page           |
| ] / [                | next / previous tab                            |
| x                    | action menu for the selected agent             |
| a                    | attach (tmux only)                             |
| p                    | send a prompt (tmux only)                      |
| n                    | launch: `KIND [DIR] [NAME] [-- ARGS...]`       |
| s / K                | stop (SIGTERM) / kill (SIGKILL), confirm with y |
| H                    | show / hide helper processes                   |
| r                    | refresh now                                    |
| ?                    | keybindings popup                              |
| q, ctrl+c            | quit                                           |

Mouse: click a list row to select it, click the main panel to focus it;
the wheel moves the selection in a list or scrolls the main panel.

The menu (`x`) lists the actions with their keys; actions that cannot
apply are dimmed with a short reason (not in tmux, lazyai's parent) and
refused with the full reason. Enter or the item's key runs it.

Launch input: DIR may use `~` and is relative to the directory lazyai was
started in.

## Refresh

Every 2 seconds (`--interval`), never more than one discovery and one
screen capture at a time. Selection follows the PID across refreshes;
changing the selection resets the main panel scroll.

## Watch flags

Computed by `internal/watch`, the same code the `list`, `show` and `watch`
commands use.

| flag      | condition                                            |
|-----------|------------------------------------------------------|
| self      | lazyai or a parent of it; cannot be stopped here     |
| high-cpu  | CPU >= 80% continuously for 30 seconds               |
| high-mem  | RSS >= 2 GiB                                         |
| long-busy | busy for 30 minutes or more                          |

Trackers are keyed by PID and reset when the process start time changes.
