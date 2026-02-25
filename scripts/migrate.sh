#!/bin/bash
# =============================================================================
# migrate.sh — Run golang-migrate for EcommerceGo services
# =============================================================================
# Usage:
#   ./scripts/migrate.sh              # migrate ALL services (up)
#   ./scripts/migrate.sh product      # migrate a single service (up)
#   ./scripts/migrate.sh product down # rollback 1 step for a service
#   ./scripts/migrate.sh product down 3 # rollback N steps
# =============================================================================
set -euo pipefail

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
BOLD="\033[1m"
GREEN="\033[32m"
YELLOW="\033[33m"
RED="\033[31m"
RESET="\033[0m"

info()    { echo -e "${BOLD}[INFO]${RESET}  $*"; }
success() { echo -e "${GREEN}[OK]${RESET}    $*"; }
warn()    { echo -e "${YELLOW}[WARN]${RESET}  $*"; }
error()   { echo -e "${RED}[ERROR]${RESET} $*" >&2; }
die()     { error "$*"; exit 1; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# ---------------------------------------------------------------------------
# Load environment (.env file at repo root)
# ---------------------------------------------------------------------------
ENV_FILE="${REPO_ROOT}/.env"
if [ -f "${ENV_FILE}" ]; then
  # shellcheck source=/dev/null
  set -o allexport
  source "${ENV_FILE}"
  set +o allexport
else
  warn ".env not found — using default values. Run ./scripts/setup.sh first."
fi

# ---------------------------------------------------------------------------
# Configuration (with defaults matching .env.example)
# ---------------------------------------------------------------------------
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-ecommerce}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-ecommerce_secret}"
POSTGRES_SSL_MODE="${POSTGRES_SSL_MODE:-disable}"

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
# $1 (optional) — service name or "all" (default: all)
# $2 (optional) — direction: "up" | "down" (default: up)
# $3 (optional) — steps for "down" (default: 1)

TARGET_SERVICE="${1:-all}"
DIRECTION="${2:-up}"
STEPS="${3:-1}"

# All services that may have migrations
ALL_SERVICES=(product cart order checkout payment user inventory campaign notification search media gateway)

# ---------------------------------------------------------------------------
# Prerequisite check
# ---------------------------------------------------------------------------
if ! command -v migrate &>/dev/null; then
  die "golang-migrate CLI not found. Install with:
    go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
  Then make sure \$GOPATH/bin is in your PATH."
fi

# ---------------------------------------------------------------------------
# Migration runner
# ---------------------------------------------------------------------------
run_migration() {
  local service="$1"
  local direction="$2"
  local steps="$3"

  local migrations_dir="${REPO_ROOT}/services/${service}/migrations"

  if [ ! -d "${migrations_dir}" ]; then
    warn "  [${service}] No migrations directory found at ${migrations_dir} — skipping."
    return 0
  fi

  # Count migration files (skip if directory is empty)
  local migration_count
  migration_count=$(find "${migrations_dir}" -maxdepth 1 -name "*.sql" | wc -l | tr -d ' ')
  if [ "${migration_count}" -eq 0 ]; then
    warn "  [${service}] migrations/ directory is empty — skipping."
    return 0
  fi

  # Determine the database name for this service
  # Convention: <SERVICE>_DB_NAME env var, falls back to <service>_db
  local db_name_var
  db_name_var="$(echo "${service}" | tr '[:lower:]' '[:upper:]')_DB_NAME"
  local db_name="${!db_name_var:-${service}_db}"

  local dsn="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${db_name}?sslmode=${POSTGRES_SSL_MODE}"

  info "  [${service}] ${direction} → database: ${db_name} (${POSTGRES_HOST}:${POSTGRES_PORT})"

  if [ "${direction}" = "up" ]; then
    if migrate -path "${migrations_dir}" -database "${dsn}" up 2>&1; then
      success "  [${service}] migration up complete."
    else
      error "  [${service}] migration failed."
      return 1
    fi
  elif [ "${direction}" = "down" ]; then
    if migrate -path "${migrations_dir}" -database "${dsn}" down "${steps}" 2>&1; then
      success "  [${service}] rolled back ${steps} step(s)."
    else
      error "  [${service}] rollback failed."
      return 1
    fi
  else
    die "Unknown direction '${direction}'. Use 'up' or 'down'."
  fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
echo ""
echo -e "${BOLD}============================================================${RESET}"
echo -e "${BOLD}  EcommerceGo — Database Migrations (${DIRECTION})${RESET}"
echo -e "${BOLD}============================================================${RESET}"
echo ""

FAILED_SERVICES=()

if [ "${TARGET_SERVICE}" = "all" ]; then
  info "Running migrations for all services..."
  echo ""
  for svc in "${ALL_SERVICES[@]}"; do
    if ! run_migration "${svc}" "${DIRECTION}" "${STEPS}"; then
      FAILED_SERVICES+=("${svc}")
    fi
  done
else
  # Validate the service name
  valid=0
  for svc in "${ALL_SERVICES[@]}"; do
    [ "${svc}" = "${TARGET_SERVICE}" ] && valid=1 && break
  done
  if [ "${valid}" -eq 0 ]; then
    die "Unknown service '${TARGET_SERVICE}'. Valid services: ${ALL_SERVICES[*]}"
  fi

  run_migration "${TARGET_SERVICE}" "${DIRECTION}" "${STEPS}" || FAILED_SERVICES+=("${TARGET_SERVICE}")
fi

echo ""

if [ "${#FAILED_SERVICES[@]}" -gt 0 ]; then
  error "Migrations failed for: ${FAILED_SERVICES[*]}"
  exit 1
fi

success "All migrations completed successfully."
echo ""
