# Caveo

Stateless Argon2id password hashing and verification service, in Go. Offloads CPU- and memory-intensive hashing off your application servers. No database, no persistence: each hash is a self-describing `$argon2id$v=19$m=...$salt$hash` string that carries its own parameters, so `verify` reads them back out of the hash rather than from config.

- Go 1.25+, `golang.org/x/crypto/argon2`
- Ships as a distroless, non-root, static image

## Structure

```
.
├── cmd/api/            # HTTP layer: handlers, routing, server lifecycle
├── internal/hasher/    # Argon2id core (pure, no HTTP)
├── Dockerfile
└── Makefile
```

## Security

- OWASP Argon2id defaults (19 MiB, 2 iterations, parallelism 1), baked in at build time.
- Salts from `crypto/rand`; comparison via `subtle.ConstantTimeCompare`.
- `verify` clamps the `m`/`t`/`p` parsed out of a submitted hash, so a crafted hash can't force a huge allocation.
- 1 MiB request body limit; 5s/10s read/write timeouts.
- Concurrent hashes are capped and shed with `503` past the limit (see Configuration).

## API

Two endpoints. Full spec at `GET /docs` (RapiDoc) or `GET /docs/openapi.yaml`.

`POST /hash`
```json
{ "password": "..." }  ->  { "hash": "$argon2id$v=19$m=19456,t=2,p=1$..." }
```

`POST /verify`
```json
{ "password": "...", "hash": "$argon2id$..." }  ->  { "match": true }
```

Errors are always JSON (`{ "error": "..." }`): `400` for bad input or a malformed hash, `500` for a hashing failure, `503` at capacity.

Health: `GET /healthz` (liveness), `GET /readyz` (readiness; `503` while draining on shutdown).

## Configuration

Environment variables. Invalid values fail fast at startup.

| Variable | Default | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | HTTP listen port. |
| `CAVEO_LOG_LEVEL` | `info` | Logging level (`debug`/`info`/`warn`/`error`). |
| `CAVEO_MAX_CONCURRENT_REQUESTS` | CPU count (`GOMAXPROCS`) | Cap on concurrent Argon2 operations; requests past the cap get `503` + `Retry-After`. Positive integer. |
| `CAVEO_DRAIN_DELAY` | `5s` | On shutdown, how long to keep serving after `/readyz` flips to `503`, so a load balancer can deregister before draining. Non-negative Go duration (`0s` disables). |

## Development

Requires Go 1.25+ and [`gotestsum`](https://github.com/gotestyourself/gotestsum) (`go install gotest.tools/gotestsum@latest`).

```
make run      # serve on :8080
make test     # unit + integration
make build    # -> bin/caveo
make clean
```

## Docker

Multi-stage build into `gcr.io/distroless/static-debian12:nonroot`, non-root (UID 65532), static binary with symbols stripped.

```
docker build -t caveo .
docker run -p 8080:8080 caveo
```