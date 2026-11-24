# Kestrel Agent

A from-scratch Go agent platform: OpenAI-compatible streaming, tool calling, ReAct runtime, hybrid RAG, state-graph workflows, observability, and evaluation.

This repository is built step by step for deep understanding, not by copying an existing project. See [the implementation plan](docs/IMPLEMENTATION_PLAN.md) for the roadmap and design decisions.

## Status

Step 0.2 — configuration, structured logging, request IDs, panic recovery, access logs, readiness, and graceful shutdown.

## Requirements

- Go 1.26+

## Quick start

```bash
go run ./cmd/server
```

In another terminal:

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

Expected response:

```json
{"service":"kestrel","status":"ok"}
```

Press `Ctrl+C` to stop the server. In-flight requests get up to
`KESTREL_HTTP_SHUTDOWN_TIMEOUT` to finish before the process exits.

## Configuration

All settings are environment variables; defaults work for local development.

| Variable | Default | Description |
|---|---|---|
| `KESTREL_ENV` | `dev` | Environment name; `prod` switches default log format to JSON |
| `KESTREL_HTTP_ADDR` | `:8080` | Listen address |
| `KESTREL_HTTP_READ_HEADER_TIMEOUT` | `5s` | Max time to read request headers |
| `KESTREL_HTTP_READ_TIMEOUT` | `15s` | Max time to read the whole request |
| `KESTREL_HTTP_WRITE_TIMEOUT` | `0` | Max response write time; `0` disables it, required for future SSE |
| `KESTREL_HTTP_IDLE_TIMEOUT` | `60s` | Keep-alive idle timeout |
| `KESTREL_HTTP_SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown budget |
| `KESTREL_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error` |
| `KESTREL_LOG_FORMAT` | `text` (`json` in prod) | `text` or `json` |

## Verify

```bash
go test ./...
go vet ./...
```

Or with Make:

```bash
make test
make vet
```

`go test -race` requires cgo. On Windows without a C toolchain, run it in Linux CI or WSL2.

## Documents

- [Learning handoff](EasyLearn.md)
- [Implementation plan](docs/IMPLEMENTATION_PLAN.md)
- [Go basics for Python developers](docs/GO-BASICS.md)
- [Step 0 walkthrough](docs/step-00.md)
- [Step 0.2 walkthrough](docs/step-00-2.md)
