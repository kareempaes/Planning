Below is a **new, learning-first tutorial** tailored to your setup:

* **Postgres in Docker containers** (you’ll talk to it via `docker exec … psql`)
* **SQLite in-memory** (for local dev outside containers + tests) using **`database/sql`**
* **Go Postgres access** using **`pgxpool`**
* **Migrations** using **`golang-migrate/migrate`** (CLI + optional library) ([GitHub][1])

Each phase has:

* **Tasks (you do)**
* **Tools used (what + why)**
* **Syntax breakdown (what it means + purpose)**
* **“Check yourself” queries** (reference + explanation)

---

## Project you’ll build (used in every phase)

A small “commerce” domain, because it naturally exercises real SQL:

**Tables**

* `users`
* `products`
* `orders`
* `order_items`
* `payments`

**Relationships**

```
users 1─∞ orders 1─∞ order_items ∞─1 products
           |
           1
           ∞
        payments
```

---

# Phase 0 — Environment & terminal workflow (Postgres container + psql inside it)

## Goal

You can run SQL scripts and inspect schema using **only the terminal**, through the Postgres container.

## Tools (what + why)

### Docker Compose

**Why:** reproducible database environment (everyone gets the same Postgres).

### `docker exec … psql`

**Why:** you don’t need Postgres installed locally; you’re using the container’s `psql`. This is a common workflow. ([DataCamp][2])

## Tasks (you do)

1. Create `docker-compose.yml` (Postgres only to start).
2. Start it: `docker compose up -d`
3. Connect using **psql inside the container**:

   * list DBs, list tables, describe tables
4. Create a `sql/` folder and run `.sql` files from inside psql.

## Commands (reference)

Assume service name `postgres`:

* Find container name:

  * `docker compose ps`
* Open psql inside container:

  * `docker exec -it <container> psql -U app -d app` ([DataCamp][2])

### `psql` meta-commands (not SQL)

These are **client commands** that help you inspect things quickly:

* `\l` list databases
* `\dt` list tables
* `\d users` describe table
* `\i sql/01_schema.sql` run a file
* `\x on` expanded output for wide rows

**Purpose:** fast feedback loop without writing introspection queries.

---

# Phase 1 — SELECT fundamentals (filters, sorting, pagination, NULL)

## Goal

You can read data reliably and understand how SQL evaluates your query.

## Syntax (breakdown)

```sql
SELECT <columns>
FROM <table>
WHERE <predicate>
ORDER BY <columns>
LIMIT <n> OFFSET <n>;
```

**What each part is for**

* `SELECT`: choose which **columns** you want (projection)
* `FROM`: choose the **source table**
* `WHERE`: filter **rows**
* `ORDER BY`: define consistent ordering (critical for pagination)
* `LIMIT/OFFSET`: basic pagination

### NULL rules (big learning point)

* `NULL` = “unknown”
* Use `IS NULL` / `IS NOT NULL` (not `= NULL`)

## Tasks (you do)

Create `sql/02_select_basics.sql` and write:

1. Query returning newest 20 users
2. Query filtering by email domain
3. Query that finds rows where a column is NULL

## Check yourself (reference + explanation)

Example pattern:

```sql
SELECT id, email, created_at
FROM users
WHERE email LIKE '%@example.com'
ORDER BY created_at DESC
LIMIT 20 OFFSET 0;
```

**Why `LIKE` here:** simple pattern matching.
**Why `ORDER BY` before pagination:** without it, “page 1” is not stable across runs.

---

# Phase 2 — Schema design (DDL): tables + constraints that prevent bad data

## Goal

You can model tables and enforce correctness at the database layer.

## Tools

* Still `psql` via `docker exec`, because schema work is easiest when repeatable.

## Core syntax (breakdown)

### CREATE TABLE

Defines structure + rules:

* columns: `name TYPE CONSTRAINTS`
* table constraints: `PRIMARY KEY`, `FOREIGN KEY`, etc.

### Constraints (purpose)

* `PRIMARY KEY`: stable identity per row
* `UNIQUE`: prevents duplicates (emails, SKUs)
* `NOT NULL`: mandatory values
* `CHECK`: domain rules (price >= 0)
* `FOREIGN KEY`: relational integrity (orders must reference real users)

## Tasks (you do)

Create `sql/01_schema.sql` with:

1. `users`
2. `products`
3. `orders`
4. `order_items`
5. `payments`

### “Why this matters”

Without constraints, bugs silently write invalid data and you only discover it later in reports.

---

# Phase 3 — INSERT / UPDATE / DELETE + seeding (DML)

## Goal

You can create realistic dev data and modify it safely.

## Syntax breakdown

### INSERT

```sql
INSERT INTO users (email) VALUES ('a@x.com');
```

**Why list columns:** schema can evolve without breaking inserts.

### Multi-row INSERT (faster seeding)

```sql
INSERT INTO products (sku, name, price_cents) VALUES
('sku_1','Apple',199),
('sku_2','Banana',99);
```

### UPDATE (always qualify)

```sql
UPDATE orders
SET status = 'paid'
WHERE id = 123 AND status = 'pending';
```

**Why include current status:** avoids accidental double transitions.

### DELETE (always qualify)

```sql
DELETE FROM products WHERE id = 5;
```

## Tasks (you do)

Create `sql/03_seed.sql`:

1. Insert 10 users
2. Insert 20 products
3. Create 30 orders, each with 1–4 items
4. Add one payment per paid order

### Optional: container init seeding

Postgres official image can run `*.sql` in `/docker-entrypoint-initdb.d` on first init. ([Docker Hub][3])
**Purpose:** automatic bootstrapping for a brand-new volume.

---

# Phase 4 — JOINs (how relational data becomes useful)

## Goal

You can combine tables to answer “real” questions.

## Join syntax (breakdown)

```sql
SELECT ...
FROM orders o
JOIN users u ON u.id = o.user_id;
```

* `JOIN` = combine matching rows
* `ON` = match rule
* `aliases` (`o`, `u`) = readability when queries grow

### JOIN types (purpose)

* `INNER JOIN` (`JOIN`) → only rows that match
* `LEFT JOIN` → keep all left rows even if no match (great for “users with zero orders”)

## Tasks (you do)

Create `sql/04_joins.sql`:

1. List last 50 orders with user email
2. List products per order (order → items → products)
3. List all users with their order counts (including 0) using LEFT JOIN

---

# Phase 5 — Aggregation (GROUP BY, HAVING) for reporting

## Goal

You can summarize large datasets into business metrics.

## Syntax breakdown

### GROUP BY

```sql
SELECT user_id, COUNT(*) AS orders
FROM orders
GROUP BY user_id;
```

* aggregate funcs: `COUNT`, `SUM`, `AVG`, `MIN`, `MAX`
* `GROUP BY` defines the buckets (“per user”, “per day”)

### HAVING vs WHERE (important)

* `WHERE` filters rows *before* grouping
* `HAVING` filters groups *after* aggregation

## Tasks (you do)

Create `sql/05_reports.sql`:

1. Revenue per day
2. Users with 3+ orders (HAVING)
3. Top 10 products by quantity sold

---

# Phase 6 — Advanced query tools (CTEs, window functions, views)

## Goal

You can write complex queries cleanly and do analytics-style ranking.

## CTE (WITH) — purpose

CTEs let you name intermediate result sets so your query stays readable.

```sql
WITH paid_orders AS (
  SELECT o.id, o.user_id, p.amount_cents, p.paid_at
  FROM orders o
  JOIN payments p ON p.order_id = o.id
)
SELECT user_id, SUM(amount_cents)
FROM paid_orders
GROUP BY user_id;
```

## Window functions — purpose

“Aggregate-like” calculations **without collapsing rows** (ranking, running totals).

Example goal: rank products by sales quantity.

## Views — purpose

Give reports a stable interface (`SELECT * FROM v_daily_revenue`) without duplicating query logic.

---

# Phase 7 — Indexes + EXPLAIN (performance and “why slow?”)

## Goal

You can diagnose slow queries and fix them correctly.

## Tools

* `EXPLAIN ANALYZE` in Postgres
* `psql \timing on`

## Concepts (purpose)

* Indexes speed up filtering and joins
* Composite indexes help multi-column filters/sorts
* `EXPLAIN ANALYZE` shows the plan + actual timing

---

# Phase 8 — Migrations with `golang-migrate/migrate` (your chosen tool)

## Goal

Your schema is versioned and repeatable: “fresh DB → migrate up → seed”.

## Tool purpose

`migrate` applies ordered `.up.sql` and `.down.sql` migrations to a database. ([GitHub][1])

## CLI workflow (tasks you do)

1. Create migration files:

   * `migrate create -ext sql -dir db/migrations -seq init_schema` ([FreeCodeCamp][4])
2. Put schema in `*_up.sql`, teardown in `*_down.sql`
3. Run:

   * `migrate -path db/migrations -database "$POSTGRES_URL" up`

### Critical note about SQLite in-memory + migrate CLI

SQLite **in-memory databases only exist inside the current process**; a CLI tool in a separate process can’t “migrate” the same in-memory DB you’re using in your app. SQLite’s shared-cache in-memory mode is still “same process only.” ([SQLite][5])

**So the practical approach is:**

* Use `migrate` CLI for **Postgres** (container DB)
* For **SQLite in-memory** (dev/test), run schema/migrations **inside your Go program** (Phase 9), or execute a `schema.sql` on startup.

---

# Phase 9 — Go integration (pgxpool for Postgres, database/sql for SQLite in-memory)

## Goal

You can run the same *business tasks* against Postgres (real) and SQLite (fast dev/test).

## Postgres in Go: `pgxpool`

**Purpose:** connection pooling + efficient Postgres access. ([Go Packages][6])

**Tasks (you do)**

1. Create a Go program that:

   * reads `POSTGRES_URL`
   * connects via `pgxpool`
   * runs a simple query (`SELECT now()` and `SELECT count(*) FROM users`)
2. Add a repository function:

   * `ListRecentOrders(ctx, limit int)`

## SQLite in Go: `database/sql` + pure-Go driver

Use `modernc.org/sqlite` (pure Go). ([Go Packages][7])

**In-memory DSN to learn**

* `file:memdb1?mode=memory&cache=shared`

  * lets multiple connections in the **same process** share the same in-memory DB ([SQLite][5])

**Tasks (you do)**

1. Add a `DB_KIND=sqlite|postgres` switch
2. When sqlite:

   * `sql.Open("sqlite", "file:memdb1?mode=memory&cache=shared")`
   * run schema setup at startup
3. Run the same “list orders” query against SQLite

### Placeholder difference (important learning point)

* Postgres uses `$1, $2…`
* SQLite often uses `?`

**Purpose:** parameterization prevents SQL injection and helps DB planning.
**Practical learning approach:** keep two versions of a query early on (one Postgres, one SQLite) so you *see* the dialect differences.

---

# Phase 10 — Testing strategy (fast + realistic)

## Goal

You have confidence without slow tests.

**Two layers**

1. **SQLite in-memory tests**: fast, great for CRUD and basic query logic
2. **Postgres integration tests** (using your container): validates Postgres-specific behavior and real query plans

This mirrors how many Go teams work in practice.

---

## Your “learning checklist” (what you should be able to explain by the end)

* Why `WHERE` vs `HAVING`
* When to use `LEFT JOIN`
* What a foreign key prevents
* What a transaction guarantees
* Why indexes speed some queries but slow some writes
* Why `migrate` CLI can’t operate on your app’s SQLite in-memory DB ([SQLite][5])
* How `pgxpool` differs from `database/sql` usage ([Go Packages][6])

---

## Next step (so this becomes a *doable* workbook)

If you want, I’ll generate the **exact task files** you’ll fill in (empty templates + prompts), like:

* `db/migrations/000001_init_schema.up.sql` (with TODOs)
* `sql/04_joins.sql` (exercise list)
* `cmd/playground/main.go` (skeleton with TODOs for pgxpool + sqlite)

Tell me what you want your container service name to be (`postgres`, `db`, etc.) and whether you prefer `Makefile` targets for common commands (migrate up/down, psql exec, seed).

[1]: https://github.com/golang-migrate/migrate?utm_source=chatgpt.com "Database migrations. CLI and Golang library."
[2]: https://www.datacamp.com/tutorial/postgresql-docker?utm_source=chatgpt.com "PostgreSQL in Docker: A Step-by-Step Guide for Beginners"
[3]: https://hub.docker.com/_/postgres?utm_source=chatgpt.com "postgres - Official Image"
[4]: https://www.freecodecamp.org/news/database-migration-golang-migrate/?utm_source=chatgpt.com "How to Perform Database Migrations using Go Migrate"
[5]: https://www.sqlite.org/inmemorydb.html?utm_source=chatgpt.com "In-Memory Databases"
[6]: https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool?utm_source=chatgpt.com "pgxpool package - github.com/jackc/pgx/v5 ..."
[7]: https://pkg.go.dev/modernc.org/sqlite?utm_source=chatgpt.com "sqlite package - modernc.org/sqlite"
