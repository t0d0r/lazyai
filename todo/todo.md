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
- [x] Screen tab with colors (`capture-pane -e`), if it stays readable when
      lines are cut to the panel width (specs/tui.md, Tabs)
- [ ] colibri server status from `coli status` or its pid file, if it has
      one
- [ ] run `make install` and confirm ~/.local/bin is on PATH in fish
- [x] add a gitleaks pre-commit hook: .githooks/pre-commit, turned on
      with `make hooks` (specs/release.md, Security)
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
- [x] run `lazyai update` through brew and apt from 0.4.0 to a later release
- [x] total memory of the shown agents next to the clock in the status
      panel (specs/tui.md)
- [x] release v0.4.1 with the memory total (specs/release.md)
- [x] mark attachable agents with @ and show memory in every list
      (specs/tui.md)
- [x] release v0.4.2 with the attachable mark and memory per row
      (specs/release.md)
- [x] directory labels show the last two directories, cut as alpha/..nd;
      @ right before the kind (specs/tui.md)
- [x] release v0.4.3 with two-directory labels (specs/release.md)
- [x] launch: failed lines stay in the history; any command on PATH
      (wrapper scripts such as claude-bst) launches and completes
      (specs/control.md, specs/tui.md)
- [x] `/` filters the list in use (specs/tui.md, Filter)
- [x] `lazyai launch` reads the same line as `n`: KIND [DIR] [NAME]
      [-- ARGS] (specs/control.md, specs/cli.md)
- [x] examples in every command's help and `lazyai examples [COMMAND]`;
      top-level help shows the launch line (specs/cli.md)
- [x] release v0.6.0 with the shared launch line and examples; the
      notes call out `--` before agent arguments (specs/release.md)
- [x] show which tmux sessions are attached: `>` mark, attached count,
      Info line, `/attached` `/detached`, confirm before `p`
      (specs/tui.md, specs/discovery.md, specs/control.md)
- [x] release v0.5.0 with the list filter, PATH commands and attached
      sessions (specs/release.md)
- [x] label rows in tmux by the full tmux session, Claude name after it;
      `list` TMUX_SESSION keeps the `lazyai-` prefix (specs/tui.md,
      specs/cli.md)
- [x] release v0.7.0 with the tmux session labels (specs/release.md)
- [x] left / right and h / l no longer move the focus; tab, shift+tab,
      1 2 3 and enter / esc do (specs/tui.md, Keys)
- [x] Processes tab shows the chain of parents above the agent
      (specs/tui.md, specs/discovery.md)
- [x] release v0.8.0 with the focus key change and the parent chain
      (specs/release.md)
- [x] v0.9.0 follow-up: Homebrew tap moved to 0.9.0, README install
      examples moved on with v0.10.0 (specs/release.md)
- [x] screen layout on v: lists as tabs and Info / Processes / Stats on
      top, Screen across the whole width below (specs/tui.md, Layouts)
- [x] screen layout: Screen panel at 80% of the height, + / - resize it
      (specs/tui.md, Layouts)
- [x] screen layout: `lazyai` in the tabbed list title, status line in
      its bottom border, no status panel (specs/tui.md, Layouts)
- [x] S and an x menu item save layout, tab and Screen size to
      settings.json for the next start (specs/tui.md, Saved settings)
- [x] release the screen layout as v0.10.0 (specs/release.md)
- [x] Screen panel title shows the last two directories and the Claude
      session name after the tmux session (specs/tui.md, Layouts)
- [x] `lazyai update` with homebrew: compare with the tap formula
      version and say "released, not yet in the Homebrew tap" instead
      of offering an upgrade brew cannot do (specs/update.md)
- [x] dashboard layouts: `v` cycles split, detail (compact lists, wide
      main panel) and lists (no main panel) (specs/tui.md, Layouts)
- [ ] try the layouts in a real terminal: switch while resizing, mouse
      in detail and lists, Screen tab after leaving lists
      (specs/tui.md, Layouts)
- [x] release the layouts with the next batch: v0.10.0 (specs/release.md)
- [x] `--layout` option or a saved layout, if wanted: saved with S
      (specs/tui.md, Saved settings)
- [ ] status panel: count agents outside tmux (`13 outside tmux`), if
      wanted (specs/tui.md)
- [ ] optional `~` short form for `lazyai-` sessions in narrow rows, if
      wanted (specs/tui.md)
- [ ] find why `lazyai update` sees the previous release for about a
      minute after publishing, while curl sees the new one
      (specs/update.md)
      (specs/tui.md)
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
- [x] launch feedback: an agent that exits at once shows its output in
      a popup; a launched agent is selected and its row is yellow for 3 s
      (specs/control.md Launch, specs/tui.md Launch feedback)
- [x] status colors: busy green, idle cyan in the rows and the detail
      state line (specs/tui.md Layout)
- [x] `!` marks an agent important: ` !` after the label, magenta row,
      toggled again with `!` or from the x menu (specs/tui.md Layout)
- [x] keep important marks across restarts in
      ~/.local/state/lazyai/important, keyed by tmux session, Claude
      session id, or process (specs/tui.md Layout)
- [x] rename an agent's tmux session: `R` in the TUI, `lazyai rename`
      in the CLI; marks follow (specs/control.md Rename, specs/tui.md)
- [x] release v0.10.1 with status colors, the `!` important mark, launch
      feedback and Screen colors; tap formula updated to 0.10.1
      (specs/release.md)
