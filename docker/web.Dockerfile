# =============================================================================
# Stage 1: Dependencies — separate layer so it is cached independently
# =============================================================================
FROM node:20-alpine AS deps

WORKDIR /app

# Copy manifests only so this layer is rebuilt only when deps change
COPY web/package.json web/package-lock.json* ./

RUN npm ci

# =============================================================================
# Stage 2: Build — compile Next.js in standalone mode
# =============================================================================
FROM node:20-alpine AS builder

WORKDIR /app

# Bring in node_modules from the deps stage
COPY --from=deps /app/node_modules ./node_modules

# Copy full Next.js source
COPY web/ .

# next.config.js must set `output: 'standalone'` for this Dockerfile to work.
# The standalone build bundles only the files needed to run in production.
ENV NEXT_TELEMETRY_DISABLED=1
ENV NODE_ENV=production

RUN npm run build

# =============================================================================
# Stage 3: Production runner — minimal footprint
# =============================================================================
FROM node:20-alpine AS production

WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# Run as non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# The standalone output contains a self-contained server.js with its own
# node_modules copy, so we only need three directories:
#   .next/standalone  → server runtime (includes node_modules subset)
#   .next/static      → compiled client assets (JS chunks, CSS)
#   public            → public static files
COPY --from=builder --chown=appuser:appgroup /app/.next/standalone ./
COPY --from=builder --chown=appuser:appgroup /app/.next/static ./.next/static
COPY --from=builder --chown=appuser:appgroup /app/public ./public

USER appuser

# Next.js storefront listens on port 3000 (configured via PORT env var)
EXPOSE 3000

HEALTHCHECK --interval=15s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:3000/api/health || exit 1

CMD ["node", "server.js"]
