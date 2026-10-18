BIN     := bin/lazyai
PREFIX  ?= $(HOME)/.local
GO      ?= go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.DEFAULT_GOAL := help

LINUX_IMAGES ?= ubuntu:24.04 debian:latest

.PHONY: help build test test-linux lint fmt check run clean install uninstall tidy

help: ## list targets
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-10s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## build bin/lazyai, stamped with the git version
	$(GO) build -ldflags "-X main.version=$(VERSION)" -o $(BIN) .

test: ## run all tests (includes a tmux integration test)
	$(GO) test ./...

test-linux: ## run the tests and an end-to-end check in Linux containers (needs docker)
	@command -v docker >/dev/null || { echo "test-linux needs docker"; exit 1; }
	@arch=$$(docker version --format '{{.Server.Arch}}') || { echo "test-linux: docker is not running"; exit 1; }; \
	set -e; rm -rf bin/linux; mkdir -p bin/linux; \
	for pkg in $$($(GO) list ./...); do \
		GOOS=linux GOARCH=$$arch CGO_ENABLED=0 $(GO) test -c -o bin/linux/$$(basename $$pkg).test $$pkg; \
	done; \
	GOOS=linux GOARCH=$$arch CGO_ENABLED=0 $(GO) build -ldflags "-X main.version=$(VERSION)" -o bin/linux/lazyai .; \
	for img in $(LINUX_IMAGES); do \
		tag=lazyai-test:$$(echo $$img | tr ':/' '--'); \
		docker build -q -t $$tag --build-arg BASE=$$img test/linux >/dev/null; \
		docker run --rm -v $(CURDIR)/bin/linux:/t:ro -v $(CURDIR)/test/linux/e2e.sh:/e2e.sh:ro $$tag /e2e.sh; \
	done

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
