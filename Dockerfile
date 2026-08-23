# syntax=docker/dockerfile:1.7
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS backend
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -o /out/dust-platform ./cmd/server

FROM alpine:3.21
RUN addgroup -S app && adduser -S -G app -u 10001 app && mkdir -p /var/lib/dust/files && chown -R app:app /var/lib/dust
COPY --from=backend /out/dust-platform /usr/local/bin/dust-platform
USER app
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=6 \
    CMD wget -q -O - http://127.0.0.1:8080/readyz || exit 1
ENTRYPOINT ["/usr/local/bin/dust-platform"]
