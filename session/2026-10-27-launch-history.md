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
