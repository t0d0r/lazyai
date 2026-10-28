# 2026-10-28 - self-update

## Done

- `lazyai update [--check] [--yes]` (specs/update.md): reads the latest
  release from the releases/latest redirect, compares it with the built
  version, and installs it the way lazyai was installed:
  - homebrew (path under /Cellar/lazyai/): `brew upgrade t0d0r/tap/lazyai`
  - deb (Linux, dpkg owns the path): download the .deb, check it against
    SHA256SUMS, `sudo apt-get install -y FILE` (no sudo as root)
  - binary (anything else): download the tarball, check it, extract
    DIR/lazyai, write a temp file next to the binary, rename over it
- Source builds (`dev`, `git describe` between tags, `-dirty`) are not
  replaced; `--check` still reports the latest release.
- internal/update (new) with tests: version parse and compare, the
  redirect, checksum mismatch and missing entry, tarball extraction
  (symlinks and other directories refused), replace in a temp dir and a
  read-only dir, method detection, a full binary update against a local
  server, and a truncated download that leaves the binary untouched.
- Checked by hand against the v0.3.0 release: macOS binary method (also
  through a symlink), Debian 13 binary method as a normal user,
  read-only /usr/local/bin error, deb method from the 0.2.1 package as
  root and as a sudo user, homebrew detected for a Cellar path.

## Decisions

- Latest release from the redirect, not the GitHub API: no JSON, no API
  rate limit.
- No automatic update check and no TUI action: lazyai uses the network
  only when `update` runs.
- Downloads follow GitHub's redirect to its asset host but only over
  HTTPS; at most 64 MiB per file, 2 minute timeout per request.

## Open

- `brew upgrade` was not run end to end: the installed formula is the
  latest, and 0.3.0 has no update command. Possible after the next two
  releases.
- Minimal containers need ca-certificates for HTTPS; the .deb does not
  depend on it.

## Files

internal/update/{update.go,update_test.go}, commands.go, specs/update.md,
specs/cli.md, specs/README.md, README.md, todo/todo.md.

## Run and verify

`make check`. By hand: `go build -ldflags "-X main.version=v0.2.1" -o
/tmp/x/lazyai .` then `/tmp/x/lazyai update --check` and
`/tmp/x/lazyai update -y`.

## Release v0.4.0

- make check, make test-linux and make test-linux LINUX_ARCH=amd64 pass
  (Ubuntu 24.04, Debian 13).
- README install examples now use v0.4.0.

- Annotated tag v0.4.0; make dist and make deb VERSION=v0.4.0 on the
  tag. gitleaks: no leaks. Pushed main and the tag.
- GitHub release v0.4.0 published with four tarballs, two debs and
  SHA256SUMS; all six pass the checksum check after download. The
  README's Debian commands, run in fresh debian:latest containers (arm64
  and amd64), install 0.4.0 and `lazyai update --check` reports it up to
  date.
- Homebrew tap formula updated to the v0.4.0 source archive (sha256
  8ca4cf4a8555b2e1a63895345d02d1b4fcaea4fed453b37f4fcb608d5f313e7c);
  style, audit, build from source and brew test pass; pushed. The
  installed lazyai 0.4.0 reports itself up to date.

## Next step

After the next release, run `lazyai update` through brew and apt from
0.4.0 to check the homebrew and deb methods end to end.

## Memory total in the status panel

- The status panel shows the total memory of the shown agents next to
  the refresh time: `16 running, 4 busy   mem 3.3G 01:59:11`
  (specs/tui.md). totalRSS in internal/tui/view.go adds each agent's
  TreeRSS and skips an agent that is already in another shown agent's
  tree; container agents are keyed by container ID and PID.
- Narrow column: drops `mem `, then shortens "running" to "run".
  Checked in tmux at 160, 100 and 80 columns.
- Test: TestTotalRSS (nested agent counted once, container PID kept
  apart from the same host PID, text in the rendered panel).

Decision: memory of the agents, not of the machine.

Files: internal/tui/{view.go,tui_test.go}, specs/tui.md, todo/todo.md.

## Release v0.4.1

- make check, make test-linux and make test-linux LINUX_ARCH=amd64 pass
  (Ubuntu 24.04, Debian 13). README install examples now use v0.4.1.
