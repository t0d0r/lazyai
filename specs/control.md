# Control

## Stop

- Only agents returned by discovery can be stopped; never arbitrary PIDs.
- Refused when the agent is protected (lazyai or a parent of it), or owned
  by another user.
- Before signalling, the PID is looked up again; its start time and command
  line must equal the listed ones, otherwise the stop is refused as "exited
  or PID reused".
- Default SIGTERM, then wait up to 5 seconds for the process to go away.
  Force sends SIGKILL.
- CLI asks `[y/N]` per PID unless `-yes`; the TUI asks `y/N` in the footer.

## Launch

- `launch [-dir D] [-name N] KIND [ARGS...]`. KIND must be a supported
  kind other than colibri and mlx-lm; ollama requires arguments. Colibri
  is refused with a pointer to its own tooling (`make run` in its project),
  because `coli` is not on PATH and needs its model and settings
  configured. mlx-lm is refused with a pointer to `mlx_lm.server` /
  `mlx_lm.chat`, which need a model. Containers are not launched.
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

## Redaction

Command lines shown on screen or in JSON replace values of arguments whose
name matches key, token, secret, password, credential or auth:
`--api-key X`, `--api-key=X`, `NAME_TOKEN=X`.
