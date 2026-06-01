#!/usr/bin/env bash
set -euo pipefail

echo "[run.sh] Starting service"

if [ -n "${DATABASE_URL:-}" ] && [ -d /app/db/migrations ] && [ -n "$(ls -A /app/db/migrations 2>/dev/null)" ]; then
  echo "[run.sh] Running DB migrations"
  goose -dir /app/db/migrations postgres "${DATABASE_URL}" up
else
  echo "[run.sh] Skipping migrations (DATABASE_URL or migrations not provided)"
fi

echo "[run.sh] Starting Go app"
exec /app/bin/app
