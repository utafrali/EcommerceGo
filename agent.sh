#!/bin/bash
# EcommerceGo Development Agent
#
# Runs agents from learn-claude-code/ directory so WORKDIR, .env, skills/,
# .tasks/, .team/ all resolve correctly.
#
# The EcommerceGo codebase path is passed via ECOMMERCEGO_ROOT env var.
# No ANTHROPIC_API_KEY needed — uses local claude CLI as backend.
#
# Usage:
#   ./agent.sh                    # autonomous loop (ecommercego_auto) — default
#   ./agent.sh ecommercego        # interactive REPL — type tasks manually
#   ./agent.sh s11                # learn-claude-code autonomous multi-agent
#   ./agent.sh s_full             # learn-claude-code full REPL
#
# Env vars:
#   INTERVAL=300   seconds between autonomous iterations (default 600)
#   MODEL_ID=...   claude model to use (default claude-sonnet-4-6)

set -e

REPO_ROOT="$(cd "$(dirname "$0")" && pwd)"
AGENT_DIR="$REPO_ROOT/learn-claude-code"
SESSION="${1:-ecommercego_auto}"
SCRIPT="$AGENT_DIR/agents/${SESSION}.py"

# ── Validate ────────────────────────────────────────────────────────────────
if [ ! -d "$AGENT_DIR" ]; then
  echo "ERROR: learn-claude-code not found at $AGENT_DIR"
  echo "Run: git clone https://github.com/shareAI-lab/learn-claude-code.git"
  exit 1
fi

if [ ! -f "$SCRIPT" ]; then
  echo "ERROR: Session script not found: $SCRIPT"
  echo "Available: ecommercego_auto (default), ecommercego, s01..s12, s_full"
  exit 1
fi

# ── Virtualenv ───────────────────────────────────────────────────────────────
if [ ! -f "$AGENT_DIR/.venv/bin/python3" ]; then
  echo "Setting up virtualenv..."
  python3 -m venv "$AGENT_DIR/.venv"
  "$AGENT_DIR/.venv/bin/pip" install -r "$AGENT_DIR/requirements.txt" -q
  echo "Done."
fi

# ── Backend detection ────────────────────────────────────────────────────────
KEY=""
if [ -f "$AGENT_DIR/.env" ]; then
  KEY=$(grep "^ANTHROPIC_API_KEY=" "$AGENT_DIR/.env" 2>/dev/null | cut -d= -f2)
fi
[ -z "$KEY" ] && KEY="${ANTHROPIC_API_KEY:-}"

if [ "$KEY" = "your-key-here" ] || [ -z "$KEY" ]; then
  BACKEND="claude CLI (no API key needed)"
else
  BACKEND="Anthropic API"
fi

# ── Banner ────────────────────────────────────────────────────────────────────
echo ""
echo "  ╔════════════════════════════════════════════╗"
echo "  ║   EcommerceGo Development Agent           ║"
echo "  ║   session : $SESSION"
echo "  ║   backend : $BACKEND"
echo "  ║   codebase: $REPO_ROOT"
echo "  ║   workdir : $AGENT_DIR"
echo "  ╚════════════════════════════════════════════╝"
echo ""
if [ "$SESSION" = "ecommercego_auto" ]; then
  echo "  Autonomous mode — agent will loop every ${INTERVAL:-600}s."
  echo "  Press Ctrl+C to stop."
else
  echo "  Type your task or 'load_skill ecommercego-dev' to start."
  echo "  /tasks  /team  /compact  to inspect state."
fi
echo ""

# ── Run from learn-claude-code/ so WORKDIR, .env, skills/ all align ──────────
cd "$AGENT_DIR"
export ECOMMERCEGO_ROOT="$REPO_ROOT"
exec .venv/bin/python3 "agents/${SESSION}.py"
