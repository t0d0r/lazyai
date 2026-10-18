# Release

## Artifacts

Each release is an annotated tag `vX.Y.Z` on main and a GitHub release of
the same name with:

- `lazyai-vX.Y.Z-OS-ARCH.tar.gz` for linux/amd64, linux/arm64,
  darwin/amd64 and darwin/arm64. Each holds one directory of the same
  name with `lazyai`, `LICENSE` and `README.md`.
- `SHA256SUMS` for those tarballs, in `sha256sum` format.

`make dist` builds them into `bin/dist`. The binaries are built with
`CGO_ENABLED=0 -trimpath -ldflags "-s -w -X main.version=VERSION"`:

- static, so one Linux binary runs on any distribution and glibc (Ubuntu,
  Debian, and others);
- `-trimpath` keeps the build machine's paths out of the binary;
- tar entries are owned by root (uid 0), so the builder's user name is not
  recorded either.

Decision: a Makefile target instead of GoReleaser. Reason: four
`go build` calls need no extra tool, and the release stays reproducible
with only Go installed.

## Runtime requirements on Linux

tmux and procps (`ps`). lsof is optional. Install from the tarball:

```sh
tar -xzf lazyai-vX.Y.Z-linux-amd64.tar.gz
install -m 0755 lazyai-vX.Y.Z-linux-amd64/lazyai ~/.local/bin/
```

`uname -m` tells which tarball: `x86_64` is amd64, `aarch64` is arm64.

## Steps

1. `make check`, `make test-linux`, and `make test-linux LINUX_ARCH=amd64`
   pass (the second arch runs emulated).
2. Clean tree; `git tag -a vX.Y.Z -m "lazyai vX.Y.Z"` on main.
3. `make dist` on the tag, so VERSION is exactly `vX.Y.Z` (no `-dirty`).
4. Push main and the tag (only when the operator asks).
5. `gh release create vX.Y.Z bin/dist/*.tar.gz bin/dist/SHA256SUMS
   --title vX.Y.Z --notes ...`.
6. Homebrew: update url and sha256 in t0d0r/homebrew-tap, or let its
   autobump workflow open the PR.

## Security

Run `gitleaks git` before pushing. Users should check downloads against
`SHA256SUMS` (`sha256sum -c SHA256SUMS --ignore-missing`). The tarballs are
not signed; the checksums only protect against corrupted downloads, not
against a compromised release page.
