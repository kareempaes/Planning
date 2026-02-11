#!/usr/bin/env bash
set -euo pipefail
docker compose exec postgres psql -U chat -d chatdb "$@"
