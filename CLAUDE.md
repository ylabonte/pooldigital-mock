# pooldigital-mock — coding standards

This file is the working agreement for changes in this repo. Claude (and any
contributor) follows it.

## What this is

A single Go binary that runs two HTTP mock servers concurrently:

- **proconip** on port 8080 — mimics a ProCon.IP pool controller
- **violet** on port 8180 — mimics a Pooldigital Violet controller

Both expose enough of the real device's wire format that the corresponding
client libraries can run integration tests against this binary unchanged.
State is in-memory only; nothing is persisted across process restarts.

## Stack

- Go 1.22+
- stdlib `net/http`
- `//go:embed` for the violet seed JSON and proconip CSV fixtures
- `github.com/charmbracelet/lipgloss` — banner box and colored output
- `github.com/spf13/pflag` — POSIX-style flags

No other third-party runtime deps. Add new ones only with strong justification.

## Architecture rules

- `internal/proconip` and `internal/violet` are **independent** — neither
  imports the other. They share only `internal/logx`.
- Each server package exposes `NewHandler(cfg ServerConfig) http.Handler`.
  No package-level globals for state.
- State is guarded by a `sync.RWMutex`. Never return a map or slice by
  reference; always return a copy.
- Wire format is contract. Any change to the JSON returned by violet or the
  CSV emitted by proconip must update fixtures and be called out in the PR
  description.
- The CLI lives in `cmd/pooldigital-mock`. `main.go` is a thin wrapper that
  calls `run(ctx, args, stdout, stderr)` so the lifecycle is testable
  without spawning a subprocess.

## Testing rules

- Every endpoint ships table-driven tests covering at least:
  happy path, auth failure (where applicable), malformed input.
- Use `httptest.NewServer` + the real HTTP client — exercises middleware
  end-to-end, not just the handler.
- Drift/render tests pin exact bytes. If a test breaks, the fixture has
  changed OR the wire contract has changed. Think before regenerating.
- Always run `go test -race ./...` before declaring work done.
- Coverage floor is **85% overall**, enforced in CI. Local check:
  `make cover`.

## Style rules

- `gofmt -s` + `goimports`, no exceptions.
- Errors wrap with `%w`; check with `errors.Is` / `errors.As`.
- No `panic` in request handlers — return an HTTP error status.
- No stdlib `log` — use `internal/logx`.
- Exported identifiers documented; unexported only when non-obvious.
- One short comment line max where intent isn't obvious; never multi-paragraph
  docstrings.

## Commit rules

- Conventional commits: `feat:`, `fix:`, `test:`, `chore:`, `ci:`, `docs:`.
- One logical change per PR; every PR green on CI before merge.
- Never `--no-verify` or `--no-gpg-sign`.

## When working with Claude

- New feature → `superpowers:test-driven-development`.
- Bug → `superpowers:systematic-debugging`.
- Before claiming "done" → `superpowers:verification-before-completion`.

## Local verification

```bash
make test    # go test -race ./...
make cover   # enforce 85% floor
make lint    # golangci-lint
make build   # static binary
make docker  # production image
```
