# TUI

The dashboard follows lazydocker: bordered panels, lists on the left, a
tabbed main panel on the right about the selected item, popups for menus,
confirmations and input, key hints in the bottom line, mouse support.

## Layout

```
+ lazyai ------------------+ + Info - Screen - Processes - Stats ------+
| 9 run, 3 busy, 1 att 3.3G| |                                          |
+--------------------------+ |  details of the selected agent           |
+ 1 Agents ----------------+ |                                          |
|>claude ai-d1 busy 2% 1G  | |                                          |
| hermes h-ai idle 1% 90M  | |                                          |
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
- Status panel (3 rows): number of agents shown, how many are busy, how
  many have a tmux client attached (only when some shown agent runs in
  tmux: `9 running, 3 busy, 2 attached`), and on the right the total
  memory of the shown agents next to the time of
  the last refresh (`mem 3.3G 01:59:11`).
  - The total adds each shown agent's tree memory (the agent and all its
    descendants, as in the rows). An agent that runs below another shown
    agent is already in that tree and is not added again. Container
    agents count by container and PID, so their PIDs do not collide with
    the host's.
  - Narrow column: the forms are tried in this order until one fits:
    `9 running, 3 busy, 2 attached` with `mem 3.3G 01:59:11`,
    `9 run, 3 busy, 2 att` with `mem ...`, the long counts without `mem `,
    `9 run, 3 busy, 2 att` without `mem `, and last `9 run, 3 busy`.
    Decision: the short words go before `mem `, so the attached count
    does not push the memory total out at common widths.
  - Decision: memory of the agents, not of the whole machine. Reason:
    lazyai is about what the agents cost; the system total is in other
    tools.
- Lists by role:
  - 1 Agents: interactive
  - 2 Services: server, daemon, and helper when shown
  - 3 Models: engine, job
- Row: a one-character gutter right before the kind: `>` (`>claude`) when
  the agent runs in a tmux pane and a client is attached to its session
  now, `@` (`@claude`) when it runs in tmux but is detached, blank
  otherwise. `a` attach and `p` prompt work on both. Decision: a
  different character, not only a color, so the state reads without
  colors and in the reversed selected row. Then the kind, role (not
  in Agents), label (the full tmux session name, else the Claude session
  name, else the last two directories of the working directory, home
  shown as `~`: `alpha/web-frontend`), `*` for lazyai's own parent;
  right-aligned status, tree CPU% and tree memory in every list. The `?`
  popup explains `>`, `@` and `*` under Marks.
  - An agent in tmux that also has a Claude session name shows it after
    the session, dimmed (`lazyai-sos-tasks hosts-various-a1`), only when
    it fits whole; otherwise it is left out. It is not dimmed in a row
    that has a style of its own (selected, flagged, busy).
  - Decision: tmux session first. Reason: it is the name `tmux ls` shows
    and `a` attaches to, and people name sessions after the work
    (`claude-ticket-123`); Claude's generated names (`hosts-various-a1`)
    say little and could be taken for another session. When the row is too narrow the
  label is cut first: a directory label keeps its parent and replaces
  the start of the last directory with `..` (`alpha/..nd`), or, with
  less room, the start of the whole label (`..ontend`); other labels
  are cut at the end. Selected row reversed in the focused list, bold
  elsewhere; flagged rows red, busy rows green. The bottom border shows
  "N of M".
  - Decision: two directories, not one. Reason: the same project
    directory name under different parents (`alpha/web-frontend`,
    `beta/web-frontend`) must stay apart.
- Heights: the three lists split the column below the status panel evenly,
  as in lazydocker, whatever their row counts; leftover rows go to the top
  lists. Longer lists scroll. When a third of the column would be under 5
  rows, the list in use takes the room and the others shrink to one row.
- An empty list says "none"; with it focused the main panel says "no
  <list> running".
- Main panel: the rest of the width, full height. Tabs are in the top
  border, the active one green. A scroll position shows in the bottom
  border when the content is longer than the panel.
- The focused panel has a green border, the others grey.
- Footer: key hints for the focused side; on the right the last message
  (red for errors) or `lazyai VERSION`.

## Tabs

- Info: state with duration, the detail fields (display.Details; the
  tmux line ends with `attached (2 clients)`, `detached, last attached
  2026-11-03 14:02` or `detached, never attached`),
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
| p                    | send a prompt (tmux only); asks first when a client is attached (control.md, Send) |
| n                    | launch: `KIND [DIR] [NAME] [-- ARGS...]`; up / down history, tab completes |
| /                    | filter the list in use; enter keeps, esc clears |
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
started in. Input popups treat both terminal backspace encodings (DEL and
ctrl+h) as backspace.

Launch history: every launch line submitted is added to a history, with
runs of spaces folded to one. A line that does not parse (unknown kind,
too many words) or whose launch fails is kept too, so the next popup
brings it back with up to be fixed; the error in the message line says
so. Up shows the previous (older) line, down the
next; past the newest line down brings back the text that was being
typed. A line used again moves to the end instead of being kept twice; the
oldest lines beyond 200 are dropped. The history is kept in
`$XDG_STATE_HOME/lazyai/launch_history` (default
`~/.local/state/lazyai/launch_history`), one line per launch, so it
survives restarts. The directory is created mode 0700 and the file is
written mode 0600 through a rename, because launch lines can carry agent
arguments. Before adding a line the file is read again, so several lazyai
instances do not drop each other's lines. A missing or unreadable file is
an empty history; a failed write shows in the message line and does not
stop the launch. The prompt popup (`p`) keeps no history: prompts are
free text sent to an agent and are not written to disk.

Launch completion: tab completes the word being typed. The first word
completes to a launch kind, or, once at least one letter is typed, to an
executable on PATH (`claude-` -> `claude-bst`), and gets a space when only
one matches.
The second word completes to a directory, resolved the same way as DIR
(`~`, relative to the start directory, symlinks followed); a single match
gets a trailing `/` so the next tab goes one level down, and hidden
directories are offered only when the word starts with `.`. With several
matches the word grows to their common prefix and the matches show under
the input until the next key. NAME and anything after `--` are not
completed.

## Filter

`/` filters the list in use (lazydocker style). The focus moves to that
list and the keys go into the filter: the rows narrow as it is typed,
up / down move the selection, backspace and ctrl+u edit. Enter keeps the
filter and returns to the normal keys; esc clears it. Esc in a list with
a kept filter clears it too.

- Match: every word of the filter appears, ignoring case and order, in
  the row's kind, role, label, working directory, Claude session name,
  tmux session, container, status, PID, or tmux state: `attached` or
  `detached` for agents in tmux (`/detached` lists the sessions nobody
  is looking at).
- Each list has its own filter; it holds across refreshes. The title
  shows it (`1 Agents /claude`), an empty result says "no match", and
  the footer shows the filter being typed.
- The status panel counts are not filtered: they describe all agents.

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
