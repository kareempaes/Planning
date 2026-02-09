# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Early-stage Go chat application supporting 1:1 and group messaging with real-time WebSocket delivery. Currently in the learning/scaffolding phase — most source files are stubs and the Go module has not been initialized yet.

## Planned Tech Stack

- **Language:** Go
- **Database:** PostgreSQL (primary), SQLite in-memory (dev/test)
- **Drivers:** `pgxpool` (Postgres), `modernc.org/sqlite` (SQLite)
- **Migrations:** `golang-migrate/migrate` (files in `db/migrations/`)
- **Realtime:** WebSocket
- **Containers:** Docker + Docker Compose

## Build & Run

No build tooling is configured yet. When it exists:
- `Makefile` — build targets (currently empty)
- `docker-compose.yml` — container orchestration (currently empty)
- `scripts/` — helper scripts for psql access and seeding (currently empty)

Go module needs to be initialized before any Go code compiles (`go mod init`).

## Architecture

Layered architecture under `internal/`:

| Layer | Package | Responsibility |
|-------|---------|---------------|
| Entry point | `cmd/app/` | Application bootstrap |
| Infrastructure | `internal/infra/` | DB connections, migrations, external adapters |
| Repository | `internal/repo/` | Data access (SQL queries) |
| Service | `internal/service/` | Business logic |

## Documentation Conventions

- Documentation is managed as an **Obsidian vault** (`.obsidian/` config present)
- Planning docs use Obsidian frontmatter (YAML) and `![[...]]` embed syntax for Mermaid diagrams
- New feature plans should follow the template in `docs/templates/Planner.md` (sections: Vision, Goal, Scope, Architecture, Workflows, Schema)
- Mermaid diagram files go in `docs/diagrams/`

## Repository Layout

- `sql/` — SQL learning/exercise files (not production schema)
- `db/migrations/` — Production migration files (golang-migrate format: `000NNN_name.up.sql` / `.down.sql`)
- `docs/PLAN.md` — Master project plan with vision, goals, and scope
