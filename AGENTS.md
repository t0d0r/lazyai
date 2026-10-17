Follow the rules in ../AGENTS.md.

Project notes:

- Go module `agentctl`, entry point `main.go`, packages under `internal/`.
- Build: `go build -o bin/agentctl .`  Test: `go test ./...`
- Dependencies come from the local Go module cache; run `go mod tidy -e`
  with `GOPROXY=off` if the network is unavailable.
