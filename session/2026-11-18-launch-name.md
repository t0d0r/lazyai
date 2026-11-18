# 2026-11-18 - launch name is the session and window name

## Done

- A NAME on the launch line, typed in `n` or given to `lazyai launch`
  (positional or `-name`), is now the tmux session name as typed, and the
  session's window has the same name. `claude . web` starts the session
  `web` with the window `web`. Before, the session was `lazyai-web` and
  the window was named after the command (`claude`), so the name typed
  showed nowhere in tmux as it was typed.
- The name is still cleaned for tmux: characters outside `[A-Za-z0-9_-]`
  become `-`, `-` at the ends is dropped (`"home work"` -> `home-work`),
  and a taken name gets `-2`, `-3` ...; the window follows the session
  (`web-2`). A name with nothing left (`...`) is refused.
- Without NAME nothing changes: session `lazyai-KIND-DIRBASENAME`, window
  named after the command.
- Sessions created by launch carry the tmux session option `@lazyai 1`
  (tmux.Mark). tmux.Launched is true for a session with the mark or with
  the `lazyai-` prefix. Attach uses it for the status-line hint and the
  window fix, which looked at the prefix only.
- Rename (`R`, `lazyai rename`) renames a window that carries the
  session's old name along with the session (tmux.RenameSession), so a
  session launched with a name keeps session and window equal. Windows
  named otherwise are left alone.
- Help text, examples, README and specs describe the new naming.
- Live check: `lazyai launch cat . lztest-N` and `-n lztest-N` gave the
  sessions `lztest-N` and `lztest-N-2`, each with a window of the same
  name, the mark set and the status-line hint in place.

## Release v0.12.0

- Checks before the tag: make check, make test-linux (arm64) and
  make test-linux LINUX_ARCH=amd64 pass on Ubuntu 24.04 and Debian 13.
- v0.12.0 tagged (annotated) on main; artifacts built with make dist and
  make deb VERSION=v0.12.0. README install examples point at v0.12.0.
- main and the tag pushed (gitleaks over the 2 unpushed commits: no
  leaks). GitHub release v0.12.0 published with the four tarballs, the
  amd64 and arm64 .deb packages and SHA256SUMS; the downloaded assets
  pass `shasum -a 256 -c SHA256SUMS`; the darwin-arm64 binary reports
  v0.12.0.
- Homebrew tap: Formula/lazyai.rb moved to the v0.12.0 source archive
  (sha256 3da2f4fab186261eb560cf47a09d3ef9ff8b8fd2efeae075fcf800905783c29f,
  the same on two downloads), committed and pushed. brew style and brew
  audit --strict clean; brew upgrade built 0.12.0 and brew test passes.
- Minor version: a launch with a NAME gives another session name than
  before, which scripts see in `tmux ls` and in the "launched" line.

## Decisions

- NAME is not prefixed. Reason: the name typed is the one to find in
  `tmux ls`, on the status line and in `attach`. The default name keeps
  `lazyai-`, so sessions nobody named are still easy to tell apart.
- A session option marks lazyai's sessions instead of the name. Reason:
  without a prefix the name no longer says who started the session. A
  side effect: a renamed session keeps the mark, so attach keeps
  refreshing its status-line hint.
- The window takes the final session name, suffix included, so the two
  are always equal.

## Open

- Sessions launched with a NAME before v0.12.0 keep their `lazyai-NAME`
  name and their window name; nothing is migrated.
- A given NAME can now equal the name of a session not started by lazyai;
  launch then takes NAME-2, as for any taken name.
- Tab completion does not understand quotes; the filter input (`/`) has
  no cursor (both from the earlier note of this day).

## Files

internal/tmux/tmux.go, internal/tmux/tmux_test.go,
internal/agent/launch.go, internal/agent/agent_test.go, commands.go,
main.go, examples.go, README.md, specs/control.md, specs/tui.md,
todo/todo.md.

## Run and verify

make check; brew upgrade lazyai && lazyai --version -> lazyai 0.12.0.
make build, then `bin/lazyai launch claude . web` and
`tmux list-windows -a -F '#{session_name} #{window_name} #{@lazyai}'`
shows `web web 1`. In the dashboard (make run): `n`, `claude . web`,
enter: the message line says `launched web (a attaches)`.

## Next step

Next open todo item.

## Security note

The name is still cleaned to letters, digits, `-` and `_` before it
reaches tmux, and tmux is run without a shell, so a name cannot inject
tmux targets or commands. Without the prefix a launched session can
share a naming pattern with sessions of other tools; scripts that kill
or attach by name pattern should match on the `@lazyai` option, not on
the name.
