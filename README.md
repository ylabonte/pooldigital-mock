# pooldigital-mock

[![CI](https://github.com/ylabonte/pooldigital-mock/actions/workflows/ci.yml/badge.svg)](https://github.com/ylabonte/pooldigital-mock/actions/workflows/ci.yml)
[![Release](https://github.com/ylabonte/pooldigital-mock/actions/workflows/release.yml/badge.svg)](https://github.com/ylabonte/pooldigital-mock/actions/workflows/release.yml)
[![Latest release](https://img.shields.io/github/v/release/ylabonte/pooldigital-mock?display_name=tag&sort=semver)](https://github.com/ylabonte/pooldigital-mock/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/ylabonte/pooldigital-mock.svg)](https://pkg.go.dev/github.com/ylabonte/pooldigital-mock)
[![Go Report Card](https://goreportcard.com/badge/github.com/ylabonte/pooldigital-mock)](https://goreportcard.com/report/github.com/ylabonte/pooldigital-mock)
[![Go version](https://img.shields.io/github/go-mod/go-version/ylabonte/pooldigital-mock)](go.mod)
[![License: MIT](https://img.shields.io/github/license/ylabonte/pooldigital-mock)](LICENSE)

One lightweight Go binary that runs two pool-controller mocks concurrently:

- **proconip** on port `8080` — mimics a [ProCon.IP](https://pooldigital.de) pool controller (CSV + basic auth)
- **violet** on port `8180` — mimics a [Pooldigital Violet](https://pooldigital.de) controller (JSON, partial auth)

Wire-compatible with the real devices for the endpoints it ships, so the matching client libraries (e.g. Home Assistant integrations) can run integration tests against this binary unchanged.

## Why

Both vendors ship Python mocks inside their respective client repos. This project bundles them together as a single static binary so you can:

- Run **both** mocks side-by-side for integration tests that talk to both controllers.
- Distribute one self-contained binary or container image — no Python runtime needed.
- See every request in a single colorful log stream while you debug a client.

## Install

### Docker

The fastest path — one command, no toolchain required:

```bash
docker run --rm -p 8080:8080 -p 8180:8180 labonte/pooldigital-mock
```

The same multi-arch image is mirrored on GitHub Container Registry if you'd rather pull from there:

```bash
docker run --rm -p 8080:8080 -p 8180:8180 ghcr.io/ylabonte/pooldigital-mock:latest
```

Both registries publish `latest` plus a tag per release (e.g. `:0.1.0` — goreleaser strips the leading `v` from image tags by convention; the corresponding git tag is `v0.1.0`); they're built from the same goreleaser run, so the digests match.

### Pre-built binary

Download from [Releases](https://github.com/ylabonte/pooldigital-mock/releases) — `linux`, `macOS`, `windows` × `amd64`, `arm64` (no `windows/arm64`).

### From source

```bash
go install github.com/ylabonte/pooldigital-mock/cmd/pooldigital-mock@latest
```

## Quickstart

```bash
pooldigital-mock
```

You'll see a startup banner and a stream of colored request lines, one per HTTP hit:

```
┌─ pooldigital-mock ───────────────┐
│ proconip   http://localhost:8080 │
│ violet     http://localhost:8180 │
└──────────────────────────────────┘

12:04:17.812  proconip  192.168.1.42    → GET    /GetState.csv     200  3ms
12:04:17.945  violet    192.168.1.42    → GET    /getReadings?ALL  200  2ms
12:04:18.012  proconip  192.168.1.42    → POST   /usrcfg.cgi       401  1ms
12:04:18.110  violet    127.0.0.1       → GET    /getConfig        200  1ms
```

Hit either endpoint:

```bash
curl -u admin:admin http://localhost:8080/GetState.csv
curl              http://localhost:8180/getReadings?ALL
```

## Flags

| Flag | Default | Env var | Purpose |
|------|---------|---------|---------|
| `--host` | `0.0.0.0` | `HOST` | bind address |
| `--proconip-port` | `8080` | `PROCONIP_MOCK_PORT` | proconip port |
| `--violet-port` | `8180` | `MYVIOLET_MOCK_PORT` | violet port |
| `--proconip-user` | `admin` | `PROCONIP_MOCK_USER` | proconip basic-auth user |
| `--proconip-pass` | `admin` | `PROCONIP_MOCK_PASS` | proconip basic-auth pass |
| `--violet-user` | `admin` | `MYVIOLET_MOCK_USER` | violet basic-auth user |
| `--violet-pass` | `admin` | `MYVIOLET_MOCK_PASS` | violet basic-auth pass |
| `--no-color` | off | `NO_COLOR` | disable ANSI colors |
| `--quiet` | off | — | suppress per-request logs |
| `--version` | — | — | print version and exit |

`Ctrl-C` triggers a graceful shutdown (both servers drain in-flight requests up to 5s).

## What gets mocked

### proconip (`:8080`)

| Method + path | Auth | What it does |
|---------------|------|--------------|
| `GET /GetState.csv` | basic | drifted sensor readings + relay state, CSV body |
| `GET /GetDmx.csv` | basic | current 16-channel DMX state |
| `POST /usrcfg.cgi` | basic | accepts `ENA=...` relay or `CH1_8=...&CH9_16=...` DMX writes |
| `GET /Command.htm?MAN_DOSAGE=t,s` | basic | manual dosage trigger |

Sensors drift on slow sine waves (pH 7.26–7.36, redox 772–802 mV, CPU temp 46–50 °C, pool water 29–31 °C) — aligned with the Violet seed so test data is consistent across both mocks.

### violet (`:8180`)

| Method + path | Auth | What it does |
|---------------|------|--------------|
| `GET /getReadings?ALL` or `?key,key…` | **none** | readings snapshot (drifted) |
| `GET /getConfig` | basic | controller config |
| `GET /setFunctionManually?KEY,ACTION,DURATION,VALUE` | basic | toggle a relay / DMX scene / cover |
| `GET /setTargetValues?target=&value=` | basic | set a target value |
| `POST /setConfig` | basic | merge JSON into config |
| `POST /setDosingParameters` | basic | merge JSON into dosing parameters |

Drift applies to `pH_value`, `orp_value`, `pot_value`, `CPU_TEMP`. Seed snapshot is captured from the public demo controller at <https://demo.myviolet.de/> and embedded in the binary.

## Run via Docker

```bash
# default ports, Docker Hub
docker run --rm -p 8080:8080 -p 8180:8180 labonte/pooldigital-mock

# pin to a release tag (recommended for CI)
docker run --rm -p 8080:8080 -p 8180:8180 labonte/pooldigital-mock:0.1.0

# same image, from GHCR
docker run --rm -p 8080:8080 -p 8180:8180 ghcr.io/ylabonte/pooldigital-mock:latest

# customise credentials + quiet output
docker run --rm -p 8080:8080 -p 8180:8180 \
  -e PROCONIP_MOCK_USER=alice -e PROCONIP_MOCK_PASS=secret \
  labonte/pooldigital-mock --quiet

# build locally
make docker && docker run --rm -p 8080:8080 -p 8180:8180 pooldigital-mock:dev
```

The production image is built from `gcr.io/distroless/static:nonroot` — no shell, no package manager, ~16 MB on disk and ~7 MB compressed at the registry. Multi-arch manifests cover `linux/amd64` and `linux/arm64`; docker picks the right one automatically.

## Develop in a container

The repo ships a [devcontainer](https://containers.dev/) so you can clone, "Reopen in Container" in VS Code (or open in [GitHub Codespaces](https://github.com/codespaces/new?repo=ylabonte/pooldigital-mock)), and have Go, `golangci-lint`, `goreleaser`, `gotestsum`, `delve`, and `goimports` preinstalled. The container forwards ports `8080` and `8180` automatically.

```bash
# from a host with the devcontainer CLI
devcontainer build --workspace-folder .
devcontainer up --workspace-folder .
devcontainer exec --workspace-folder . make test
```

A `.vscode/` set is checked in too: `launch.json` has seven run/debug configurations (default + custom ports + quiet, plus test-by-name and attach-by-pid), `tasks.json` wires the Make targets to `Cmd/Ctrl+Shift+B`, and `settings.json` enforces `goimports` + format-on-save + golangci-lint workspace integration.

## Development

```bash
make test         # go test -race ./...
make cover        # race + coverage, fail if total < 85%
make cover-html   # render coverage.html
make lint         # golangci-lint
make vet          # go vet ./...
make fmt          # gofmt -s -w .
make build        # static binary → ./pooldigital-mock
make run          # go run ./cmd/pooldigital-mock
make docker       # build the production OCI image (tag pooldigital-mock:dev)
make docker-run   # build + run the image with both ports forwarded
make help         # list every target with its one-line summary
```

See [`CLAUDE.md`](./CLAUDE.md) for coding standards (architecture rules, testing rules, commit style, when to invoke each Claude superpower).

## License

MIT — see [`LICENSE`](./LICENSE).
