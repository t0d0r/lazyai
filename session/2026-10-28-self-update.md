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
- Annotated tag v0.4.1; make dist and make deb on the tag; gitleaks: no
  leaks. Pushed main and the tag. GitHub release v0.4.1 published; all
  six files pass the checksum check after download.
- First real self-update, from the published v0.4.0: deb method as a
  sudo user and binary method in ~/.local/bin, on Debian 13 arm64 and
  amd64, both end at v0.4.1. Homebrew tap formula updated to v0.4.1
  (sha256 5144f42765f9cdbcea917234639da1f5e095868e7e7ea466d9edc97891da2daf,
  style and audit pass, pushed); `lazyai update -y` on macOS ran brew
  upgrade from 0.4.0 to 0.4.1 and brew test passes.

Run and verify: `make check`; `lazyai update --check`.

Next step: try lazyai on a real Linux desktop with real agents
(todo/todo.md).

## Attachable mark and memory per row

- Dashboard rows start with `@` when the agent runs in a tmux pane, so
  attach (`a`) and prompt (`p`) work on it; other rows have a blank
  gutter. The `?` popup has a Marks group for `@` and `*`. The mark uses
  the same check as the action menu (disabledReason).
- Every list now shows tree memory after status and CPU, not only
  Models.
- Checked in tmux at 160 and 80 columns; at 80 the label is cut first.
- Test: TestAttachMark (mark only on the agent in tmux, legend in help).

Files: internal/tui/{view.go,keys.go,tui_test.go}, specs/tui.md,
todo/todo.md.

## Release v0.4.2

- make check, make test-linux and make test-linux LINUX_ARCH=amd64 pass
  (Ubuntu 24.04, Debian 13). README install examples now use v0.4.2.
- Annotated tag v0.4.2; make dist and make deb on the tag; gitleaks: no
  leaks. Pushed main and the tag. GitHub release v0.4.2 published; all
  six files pass the checksum check after download.
- Homebrew tap formula updated to v0.4.2 (sha256
  fe21511066e704bec767d1d970b6143c3578febedebfca65ce124fd3bab6be8a),
  style and audit pass, pushed. `lazyai update -y` on macOS ran brew
  upgrade from 0.4.1 to 0.4.2; brew test passes.
- Seen right after publishing: on a Linux host with the v0.4.1 binary,
  `lazyai update --check` said up to date twice while curl from the same
  host already got the v0.4.2 redirect; the third run found v0.4.2.
  GitHub serves the old redirect to some clients for a short time.
  Recorded in specs/update.md.

Open: if the stale redirect becomes a nuisance, try request headers
(Accept, Cache-Control: no-cache) or the releases API, and confirm on
the next release.

Run and verify: `make check`; `lazyai update --check`.

Next step: try lazyai on a real Linux desktop with real agents
(todo/todo.md).

## Two-directory labels

- Rows labelled by their working directory show the last two
  directories (`alpha/web-frontend`, home as `~`) instead of the
  basename, so the same project name under different parents stays
  apart. Session names and tmux sessions are unchanged.
- Narrow rows: shortenPath keeps the parent and replaces the start of
  the last directory with `..` (`alpha/..nd`); with less room, the start
  of the whole label (`..ontend`). Status, CPU and memory stay.
- Checked in tmux at 160 and 100 columns against the live agents.
- Test: TestPathLabel.

Files: internal/tui/{view.go,keys.go,tui_test.go}, specs/tui.md,
todo/todo.md.

## Release v0.4.3

- make check, make test-linux and make test-linux LINUX_ARCH=amd64 pass
  (Ubuntu 24.04, Debian 13). README install examples now use v0.4.3.
- Annotated tag v0.4.3; make dist and make deb on the tag; gitleaks: no
  leaks; no real directory or session names in the pushed commits.
  Pushed main and the tag. GitHub release v0.4.3 published; all six
  files pass the checksum check after download.
- Homebrew tap formula updated to v0.4.3 (sha256
  413328225747fc0bb2eaee7b77e2746c614b08beb42dd7e31cd6cbc995684077),
  style and audit pass, pushed. `lazyai update -y` on macOS ran brew
  upgrade from 0.4.2 to 0.4.3; brew test passes.
- Stale releases/latest seen again on the Linux host right after
  publishing (v0.4.2 reported as up to date). A minute later curl with
  Go-like headers (Go user agent, no Accept, gzip, with and without
  Cache-Control: no-cache) and lazyai itself all got v0.4.3, so request
  headers are not the cause. Untested idea: the static binary uses Go's
  own DNS resolver and may reach another GitHub edge than curl.

Open: stale latest-release answer for about a minute after a release.
To investigate on the next release: run `lazyai update --check` and
`curl -sI` on the Linux host within seconds of publishing, and compare
the IPs Go and curl connect to.

Run and verify: `make check`; `lazyai update --check`.

Next step: try lazyai on a real Linux desktop with real agents
(todo/todo.md).
- The `@` mark sits right before the kind (`@claude`), and the gutter is
  one character wide, so labels get one more cell.
- Examples in specs, tests and notes use invented directory names
  (`alpha/web-frontend`), not names from real working directories.
