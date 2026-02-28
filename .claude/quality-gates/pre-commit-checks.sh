#!/bin/bash
# Master Agent Quality Gates - Pre-Commit Checks
# Run before every commit to catch issues early

set -e

PROJECT_ROOT="/Users/ugurtafrali/Dev/EcommerceGo"
REPORT_FILE="/tmp/quality-gate-report.txt"

echo "🔍 Master Agent Quality Gates - Running Pre-Commit Checks..."
echo "=============================================================="
echo "" > "$REPORT_FILE"

FAILED_CHECKS=0

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 1. FRONTEND BUILD CHECK
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo "1. Checking frontend build..."
cd "$PROJECT_ROOT/web"
if npm run build > /tmp/frontend-build.log 2>&1; then
  echo "   ✅ Frontend builds successfully"
else
  echo "   ❌ Frontend build FAILED"
  echo "FAIL: Frontend build" >> "$REPORT_FILE"
  FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 2. FRONTEND VISUAL CHECK (Screenshot Critical Pages)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo "2. Visual regression check..."
if command -v npx &> /dev/null; then
  # Check if dev server is running
  if curl -s http://localhost:3000 > /dev/null 2>&1; then
    # TODO: Add visual regression tests with Playwright
    echo "   ⚠️  Visual checks skipped (dev server must be running)"
  else
    echo "   ⚠️  Dev server not running, skipping visual checks"
  fi
else
  echo "   ⚠️  npx not available, skipping visual checks"
fi

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 3. BANNED WORDS CHECK (Modanisa, competitor names)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo "3. Checking for banned words in code..."
BANNED_WORDS=("Modanisa" "modanisa" "MODANISA")
cd "$PROJECT_ROOT"

for word in "${BANNED_WORDS[@]}"; do
  if git diff --cached --name-only | xargs grep -l "$word" 2>/dev/null; then
    echo "   ❌ BANNED WORD FOUND: $word"
    echo "FAIL: Banned word '$word' found in staged files" >> "$REPORT_FILE"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
  fi
done

if [ "$FAILED_CHECKS" -eq 0 ]; then
  echo "   ✅ No banned words found"
fi

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 4. CSS ANIMATION CHECK (Missing Tailwind animations)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo "4. Checking for missing CSS animations..."
cd "$PROJECT_ROOT/web"

# Find all animate- classes in components
USED_ANIMATIONS=$(grep -roh 'animate-[a-zA-Z-]*' src/components/ | sort -u)
# Get defined animations from tailwind.config.ts (macOS compatible)
DEFINED_ANIMATIONS=$(grep -o "'[^']*':" tailwind.config.ts | sed "s/'//g" | sed "s/://g" | grep -v "^[0-9]")

MISSING_ANIMATIONS=0
for anim in $USED_ANIMATIONS; do
  anim_name="${anim#animate-}"
  if ! echo "$DEFINED_ANIMATIONS" | grep -q "$anim_name"; then
    echo "   ❌ Missing animation: $anim"
    echo "FAIL: Missing Tailwind animation '$anim'" >> "$REPORT_FILE"
    MISSING_ANIMATIONS=$((MISSING_ANIMATIONS + 1))
  fi
done

if [ "$MISSING_ANIMATIONS" -eq 0 ]; then
  echo "   ✅ All animations defined"
else
  FAILED_CHECKS=$((FAILED_CHECKS + MISSING_ANIMATIONS))
fi

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 5. ACCESSIBILITY CHECKS (Basic ARIA, alt text)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo "5. Basic accessibility checks..."
cd "$PROJECT_ROOT/web/src"

# Check for buttons without aria-label
BUTTONS_NO_ARIA=$(grep -r '<button' . | grep -v 'aria-label' | grep -v 'aria-labelledby' | wc -l)
if [ "$BUTTONS_NO_ARIA" -gt 10 ]; then
  echo "   ⚠️  Found $BUTTONS_NO_ARIA buttons without aria-label (threshold: 10)"
else
  echo "   ✅ Accessibility baseline met"
fi

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 6. BACKEND SERVICE COMPILATION
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo "6. Checking backend services compile..."
cd "$PROJECT_ROOT"

SERVICES=("product" "cart" "order" "user" "gateway")
for svc in "${SERVICES[@]}"; do
  if [ -d "services/$svc" ]; then
    cd "$PROJECT_ROOT/services/$svc"
    if go build ./... > /dev/null 2>&1; then
      echo "   ✅ $svc service compiles"
    else
      echo "   ❌ $svc service FAILS to compile"
      echo "FAIL: $svc service compilation failed" >> "$REPORT_FILE"
      FAILED_CHECKS=$((FAILED_CHECKS + 1))
    fi
  fi
done

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 7. COMMIT MESSAGE CHECK
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo "7. Checking commit message format..."
# This will be run by git hook, skip in manual mode
echo "   ⏭️  Skipped (run by git hook)"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# FINAL REPORT
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo ""
echo "=============================================================="
if [ "$FAILED_CHECKS" -eq 0 ]; then
  echo "✅ All quality gates PASSED"
  echo ""
  exit 0
else
  echo "❌ $FAILED_CHECKS quality gates FAILED"
  echo ""
  echo "Failed checks:"
  cat "$REPORT_FILE"
  echo ""
  echo "Fix these issues before committing."
  exit 1
fi
