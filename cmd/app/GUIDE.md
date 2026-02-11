# cmd/app/ — Application Entry Point

This is where the application boots. `main.go` is responsible for:

1. Loading configuration
2. Creating infrastructure (DB connection)
3. Running migrations
4. Wiring the dependency graph
5. Starting the HTTP server

## `main.go` — STUB (empty `func main()`)

### Bootstrap Order

```go
func main() {
    // 1. Load config (env vars, flags, or config file)
    //    - DB driver (postgres vs sqlite)
    //    - DB DSN (connection string or ":memory:")
    //    - Server port
    //    - JWT signing key
    //    - Migration path

    // 2. Create DB connection
    //    db, err := infra.NewDB(ctx, infra.Postgres, dsn)

    // 3. Run migrations
    //    infra.RunMigrations(db, "pgx", "db/migrations")

    // 4. Create repository store
    //    store, err := repo.NewStore(repo.SQLStore, db)

    // 5. Create service registry
    //    registry, err := service.NewRegistry(service.DefaultRegistry, store)

    // 6. Set up HTTP router and handlers
    //    - Mount auth routes     → registry.Auth
    //    - Mount user routes     → registry.Users
    //    - Mount conversation routes → registry.Conversations
    //    - Mount message routes  → registry.Messages
    //    - Mount moderation routes → registry.Moderation
    //    - Mount WebSocket endpoint

    // 7. Start server
    //    http.ListenAndServe(":8080", router)
}
```

### Why This Order Matters

Each step depends on the previous one:
- You can't create repos without a DB connection
- You can't create services without repos
- You can't create handlers without services

This is **constructor-based dependency injection** — no global variables, no service locators, no init() magic. Every dependency is explicitly passed through constructors. This makes the app easy to test (swap any layer with a mock) and easy to reason about (follow the wiring from main).

### Configuration

Not yet implemented. Options:
- **Environment variables** — simplest, good for containers (`os.Getenv`)
- **Flags** — good for CLI usage (`flag.String`)
- **Config file** — good for complex setups (YAML/TOML)

For dev, you'll typically use:
```
DB_DRIVER=sqlite
DB_DSN=:memory:
PORT=8080
JWT_SECRET=dev-secret
```

For production:
```
DB_DRIVER=pgx
DB_DSN=host=localhost port=5432 user=app password=... dbname=chat sslmode=require
PORT=8080
JWT_SECRET=<long-random-string>
```

### HTTP Handlers (not yet created)

The handler layer will likely live in `internal/handler/` or be defined directly in `main.go` for simplicity. Each handler:
1. Parses the HTTP request (path params, query params, JSON body)
2. Extracts the authenticated user ID from the JWT (via middleware)
3. Calls the appropriate service method
4. Maps the result (or error) to an HTTP response

The handler layer is the **only place** that knows about `net/http`. Services and repos are HTTP-agnostic.
