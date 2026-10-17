# Discovery

## Process snapshot

`ps -axww -o pid=,ppid=,uid=,pcpu=,rss=,tty=,stat=,lstart=,args=` with
`LC_ALL=C`. The first 12 words are fixed columns (lstart is 5 words); the
rest is the command line with its spacing kept. Malformed lines are skipped.

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
| aider        | basename `aider`                                                |
| cursor-agent | basename `cursor-agent`, `/cursor-cli/`                         |
| hermes       | basename `hermes`, `/hermes-agent/` (Hermes Agent)              |
| ollama       | basename `ollama`                                               |
| colibri      | basename `coli`; or a path containing `colibri` whose basename is an engine: `qwen<N>...`, `qwenimage`, `colibri`, `glm...` |

argv[0] inside a macOS app bundle (`.app/Contents/`) is excluded, so
Claude.app, Cursor.app and their helpers are skipped. Homebrew's framework
Python (`.../Python.app/Contents/MacOS/Python`) is not a desktop app and is
not excluded; excluding it once hid Hermes.

A bare `qwen36` outside a colibri path is not matched, so other tools with
model-like names are not picked up.

Roles:

- claude: `bg-pty-host` or `bg-spare` -> helper; `daemon` -> daemon
- codex: `app-server` with `daemon` -> daemon; `app-server`, `mcp-server`
  -> server
- opencode: `serve`, `web` -> server
- hermes: `gateway`, `proxy`, `lsp` -> server
- ollama: `serve` -> server; `runner` -> engine
- colibri: a native engine binary -> engine; `coli` with `serve`, `web`,
  `cluster`, `start` -> server; `chat`, `run` or no subcommand ->
  interactive; any other subcommand (`tune`, `bench`, `build`, ...) -> job
- everything else -> interactive

Helpers are hidden by default in the TUI and `list`. Engines are shown,
because that is where model CPU and memory use shows up.

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
