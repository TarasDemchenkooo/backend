# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26.4
ARG MIGRATE_VERSION=v4.18.3

FROM golang:${GO_VERSION}-alpine AS build

ENV CGO_ENABLED=0 GOWORK=off GOFLAGS=-trimpath

WORKDIR /src
COPY pkg/observability/go.mod pkg/observability/go.sum pkg/observability/
COPY services/auth/go.mod services/auth/go.sum services/auth/

WORKDIR /src/services/auth
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /src
COPY pkg/observability pkg/observability
COPY services/auth services/auth

WORKDIR /src/services/auth
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -o /out/auth ./cmd/server

FROM migrate/migrate:${MIGRATE_VERSION} AS migrations

COPY services/auth/migrations /migrations
USER 65532:65532

FROM gcr.io/distroless/static-debian12:nonroot AS app

COPY --from=build /out/auth /auth
USER nonroot:nonroot
EXPOSE 8080 9090
ENTRYPOINT ["/auth"]
