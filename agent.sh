#!/bin/bash
# EcommerceGo Development Agent
# Runs learn-claude-code s_full.py from the EcommerceGo root directory.
#
# Usage:
#   ./agent.sh          # interactive session
#   ./agent.sh s11      # run a specific session (s01–s12, s_full)
#
# The agent operates on the EcommerceGo codebase (this directory).
# Skills are loaded from learn-claude-code/skills/.

set -e

REPO_ROOT="$(cd "$(dirname "$0")" && pwd)"
AGENT_DIR="$REPO_ROOT/learn-claude-code"
SESSION="${1:-s_full}"

if [ ! -f "$AGENT_DIR/.venv/bin/python3" ]; then
  echo "Setting up virtualenv..."
  python3 -m venv "$AGENT_DIR/.venv"
  "$AGENT_DIR/.venv/bin/pip" install -r "$AGENT_DIR/requirements.txt" -q
fi

if [ ! -f "$AGENT_DIR/.env" ]; then
  echo "ERROR: $AGENT_DIR/.env not found."
  echo "Copy $AGENT_DIR/.env.example → $AGENT_DIR/.env and set ANTHROPIC_API_KEY."
  exit 1
fi

# Symlink the skills and .env into the working directory so the agent finds them.
# The agent uses WORKDIR/skills and reads .env from its own script directory.
ln -sfn "$AGENT_DIR/skills" "$REPO_ROOT/skills" 2>/dev/null || true

echo ""
echo "  ╔══════════════════════════════════════╗"
echo "  ║   EcommerceGo Development Agent      ║"
echo "  ║   session: $SESSION"
echo "  ║   workdir: $REPO_ROOT"
echo "  ╚══════════════════════════════════════╝"
echo ""

# Run from EcommerceGo root so the agent operates on the codebase.
cd "$REPO_ROOT"
exec "$AGENT_DIR/.venv/bin/python3" "$AGENT_DIR/agents/${SESSION}.py"
