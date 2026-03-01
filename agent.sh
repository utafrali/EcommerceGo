#!/bin/bash
# EcommerceGo Development Agent
#
# Runs learn-claude-code s_full.py (or any session) from the learn-claude-code/
# directory so WORKDIR, .env, skills/, .tasks/, .team/ all resolve correctly.
#
# The EcommerceGo codebase path is passed to the agent via the ECOMMERCEGO_ROOT
# env var and is documented in the ecommercego-dev skill.
#
# Usage:
#   ./agent.sh              # full agent (s_full) — recommended
#   ./agent.sh s11          # autonomous multi-agent with task claiming
#   ./agent.sh s07          # task-system only
#
# First time setup:
#   1. Set ANTHROPIC_API_KEY in learn-claude-code/.env
#   2. Run ./agent.sh

set -e

REPO_ROOT="$(cd "$(dirname "$0")" && pwd)"
AGENT_DIR="$REPO_ROOT/learn-claude-code"
SESSION="${1:-ecommercego}"
SCRIPT="$AGENT_DIR/agents/${SESSION}.py"

# ── Validate ────────────────────────────────────────────────────────────────
if [ ! -d "$AGENT_DIR" ]; then
  echo "ERROR: learn-claude-code not found at $AGENT_DIR"
  echo "Run: git clone https://github.com/shareAI-lab/learn-claude-code.git"
  exit 1
fi

if [ ! -f "$SCRIPT" ]; then
  echo "ERROR: Session script not found: $SCRIPT"
  echo "Available: s01..s12, s_full"
  exit 1
fi

# ── Virtualenv ───────────────────────────────────────────────────────────────
if [ ! -f "$AGENT_DIR/.venv/bin/python3" ]; then
  echo "Setting up virtualenv..."
  python3 -m venv "$AGENT_DIR/.venv"
  "$AGENT_DIR/.venv/bin/pip" install -r "$AGENT_DIR/requirements.txt" -q
  echo "Done."
fi

# ── API Key check ─────────────────────────────────────────────────────────────
if [ -f "$AGENT_DIR/.env" ]; then
  KEY=$(grep "^ANTHROPIC_API_KEY=" "$AGENT_DIR/.env" | cut -d= -f2)
  if [ "$KEY" = "your-key-here" ] || [ -z "$KEY" ]; then
    if [ -z "$ANTHROPIC_API_KEY" ]; then
      echo "ERROR: ANTHROPIC_API_KEY not set."
      echo "Edit $AGENT_DIR/.env and set your real API key."
      exit 1
    fi
  fi
fi

# ── Banner ────────────────────────────────────────────────────────────────────
echo ""
echo "  ╔════════════════════════════════════════════╗"
echo "  ║   EcommerceGo Development Agent           ║"
echo "  ║   session : $SESSION"
echo "  ║   codebase: $REPO_ROOT"
echo "  ║   workdir : $AGENT_DIR"
echo "  ╚════════════════════════════════════════════╝"
echo ""
echo "  Type your task or 'load_skill ecommercego-dev' to start."
echo "  /tasks  /team  /compact  to inspect state."
echo ""

# ── Run from learn-claude-code/ so WORKDIR, .env, skills/ all align ──────────
cd "$AGENT_DIR"
export ECOMMERCEGO_ROOT="$REPO_ROOT"
exec .venv/bin/python3 "agents/${SESSION}.py"
