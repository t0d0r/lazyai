# Control

## Stop

- Only agents returned by discovery can be stopped; never arbitrary PIDs.
- Refused when the agent is protected (lazyai or a parent of it), owned
  by another user, or reported by docker from inside a container
  (`foreign_pid`; its PID is not one of this machine's). The message
  points to `docker stop CONTAINER`.
- Before signalling, the PID is looked up again; its start time and command
  line must equal the listed ones, otherwise the stop is refused as "exited
  or PID reused".
- Default SIGTERM, then wait up to 5 seconds for the process to go away.
  Force sends SIGKILL.
- CLI asks `[y/N]` per PID unless `-yes`; the TUI asks `y/N` in the footer.

## Launch

- `launch [-dir D] [-name N] KIND [DIR] [NAME] [-- ARGS...]`, the same
  line as `n` in the dashboard, read by one parser (agent.ParseLaunch).
  DIR may use `~` and is relative to `-dir` (default: the current
  directory); NAME names the session; agent arguments go after `--`. A
  word before `--` that starts with `-` is a usage error that says
  arguments go after `--`; so is a fourth word. NAME given both as
  `-name` and positionally is a usage error.
  - Decision: one launch line for CLI and TUI. Reason: a line typed in
    `n` must work unchanged on the command line. This changes the CLI:
    agent arguments now need `--` (`lazyai launch ollama -- run
    llama3.2`); up to v0.5.0 everything after KIND went to the agent.
- KIND must be a supported
  kind other than colibri and mlx-lm; ollama requires arguments. Colibri
  is refused with a pointer to its own tooling (`make run` in its project),
  because `coli` is not on PATH and needs its model and settings
  configured. mlx-lm is refused with a pointer to `mlx_lm.server` /
  `mlx_lm.chat`, which need a model. Containers are not launched.
- KIND may also be any other command name found on PATH, for example a
  wrapper script such as `claude-bst` that runs claude with another
  profile. Only a bare name is accepted: no `/` and no leading `-`, so
  the command always comes from PATH. What it starts is classified by
  discovery as usual (a wrapper that execs claude shows as claude).
  - Decision: accept commands on PATH instead of a fixed list. Reason:
    agents are often started through wrapper scripts with other
    profiles or settings; the operator types the line, with the same
    rights as in a shell, so the list of kinds gave no protection.
- The command is resolved with `PATH` lookup and run directly by
  `tmux new-session -d -s SESSION -c DIR -- PATH ARGS...`. With no ARGS
  tmux would hand the single word to the shell, so it runs `env PATH`
  instead, two words that tmux runs directly; env then execs the agent in
  its own place. A PATH containing `=` is refused there, because env would
  take it for a variable. No shell is involved, so nothing in names,
  paths or arguments is expanded.
- Session name is `lazyai-` plus the name (default `KIND-DIRBASENAME`)
  with characters outside `[A-Za-z0-9_-]` replaced by `-`. If taken, `-2`,
  `-3` ... is appended.
- The tmux session ends when the agent exits.
- Early exit: the session is created with `remain-on-exit on` in the
  same tmux call (`new-session ... ; set-option -w -t =SESSION:
  remain-on-exit on`, targeted so that inside tmux the option does not
  land on lazyai's own window). For 1.5 s (tmux.LaunchWatch) lazyai
  checks `#{pane_dead}` every 100 ms:
  - pane dead: read its whole output (`capture-pane -p -J -S -`), kill
    the session and fail with `KIND exited right after start (status N):
    LAST LINE`, keeping the last 20 non-empty lines without tmux's
    `Pane is dead` line (tmux.ExitError). The CLI prints the message and
    exits 1.
  - still running: set `remain-on-exit off`, so the session ends with
    the agent as before.
  Reason: without it a command that fails at once (unknown option,
  missing login, bad arguments) takes its session and its error message
  with it, and the launch looks successful. A command that fails later
  than 1.5 s is not caught; its session ends as before.

## Attach

- Requires the agent to run in a tmux pane.
- Outside tmux: `tmux attach-session -t =SESSION` and return on detach.
  Inside tmux: `tmux switch-client -t =SESSION`.
- The `=` prefix forces an exact session name match.
- Status line hint: `lazyai-*` sessions get a session-only `status-right`
  of `PREFIX d: detach, back to lazyai | HH:MM`, where PREFIX is the tmux
  prefix in readable form (`ctrl+b`). It is set at launch and again at
  every attach; when lazyai itself runs inside tmux the hint is
  `PREFIX L: back to lazyai` (last session), because detaching would leave
  tmux altogether. Global options and sessions not started by lazyai are
  never changed. Failures to set it are ignored.

## Send

- Requires a tmux pane. `tmux send-keys -t PANE -l -- TEXT`, a 150 ms pause,
  then `send-keys Enter`. `-l` sends the text literally.
- CLI targets: PID, tmux session name with or without `lazyai-`, or
  Claude session name; ambiguous names are refused.
- TUI: when a client is attached to the session, `p` first asks
  "SESSION is attached (N clients); someone may be typing there. Send a
  prompt anyway?" and opens the prompt popup only on `y`. Reason: text
  typed by a person in the pane and the sent prompt would mix, and Enter
  would submit both. The CLI `send` does not ask: it runs from scripts
  that decided already.

## Rename

- Requires the agent to run in a tmux session.
  `tmux rename-session -t =OLD -- NEW`.
- NEW is taken as given, with no `lazyai-` prefix added: letters, digits,
  `-` and `_`, 1 to 100 characters. Reason: tmux turns `:` and `.` into
  `_` and allows spaces, which make the session hard to name in `a`,
  `attach` and scripts. Renaming to the current name does nothing.
- A NEW that names another session is refused (`a tmux session named NEW
  exists already`), as are tmux's own errors, with tmux's message.
- Important marks of OLD move to NEW (tui.md, Important mark).
- A session renamed away from `lazyai-` keeps the status-line hint set at
  launch, but attach no longer refreshes it, as for any session not
  started by lazyai.
- CLI targets as for Send.

## Redaction

Command lines shown on screen or in JSON replace values of arguments whose
name matches key, token, secret, password, credential or auth:
`--api-key X`, `--api-key=X`, `NAME_TOKEN=X`.
