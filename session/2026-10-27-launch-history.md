# 2026-10-27 - launch history and completion

Details of the work are in session/2026-10-18-bootstrap.md, section
"2026-10-27 - launch history and completion".

## Publish

- Operator asked to publish and chose "push main only": no tag, no
  release.
- gitleaks over the new commit: no leaks. Pushed main
  (200b569..4fd7656).

Open: the feature is not in a release yet; v0.2.1 is the latest tag.

Run and verify: `make check`.

Next step: try the launch popup in a real terminal, then cut the next
release (specs/release.md) when the operator asks.

## 2026-10-28 - release v0.3.0

- Operator asked to push to a release, then to add a tag and bump the
  version. The version comes from the tag (ldflags from git describe),
  so the bump is the tag v0.3.0: minor, since the change is a feature.
- Checks: make check, make test-linux (arm64) and make test-linux
  LINUX_ARCH=amd64 pass.
- README install examples now use v0.3.0.

Files: README.md, todo/todo.md, this note.

- Annotated tag v0.3.0 on 376741b. make dist and make deb
  VERSION=v0.3.0 on the tag; the darwin-arm64 binary reports v0.3.0.
- gitleaks over the unpushed commit: no leaks. Pushed main
  (a4ca9a6..376741b) and the tag v0.3.0.
- GitHub release v0.3.0 published with four tarballs, two debs and
  SHA256SUMS. Downloaded all assets back from the release: all six pass
  the checksum check.

Next step: update url and sha256 in the t0d0r/homebrew-tap formula (or
let its autobump workflow open the PR), then try the launch popup in a
real terminal.
