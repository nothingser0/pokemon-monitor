# ============================================================
# Pokemon GO Coordinate Monitor - Docker Image (Multi-Arch)
# ============================================================

# Stage 1: Build binary using native Go cross-compilation ($BUILDPLATFORM)
FROM --platform=$BUILDPLATFORM golang:alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /build

COPY go.mod ./
COPY main.go ./

# Fast multi-arch cross-compilation without QEMU overhead
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -buildvcs=false -ldflags="-s -w" -o pokemon-monitor .

# Stage 2: Minimal runtime image
FROM alpine:3.20

# ca-certificates for secure HTTPS requests (API & Discord)
# tzdata for proper timezone parsing (e.g. Asia/Jakarta)
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

RUN mkdir -p /app/data

COPY --from=builder /build/pokemon-monitor .

ENV STATE_FILE=/app/data/.state.json

ENTRYPOINT ["./pokemon-monitor"]
