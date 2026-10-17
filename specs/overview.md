# Overview

## Goal

One tool to see and control every AI coding agent running locally for the
current user on this Mac.

## Scope

- Agents: Claude Code, Codex, opencode, Gemini CLI, aider, cursor-agent,
  ollama.
- Actions: list and inspect, watch status, stop or kill, launch in tmux,
  attach, send a prompt.
- Interfaces: an interactive TUI (default) and CLI subcommands over the same
  core, so actions can be scripted and tested.

## Non-goals

- Desktop apps (Claude.app, Cursor.app). They are excluded from discovery.
- Other users' processes, remote machines, and containers.
- Reading or changing agent conversations, settings or credentials.

## Decisions

- Go with bubbletea and lipgloss. Reason: single binary, TUI requested; the
  libraries were already in the local module cache, so no network is needed.
- tmux is the control channel for launch, attach and send. Reason: it gives
  a real terminal per agent that can be attached to and typed into, without
  hooking into each agent's private protocol.
- Discovery is read-only and uses `ps` and `lsof`. Reason: works for every
  agent without integration; agent-specific metadata (Claude session files)
  only enriches it.
- No daemon or server. Reason: nothing to secure on a socket or port; the
  TUI polls every 2 seconds.

## Platform

macOS only for now: `ps` lstart format, `lsof` for working directories,
`ttysNNN` terminal names.
