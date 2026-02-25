# =============================================================================
# Stage 1: Build
# =============================================================================
FROM golang:1.22-alpine AS builder

# Build argument — passed via docker compose build-arg or --build-arg
ARG SERVICE_NAME
ENV SERVICE_NAME=${SERVICE_NAME}

# Install certificates for HTTPS calls during build (e.g. go mod download)
RUN apk add --no-cache ca-certificates git

WORKDIR /workspace

# Copy shared pkg module first so it can be resolved from the workspace
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/${SERVICE_NAME}/go.mod services/${SERVICE_NAME}/go.sum ./services/${SERVICE_NAME}/

# Download dependencies before copying source (improves layer caching)
WORKDIR /workspace/services/${SERVICE_NAME}
RUN go mod download

# Copy full source tree
WORKDIR /workspace
COPY pkg/ ./pkg/
COPY services/${SERVICE_NAME}/ ./services/${SERVICE_NAME}/

# Build the binary — static, no CGO so the distroless image works
WORKDIR /workspace/services/${SERVICE_NAME}
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

# Lightweight healthcheck — relies on the /health endpoint every service
# exposes via pkg/health. Adjust the port if your service uses a non-default.
HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/app/service", "-healthcheck"]

ENTRYPOINT ["/app/service"]
