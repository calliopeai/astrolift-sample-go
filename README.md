# astrolift-sample-go

Minimal, real Go single web service — the **language-coverage** fixture (Go)
for the Astrolift validation campaign ([spec 42](../specs/42-sample-fixtures.md),
row **R3**).

## Topology

| | |
|---|---|
| Language | Go 1.22 (standard library only) |
| Structural kind | `deployment` (single workload, public) |
| Image | `docker.io/calliopeai/astrolift-sample-go` |
| Manifest | `manifests/single.toml` |

A single stateless HTTP service. No managed services are attached — this
fixture exercises the plain Go build-and-serve path and the common debug
contract, nothing more.

## HTTP surface

| Route | Purpose |
|---|---|
| `GET /api/stars` | Sample JSON star/constellation catalogue (a nod to Astrolift). |
| `GET /health` | Readiness/liveness probe. `200` once serving. Drives the manifest `healthcheck` (`kind="http"`, `value="/health"`). |
| `GET /debug` | Diagnostic surface per spec 42 §3.1 (see below). |

### `GET /debug`

```json
{
  "app": "sample-go",
  "version": "main-abc1234",
  "kind": "deployment",
  "hostname": "sample-go-xxxxx",
  "uptime_s": 123,
  "now": "2026-06-22T00:00:00Z",
  "env_seen": ["GIT_SHA", "LOG_LEVEL", "PORT"],
  "bindings": {},
  "build": {
    "image": "docker.io/calliopeai/astrolift-sample-go",
    "built_at": "2026-06-22T00:00:00Z",
    "commit": "main-abc1234"
  }
}
```

- `env_seen` is an allow-list of non-secret env keys that are present — never
  a dump of the environment, so platform-injected secrets are not echoed.
- `bindings` is `{}`: this fixture attaches no managed services.
- `version` / `build.commit` come from the `GIT_SHA` stamped at build time
  (`-ldflags -X main.version`), falling back to the `GIT_SHA` env var, then
  `dev` for local runs.

A structured startup banner is logged at boot (app, version, kind, hostname,
listen address, bindings).

## Configuration

| Env var | Default | Meaning |
|---|---|---|
| `PORT` | `8080` | Listen port. |
| `GIT_SHA` | `dev` | Build commit (normally stamped at build time). |
| `LOG_LEVEL` | — | Reported in `env_seen` if set; not otherwise consumed. |

## Run locally

```sh
go run .
# in another shell:
curl -s localhost:8080/api/stars | head
curl -s localhost:8080/health
curl -s localhost:8080/debug
```

## Build the image

```sh
docker build --build-arg GIT_SHA=$(git rev-parse --short HEAD) \
  -t docker.io/calliopeai/astrolift-sample-go:dev .
docker run --rm -p 8080:8080 docker.io/calliopeai/astrolift-sample-go:dev
```

CI (`.github/workflows/build.yml`) builds on push to `main` and pushes
`main-<sha7>` + `latest` to Docker Hub using the org secrets
`DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN`.

## Variant → matrix map

| Manifest | Matrix cell (spec 42 §6) |
|---|---|
| `manifests/single.toml` | `deployment (single)` × `go` |
