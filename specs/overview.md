# Overview

## Goal

One tool to see and control every AI coding agent running locally for the
current user on this machine, on macOS or Linux.

## Scope

- Agents: Claude Code, Codex, opencode, Gemini CLI, Qwen Code, Grok CLI,
  GitHub Copilot CLI, aider, cursor-agent, Hermes Agent; also when run with
  `docker run` / `podman run`.
- Local model runtimes: ollama, mlx-lm, Colibri. Their engine processes are shown,
  because agents drive them and they hold most of the CPU and memory.
- Actions: list and inspect, watch status, stop or kill, launch in tmux,
  attach, send a prompt.
- Interfaces: an interactive TUI (default) and CLI subcommands over the same
  core, so actions can be scripted and tested.

## Non-goals

- Desktop apps (Claude.app, Cursor.app). They are excluded from discovery.
- Other users' processes and remote machines. Agents in docker containers
  are listed and watched (see discovery.md, Containers), but lazyai does
  not start, stop or enter containers itself.
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

macOS and Linux. Both use `ps` (procps on Linux) and tmux; the
differences are listed in discovery.md (Linux). Tested on Ubuntu 24.04 and
the latest Debian, arm64 and amd64, in docker with `make test-linux`: the
unit tests plus an
end-to-end run of launch, list, show, send, the dashboard, stop and kill
against fake agents.

Decision: on Linux, read `/proc` only where `ps` falls short (recent CPU and
working directory) instead of replacing `ps`. Reason: one
parser and one start-time format on both systems, and Claude Code itself
uses `ps -o lstart` for `procStart` on every Unix.
