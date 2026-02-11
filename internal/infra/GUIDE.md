# infra/ — Infrastructure Layer

This package owns external connections — databases, migration runners, and (eventually) the WebSocket server. It has **no knowledge of business logic**. Its job is to provide initialized, ready-to-use infrastructure that the repo and service layers consume.

## Files

### `db.go` — COMPLETE

| Export | Purpose |
|--------|---------|
| `DBConfig` | Holds the driver name (`"pgx"` or `"sqlite"`) and DSN string. |
| `OpenDB()` | Opens a `*sql.DB` connection and pings it with a 5-second timeout. Returns an error if the database is unreachable. |

**Why `database/sql` instead of raw `pgxpool`:** Both `pgx/v5/stdlib` and `modernc.org/sqlite` register as `database/sql` drivers. Using the standard interface means the repo layer writes one set of SQL that works against both Postgres and SQLite. The trade-off is losing pgx-specific features (COPY, custom types) — acceptable for this stage.

### `state.go` — COMPLETE

| Export | Purpose |
|--------|---------|
| `DriverType` | Typed `iota` enum: `Postgres` (0) and `SQLite` (1). |
| `NewDB()` | Factory function. Takes a `DriverType` and DSN, returns a configured `*sql.DB`. |

**Why a factory pattern here:** The caller (main.go) decides which database to use based on configuration (env vars, flags). The factory encapsulates driver-specific setup:

- **Postgres:** 25 max connections, 10 idle, 5-min lifetime, 1-min idle timeout — standard production pooling
- **SQLite:** 1 max connection — SQLite doesn't support concurrent writers, so a single connection prevents "database is locked" errors

**Driver imports live here** (`_ "github.com/jackc/pgx/v5/stdlib"` and `_ "modernc.org/sqlite"`). Blank imports register the drivers with `database/sql` at init time. They must be in a file that gets compiled — `state.go` is the right place since it's the entry point for DB creation.

### `migration.go` — COMPLETE

| Export | Purpose |
|--------|---------|
| `RunMigrations()` | Takes a `*sql.DB`, driver name, and path to migration files. Applies all pending up-migrations using `golang-migrate`. |

**How it works:**
1. Creates a driver-specific `database.Driver` adapter (postgres or sqlite)
2. Points `golang-migrate` at the `db/migrations/` directory via the `file://` source
3. Calls `m.Up()` — applies all unapplied migrations in order
4. `migrate.ErrNoChange` is swallowed (not an error — just means everything is already up to date)

**When to call it:** Once at application startup, after `NewDB()` returns a connection but before any repo operations.

### `ws.go` — STUB (needs implementation)

This file will hold the WebSocket infrastructure. What it needs:

| Component | Purpose |
|-----------|---------|
| `Hub` struct | Central connection registry. Tracks which users are connected and maps user IDs to WebSocket connections. |
| `Client` struct | Represents a single WebSocket connection. Holds the connection, user ID, and send channel. |
| `NewHub()` | Constructor for the hub. |
| `Hub.Run()` | Goroutine that processes register/unregister/broadcast events. |

**Why the Hub pattern:** WebSocket connections are stateful and long-lived. The hub acts as a centralized broker:
- When a message is sent via REST API, the service publishes an event to the hub
- The hub looks up which recipients are connected and pushes the message frame
- When a client disconnects, the hub removes them and broadcasts a presence update

**Dependencies:** You'll likely need `gorilla/websocket` or `nhooyr.io/websocket`. The hub should be passed to the service layer (or use an event channel) so services can trigger real-time pushes without importing the WebSocket library directly.

## Wiring Order

```
main.go calls:
  1. infra.NewDB(ctx, driver, dsn)     → *sql.DB
  2. infra.RunMigrations(db, ...)       → ensures schema is up to date
  3. repo.NewStore(SQLStore, db)        → *repo.Store
  4. service.NewRegistry(Default, store) → *service.Registry
  5. set up HTTP routes + WebSocket hub
  6. start server
```
