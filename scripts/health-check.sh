#!/bin/bash
# Master Agent Health Check - Tüm servisleri kontrol et

echo "🔍 HEALTH CHECK STARTING..."
echo ""

ERRORS=0

# 1. Docker kontrol
echo "1️⃣  Docker daemon..."
if docker ps &>/dev/null; then
  echo "   ✅ Docker çalışıyor"
else
  echo "   ❌ Docker çalışmıyor!"
  ((ERRORS++))
fi

# 2. Backend servisleri kontrol
echo ""
echo "2️⃣  Backend Services..."
for service in product:8001 cart:8002 order:8003 checkout:8004 gateway:8080; do
  name="${service%:*}"
  port="${service#*:}"
  if curl -sf "http://localhost:${port}/health/live" &>/dev/null; then
    echo "   ✅ ${name} (${port})"
  else
    echo "   ❌ ${name} (${port}) - NOT RESPONDING"
    ((ERRORS++))
  fi
done

# 3. BFF kontrol
echo ""
echo "3️⃣  BFF (Backend for Frontend)..."
if curl -sf "http://localhost:3001/health" &>/dev/null; then
  echo "   ✅ BFF çalışıyor (3001)"
else
  echo "   ❌ BFF çalışmıyor (3001)"
  ((ERRORS++))
fi

# 4. Web Frontend kontrol
echo ""
echo "4️⃣  Web Frontend..."
if curl -sf "http://localhost:3100/" &>/dev/null; then
  echo "   ✅ Web çalışıyor (3100)"
else
  echo "   ❌ Web çalışmıyor (3100)"
  ((ERRORS++))
fi

# 5. Ürün datasını kontrol
echo ""
echo "5️⃣  Product Data..."
PRODUCT_COUNT=$(curl -sf "http://localhost:3001/api/products?page=1&limit=1" | jq -r '.data | length' 2>/dev/null || echo "0")
if [ "$PRODUCT_COUNT" -gt 0 ]; then
  echo "   ✅ ${PRODUCT_COUNT} ürün mevcut"
else
  echo "   ❌ Hiç ürün yok - seed script çalıştırılmalı!"
  ((ERRORS++))
fi

# 6. Dashboard kontrol
echo ""
echo "6️⃣  Dashboard..."
if curl -sf "http://localhost:3333/" &>/dev/null; then
  echo "   ✅ Dashboard çalışıyor (3333)"
else
  echo "   ⚠️  Dashboard çalışmıyor (3333) - optional"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [ $ERRORS -eq 0 ]; then
  echo "✅ TÜM SERVİSLER SAĞLIKLI!"
else
  echo "❌ ${ERRORS} SORUN TESPİT EDİLDİ!"
  exit 1
fi
