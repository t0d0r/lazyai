# Update

`lazyai update` replaces the running lazyai with the latest GitHub
release, in the way it was installed.

```
lazyai update [--check] [--yes]
```

- `--check`, `-c`: report the installed and the latest version, change
  nothing.
- `--yes`, `-y`: do not ask for confirmation.

## Latest release

The latest release is the tag that
`https://github.com/t0d0r/lazyai/releases/latest` redirects to
(`.../releases/tag/vX.Y.Z`). lazyai reads the `Location` header and does
not follow it.

Right after a release is published, GitHub can still answer some
clients with the previous tag for a minute or two (seen for v0.4.2: Go's
client got v0.4.1 twice while curl already got v0.4.2). `update` then
says the installed version is up to date; running it again a little
later finds the new release.

Decision: the redirect, not the GitHub API. Reason: no JSON and no API
rate limit for unauthenticated users; the release page is what a person
would open.

## Versions

A release version is `vX.Y.Z` or `X.Y.Z` (Homebrew builds without the
`v`). Versions compare by number: major, then minor, then patch.

- Installed version equal to or newer than the latest: print
  `lazyai VERSION is up to date` and exit 0.
- Installed version is not a release version (`dev`, or `git describe`
  output such as `v0.3.0-2-gabc1234` or `v0.3.0-dirty`): this is a build
  from source. `--check` reports the latest version; `update` refuses
  and says to update the source (`git pull`, `make install`). Exit 1.
- Otherwise print `update available: OLD -> NEW`. With `--check`, exit 0
  here. Without it, ask `Update lazyai OLD -> NEW (METHOD)? [y/N]` unless
  `--yes`; an answer other than `y` changes nothing and exits 0.

## Install methods

The method follows from the path of the running executable after
resolving symlinks.

| method   | detected by                                        | update                                         |
|----------|----------------------------------------------------|------------------------------------------------|
| homebrew | path contains `/Cellar/lazyai/`                    | run `brew upgrade t0d0r/tap/lazyai`            |
| deb      | Linux, and `dpkg-query -S PATH` succeeds           | download and check the .deb, `apt install` it  |
| binary   | anything else                                      | download and check the tarball, replace PATH   |

- homebrew: brew builds from the tap formula, so a release the tap has
  not picked up yet is reported by brew as up to date.
- deb: downloads `lazyai_X.Y.Z_ARCH.deb` and `SHA256SUMS` into a new
  temporary directory (mode 0755, file 0644, so apt's sandbox user can
  read it), checks the sum, then runs `sudo apt-get install -y FILE` with
  the file's absolute path (without sudo when already root), with the
  terminal attached so sudo can ask for the password. The directory is
  removed afterwards.
- binary: downloads `lazyai-vX.Y.Z-OS-ARCH.tar.gz` and `SHA256SUMS`,
  checks the sum, takes the regular file `lazyai-vX.Y.Z-OS-ARCH/lazyai`
  from the tarball, writes it to a temporary file next to PATH with mode
  0755 and renames it over PATH. A running lazyai keeps the old file
  open, so the replacement is safe. When the directory is not writable,
  the update fails and names the directory (rerun with sudo, or install
  the release by hand).

OS and ARCH are the ones lazyai was built for (`runtime.GOOS`,
`runtime.GOARCH`). A platform without a release asset fails with a
message.

Done: `updated lazyai OLD -> NEW (PATH)` for the binary method; brew and
apt print their own output.

## Limits

- Downloads start at github.com over HTTPS and follow GitHub's redirect
  to its asset host; a redirect to anything other than HTTPS is refused.
  Each request has a 2 minute timeout and each file at most 64 MiB.
- A file missing from `SHA256SUMS` or with a different sum stops the
  update before anything is installed.
- HTTPS needs the system's CA certificates (`ca-certificates` on Debian
  and Ubuntu, present on normal installs; minimal container images lack
  it).

## Decisions

- No automatic check. lazyai uses the network only when `update` runs,
  so running the dashboard never contacts a server. Reason: lazyai
  inspects local processes and should not announce itself.
- The TUI has no update action. Replacing the binary under a running
  dashboard is confusing; `lazyai update` is run from a shell.
- A source build is never replaced: it may have local changes, and its
  owner updates it with git.

## Security

`SHA256SUMS` comes from the same release as the files, so the check
catches corrupted or truncated downloads, not a compromised release page.
TLS to github.com is what authenticates the download. Release assets are
not signed (see release.md).

## Acceptance

- Unit tests: version parsing and comparison, the latest-release
  redirect, checksum mismatch and missing entry, tarball extraction (only
  the expected regular file), replacing a file in a temporary directory,
  method detection by path.
- By hand: a binary built as an older release version updates itself to
  the latest release; `--check` on a current version says it is up to
  date; a dev build refuses.

Checked against the v0.3.0 release: binary method on macOS (also through
a symlink) and on Debian 13 as a normal user in ~/.local/bin; a read-only
/usr/local/bin fails with the message above; deb method from the 0.2.1
package as root and as a sudo user; homebrew detected for a Cellar path
reached through a symlink.

Checked from the published v0.4.0 to v0.4.1: deb as a sudo user and
binary in ~/.local/bin on Debian 13 (arm64 and amd64), and homebrew on
macOS (`brew upgrade` built and installed 0.4.1).
