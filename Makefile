BIN     := bin/lazyai
PREFIX  ?= $(HOME)/.local
GO      ?= go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.DEFAULT_GOAL := help

LINUX_IMAGES ?= ubuntu:24.04 debian:latest
# LINUX_ARCH defaults to docker's own; amd64 on an arm64 Mac runs emulated.
LINUX_ARCH ?=
DIST_PLATFORMS ?= linux/amd64 linux/arm64 darwin/amd64 darwin/arm64
DEB_ARCHES ?= amd64 arm64
DEB_IMAGE ?= debian:latest
DEB_MAINTAINER ?= $(shell git config user.name) <$(shell git config user.email)>

.PHONY: help build test test-linux dist deb lint fmt check run clean install uninstall tidy

help: ## list targets
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-10s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## build bin/lazyai, stamped with the git version
	$(GO) build -ldflags "-X main.version=$(VERSION)" -o $(BIN) .

test: ## run all tests (includes a tmux integration test)
	$(GO) test ./...

test-linux: ## run the tests and an end-to-end check in Linux containers (needs docker)
	@command -v docker >/dev/null || { echo "test-linux needs docker"; exit 1; }
	@arch=$$(docker version --format '{{.Server.Arch}}') || { echo "test-linux: docker is not running"; exit 1; }; \
	[ -z "$(LINUX_ARCH)" ] || arch=$(LINUX_ARCH); \
	set -e; rm -rf bin/linux; mkdir -p bin/linux; \
	for pkg in $$($(GO) list ./...); do \
		GOOS=linux GOARCH=$$arch CGO_ENABLED=0 $(GO) test -c -o bin/linux/$$(basename $$pkg).test $$pkg; \
	done; \
	GOOS=linux GOARCH=$$arch CGO_ENABLED=0 $(GO) build -ldflags "-X main.version=$(VERSION)" -o bin/linux/lazyai .; \
	for img in $(LINUX_IMAGES); do \
		tag=lazyai-test:$$(echo $$img | tr ':/' '--')-$$arch; \
		echo "== $$img $$arch"; \
		docker build -q --platform linux/$$arch -t $$tag --build-arg BASE=$$img test/linux >/dev/null; \
		docker run --rm --platform linux/$$arch -v $(CURDIR)/bin/linux:/t:ro -v $(CURDIR)/test/linux/e2e.sh:/e2e.sh:ro $$tag /e2e.sh; \
	done

dist: ## build release tarballs and SHA256SUMS into bin/dist
	@set -e; rm -rf bin/dist; mkdir -p bin/dist; \
	case "$$(tar --version)" in \
		*bsdtar*) own="--uid 0 --gid 0 --uname root --gname root";; \
		*) own="--owner=0 --group=0 --numeric-owner";; \
	esac; \
	for p in $(DIST_PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; name=lazyai-$(VERSION)-$$os-$$arch; \
		mkdir bin/dist/$$name; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 $(GO) build -trimpath \
			-ldflags "-s -w -X main.version=$(VERSION)" -o bin/dist/$$name/lazyai .; \
		cp LICENSE README.md bin/dist/$$name/; \
		COPYFILE_DISABLE=1 tar $$own -czf bin/dist/$$name.tar.gz -C bin/dist $$name; \
		rm -rf bin/dist/$$name; \
	done; \
	cd bin/dist; \
	if command -v sha256sum >/dev/null; then sha256sum *.tar.gz; else shasum -a 256 *.tar.gz; fi > SHA256SUMS; \
	cat SHA256SUMS

deb: ## package the linux tarballs in bin/dist as .deb files (needs docker; run dist first)
	@command -v docker >/dev/null || { echo "deb needs docker for dpkg-deb"; exit 1; }
	@set -e; ver=$$(echo $(VERSION) | sed 's/^v//; s/-/+/; s/-/./g'); \
	for arch in $(DEB_ARCHES); do \
		tgz=bin/dist/lazyai-$(VERSION)-linux-$$arch.tar.gz; \
		[ -f $$tgz ] || { echo "deb: $$tgz missing; run make dist"; exit 1; }; \
		root=bin/dist/deb-$$arch; rm -rf $$root; \
		mkdir -p $$root/DEBIAN $$root/usr/bin $$root/usr/share/doc/lazyai $$root/usr/share/lintian/overrides; \
		echo "lazyai: statically-linked-binary [usr/bin/lazyai]" > $$root/usr/share/lintian/overrides/lazyai; \
		tar -xzf $$tgz -O lazyai-$(VERSION)-linux-$$arch/lazyai > $$root/usr/bin/lazyai; \
		chmod 0755 $$root/usr/bin/lazyai; \
		{ echo "Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/"; \
		  echo "Upstream-Name: lazyai"; echo "Source: https://github.com/t0d0r/lazyai"; echo; \
		  echo "Files: *"; echo "License: BSD-2-Clause"; echo; \
		  sed 's/^$$/./; s/^/ /' LICENSE; } > $$root/usr/share/doc/lazyai/copyright; \
		printf 'lazyai (%s) unstable; urgency=medium\n\n  * Release %s, see https://github.com/t0d0r/lazyai/releases\n\n -- %s  %s\n' \
			"$$ver" "$(VERSION)" "$(DEB_MAINTAINER)" "$$(LC_ALL=C date -R 2>/dev/null || LC_ALL=C date '+%a, %d %b %Y %H:%M:%S %z')" \
			| gzip -9n > $$root/usr/share/doc/lazyai/changelog.gz; \
		chmod 0644 $$root/usr/share/doc/lazyai/* $$root/usr/share/lintian/overrides/lazyai; \
		{ echo "Package: lazyai"; echo "Version: $$ver"; echo "Architecture: $$arch"; \
		  echo "Maintainer: $(DEB_MAINTAINER)"; \
		  echo "Installed-Size: $$(du -sk $$root/usr | cut -f1)"; \
		  echo "Depends: tmux, procps"; echo "Suggests: lsof"; \
		  echo "Section: utils"; echo "Priority: optional"; \
		  echo "Homepage: https://github.com/t0d0r/lazyai"; \
		  echo "Description: dashboard to see and control local AI coding agents"; \
		  echo " lazyai lists Claude Code, Codex, opencode, Gemini CLI and other agents"; \
		  echo " running for the current user, also inside docker containers, and can"; \
		  echo " launch them in tmux, attach, send a prompt, and stop them."; \
		  echo " The binary is static and has no library dependencies."; } > $$root/DEBIAN/control; \
		docker run --rm --user $$(id -u):$$(id -g) -v $(CURDIR)/bin/dist:/dist $(DEB_IMAGE) \
			dpkg-deb --root-owner-group -Zxz --build /dist/deb-$$arch /dist/lazyai_$${ver}_$$arch.deb; \
		rm -rf $$root; \
	done; \
	cd bin/dist; \
	if command -v sha256sum >/dev/null; then sha256sum *.tar.gz *.deb; else shasum -a 256 *.tar.gz *.deb; fi > SHA256SUMS; \
	cat SHA256SUMS

lint: ## check gofmt and run go vet
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi
	$(GO) vet ./...

fmt: ## format all Go files
	gofmt -w .

check: lint test ## lint and test; must pass before a commit

run: build ## build and start the dashboard
	./$(BIN)

install: build ## copy the binary to $(PREFIX)/bin
	install -d $(PREFIX)/bin
	install -m 0755 $(BIN) $(PREFIX)/bin/lazyai

uninstall: ## remove the binary from $(PREFIX)/bin
	rm -f $(PREFIX)/bin/lazyai

tidy: ## tidy go.mod from the local module cache only
	GOPROXY=off $(GO) mod tidy -e

clean: ## remove build output
	rm -rf bin
