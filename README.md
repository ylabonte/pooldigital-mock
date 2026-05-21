# pooldigital-mock

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

### Pre-built binary

Download from [Releases](https://github.com/ylabonte/pooldigital-mock/releases) (linux/macos/windows, amd64 + arm64).

### Docker

```bash
docker run --rm -p 8080:8080 -p 8180:8180 \
  ghcr.io/ylabonte/pooldigital-mock:latest
```

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

Sensors drift on slow sine waves (pH 7.30–7.50, redox 675–725 mV, CPU temp 28–32 °C, pump temp 26–28 °C).

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
# default ports
docker run --rm -p 8080:8080 -p 8180:8180 ghcr.io/ylabonte/pooldigital-mock:latest

# customise credentials
docker run --rm -p 8080:8080 -p 8180:8180 \
  -e PROCONIP_MOCK_USER=alice -e PROCONIP_MOCK_PASS=secret \
  ghcr.io/ylabonte/pooldigital-mock:latest --quiet

# build locally
make docker && docker run --rm -p 8080:8080 -p 8180:8180 pooldigital-mock:dev
```

The production image is built from `gcr.io/distroless/static:nonroot` — no shell, no package manager, ~10 MB compressed.

## Develop in a container

The repo ships a [devcontainer](https://containers.dev/) so you can clone, "Reopen in Container" in VS Code (or open in GitHub Codespaces), and have Go, golangci-lint, goreleaser, gotestsum, and delve preinstalled.

```bash
# from a host with the devcontainer CLI
devcontainer build --workspace-folder .
devcontainer up --workspace-folder .
devcontainer exec --workspace-folder . make test
```

## Development

```bash
make test    # go test -race ./...
make cover   # enforce 85% coverage floor
make lint    # golangci-lint
make build   # produces ./pooldigital-mock
make run     # go run ./cmd/pooldigital-mock
```

See [`CLAUDE.md`](./CLAUDE.md) for coding standards.

## License

MIT — see [`LICENSE`](./LICENSE).
