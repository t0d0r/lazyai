# Todo

## Done

- [x] process snapshot and parsing (specs/discovery.md)
- [x] agent classification and roles (specs/discovery.md)
- [x] Claude session metadata with PID-reuse check (specs/discovery.md)
- [x] tmux launch, pane mapping, send (specs/control.md)
- [x] safe stop with identity re-check and self protection (specs/control.md)
- [x] CLI: list, stop, launch, attach, send (specs/control.md)
- [x] TUI dashboard with watch flags (specs/tui.md)
- [x] unit tests and a tmux integration test
- [x] Makefile with the standard targets from ../AGENTS.md
- [x] CLI parity with the TUI: show, watch, kill, flags in list, stop by
      name (specs/cli.md)
- [x] Unix-style CLI: -h/--help everywhere, long and short options,
      options after arguments, --version, exit status 2 for usage errors
      (specs/cli.md)
- [x] rename agentctl -> lazyai
- [x] lazydocker-style dashboard: Agents / Services / Models panels, Info,
      Screen, Processes and Stats tabs, x menu, ? help, mouse
      (specs/tui.md)
- [x] process tree per agent with tree CPU and memory (specs/discovery.md)

- [x] fix: Homebrew framework Python (Python.app) was excluded as a desktop
      app, hiding Python agents such as Hermes (specs/discovery.md)
- [x] kinds hermes, qwen (Qwen Code), colibri; roles job and engine
      (specs/discovery.md)
- [x] kinds grok, copilot, mlx-lm; agents in docker/podman run; desktop
      apps with spaces in the bundle name (specs/discovery.md)
- [x] Linux support, tested in docker on Ubuntu 24.04 and Debian 13 with
      make test-linux (specs/overview.md, specs/discovery.md)
- [x] agents in docker containers: docker run IMAGE CMD, exec, compose
      run/exec, sandbox run, and agents typed into a container shell via
      docker ps/top (specs/discovery.md, Containers)

- [x] TUI launch/prompt input accepts both DEL and ctrl+h backspace encodings
      (specs/tui.md)
- [x] `list` makes tmux membership explicit with `IN_TMUX=yes/no` and keeps
      the matched session in `TMUX_SESSION` (specs/cli.md)

- [x] launch popup: up/down history kept across restarts, tab completes
      the kind and the directory (specs/tui.md)

## Next

- [ ] verify attach by hand from a real terminal, outside and inside tmux
- [ ] verify send into a real Claude Code session in tmux (paste and Enter
      timing)
- [ ] check classification against real gemini, aider, cursor-agent and
      ollama processes (only claude, codex, opencode were running)
- [ ] decide opencode launch: the fish alias runs it in docker, lazyai
      runs the Homebrew binary
- [ ] Codex status and session id from ~/.codex metadata (needs a spec)
- [ ] verify Hermes server roles (gateway, proxy, lsp) and Qwen Code
      against real processes; only interactive Hermes has been seen
- [ ] verify copilot, mlx-lm and docker-run opencode against real
      processes; only grok was running when they were added
- [ ] try the dashboard in a real terminal: mouse, attach and return,
      colors in light and dark themes
- [ ] Screen tab with colors (`capture-pane -e`), if it stays readable when
      lines are cut to the panel width
- [ ] colibri server status from `coli status` or its pid file, if it has
      one
- [ ] run `make install` and confirm ~/.local/bin is on PATH in fish
- [ ] add a gitleaks pre-commit hook
- [ ] module path github.com/t0d0r/lazyai so `go install
      github.com/t0d0r/lazyai@latest` works
- [x] LICENSE: BSD-2-Clause
- [x] release v0.1.0 and Homebrew formula in t0d0r/homebrew-tap
- [ ] for each release: tag vX.Y.Z, then update url and sha256 in the tap
      formula (or let the tap's autobump workflow open the PR)
      (specs/release.md)
- [x] prebuilt static binaries for linux and macOS: `make dist`, attached
      to the GitHub release (specs/release.md)
- [x] release v0.2.0 with Linux and container support on GitHub
      (specs/release.md)
- [x] .deb packages for amd64 and arm64: `make deb`, added to the v0.2.0
      release (specs/release.md, Debian packages)
- [x] release v0.3.0 with launch history and tab completion
      (specs/release.md); tap formula updated to 0.3.0
- [x] `lazyai update`: self-update to the latest release by brew, apt
      or binary replacement (specs/update.md)
- [x] release v0.4.0 with `lazyai update` (specs/release.md)
- [ ] run `lazyai update` through brew and apt from 0.4.0 to a later release
- [x] total memory of the shown agents next to the clock in the status
      panel (specs/tui.md)
      (specs/update.md)
- [ ] man page (lintian no-manual-page), for example generated from
      `lazyai help`
- [ ] APT repository, if wanted: needs a signing key and hosting
      (specs/release.md)
- [ ] set the GitHub description and topics
- [ ] try lazyai on a real Linux desktop with real agents: Claude session
      file status (procStart in UTC from procps), npm-installed agents,
      docker-run opencode under rootful docker (specs/discovery.md, Linux)
- [ ] check that the Homebrew formula builds and passes brew test on
      Linuxbrew, then release with Linux support
- [x] make test-linux on amd64 too: `LINUX_ARCH=amd64` (emulated);
      passes on Ubuntu 24.04 and Debian 13
- [ ] decide whether to run make test-linux in CI
- [ ] agents in containers on a Linux host (docker top gives host PIDs;
      only unit-tested): try it in a docker-in-docker test image
      (specs/discovery.md, Docker containers)
- [ ] podman containers: podman top takes AIX-style descriptors, not ps
      options
- [ ] stop for container agents through docker (docker exec CTR kill),
      if it is wanted; refused for now (specs/control.md)
- [ ] try a real Claude Code in a container and a VS Code devcontainer
