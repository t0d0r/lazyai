# Discovery

## Process snapshot

`ps -axww -o pid=,ppid=,uid=,pcpu=,rss=,tty=,stat=,lstart=,args=` with
`LC_ALL=C`. The first 12 words are fixed columns (lstart is 5 words); the
rest is the command line with its spacing kept. Malformed lines are skipped.

## Classification

From the command line, using argv[0], and for interpreters (node, bun, deno,
bash, sh, zsh, python*) also the first non-flag argument:

| kind         | matched by                                                      |
|--------------|-----------------------------------------------------------------|
| claude       | basename `claude`, `/.local/share/claude/versions/`, `/@anthropic-ai/claude-code/` |
| codex        | basename `codex`, `/@openai/codex/`                             |
| opencode     | basename `opencode`                                             |
| gemini       | basename `gemini`/`gemini.js`, `/gemini-cli/`                   |
| aider        | basename `aider`                                                |
| cursor-agent | basename `cursor-agent`, `/cursor-cli/`                         |
| ollama       | basename `ollama`                                               |

Anything whose argv[0] contains `.app/Contents/` is excluded.

Roles:

- claude: `bg-pty-host` or `bg-spare` -> helper; `daemon` -> daemon
- codex: `app-server` with `daemon` -> daemon; `app-server`, `mcp-server`
  -> server
- opencode: `serve`, `web` -> server
- ollama: `serve` -> server; `runner` -> helper
- everything else -> interactive

Helpers are hidden by default in the TUI and `list`.

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

## Self protection

agentctl's own PID and all its ancestors are marked `protected`.

## Output

`list -json` prints the agents with the raw command line omitted and a
redacted `command` field instead (see control.md, Redaction).
