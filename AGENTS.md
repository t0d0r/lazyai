Follow the rules in ../AGENTS.md.

Project notes:

- Go module `agentctl`, entry point `main.go`, packages under `internal/`.
- `make check` before every commit; `make build`, `make run`, `make help`
  for the rest.
- `make test` includes a tmux integration test that creates and removes an
  `agentctl-test-*` session on the default tmux server.
- Dependencies come from the local Go module cache; `make tidy` updates
  go.mod without the network.
