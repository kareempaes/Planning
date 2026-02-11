#!/usr/bin/env bash
set -euo pipefail

docker compose exec postgres psql -U chat -d chatdb <<'SQL'
INSERT INTO users (id, email, password_hash, display_name, status, created_at, updated_at)
VALUES
  ('00000000-0000-0000-0000-000000000001', 'alice@example.com',
   '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
   'Alice', 'offline', now(), now()),
  ('00000000-0000-0000-0000-000000000002', 'bob@example.com',
   '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
   'Bob', 'offline', now(), now())
ON CONFLICT DO NOTHING;
SQL

echo "Seed complete: alice@example.com and bob@example.com (password: password123)"
