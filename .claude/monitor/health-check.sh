#!/bin/bash
# EcommerceGo Health Monitor - Round 26
# Continuously monitors all services and restarts failed ones

SERVICES=(
  "product:8001"
  "cart:8002"
  "order:8003"
  "checkout:8004"
  "payment:8005"
  "user:8006"
  "inventory:8007"
  "campaign:8008"
  "notification:8009"
  "search:8010"
  "media:8011"
  "gateway:8080"
)

STATUS_FILE="/Users/ugurtafrali/Dev/EcommerceGo/.claude/pipeline/status.json"
PROJECT_ROOT="/Users/ugurtafrali/Dev/EcommerceGo"

check_service() {
  local service=$1
  local port=$2

  # Try health endpoint
  response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:${port}/health/live 2>/dev/null)

  if [ "$response" = "200" ]; then
    echo "UP"
  else
    echo "DOWN"
  fi
}

start_service() {
  local service=$1
  local port=$2

  echo "[$(date +%H:%M:%S)] Starting $service service on port $port..."

  cd "$PROJECT_ROOT/services/$service"
  nohup go run cmd/server/main.go > "/tmp/${service}-service.log" 2>&1 &

  sleep 3

  # Verify startup
  status=$(check_service "$service" "$port")
  if [ "$status" = "UP" ]; then
    echo "[$(date +%H:%M:%S)] ✅ $service service started successfully"
    return 0
  else
    echo "[$(date +%H:%M:%S)] ❌ $service service failed to start"
    return 1
  fi
}

monitor_loop() {
  echo "============================================"
  echo "EcommerceGo Health Monitor - Round 26"
  echo "============================================"
  echo ""

  while true; do
    echo "[$(date +%H:%M:%S)] Health check cycle..."

    for entry in "${SERVICES[@]}"; do
      IFS=':' read -r service port <<< "$entry"

      status=$(check_service "$service" "$port")

      if [ "$status" = "DOWN" ]; then
        echo "[$(date +%H:%M:%S)] ⚠️  $service is DOWN, attempting restart..."
        start_service "$service" "$port"
      else
        echo "[$(date +%H:%M:%S)] ✅ $service is UP"
      fi
    done

    echo ""
    sleep 30  # Check every 30 seconds
  done
}

# Start monitoring
monitor_loop
