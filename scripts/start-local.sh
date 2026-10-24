#!/usr/bin/env sh
set -eu

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

if [ ! -f .env ]; then
  cp .env.example .env
fi

if ! docker ps --format '{{.Names}}' | grep -qx ltseries-mongo; then
  if docker ps -a --format '{{.Names}}' | grep -qx ltseries-mongo; then
    docker start ltseries-mongo >/dev/null
  else
    docker run -d --name ltseries-mongo -p 127.0.0.1:27019:27017 -v ltseries_mongo_data:/data/db mongo:7 >/dev/null
  fi
fi

printf '%s\n' "MongoDB: mongodb://127.0.0.1:27019"
printf '%s\n' "Backend: cd backend && go run ./cmd/server"
printf '%s\n' "Frontend: cd LeriesStudio && npm ci && PORT=3108 npm run dev"
