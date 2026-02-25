#!/bin/bash
# =============================================================================
# seed.sh — Load sample data into EcommerceGo via the Product Service API
# =============================================================================
# Usage:
#   ./scripts/seed.sh                 # seeds against localhost defaults
#   PRODUCT_BASE_URL=http://localhost:8001 ./scripts/seed.sh
# =============================================================================
set -euo pipefail

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
BOLD="\033[1m"
GREEN="\033[32m"
YELLOW="\033[33m"
RED="\033[31m"
CYAN="\033[36m"
RESET="\033[0m"

info()    { echo -e "${BOLD}[INFO]${RESET}   $*"; }
success() { echo -e "${GREEN}[OK]${RESET}     $*"; }
warn()    { echo -e "${YELLOW}[WARN]${RESET}   $*"; }
error()   { echo -e "${RED}[ERROR]${RESET}  $*" >&2; }
die()     { error "$*"; exit 1; }
step()    { echo -e "\n${CYAN}${BOLD}--- $* ---${RESET}"; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# ---------------------------------------------------------------------------
# Load .env (if present)
# ---------------------------------------------------------------------------
ENV_FILE="${REPO_ROOT}/.env"
if [ -f "${ENV_FILE}" ]; then
  # shellcheck source=/dev/null
  set -o allexport
  source "${ENV_FILE}"
  set +o allexport
fi

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
PRODUCT_HTTP_PORT="${PRODUCT_HTTP_PORT:-8001}"
PRODUCT_BASE_URL="${PRODUCT_BASE_URL:-http://localhost:${PRODUCT_HTTP_PORT}}"
API_BASE="${PRODUCT_BASE_URL}/api/v1"

# Max seconds to wait for the product service to be reachable
WAIT_TIMEOUT=30

# ---------------------------------------------------------------------------
# Prerequisite check
# ---------------------------------------------------------------------------
if ! command -v curl &>/dev/null; then
  die "curl is required but not found. Install curl and try again."
fi

# ---------------------------------------------------------------------------
# Wait for product service to be ready
# ---------------------------------------------------------------------------
info "Checking product service at ${PRODUCT_BASE_URL}/health/live..."

elapsed=0
while ! curl -sf "${PRODUCT_BASE_URL}/health/live" > /dev/null 2>&1; do
  if [ "${elapsed}" -ge "${WAIT_TIMEOUT}" ]; then
    die "Product service did not become reachable within ${WAIT_TIMEOUT}s.
  Make sure the service is running:
    docker compose up -d product
  or locally:
    cd services/product && go run ./cmd/server"
  fi
  printf "  waiting (%ds)...\r" "${elapsed}"
  sleep 2
  elapsed=$((elapsed + 2))
done

success "Product service is reachable."

# ---------------------------------------------------------------------------
# Helper: POST JSON and extract the id from the response
# ---------------------------------------------------------------------------
post_json() {
  local endpoint="$1"
  local payload="$2"
  local label="$3"

  local response
  response=$(curl -sf \
    -X POST \
    -H "Content-Type: application/json" \
    -d "${payload}" \
    "${API_BASE}${endpoint}" 2>&1) || {
    error "  POST ${endpoint} failed — ${response}"
    return 1
  }

  # Extract .data.id using basic shell parsing (no jq required)
  local id
  id=$(echo "${response}" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4 || true)

  if [ -z "${id}" ]; then
    warn "  Could not parse id from response for ${label}. Response: ${response}"
  else
    success "  Created ${label} (id: ${id})"
  fi

  echo "${id}"
}

echo ""
echo -e "${BOLD}============================================================${RESET}"
echo -e "${BOLD}  EcommerceGo — Seed Sample Data${RESET}"
echo -e "${BOLD}============================================================${RESET}"

# ===========================================================================
# Brands
# ===========================================================================
step "Creating brands"

brand_nike_id=$(post_json "/products" '{
  "name": "Nike",
  "description": "Just Do It",
  "base_price": 0,
  "currency": "USD"
}' "Brand: Nike (placeholder — replace when brand endpoint exists)" || true)

# If a dedicated brand endpoint exists at /api/v1/brands, use:
# brand_nike_id=$(post_json "/brands" '{"name":"Nike","slug":"nike","description":"Just Do It"}' "Brand: Nike" || true)
# For now we track the concept via product metadata

# ===========================================================================
# Categories (placeholder — create via categories endpoint when available)
# ===========================================================================
step "Noting categories"
info "  Categories endpoint not yet available — IDs will be null in products."

CATEGORY_SNEAKERS=""
CATEGORY_APPAREL=""
CATEGORY_ACCESSORIES=""

# ===========================================================================
# Products — Sneakers
# ===========================================================================
step "Creating products (Sneakers)"

post_json "/products" "{
  \"name\": \"Air Max 270\",
  \"description\": \"Nike Air Max 270 running shoe with large Air unit for max cushioning.\",
  \"category_id\": ${CATEGORY_SNEAKERS:+\"${CATEGORY_SNEAKERS}\"}$([ -z "${CATEGORY_SNEAKERS}" ] && echo 'null'),
  \"base_price\": 15000,
  \"currency\": \"USD\",
  \"metadata\": {
    \"brand\": \"Nike\",
    \"sizes\": [\"7\", \"8\", \"9\", \"10\", \"11\", \"12\"],
    \"colors\": [\"black\", \"white\", \"red\"]
  }
}" "Product: Air Max 270" || true

post_json "/products" "{
  \"name\": \"Ultraboost 22\",
  \"description\": \"Adidas Ultraboost 22 with responsive Boost midsole and Primeknit upper.\",
  \"category_id\": null,
  \"base_price\": 18000,
  \"currency\": \"USD\",
  \"metadata\": {
    \"brand\": \"Adidas\",
    \"sizes\": [\"7\", \"8\", \"9\", \"10\", \"11\"],
    \"colors\": [\"core black\", \"white\", \"solar yellow\"]
  }
}" "Product: Ultraboost 22" || true

post_json "/products" "{
  \"name\": \"Chuck Taylor All Star\",
  \"description\": \"Converse Chuck Taylor All Star — the iconic canvas high-top.\",
  \"category_id\": null,
  \"base_price\": 6500,
  \"currency\": \"USD\",
  \"metadata\": {
    \"brand\": \"Converse\",
    \"sizes\": [\"6\", \"7\", \"8\", \"9\", \"10\", \"11\", \"12\"],
    \"colors\": [\"optical white\", \"black\", \"navy\"]
  }
}" "Product: Chuck Taylor All Star" || true

# ===========================================================================
# Products — Apparel
# ===========================================================================
step "Creating products (Apparel)"

post_json "/products" "{
  \"name\": \"Dri-FIT Training T-Shirt\",
  \"description\": \"Nike Dri-FIT short-sleeve training top with sweat-wicking fabric.\",
  \"category_id\": null,
  \"base_price\": 3500,
  \"currency\": \"USD\",
  \"metadata\": {
    \"brand\": \"Nike\",
    \"sizes\": [\"XS\", \"S\", \"M\", \"L\", \"XL\", \"2XL\"],
    \"colors\": [\"black\", \"white\", \"navy\", \"red\"],
    \"gender\": \"unisex\"
  }
}" "Product: Dri-FIT Training T-Shirt" || true

post_json "/products" "{
  \"name\": \"Essentials Hoodie\",
  \"description\": \"Adidas Essentials fleece hoodie — warm, comfortable, everyday wear.\",
  \"category_id\": null,
  \"base_price\": 6500,
  \"currency\": \"USD\",
  \"metadata\": {
    \"brand\": \"Adidas\",
    \"sizes\": [\"S\", \"M\", \"L\", \"XL\", \"2XL\"],
    \"colors\": [\"legend ink\", \"white\", \"black\"],
    \"gender\": \"unisex\"
  }
}" "Product: Essentials Hoodie" || true

post_json "/products" "{
  \"name\": \"Classic Fit Oxford Shirt\",
  \"description\": \"Polo Ralph Lauren classic fit Oxford shirt — wrinkle-resistant cotton blend.\",
  \"category_id\": null,
  \"base_price\": 8900,
  \"currency\": \"USD\",
  \"metadata\": {
    \"brand\": \"Polo Ralph Lauren\",
    \"sizes\": [\"S\", \"M\", \"L\", \"XL\", \"2XL\"],
    \"colors\": [\"white\", \"blue\", \"light pink\"],
    \"gender\": \"men\"
  }
}" "Product: Classic Fit Oxford Shirt" || true

# ===========================================================================
# Products — Accessories
# ===========================================================================
step "Creating products (Accessories)"

post_json "/products" "{
  \"name\": \"Reversible Bucket Hat\",
  \"description\": \"Nike reversible bucket hat with Dri-FIT technology.\",
  \"category_id\": null,
  \"base_price\": 3200,
  \"currency\": \"USD\",
  \"metadata\": {
    \"brand\": \"Nike\",
    \"sizes\": [\"S/M\", \"L/XL\"],
    \"colors\": [\"black/white\", \"navy/grey\"]
  }
}" "Product: Reversible Bucket Hat" || true

post_json "/products" "{
  \"name\": \"Running Belt\",
  \"description\": \"Lightweight zip-up running waist pack — holds phone, keys, and cards.\",
  \"category_id\": null,
  \"base_price\": 2200,
  \"currency\": \"USD\",
  \"metadata\": {
    \"brand\": \"Generic Sport\",
    \"sizes\": [\"one size\"],
    \"colors\": [\"black\", \"grey\", \"blue\"]
  }
}" "Product: Running Belt" || true

post_json "/products" "{
  \"name\": \"Wireless Sport Earbuds\",
  \"description\": \"IPX5 waterproof Bluetooth 5.3 earbuds with 8-hour battery and charging case.\",
  \"category_id\": null,
  \"base_price\": 12900,
  \"currency\": \"USD\",
  \"metadata\": {
    \"brand\": \"SoundFit\",
    \"connectivity\": \"Bluetooth 5.3\",
    \"battery_hours\": 8,
    \"waterproof\": \"IPX5\"
  }
}" "Product: Wireless Sport Earbuds" || true

# ===========================================================================
# Summary
# ===========================================================================
echo ""
echo -e "${BOLD}============================================================${RESET}"
echo -e "${GREEN}${BOLD}  Seeding complete!${RESET}"
echo -e "${BOLD}============================================================${RESET}"
echo ""
info "Products are available at:"
echo "  GET ${API_BASE}/products"
echo "  GET ${API_BASE}/products?page=1&per_page=10"
echo ""
info "Note: prices are stored in cents (e.g. 15000 = \$150.00 USD)"
echo ""
