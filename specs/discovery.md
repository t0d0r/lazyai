# Discovery

## Process snapshot

`ps -axww -o pid=,ppid=,uid=,pcpu=,rss=,tty=,stat=,lstart=,args=` with
`LC_ALL=C`. The first 12 words are fixed columns (lstart is 5 words); the
rest is the command line with its spacing kept. Malformed lines are skipped. No terminal is `??` on macOS, `?` on Linux;
terminal names are `ttys003` on macOS, `pts/3` on Linux.

## Classification

From the command line, using argv[0], and for interpreters (node, bun, deno,
bash, sh, zsh, python*, compared case-insensitively) also the first
non-flag argument:

| kind         | matched by                                                      |
|--------------|-----------------------------------------------------------------|
| claude       | basename `claude`, `/.local/share/claude/versions/`, `/@anthropic-ai/claude-code/` |
| codex        | basename `codex`, `/@openai/codex/`                             |
| opencode     | basename `opencode`                                             |
| gemini       | basename `gemini`/`gemini.js`, `/gemini-cli/`                   |
| qwen         | basename `qwen`, `/@qwen-code/` (Qwen Code)                     |
| grok         | basename `grok`, `/.grok/` (xAI Grok CLI; the binary is `~/.grok/downloads/grok-<version>-...`) |
| copilot      | basename `copilot`, `/copilot-cli/`, `/@github/copilot/` (GitHub Copilot CLI) |
| aider        | basename `aider`                                                |
| cursor-agent | basename `cursor-agent`, `/cursor-cli/`                         |
| hermes       | basename `hermes`, `/hermes-agent/` (Hermes Agent)              |
| ollama       | basename `ollama`                                               |
| mlx-lm       | basename `mlx_lm` or `mlx_lm.<command>` (Apple MLX LM)          |
| colibri      | basename `coli`; or a path containing `colibri` whose basename is an engine: `qwen<N>...`, `qwenimage`, `colibri`, `glm...` |

A program inside a macOS app bundle (`.app/Contents/`) is excluded, so
Claude.app, Cursor.app, ChatGPT.app, "Grok Bot.app" and their helpers are
skipped. The check runs on the command line up to the first option, so
bundle names with spaces are caught. Homebrew's framework Python
(`.../Python.app/Contents/MacOS/Python`) is not a desktop app and is not
excluded; excluding it once hid Hermes.

Editor plugins are not agents: the Copilot language server of copilot.vim
(`node .../copilot.vim/dist/agent.js`) is not matched.

## Containers

`docker run` and `podman run` are matched when the image's repository name
(last path segment, without tag or digest) is a supported kind, for
example `ghcr.io/anomalyco/opencode:latest` -> opencode. Options that take
a separate value (`-v`, `-e`, `-w`, `-p`, `--name`, ...) are skipped so
their values are not taken for the image. A detached container (`-d`,
`--detach`, or `d` among combined short flags such as `-dit`) is a server,
otherwise interactive. The process lazyai sees and stops is the docker
client on this Mac; with `-it` it forwards signals to the container.

A bare `qwen36` outside a colibri path is not matched, so other tools with
model-like names are not picked up.

Roles:

- claude: `bg-pty-host` or `bg-spare` -> helper; `daemon` -> daemon
- codex: `app-server` with `daemon` -> daemon; `app-server`, `mcp-server`
  -> server
- opencode: `serve`, `web` -> server
- hermes: `gateway`, `proxy`, `lsp` -> server
- copilot: `--acp` -> server; `-p`, `--prompt` -> job
- ollama: `serve` -> server; `runner` -> engine
- mlx-lm: `server` -> server; `chat` or no command -> interactive; any
  other command (`generate`, `lora`, `convert`, ...) -> job
- colibri: a native engine binary -> engine; `coli` with `serve`, `web`,
  `cluster`, `start` -> server; `chat`, `run` or no subcommand ->
  interactive; any other subcommand (`tune`, `bench`, `build`, ...) -> job
- everything else -> interactive

Helpers are hidden by default in the TUI and `list`. Engines are shown,
because that is where model CPU and memory use shows up.

A shell running a command string (`sh -c CMD`, `bash -lc CMD`, any short
flag group with `c`, and `python -c CODE`) is not an agent: the agent is
its child, or replaces the shell when the shell execs it. Linux `/bin/sh`
(dash) does not exec, so without this rule such agents were listed twice.
`sh SCRIPT` still matches the script.

## Linux

- CPU: Linux `ps` reports CPU averaged over the life of the process, so a
  long-idle agent that just got busy would look idle. Each snapshot
  replaces it with usage since the previous snapshot, from utime + stime
  in `/proc/<pid>/stat` (USER_HZ 100). The start time in that file tells a
  reused PID apart. The first snapshot of a run has nothing to compare
  with and samples for 250 ms. A process missing from either sample keeps
  the `ps` value. macOS `ps` already reports recent usage.
- Working directory: `/proc/<pid>/cwd`, then `lsof` for what that leaves
  out, only if `lsof` is installed (many Linux systems lack it).
- Containers: a process in a PID namespace nested below lazyai's own (the
  NSpid line of `/proc/<pid>/status` has more entries) runs inside a
  container. On Linux those processes are visible on the host, unlike
  macOS where containers run in a VM, so they are dropped; the `docker run`
  client stands for them (see Containers). If lazyai itself runs in a
  container, processes in that same container are kept.
- tmux: `list-panes` is called with `-u`. Without a UTF-8 locale (common in
  containers, cron and ssh sessions) tmux 3.4 prints the tab separators,
  and non-ASCII session names, as `_`.

## Status

- Claude interactive sessions: read `~/.claude/sessions/<pid>.json`
  (`$CLAUDE_CONFIG_DIR/sessions` if set) for `sessionId`, `cwd`, `name`,
  `status`, `statusUpdatedAt`. The file is used only if its `pid` matches and
  its `procStart` equals the process start time. Claude writes `procStart`
  in UTC; both UTC and local time are accepted. `*.key` files are never read.
- Everyone else: busy when CPU >= 2%, otherwise idle.
- Session id falls back to `--resume <id>` from the command line.

## Working directory

Claude session file first, otherwise one `lsof -a -d cwd -Fn -p ...` call
for all agents that still lack one.

## tmux mapping

`tmux list-panes -a` gives session, pane id and tty for each pane. An agent
whose tty equals a pane tty belongs to that pane. No tmux server means no
panes, not an error.

## Process tree

Each agent carries its descendants (depth first, with depth, CPU, memory
and redacted command) and TreeCPU / TreeRSS, the sums over the agent and
all descendants. An agent below another agent (for example a colibri
engine started by Hermes through make) is both its own row and part of
the parent's tree.

## Self protection

lazyai's own PID and all its ancestors are marked `protected`.

## Output

`list -json` prints the agents with the raw command line omitted and a
redacted `command` field instead (see control.md, Redaction).
