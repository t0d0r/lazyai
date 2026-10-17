BIN     := bin/lazyai
PREFIX  ?= $(HOME)/.local
GO      ?= go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.DEFAULT_GOAL := help

.PHONY: help build test lint fmt check run clean install uninstall tidy

help: ## list targets
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-10s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## build bin/lazyai, stamped with the git version
	$(GO) build -ldflags "-X main.version=$(VERSION)" -o $(BIN) .

test: ## run all tests (includes a tmux integration test)
	$(GO) test ./...

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
