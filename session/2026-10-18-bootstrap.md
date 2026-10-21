# 2026-10-18 bootstrap

## Done

- Created the project: Go TUI plus CLI to list, watch, launch, attach to,
  send prompts to, and stop local AI agents.
- Operator choices: all four capability groups, TUI, Go; agents Claude
  Code, Codex and the others (opencode, gemini, aider, cursor-agent,
  ollama); Claude Desktop excluded.
- Found that Claude Code writes `~/.claude/sessions/<pid>.json` with session
  id, cwd, name and busy/idle status; `procStart` there is UTC.
- Verified on the live system: discovery of claude, codex and opencode
  processes; launch of a Claude session in tmux; TUI render and detail
  view in a 160x34 tmux window; stop through the TUI confirm flow.
- Added a Makefile (help, build, test, lint, fmt, check, run, install,
  uninstall, tidy, clean). The parent ../AGENTS.md now requires a Makefile
  with standard targets in every project.
- CLI now covers every TUI action: `show`, `watch` (change events, text or
  JSON lines), `kill`, FLAGS column in `list`, and `stop`/`kill` by PID or
  name. Tracking and flags moved from internal/tui to internal/watch so
  both interfaces share them. Verified live: watch reported started,
  busy -> idle and exited for a test Claude session launched and killed by
  name.
- Fixed `lazyai -h` (it went to the tui flag parser and never printed
  the usage). The CLI is now Unix-style: commands live in a table in
  commands.go that drives parsing and generated help; flags.go wraps
  flag.FlagSet with long/short options and argument permutation;
  `--version` is stamped by `make build`; exit status 2 for usage errors.
  `stop` now reads confirmations through one buffered reader, so piped
  answers for several agents are no longer lost.
- Operator reported Colibri running Qwen (driven by Hermes Agent) was not
  shown. Causes: the `.app/Contents/` desktop-app exclusion also matched
  Homebrew's framework Python, hiding Hermes; and Hermes, Colibri and
  Qwen Code were not known kinds. Fixed the exclusion, added the three
  kinds and roles `job` and `engine` (ollama runners moved from helper to
  engine). Verified live: hermes interactive, `coli tune` job, `qwen36`
  engine flagged high-mem.

## Decisions

- tmux is the only control channel for attach and send. Agents started in
  plain terminal tabs can be listed and stopped only.
- Status for non-Claude agents is a CPU heuristic (>= 2%).
- Durations for CPU-based status appear only after a change is observed.
- Helper processes (Claude pty hosts and spares, ollama runners) are hidden
  by default.

## Open

- Attach was not exercised with a real terminal client (no client in the
  agent's shell). Send was tested against `cat`, not a real agent.
- gemini, aider, cursor-agent and ollama were not running; their
  classification is tested with sample command lines only.

- Renamed the project from agentctl to lazyai (directory ../lazyai, Go
  module, binary, tmux prefix `lazyai-`). Older lines in this note were
  rewritten with the new name.
- Rebuilt the TUI after lazydocker: Agents / Services / Models list panels
  plus a status panel on the left; Info, Screen (live tmux pane), Processes
  (agent's process tree) and Stats (ASCII CPU and memory charts) tabs on
  the right; x action menu, ? keybindings, confirm and input popups
  overlaid in the middle; mouse select and scroll. Discovery now records
  each agent's descendants and tree totals; the watcher keeps 150 samples
  of history. Verified in a 160x40 tmux window against the live system
  and a throwaway Claude session (Screen showed its trust prompt).
- Operator screenshot in iTerm: Agents took all spare rows and Services /
  Models were squeezed. Lists now split the column evenly; only very
  short terminals give the list in use the room.
- Operator asked how to leave an attached session; lazyai sessions now
  show `ctrl+b d: detach, back to lazyai` (or `ctrl+b L` when lazyai runs
  inside tmux) in their own tmux status-right. Verified by attaching in a
  pseudo terminal. tmux option commands need `=name:` as the target.
- Published at operator request: branch renamed master -> main, remote
  origin git@github.com:t0d0r/lazyai.git, pushed. Before pushing, the
  whole history was scanned for client names, home paths, session ids,
  emails other than the commit author, and token patterns; nothing found.
- Added a README screenshot (lazyai.png) and pushed it.
- License: operator chose BSD-2-Clause (FreeBSD), copyright Todor
  Dragnev.
- Released v0.1.0 (annotated tag on 1adce7f; source tarball sha256
  866462d199845ba14d215beba2abe9585e2e95b64416123f0bb1007096c09c14).
- Homebrew: tap created with `brew tap-new t0d0r/tap` (local checkout at
  $(brew --repository)/Library/Taps/t0d0r/homebrew-tap, outside ~/work/AI),
  Formula/lazyai.rb added, `brew style`, `brew audit --strict --online`,
  `brew install --build-from-source` and `brew test` all pass, pushed to
  git@github.com:t0d0r/homebrew-tap.git. lazyai 0.1.0 is now installed at
  /opt/homebrew/bin/lazyai. The tap's CI builds formulae only on pull
  requests; its daily autobump workflow opens a PR for new tags.
- README: animated GIF of the dashboard kept on local branch
  docs/readme-gif (not pushed, at operator request); "Why?" section added
  on main from the operator's text, translated to English.
- Operator noticed grok was missing. Survey of the machine: grok CLI
  (2 running, one for 11 days), Copilot CLI and mlx-lm installed, opencode
  run through a docker alias; Grok Bot.app, ChatGPT.app and copilot.vim
  correctly not agents. Added kinds grok, copilot, mlx-lm, docker/podman
  run of known images, and a desktop-app check that handles bundle names
  with spaces. Verified live: both grok sessions listed.
- Released v0.1.1 (annotated tag on 13a8d86; tarball sha256
  108c32db288a2ca7f519b2d3fb17e2a82185cfff4c4804884a4b81fdd629b686).
  Tap formula updated (4dfda82), style and audit clean, upgraded locally
  from source, brew test passes, installed lazyai reports 0.1.1 and
  lists both grok sessions.
- Pushed branch docs/readme-gif (d688199) to origin at operator request,
  after checking its two frames show nothing beyond the public
  screenshot. It branches from d4914fc, two commits behind main, and
  changes only the README image line plus lazyai.gif, so it merges
  cleanly. Not merged into main.

## Files

main.go, commands.go, flags.go, main_test.go, cli_test.go,
internal/agent/*, internal/tmux/*, internal/tui/{tui,view,keys,chart}.go,
internal/watch/*, internal/display/display.go, Makefile, README.md,
AGENTS.md, specs/*, todo/todo.md.

## Run and verify

```sh
make check
make run
```

`make install` was not run; it copies the binary to ~/.local/bin.

The tmux test creates and removes a session named `lazyai-test-*` on the
default tmux server.

## Next step

Run `bin/lazyai` in a normal terminal, launch a Claude session with `n`,
watch it in the Screen tab (`]`), attach with `a`, detach with ctrl+b d,
send a prompt with `p`, and try the mouse.

## Linux support (later the same day)

Operator asked to make lazyai work on Linux and test it in docker on
Ubuntu 24.04 and the latest Debian.

Done:

- `ps` works as is with procps (same options, same lstart format). No
  terminal is `?` on Linux, now handled.
- tmux pane ttys are `/dev/pts/N`; they were cut to `N` with
  filepath.Base, so no agent matched its pane. Now only `/dev/` is
  stripped, which gives `pts/N` (Linux) and `ttys003` (macOS), as ps.
- tmux 3.4 without a UTF-8 locale prints tabs in `-F` output as `_`, which
  broke Panes in containers; `tmux -u list-panes` fixes it.
- Single-word launches went through tmux's `sh -c`. Linux dash does not
  exec, so every launched agent was listed twice. Launch now runs
  `env PATH` (two words, exec'd directly); classification also ignores
  shells running a command string (`sh -c`, `bash -lc`, `python -c`).
- CPU on Linux: ps gives a lifetime average, so status would stay idle.
  Snapshot now replaces it with usage since the previous snapshot from
  /proc/<pid>/stat (first snapshot samples 250 ms).
- Working directory from /proc/<pid>/cwd; lsof only as a fallback and only
  if installed.
- Processes inside containers (deeper PID namespace by NSpid) are dropped
  from the agent list on Linux; the docker client stands for them.
  Checked by hand with `unshare -p` in a privileged container.
- `make test-linux`: cross-compiles the test binaries and bin/linux/lazyai
  for the docker server's arch, builds test/linux/Dockerfile per image in
  LINUX_IMAGES (default ubuntu:24.04 debian:latest) and runs
  test/linux/e2e.sh: unit tests, then fake claude (echo loop) and codex
  (busy loop) launched in tmux, list/json checks (tmux session, cwd, pts
  tty, idle/busy, CPU > 50), show, send, the dashboard drawn in a tmux
  pane and quit with q, stop and kill. Passes on Ubuntu 24.04.5 (tmux 3.4)
  and Debian 13 trixie (tmux 3.5a), arm64.
- Specs: overview (goal, Platform, decision to keep ps and add /proc),
  discovery (shell -c rule, new Linux section), control (env for single
  word launch). README requirements and test-linux.
- macOS checked again: make check passes, live `bin/lazyai list` as before.

Decided: keep ps on Linux and read /proc only for CPU, cwd and PID
namespace. Claude Code writes procStart with `LC_ALL=C TZ=UTC ps -o
lstart=` on every Unix (seen in its binary), so session files match on
Linux without changes.

Open: not tried with real agents on a Linux desktop; amd64 not tested;
Homebrew on Linux not tried.

Files: internal/agent/{proc,procfs,procfs_test,cwd,discover,classify,
agent_test}.go, internal/tmux/{tmux,tmux_test}.go, Makefile,
test/linux/{Dockerfile,e2e.sh}, README.md, specs/{overview,discovery,
control}.md, todo/todo.md.

Run and verify: `make check` (macOS), `make test-linux` (docker).

Next step: run lazyai on a real Linux machine with a logged-in Claude Code
session and confirm status comes from the session file.

## Agents in docker containers (macOS)

Operator noticed that claude inside docker was not detected. They run it
on macOS as `docker run IMAGE claude`, with `docker exec`/compose, and by
typing claude into a shell in a container or devcontainer. Only
`docker run` of an image named after an agent was recognized.

Done:

- internal/agent/client.go: docker/podman/docker-compose command lines
  (run, exec, compose run/exec, sandbox run, `docker container ...`,
  global options such as --context). Kind from the command in the
  container (looking through `bash -lc ...`), else the image or service.
- internal/agent/container.go: `docker ps` plus parallel `docker top`
  per container; agents found there are folded into their client row
  (CPU, memory and status from the agent, PID and pane from the client)
  or get their own row with ForeignPID and key SHORTID/PID. A shell
  client in tmux hands its pane to the one interactive agent in its
  container, so attach and send work. Stop refuses ForeignPID agents.
  CPU over a 30 s window, since docker top gives whole CPU seconds.
- Agent.Key() replaces the PID as the identity in watch, TUI selection
  and resolve; AGENT accepts a container name or SHORTID/PID.
- Linux: removed last turn's NSpid filter. Host rows are matched to
  docker's report by PID and command line instead, so nothing docker
  cannot see is hidden.
- Verified live on this Mac with a throwaway image of fake agents:
  docker run IMAGE claude, docker exec -it CTR codex, claude typed into
  `docker exec -it CTR bash`, docker compose run app claude. All listed
  with `docker:NAME`; send into the typed claude arrived; stop of it was
  refused; stop of the docker run row ended its container. A Docker
  query adds about 10-30 ms per refresh. Test containers, sessions and
  images were removed afterwards.
- make check and make test-linux pass (no docker inside those images, so
  container discovery is a no-op there).

Open: Linux host with docker (host PIDs) only unit-tested; podman top not
supported; no real Claude Code in a container tried yet.

Files: internal/agent/{client,container,container_test,classify,discover,
control,procfs,procfs_test}.go, internal/watch/{watch,watch_test}.go,
internal/tui/{tui,view,keys}.go, internal/display/display.go,
commands.go, specs/{discovery,control,cli,overview}.md, README.md,
todo/todo.md.

Next step: try lazyai against a real devcontainer with Claude Code.

## Release v0.2.0 with Linux binaries

Operator copied the macOS bin/lazyai to an Ubuntu machine ("Exec format
error") and asked for a GitHub release for Ubuntu and Debian.

Done:

- Gitleaks scan of all 20 commits and the working tree: no leaks; no
  personal paths or emails in tracked files (test fixtures use
  /Users/u).
- Makefile `dist`: static (CGO_ENABLED=0) binaries for linux/amd64,
  linux/arm64, darwin/amd64, darwin/arm64, built with -trimpath and
  stripped, packed as lazyai-VERSION-OS-ARCH.tar.gz (entries owned by
  root) plus SHA256SUMS in bin/dist. Checked: the linux binary is a
  static ELF and holds no local paths.
- Makefile `test-linux` takes LINUX_ARCH; `make test-linux
  LINUX_ARCH=amd64` passes on Ubuntu 24.04 and Debian 13 under emulation
  (about 2 minutes).
- specs/release.md (new): artifacts, build flags, steps, security.
  README install section shows the download and checksum check.

Decision: Makefile target instead of GoReleaser (no extra tool).

- Commit c24bd53; annotated tag v0.2.0 on it. make dist on the tag gives
  lazyai-v0.2.0-{linux,darwin}-{amd64,arm64}.tar.gz; each linux tarball
  checked in Ubuntu 24.04 and Debian 13 containers of its arch
  (sha256sum -c OK, --version v0.2.0, list runs). gitleaks clean.
- Pushed main (e63e4c9..c24bd53) and tag v0.2.0 at operator request.

- Operator logged gh in. GitHub release v0.2.0 published with the four
  tarballs and SHA256SUMS (checked again before upload). Downloaded back
  from the public release URL in a Debian amd64 container: checksum OK,
  --version prints v0.2.0.

Open: tap formula not yet bumped to v0.2.0 (autobump may open a PR).

Files: Makefile, specs/{release,README,overview}.md, README.md,
todo/todo.md.

Run and verify: `make check`, `make test-linux`, `make test-linux
LINUX_ARCH=amd64`, `make dist`.

Next step: operator installs v0.2.0 on their Ubuntu machine and runs it
with real agents; bump the tap formula to v0.2.0.

## Debian packages

Operator asked for a deb package, "universal, static?". Answer: the
binary is static, so one package per architecture installs on any
Debian or Ubuntu release; a single package for both CPUs is not how
dpkg works (decision recorded in specs/release.md).

Done:

- Makefile `deb`: takes the binary from each linux tarball unchanged,
  writes control, copyright, changelog.gz and a lintian override, and
  runs dpkg-deb --root-owner-group in a debian:latest container as the
  calling user. Rewrites SHA256SUMS over tarballs and debs; tarball
  lines unchanged, so the published sums stay valid.
- lazyai_0.2.0_{amd64,arm64}.deb: install (pulling tmux and procps),
  run --version and list, and remove on Ubuntu 24.04 and 20.04, Debian
  13 and 11, both arches. lintian: only no-manual-page.
- Maintainer field comes from git config (same identity as the
  commits); nothing new is published about the operator.

Files: Makefile, specs/release.md, README.md, todo/todo.md.

Run and verify: `make dist` on the tag, then `make deb VERSION=v0.2.0`.

- Uploaded both debs and the new SHA256SUMS to the v0.2.0 release
  (--clobber replaced only SHA256SUMS; tarballs untouched) and added a
  Debian and Ubuntu section to the release notes. Installed from the
  public URL on a fresh Ubuntu 24.04 amd64: checksum OK, v0.2.0, tmux
  pulled in.

Next step: operator installs the .deb on their Ubuntu machine and runs
lazyai with real agents; bump the tap formula to v0.2.0.

## 2026-10-21 - TUI backspace compatibility

Done:

- Operator reported that Backspace did not delete characters in the `n` launch modal.
- Bubble Tea distinguishes DEL (`tea.KeyBackspace`) from the ctrl+h byte emitted by some terminal configurations (`tea.KeyCtrlH`). The modal handled only DEL.
- Added coverage for both encodings and made `updateInput` delete one rune for either key.
- Verified the regression test fails before the implementation change and passes after it.

Files: internal/tui/{tui.go,tui_test.go}, specs/tui.md, todo/todo.md.

Run and verify: `make check`.

Next step: run `bin/lazyai` in the affected terminal and press `n`, type text, then Backspace to confirm the terminal emits either supported encoding.
