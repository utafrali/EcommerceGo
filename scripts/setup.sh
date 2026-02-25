#!/bin/bash
# =============================================================================
# setup.sh — Initial developer setup for EcommerceGo
# =============================================================================
# Usage: ./scripts/setup.sh
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

# Resolve the repository root (directory containing this script's parent)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${REPO_ROOT}"

echo ""
echo -e "${BOLD}============================================================${RESET}"
echo -e "${BOLD}  EcommerceGo — Developer Setup${RESET}"
echo -e "${BOLD}============================================================${RESET}"
echo ""

# ---------------------------------------------------------------------------
# 1. Check prerequisites
# ---------------------------------------------------------------------------
info "Checking prerequisites..."

check_cmd() {
  local cmd="$1"
  local friendly="${2:-$cmd}"
  local install_hint="${3:-}"
  if command -v "${cmd}" &>/dev/null; then
    local version
    version="$(${cmd} --version 2>&1 | head -1 || true)"
    success "${friendly} found — ${version}"
  else
    error "${friendly} not found."
    [ -n "${install_hint}" ] && echo "       ${install_hint}"
    MISSING_PREREQS=1
  fi
}

MISSING_PREREQS=0

check_cmd go        "Go"            "Install from https://go.dev/dl/"
check_cmd node      "Node.js"       "Install from https://nodejs.org/ (v20+ recommended)"
check_cmd npm       "npm"           "Comes with Node.js"
check_cmd docker    "Docker"        "Install from https://docs.docker.com/get-docker/"
check_cmd docker    "docker compose" # compose is now a docker subcommand

# Verify docker compose v2 is available (not docker-compose v1)
if docker compose version &>/dev/null 2>&1; then
  success "docker compose (v2) found — $(docker compose version | head -1)"
else
  error "docker compose v2 not found. Please upgrade Docker Desktop or install the compose plugin."
  MISSING_PREREQS=1
fi

if [ "${MISSING_PREREQS}" -ne 0 ]; then
  die "One or more prerequisites are missing. Please install them and re-run setup."
fi

echo ""

# ---------------------------------------------------------------------------
# 2. Copy .env.example → .env  (if .env does not already exist)
# ---------------------------------------------------------------------------
info "Checking environment file..."

if [ -f "${REPO_ROOT}/.env" ]; then
  warn ".env already exists — skipping copy. Edit it manually if needed."
else
  cp "${REPO_ROOT}/.env.example" "${REPO_ROOT}/.env"
  success "Copied .env.example → .env"
  warn "Review and update ${REPO_ROOT}/.env with your local secrets before proceeding."
fi

echo ""

# ---------------------------------------------------------------------------
# 3. Sync Go workspace
# ---------------------------------------------------------------------------
info "Syncing Go workspace (go work sync)..."

if [ -f "${REPO_ROOT}/go.work" ]; then
  go work sync
  success "Go workspace synced."
else
  warn "No go.work found at repo root. Skipping go work sync."
  warn "If services share the pkg module, consider running: go work init && go work use ./pkg ./services/*"
fi

echo ""

# ---------------------------------------------------------------------------
# 4. Download Go module dependencies for each service
# ---------------------------------------------------------------------------
info "Downloading Go module dependencies..."

SERVICES=(product cart order checkout payment user inventory campaign notification search media gateway)

for svc in "${SERVICES[@]}"; do
  svc_dir="${REPO_ROOT}/services/${svc}"
  if [ -f "${svc_dir}/go.mod" ]; then
    info "  go mod download — ${svc}"
    (cd "${svc_dir}" && go mod download -x 2>/dev/null | tail -1 || true)
  fi
done

success "Go dependencies ready."
echo ""

# ---------------------------------------------------------------------------
# 5. Install npm dependencies for BFF and Web
# ---------------------------------------------------------------------------
info "Installing Node.js dependencies..."

for app_dir in bff web cms; do
  full_path="${REPO_ROOT}/${app_dir}"
  if [ -f "${full_path}/package.json" ]; then
    info "  npm ci — ${app_dir}"
    (cd "${full_path}" && npm ci --silent)
    success "  ${app_dir} dependencies installed."
  fi
done

echo ""

# ---------------------------------------------------------------------------
# 6. Start infrastructure containers
# ---------------------------------------------------------------------------
info "Starting infrastructure (postgres, redis, kafka, elasticsearch, minio, mailhog)..."
docker compose -f "${REPO_ROOT}/docker-compose.infra.yml" up -d

echo ""
info "Waiting for infrastructure to become healthy..."

# Poll until all services in the infra compose file are healthy / running
wait_healthy() {
  local service="$1"
  local max_attempts="${2:-30}"
  local attempt=0

  while [ "${attempt}" -lt "${max_attempts}" ]; do
    local state
    state=$(docker inspect --format='{{.State.Health.Status}}' "ecommerce-${service}" 2>/dev/null || echo "missing")
    if [ "${state}" = "healthy" ]; then
      success "  ${service} is healthy."
      return 0
    fi
    attempt=$((attempt + 1))
    printf "  waiting for %s (%d/%d)...\r" "${service}" "${attempt}" "${max_attempts}"
    sleep 3
  done

  warn "  ${service} did not become healthy within timeout — check logs with:"
  warn "    docker compose -f docker-compose.infra.yml logs ${service}"
  return 1
}

wait_healthy postgres 30
wait_healthy redis    20
wait_healthy kafka    40
wait_healthy elasticsearch 40

echo ""

# ---------------------------------------------------------------------------
# 7. Run database migrations
# ---------------------------------------------------------------------------
info "Running database migrations..."

if command -v migrate &>/dev/null; then
  "${SCRIPT_DIR}/migrate.sh"
else
  warn "golang-migrate not found. Skipping migrations."
  warn "Install with: go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
  warn "Then run: ./scripts/migrate.sh"
fi

echo ""

# ---------------------------------------------------------------------------
# 8. Done — print access URLs
# ---------------------------------------------------------------------------
# Source .env so we can read port values
# shellcheck source=/dev/null
set -o allexport
[ -f "${REPO_ROOT}/.env" ] && source "${REPO_ROOT}/.env"
set +o allexport

GATEWAY_HTTP_PORT="${GATEWAY_HTTP_PORT:-8080}"
BFF_PORT="${BFF_PORT:-3001}"
WEB_PORT="${WEB_PORT:-3000}"
MINIO_CONSOLE_PORT="${MINIO_CONSOLE_PORT:-9001}"
ELASTICSEARCH_PORT="${ELASTICSEARCH_PORT:-9200}"

echo ""
echo -e "${BOLD}============================================================${RESET}"
echo -e "${GREEN}${BOLD}  Setup complete!${RESET}"
echo -e "${BOLD}============================================================${RESET}"
echo ""
echo -e "  ${BOLD}Infrastructure:${RESET}"
echo    "    PostgreSQL     →  localhost:${POSTGRES_PORT:-5432}"
echo    "    Redis          →  localhost:${REDIS_PORT:-6379}"
echo    "    Kafka          →  localhost:9092"
echo    "    Elasticsearch  →  http://localhost:${ELASTICSEARCH_PORT}"
echo    "    MinIO API      →  http://localhost:${MINIO_API_PORT:-9000}"
echo    "    MinIO Console  →  http://localhost:${MINIO_CONSOLE_PORT}"
echo    "    MailHog UI     →  http://localhost:8025"
echo ""
echo -e "  ${BOLD}Application (start with: docker compose up -d):${RESET}"
echo    "    API Gateway    →  http://localhost:${GATEWAY_HTTP_PORT}"
echo    "    BFF            →  http://localhost:${BFF_PORT}"
echo    "    Storefront     →  http://localhost:${WEB_PORT}"
echo ""
echo -e "  ${BOLD}Next steps:${RESET}"
echo    "    1. Edit .env with your secrets"
echo    "    2. docker compose up -d          # start all services"
echo    "    3. ./scripts/seed.sh             # load sample data"
echo    "    4. make test                     # run tests"
echo ""
