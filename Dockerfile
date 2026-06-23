# Multi-stage build for the astrolift-sample-go fixture (spec 42, R3).
#
#   stage 1 (build):   golang:1.22 compiles a static binary, stamping the
#                      commit + build time into main.version / main.builtAt
#                      via -ldflags from the GIT_SHA build-arg.
#   stage 2 (runtime): distroless static — no shell, no package manager,
#                      runs as nonroot. The binary is fully static (CGO off).

# ---- build ----
FROM golang:1.22 AS build
WORKDIR /src

# GIT_SHA is supplied by CI (--build-arg GIT_SHA=<sha>); defaults to "dev"
# for local `docker build` so the image is still self-describing.
ARG GIT_SHA=dev

# stdlib-only module: copy go.mod first for layer caching, then sources.
COPY go.mod ./
COPY main.go ./

# Static, stripped binary. The ldflags inject the commit and an RFC3339 build
# timestamp into the variables read by /debug (spec 42 §3.1 "build" block).
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags "-s -w -X main.version=${GIT_SHA} -X main.builtAt=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /out/sample-go .

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

# Carried into the binary's /debug "build.commit"; also exposed as an env var
# so it is visible to anything inspecting the container environment.
ARG GIT_SHA=dev
ENV GIT_SHA=${GIT_SHA}
ENV PORT=8080

COPY --from=build /out/sample-go /usr/local/bin/sample-go

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/sample-go"]
