# 2026-11-03 - parent chain in the Processes tab

## Done

- Discovery records each agent's parents (agent.Agent.Parents, JSON
  `parents`): topmost process found first, direct parent last, commands
  redacted. The walk stops at PID 0, a PPID missing from the snapshot,
  or a loop (specs/discovery.md, Process tree).
- Processes tab: parents dimmed above the agent, one indent step per
  level; the agent in bold; descendants indented below it. The totals
  still cover the agent and its descendants and say how many parents
  above are not counted (specs/tui.md, Tabs).
- Checked live: a Claude agent in tmux shows `launchd -> tmux -> claude`;
  every host agent got parents, the deepest chain has 5 levels.
- Tests: TestParents (order, redaction, missing parent, loop),
  TestBuildClaudeSessionAndTmux (parents not in totals),
  TestProcessesParents (indent, order, totals line).

## Decisions

- Parents in the same tab, not a new one: one tree read top to bottom
  answers who started the agent and what it runs.
- Parents stay out of the tree totals: the totals are what the agent
  costs.
- No user column: lazyai lists the user's own agents, and a uid per
  parent would add width for little gain.

## Open

- Not released, together with the focus key change.
- Container agents with docker PIDs have no parents.

## Files

internal/agent/{discover.go,agent_test.go}, internal/tui/{view.go,
tui_test.go}, specs/{tui.md,discovery.md}, todo/todo.md.

## Run and verify

`make check`; `make run`, select an agent, `]` to Processes;
`bin/lazyai list --json` has `parents`.

## Next step

Release the focus key change and the parent chain (specs/release.md).
