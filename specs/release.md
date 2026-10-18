# Release

## Artifacts

Each release is an annotated tag `vX.Y.Z` on main and a GitHub release of
the same name with:

- `lazyai-vX.Y.Z-OS-ARCH.tar.gz` for linux/amd64, linux/arm64,
  darwin/amd64 and darwin/arm64. Each holds one directory of the same
  name with `lazyai`, `LICENSE` and `README.md`.
- `lazyai_X.Y.Z_ARCH.deb` for amd64 and arm64 (see Debian packages).
- `SHA256SUMS` for the tarballs and packages, in `sha256sum` format.

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

## Debian packages

`make deb` (after `make dist`, same VERSION) wraps the binary from each
linux tarball, byte for byte, in a .deb built by `dpkg-deb
--root-owner-group` in a `debian:latest` container (`DEB_IMAGE`), so the
Mac needs no Debian tools. It then rewrites `SHA256SUMS` over the
tarballs and packages; the tarball lines do not change.

- Package `lazyai`, version VERSION without the `v` (a build between
  tags becomes `0.2.0+3.gHASH`), Architecture amd64 or arm64.
- Depends: `tmux, procps`; Suggests: `lsof`. No libc dependency, since
  the binary is static.
- Files: `/usr/bin/lazyai`, `/usr/share/doc/lazyai/{copyright,
  changelog.gz}` and a lintian override for `statically-linked-binary`.
- Maintainer: `DEB_MAINTAINER`, by default git's user.name and
  user.email, the same identity as in the commits.

Decision: one package per architecture, not one "all" package. Reason:
dpkg installs a package only on its architecture, and `all` is for
architecture-independent files; a fat package choosing a binary at
install time would bypass that. The static binary is what makes each
package install on any Debian or Ubuntu release.

Decision: no APT repository for now. Reason: a repository needs a
signing key kept safe and a place to host it; the .deb from the release
page is installed with `sudo apt install ./FILE.deb`, which still
resolves the dependencies.

Checked for v0.2.0: installs, runs and removes on Ubuntu 24.04 and
20.04, Debian 13 and 11, on amd64 and arm64; lintian reports only
`no-manual-page`.

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
3. `make dist` on the tag, so VERSION is exactly `vX.Y.Z` (no `-dirty`),
   then `make deb VERSION=vX.Y.Z`.
4. Push main and the tag (only when the operator asks).
5. `gh release create vX.Y.Z bin/dist/*.tar.gz bin/dist/*.deb bin/dist/SHA256SUMS
   --title vX.Y.Z --notes ...`.
6. Homebrew: update url and sha256 in t0d0r/homebrew-tap, or let its
   autobump workflow open the PR.

## Security

Run `gitleaks git` before pushing. Users should check downloads against
`SHA256SUMS` (`sha256sum -c SHA256SUMS --ignore-missing`). The tarballs are
not signed; the checksums only protect against corrupted downloads, not
against a compromised release page.
