# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.26

FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /src
ENV CGO_ENABLED=0 GOFLAGS=-trimpath

COPY backend/go.mod backend/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY backend/ .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /out/ledger ./cmd/ledger

FROM gcr.io/distroless/base-debian12:nonroot

WORKDIR /app
COPY --from=builder /out/ledger /app/ledger

EXPOSE 9095

ENTRYPOINT ["/app/ledger"]
