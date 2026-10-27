# 2026-10-27 - launch history and completion

Details of the work are in session/2026-10-18-bootstrap.md, section
"2026-10-27 - launch history and completion".

## Publish

- Main pushed without a tag or release. gitleaks over the new commit: no
  leaks. Pushed main (200b569..4fd7656).

Run and verify: `make check`.

## 2026-10-28 - release v0.3.0

- The version comes from the tag (ldflags from git describe), so the
  bump is the tag v0.3.0: minor, since the change is a feature.
- Checks: make check, make test-linux (arm64) and make test-linux
  LINUX_ARCH=amd64 pass.
- README install examples now use v0.3.0.
- Annotated tag v0.3.0. make dist and make deb VERSION=v0.3.0 on the
  tag; the darwin-arm64 binary reports v0.3.0.
- gitleaks over the unpushed commit: no leaks. Pushed main and the tag
  v0.3.0.
- GitHub release v0.3.0 published with four tarballs, two debs and
  SHA256SUMS. Downloaded all assets back from the release: all six pass
  the checksum check.

Files: README.md, todo/todo.md, this note.

## 2026-10-28 - commit messages and notes describe the project

- ../AGENTS.md: commit messages and session notes describe the project,
  not the conversation (no who asked, chose, approved or reported).
- History rewritten from 4fd7656: the message of the "record the push of
  launch history" commit reworded (now 16f0e3a); later commits rebased
  on it, so v0.3.0 now points at 32e8e82 (was 376741b). Trees are
  unchanged, so the v0.3.0 release assets and SHA256SUMS stay valid.
  Main and the tag force-pushed.
- Older commits are untouched: the Homebrew formula pins the sha256 of
  GitHub's v0.2.1 source archive, which embeds the commit id, so moving
  v0.2.1 or earlier tags would break brew install.
- Both session notes rewritten to the new rule.

Files: session/2026-10-18-bootstrap.md, this note.

Run and verify: `make check`.

Next step: update url and sha256 in the t0d0r/homebrew-tap formula to
v0.3.0 (or let its autobump workflow open the PR), then try the launch
popup in a real terminal.
