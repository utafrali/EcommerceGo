# =============================================================================
# Stage 1: Build
# =============================================================================
FROM golang:1.23-alpine AS builder

# Build argument — passed via docker compose build-arg or --build-arg
ARG SERVICE_NAME
ARG SERVICE_PATH=${SERVICE_NAME}
ENV SERVICE_NAME=${SERVICE_NAME}
ENV SERVICE_PATH=${SERVICE_PATH}

# Disable Go workspace mode — each service builds independently using its own
# go.mod replace directive to resolve the shared pkg module.
ENV GOWORK=off

# Install certificates for HTTPS calls during build (e.g. go mod download)
RUN apk add --no-cache ca-certificates git

WORKDIR /workspace

# Copy shared pkg module first so it can be resolved via replace directive
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/${SERVICE_PATH}/go.mod services/${SERVICE_PATH}/go.sum ./services/${SERVICE_PATH}/

# Download dependencies before copying source (improves layer caching)
WORKDIR /workspace/services/${SERVICE_PATH}
RUN go mod download

# Copy full source tree
WORKDIR /workspace
COPY pkg/ ./pkg/
COPY services/${SERVICE_PATH}/ ./services/${SERVICE_PATH}/

# Build the binary — static, no CGO so the distroless image works
WORKDIR /workspace/services/${SERVICE_PATH}
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -ldflags="-w -s" \
      -o /out/${SERVICE_NAME} \
      ./cmd/server

# =============================================================================
# Stage 2: Run (distroless — minimal attack surface, no shell)
# =============================================================================
FROM gcr.io/distroless/static-debian12

ARG SERVICE_NAME
ENV SERVICE_NAME=${SERVICE_NAME}

# Run as non-root user (distroless ships uid 65532 "nonroot" by default)
USER nonroot:nonroot

# Copy the compiled binary from the builder stage
COPY --from=builder --chown=nonroot:nonroot /out/${SERVICE_NAME} /app/service

# Expose both HTTP and gRPC ports (the actual port numbers are configured via
# env vars at runtime — these are just documentation / docker metadata)
EXPOSE 8080 9090

# Healthcheck disabled at Dockerfile level — distroless has no shell/curl.
# Use docker-compose healthcheck with `wget` from a sidecar, or rely on
# orchestrator-level probes (Kubernetes liveness/readiness).
# HEALTHCHECK NONE

ENTRYPOINT ["/app/service"]
