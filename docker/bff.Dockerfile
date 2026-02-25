# =============================================================================
# Stage 1: Build — install all deps and compile TypeScript
# =============================================================================
FROM node:20-alpine AS builder

WORKDIR /app

# Copy manifest files first so npm ci can be cached independently of source
COPY bff/package.json bff/package-lock.json* ./

# Install all dependencies (including devDependencies for tsc)
RUN npm ci

# Copy source and config files
COPY bff/ .

# Compile TypeScript → JavaScript (output goes to ./dist)
RUN npm run build

# =============================================================================
# Stage 2: Production — lean image with only runtime deps
# =============================================================================
FROM node:20-alpine AS production

WORKDIR /app

# Copy only production manifests to install --omit=dev dependencies
COPY bff/package.json bff/package-lock.json* ./

RUN npm ci --omit=dev && npm cache clean --force

# Copy compiled output from builder
COPY --from=builder /app/dist ./dist

# Run as non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# BFF listens on port 3001 (configured via BFF_PORT env var)
EXPOSE 3001

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:3001/health || exit 1

CMD ["node", "dist/index.js"]
